use alloy_primitives::{address, Address, B256, U256};
use arbos::tx_processor::{
    EndTxNormalParams, GasChargingError, GasChargingParams, RevertedTxAction, TxProcessor,
};

const ARBOS_V30: u64 = 30;
const ARBOS_V50: u64 = 50;
const ONE_GWEI: u64 = 1_000_000_000;
const BATCH_POSTER: Address = arbos::l1_pricing::BATCH_POSTER_ADDRESS;

fn default_charging_params() -> GasChargingParams {
    GasChargingParams {
        base_fee: U256::from(ONE_GWEI),
        poster_cost: U256::ZERO,
        is_gas_estimation: false,
        is_eth_call: false,
        skip_l1_charging: false,
        min_base_fee: U256::from(ONE_GWEI / 10),
        per_block_gas_limit: 32_000_000,
        per_tx_gas_limit: 32_000_000,
        arbos_version: ARBOS_V30,
    }
}

#[test]
fn new_tx_processor_with_batch_poster_coinbase_is_not_delayed_inbox() {
    let p = TxProcessor::new(BATCH_POSTER);
    assert!(!p.drop_tip(9));
}

#[test]
fn new_tx_processor_with_other_coinbase_is_delayed_inbox() {
    let p = TxProcessor::new(address!("00000000000000000000000000000000DEADBEEF"));
    assert!(p.drop_tip(9));
    assert!(p.drop_tip(11));
}

#[test]
fn drop_tip_version_gating() {
    let p = TxProcessor::new(BATCH_POSTER);
    assert!(p.drop_tip(0));
    assert!(p.drop_tip(8));
    assert!(!p.drop_tip(9));
    assert!(p.drop_tip(10));
    assert!(p.drop_tip(30));
}

#[test]
fn paid_gas_price_uses_base_fee_except_v9() {
    let p = TxProcessor::new(BATCH_POSTER);
    let bf = U256::from(100u64);
    let gp = U256::from(150u64);
    assert_eq!(p.get_paid_gas_price(0, bf, gp), bf);
    assert_eq!(p.get_paid_gas_price(8, bf, gp), bf);
    assert_eq!(p.get_paid_gas_price(9, bf, gp), gp);
    assert_eq!(p.get_paid_gas_price(10, bf, gp), bf);
    assert_eq!(p.get_paid_gas_price(30, bf, gp), bf);
}

#[test]
fn gas_price_op_pre_v3_returns_raw_price() {
    let p = TxProcessor::new(BATCH_POSTER);
    let bf = U256::from(100u64);
    let gp = U256::from(150u64);
    assert_eq!(p.gas_price_op(0, bf, gp), gp);
    assert_eq!(p.gas_price_op(2, bf, gp), gp);
    assert_eq!(p.gas_price_op(3, bf, gp), bf);
}

#[test]
fn push_pop_program_tracks_depth() {
    let mut p = TxProcessor::new(BATCH_POSTER);
    let addr = address!("AAAA000000000000000000000000000000000000");
    assert!(!p.is_reentrant(&addr));

    p.push_program(addr);
    assert!(!p.is_reentrant(&addr));
    p.push_program(addr);
    assert!(p.is_reentrant(&addr));
    p.push_program(addr);
    assert!(p.is_reentrant(&addr));

    p.pop_program(addr);
    assert!(p.is_reentrant(&addr));
    p.pop_program(addr);
    assert!(!p.is_reentrant(&addr));
    p.pop_program(addr);
    assert!(!p.is_reentrant(&addr));
}

#[test]
fn push_distinct_programs_dont_interfere() {
    let mut p = TxProcessor::new(BATCH_POSTER);
    let a = address!("AAAA000000000000000000000000000000000000");
    let b = address!("BBBB000000000000000000000000000000000000");
    p.push_program(a);
    p.push_program(b);
    assert!(!p.is_reentrant(&a));
    assert!(!p.is_reentrant(&b));
    p.push_program(a);
    assert!(p.is_reentrant(&a));
    assert!(!p.is_reentrant(&b));
}

#[test]
fn reverted_tx_hook_no_hash_returns_none() {
    let p = TxProcessor::new(BATCH_POSTER);
    assert_eq!(
        p.reverted_tx_hook(None, Some(50_000), false),
        RevertedTxAction::None
    );
    assert_eq!(p.reverted_tx_hook(None, None, true), RevertedTxAction::None);
}

#[test]
fn reverted_tx_hook_pre_recorded_returns_revert_with_adjusted_gas() {
    let p = TxProcessor::new(BATCH_POSTER);
    let action = p.reverted_tx_hook(Some(B256::repeat_byte(0xAB)), Some(50_000), false);
    assert_eq!(
        action,
        RevertedTxAction::PreRecordedRevert {
            gas_to_consume: 50_000 - 21_000,
        }
    );
}

#[test]
fn reverted_tx_hook_filtered_returns_filtered_action() {
    let p = TxProcessor::new(BATCH_POSTER);
    let action = p.reverted_tx_hook(Some(B256::repeat_byte(0xCD)), None, true);
    assert_eq!(action, RevertedTxAction::FilteredTx);
}

#[test]
fn gas_charging_zero_base_fee_skips_l1_cost() {
    let mut p = TxProcessor::new(BATCH_POSTER);
    let mut gas = 1_000_000u64;
    let mut params = default_charging_params();
    params.base_fee = U256::ZERO;
    params.poster_cost = U256::from(1_000_000u64);

    p.gas_charging_hook(&mut gas, 21_000, &params).unwrap();
    assert_eq!(p.nonrefundable_gas(), 0);
}

#[test]
fn gas_charging_skip_l1_charging_skips_poster_cost() {
    let mut p = TxProcessor::new(BATCH_POSTER);
    let mut gas = 1_000_000u64;
    let mut params = default_charging_params();
    params.poster_cost = U256::from(1_000_000u64);
    params.skip_l1_charging = true;

    p.gas_charging_hook(&mut gas, 21_000, &params).unwrap();
    assert_eq!(p.nonrefundable_gas(), 0);
}

#[test]
fn gas_charging_charges_poster_gas_when_cost_present() {
    let mut p = TxProcessor::new(BATCH_POSTER);
    let mut gas = 1_000_000u64;
    let mut params = default_charging_params();
    params.poster_cost = U256::from(100_000u64) * U256::from(ONE_GWEI);

    p.gas_charging_hook(&mut gas, 21_000, &params).unwrap();
    assert_eq!(p.nonrefundable_gas(), 100_000);
    assert_eq!(gas, 1_000_000 - 100_000);
}

#[test]
fn gas_charging_intrinsic_too_low_returns_error() {
    let mut p = TxProcessor::new(BATCH_POSTER);
    let mut gas = 50u64;
    let mut params = default_charging_params();
    params.poster_cost = U256::from(100_000u64) * U256::from(ONE_GWEI);

    let err = p.gas_charging_hook(&mut gas, 21_000, &params).unwrap_err();
    assert!(matches!(err, GasChargingError::IntrinsicGasTooLow));
}

#[test]
fn gas_charging_holds_compute_gas_above_per_block_limit() {
    let mut p = TxProcessor::new(BATCH_POSTER);
    let mut gas = 100_000_000u64;
    let mut params = default_charging_params();
    params.per_block_gas_limit = 32_000_000;

    p.gas_charging_hook(&mut gas, 21_000, &params).unwrap();
    assert_eq!(gas, 32_000_000);
    assert_eq!(p.held_gas(), 100_000_000 - 32_000_000);
}

#[test]
fn gas_charging_per_tx_limit_at_v50_uses_per_tx() {
    let mut p = TxProcessor::new(BATCH_POSTER);
    let mut gas = 100_000_000u64;
    let mut params = default_charging_params();
    params.arbos_version = ARBOS_V50;
    params.per_block_gas_limit = u64::MAX;
    params.per_tx_gas_limit = 1_000_000;
    let intrinsic = 21_000u64;

    p.gas_charging_hook(&mut gas, intrinsic, &params).unwrap();
    assert_eq!(gas, params.per_tx_gas_limit - intrinsic);
}

#[test]
fn gas_charging_eth_call_does_not_hold_gas() {
    let mut p = TxProcessor::new(BATCH_POSTER);
    let mut gas = 100_000_000u64;
    let mut params = default_charging_params();
    params.is_eth_call = true;
    params.per_block_gas_limit = 1_000;

    p.gas_charging_hook(&mut gas, 21_000, &params).unwrap();
    assert_eq!(gas, 100_000_000);
    assert_eq!(p.held_gas(), 0);
}

#[test]
fn end_tx_normal_distributes_total_to_network_when_no_infra() {
    let p = TxProcessor::new(BATCH_POSTER);
    let params = EndTxNormalParams {
        gas_used: 100_000,
        gas_price: U256::from(ONE_GWEI),
        base_fee: U256::from(ONE_GWEI),
        coinbase: BATCH_POSTER,
        network_fee_account: address!("00000000000000000000000000000000000A4B05"),
        infra_fee_account: Address::ZERO,
        min_base_fee: U256::from(ONE_GWEI / 2),
        arbos_version: ARBOS_V30,
    };
    let dist = p.compute_end_tx_fee_distribution(&params);
    assert_eq!(
        dist.network_fee_amount,
        U256::from(100_000u64) * U256::from(ONE_GWEI)
    );
    assert_eq!(dist.infra_fee_amount, U256::ZERO);
    assert_eq!(dist.poster_fee_amount, U256::ZERO);
}
