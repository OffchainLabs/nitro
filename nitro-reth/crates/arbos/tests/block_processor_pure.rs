use alloy_primitives::{Address, B256, U256};
use arbos::block_processor::{
    create_new_header, finalize_block_header_info, BlockProductionState, NoopSequencingHooks,
    SequencingHooks, TxAction, TxOutcome, TxResult,
};
use arbos::internal_tx::L1Info;

fn l1_info(ts: u64, poster_byte: u8) -> L1Info {
    L1Info {
        poster: Address::repeat_byte(poster_byte),
        l1_block_number: 100,
        l1_timestamp: ts,
    }
}

#[test]
fn create_new_header_none_l1_info_uses_zero_coinbase_and_parent_ts() {
    let r = create_new_header(
        None,
        B256::repeat_byte(0xAA),
        42,
        1_700_000_000,
        &[],
        B256::ZERO,
        U256::from(1_000_000_000u64),
    );
    assert_eq!(r.coinbase, Address::ZERO);
    assert_eq!(r.timestamp, 1_700_000_000);
    assert_eq!(r.number, 43);
    assert_eq!(r.extra_data.len(), 32);
    assert_eq!(r.difficulty, U256::from(1));
    assert_eq!(r.base_fee, U256::from(1_000_000_000u64));
}

#[test]
fn create_new_header_uses_l1_info_when_newer_timestamp() {
    let info = l1_info(1_700_000_500, 0xCC);
    let r = create_new_header(
        Some(&info),
        B256::ZERO,
        10,
        1_700_000_000,
        &[],
        B256::ZERO,
        U256::ZERO,
    );
    assert_eq!(r.timestamp, 1_700_000_500);
    assert_eq!(r.coinbase, Address::repeat_byte(0xCC));
}

#[test]
fn create_new_header_clamps_timestamp_to_parent_if_l1_older() {
    let info = l1_info(1_000_000, 0xCC);
    let r = create_new_header(Some(&info), B256::ZERO, 5, 9_999_999, &[], B256::ZERO, U256::ZERO);
    assert_eq!(r.timestamp, 9_999_999);
}

#[test]
fn create_new_header_truncates_parent_extra_to_32_bytes() {
    let mut prev = vec![0u8; 64];
    prev[0..32].copy_from_slice(&[0xAB; 32]);
    prev[32..64].copy_from_slice(&[0xEE; 32]);
    let r = create_new_header(None, B256::ZERO, 0, 0, &prev, B256::ZERO, U256::ZERO);
    assert_eq!(r.extra_data.len(), 32);
    assert_eq!(r.extra_data, vec![0xAB; 32]);
}

#[test]
fn create_new_header_pads_short_parent_extra_with_zeros() {
    let r = create_new_header(None, B256::ZERO, 0, 0, &[0x01, 0x02], B256::ZERO, U256::ZERO);
    let mut expected = vec![0u8; 32];
    expected[0] = 0x01;
    expected[1] = 0x02;
    assert_eq!(r.extra_data, expected);
}

#[test]
fn finalize_block_header_info_preserves_all_fields() {
    let info = finalize_block_header_info(B256::repeat_byte(0xAB), 7, 100, 30);
    assert_eq!(info.send_root, B256::repeat_byte(0xAB));
    assert_eq!(info.send_count, 7);
    assert_eq!(info.l1_block_number, 100);
    assert_eq!(info.arbos_format_version, 30);
}

#[test]
fn compute_data_gas_base_fee_zero_returns_zero() {
    assert_eq!(
        BlockProductionState::compute_data_gas(U256::from(1000u64), U256::ZERO, 21_000),
        0
    );
}

#[test]
fn compute_data_gas_clamps_at_tx_gas_limit() {
    let data_gas = BlockProductionState::compute_data_gas(
        U256::from(1_000_000u64),
        U256::from(1u64),
        100_000,
    );
    assert_eq!(data_gas, 100_000);
}

#[test]
fn compute_data_gas_divides_poster_by_base_fee() {
    let data_gas =
        BlockProductionState::compute_data_gas(U256::from(10_000u64), U256::from(10u64), 50_000);
    assert_eq!(data_gas, 1_000);
}

#[test]
fn production_state_first_action_is_start_block() {
    let mut s = BlockProductionState::new(30_000_000, 30, 1_700_000_000, U256::from(1));
    let mut hooks = NoopSequencingHooks;
    assert!(matches!(s.next_tx_action(&mut hooks), TxAction::ExecuteStartBlock));
}

#[test]
fn production_state_after_start_block_drains_sequencer() {
    let mut s = BlockProductionState::new(30_000_000, 30, 0, U256::from(1));
    let mut hooks = NoopSequencingHooks;
    let _ = s.next_tx_action(&mut hooks);
    assert!(matches!(s.next_tx_action(&mut hooks), TxAction::Done));
}

struct OneTx(Option<Vec<u8>>);
impl SequencingHooks for OneTx {
    fn next_tx_to_sequence(&mut self) -> Option<Vec<u8>> {
        self.0.take()
    }
    fn pre_tx_filter(&self, _: &[u8]) -> Result<(), String> {
        Ok(())
    }
    fn post_tx_filter(&self, _: &[u8], _: &[u8]) -> Result<(), String> {
        Ok(())
    }
    fn discard_invalid_txs_early(&self) -> bool {
        false
    }
}

#[test]
fn production_state_returns_user_tx_from_hooks() {
    let mut s = BlockProductionState::new(30_000_000, 30, 0, U256::from(1));
    let mut hooks = OneTx(Some(vec![0xAB, 0xCD]));
    let _ = s.next_tx_action(&mut hooks);
    match s.next_tx_action(&mut hooks) {
        TxAction::ExecuteUserTx(bytes) => assert_eq!(bytes, vec![0xAB, 0xCD]),
        other => panic!("expected user tx, got {other:?}"),
    }
}

#[test]
fn production_state_blocks_user_tx_when_gas_exhausted() {
    let mut s = BlockProductionState::new(10_000, 30, 0, U256::from(1));
    let mut hooks = OneTx(Some(vec![0xAB]));
    let _ = s.next_tx_action(&mut hooks);
    assert!(matches!(s.next_tx_action(&mut hooks), TxAction::Done));
}

#[test]
fn production_state_queues_redeems_fifo() {
    let mut s = BlockProductionState::new(30_000_000, 30, 0, U256::from(1));
    let mut hooks = OneTx(Some(vec![0xAB]));
    let _ = s.next_tx_action(&mut hooks);
    let _ = s.next_tx_action(&mut hooks);
    s.record_tx_outcome(
        &TxAction::ExecuteUserTx(vec![0xAB]),
        TxOutcome::Success(TxResult {
            gas_used: 25_000,
            data_gas: 4_000,
            evm_success: true,
            scheduled_txs: vec![vec![0xCC], vec![0xDD]],
            evm_error: None,
        }),
    )
    .expect("record");
    let mut noop = NoopSequencingHooks;
    match s.next_tx_action(&mut noop) {
        TxAction::ExecuteRedeem(b) => assert_eq!(b, vec![0xCC]),
        other => panic!("expected redeem, got {other:?}"),
    }
    match s.next_tx_action(&mut noop) {
        TxAction::ExecuteRedeem(b) => assert_eq!(b, vec![0xDD]),
        other => panic!("expected redeem, got {other:?}"),
    }
}

#[test]
fn production_state_rejects_for_block_gas_only_in_old_arbos() {
    let s_old = BlockProductionState::new(10_000, 30, 0, U256::from(1));
    let s_new = BlockProductionState::new(10_000, 50, 0, U256::from(1));
    let big = 1_000_000u64;
    assert!(!s_old.should_reject_for_block_gas(big, true));
    let mut s_old_after = BlockProductionState::new(10_000, 30, 0, U256::from(1));
    s_old_after
        .record_tx_outcome(
            &TxAction::ExecuteUserTx(vec![0xAB]),
            TxOutcome::Success(TxResult {
                gas_used: 21_000,
                data_gas: 0,
                evm_success: true,
                scheduled_txs: vec![],
                evm_error: None,
            }),
        )
        .unwrap();
    assert!(s_old_after.should_reject_for_block_gas(big, true));
    assert!(!s_new.should_reject_for_block_gas(big, true));
    assert!(!s_old_after.should_reject_for_block_gas(big, false));
}

#[test]
fn production_state_internal_tx_error_returns_err() {
    let mut s = BlockProductionState::new(30_000_000, 30, 0, U256::from(1));
    let err = s.record_tx_outcome(
        &TxAction::ExecuteStartBlock,
        TxOutcome::Success(TxResult {
            gas_used: 1000,
            data_gas: 0,
            evm_success: false,
            scheduled_txs: vec![],
            evm_error: Some("boom".to_string()),
        }),
    );
    assert!(err.is_err());
}

#[test]
fn production_state_invalid_user_tx_counts_as_user_tx_processed() {
    let mut s = BlockProductionState::new(30_000_000, 30, 0, U256::from(1));
    s.record_tx_outcome(
        &TxAction::ExecuteUserTx(vec![0xAB]),
        TxOutcome::Invalid("bad".into()),
    )
    .unwrap();
    assert_eq!(s.user_txs_processed(), 1);
    assert_eq!(s.block_gas_left, 30_000_000 - 21_000);
}

#[test]
fn production_state_track_deposits_and_withdrawals() {
    let mut s = BlockProductionState::new(30_000_000, 30, 0, U256::from(1));
    s.track_deposit(U256::from(1_000u64));
    s.track_deposit(U256::from(500u64));
    s.track_withdrawal(U256::from(200u64));
    assert_eq!(s.expected_balance_delta, 1_300);
}

#[test]
fn production_state_verify_balance_delta_exact_match() {
    let mut s = BlockProductionState::new(30_000_000, 30, 0, U256::from(1));
    s.track_deposit(U256::from(100u64));
    assert!(s.verify_balance_delta(100, false).is_ok());
}

#[test]
fn production_state_verify_balance_delta_excess_is_error() {
    let s = BlockProductionState::new(30_000_000, 30, 0, U256::from(1));
    assert!(s.verify_balance_delta(500, false).is_err());
}

#[test]
fn production_state_verify_balance_delta_burn_ok_except_in_debug() {
    let mut s = BlockProductionState::new(30_000_000, 30, 0, U256::from(1));
    s.track_deposit(U256::from(1_000u64));
    assert!(s.verify_balance_delta(500, false).is_ok());
    assert!(s.verify_balance_delta(500, true).is_err());
}

#[test]
fn production_state_set_arbos_version_updates_gate() {
    let mut s = BlockProductionState::new(10_000, 30, 0, U256::from(1));
    s.set_arbos_version(50);
    assert_eq!(s.arbos_version(), 50);
    assert!(!s.should_reject_for_block_gas(1_000_000, true));
}
