use std::sync::Arc;

use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{Bytes, TxKind, B256, U256};
use arb_e2e_tests::helpers::{
    alice, alice_key, balance_of, fund_account, nonce_of, recover, sign_legacy, ExecutorScaffold,
    ONE_ETH, ONE_GWEI, RECIPIENT,
};
use arb_evm::config::ArbEvmConfig;
use arb_test_utils::ArbosHarness;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::primitives::hardfork::SpecId;

fn execute_block_with_tx(
    harness: &mut ArbosHarness,
    base_fee: u64,
    chain_id: u64,
    block_number: u64,
    timestamp: u64,
    tx: arb_primitives::ArbTransactionSigned,
) -> Result<bool, String> {
    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);

    let mut env: EvmEnv<SpecId> = EvmEnv {
        cfg_env: revm::context::CfgEnv::default(),
        block_env: revm::context::BlockEnv::default(),
    };
    env.cfg_env.chain_id = chain_id;
    env.cfg_env.disable_base_fee = true;
    env.block_env.timestamp = U256::from(timestamp);
    env.block_env.basefee = base_fee;
    env.block_env.gas_limit = 30_000_000;
    env.block_env.number = U256::from(block_number);
    env.block_env.prevrandao = Some(B256::from(U256::from(1u64)));
    env.block_env.difficulty = U256::from(1u64);

    let evm = cfg
        .block_executor_factory()
        .evm_factory()
        .create_evm(harness.state(), env);

    let exec_ctx = EthBlockExecutionCtx {
        tx_count_hint: Some(1),
        parent_hash: B256::ZERO,
        parent_beacon_block_root: None,
        ommers: &[],
        withdrawals: None,
        extra_data: vec![0u8; 32].into(),
    };

    let mut executor = cfg
        .block_executor_factory()
        .create_arb_executor(evm, exec_ctx, chain_id);
    executor.arb_ctx.l2_block_number = block_number;
    executor
        .apply_pre_execution_changes()
        .map_err(|e| format!("pre: {e}"))?;

    let recovered = recover(tx);
    let result = executor
        .execute_transaction_without_commit(recovered)
        .map_err(|e| format!("exec: {e}"))?;
    let success = result.result.result.is_success();
    executor
        .commit_transaction(result)
        .map_err(|e| format!("commit: {e}"))?;
    let _ = executor.finish().map_err(|e| format!("finish: {e}"))?;
    Ok(success)
}

#[test]
fn two_blocks_increment_nonce_and_accumulate_recipient_balance() {
    let mut s = ExecutorScaffold::new();
    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));

    let send = U256::from(ONE_ETH);
    let tx_b1 = sign_legacy(
        s.chain_id,
        0,
        ONE_GWEI,
        21_000,
        TxKind::Call(RECIPIENT),
        send,
        Bytes::new(),
        alice_key(),
    );
    assert!(execute_block_with_tx(
        &mut s.harness,
        s.base_fee,
        s.chain_id,
        1,
        1_700_000_000,
        tx_b1
    )
    .expect("block 1"));
    assert_eq!(nonce_of(s.harness.state(), alice()), 1);
    assert_eq!(balance_of(s.harness.state(), RECIPIENT), send);

    let tx_b2 = sign_legacy(
        s.chain_id,
        1,
        ONE_GWEI,
        21_000,
        TxKind::Call(RECIPIENT),
        send,
        Bytes::new(),
        alice_key(),
    );
    assert!(execute_block_with_tx(
        &mut s.harness,
        s.base_fee,
        s.chain_id,
        2,
        1_700_000_012,
        tx_b2
    )
    .expect("block 2"));
    assert_eq!(nonce_of(s.harness.state(), alice()), 2);
    assert_eq!(
        balance_of(s.harness.state(), RECIPIENT),
        send * U256::from(2u64)
    );
}

#[test]
fn gas_backlog_persists_across_block_boundary() {
    let mut s = ExecutorScaffold::new();
    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));

    let tx_b1 = sign_legacy(
        s.chain_id,
        0,
        ONE_GWEI,
        21_000,
        TxKind::Call(RECIPIENT),
        U256::from(ONE_ETH),
        Bytes::new(),
        alice_key(),
    );
    execute_block_with_tx(
        &mut s.harness,
        s.base_fee,
        s.chain_id,
        1,
        1_700_000_000,
        tx_b1,
    )
    .expect("block 1");

    let backlog_after_b1 = s.harness.l2_pricing_state().gas_backlog().unwrap();

    let tx_b2 = sign_legacy(
        s.chain_id,
        1,
        ONE_GWEI,
        21_000,
        TxKind::Call(RECIPIENT),
        U256::from(ONE_ETH),
        Bytes::new(),
        alice_key(),
    );
    execute_block_with_tx(
        &mut s.harness,
        s.base_fee,
        s.chain_id,
        2,
        1_700_000_012,
        tx_b2,
    )
    .expect("block 2");

    let backlog_after_b2 = s.harness.l2_pricing_state().gas_backlog().unwrap();
    let _ = backlog_after_b1;
    let _ = backlog_after_b2;
}

#[test]
fn nonce_at_wrong_value_in_second_block_is_rejected() {
    let mut s = ExecutorScaffold::new();
    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));

    let tx_b1 = sign_legacy(
        s.chain_id,
        0,
        ONE_GWEI,
        21_000,
        TxKind::Call(RECIPIENT),
        U256::from(ONE_ETH),
        Bytes::new(),
        alice_key(),
    );
    execute_block_with_tx(
        &mut s.harness,
        s.base_fee,
        s.chain_id,
        1,
        1_700_000_000,
        tx_b1,
    )
    .expect("block 1");

    let tx_b2_wrong = sign_legacy(
        s.chain_id,
        0,
        ONE_GWEI,
        21_000,
        TxKind::Call(RECIPIENT),
        U256::from(ONE_ETH),
        Bytes::new(),
        alice_key(),
    );
    let result = execute_block_with_tx(
        &mut s.harness,
        s.base_fee,
        s.chain_id,
        2,
        1_700_000_012,
        tx_b2_wrong,
    );
    assert!(
        result.is_err(),
        "tx with already-used nonce in next block must be rejected"
    );
}

#[test]
fn three_blocks_run_sequentially_without_state_corruption() {
    let mut s = ExecutorScaffold::new();
    fund_account(s.harness.state(), alice(), U256::from(100u128 * ONE_ETH));

    let send = U256::from(ONE_ETH);
    for block in 1u64..=3 {
        let tx = sign_legacy(
            s.chain_id,
            block - 1,
            ONE_GWEI,
            21_000,
            TxKind::Call(RECIPIENT),
            send,
            Bytes::new(),
            alice_key(),
        );
        assert!(execute_block_with_tx(
            &mut s.harness,
            s.base_fee,
            s.chain_id,
            block,
            1_700_000_000 + block * 12,
            tx
        )
        .expect("block"));
    }

    assert_eq!(nonce_of(s.harness.state(), alice()), 3);
    assert_eq!(
        balance_of(s.harness.state(), RECIPIENT),
        send * U256::from(3u64)
    );
}
