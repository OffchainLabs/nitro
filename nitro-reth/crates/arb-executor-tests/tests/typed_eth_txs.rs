use std::sync::Arc;

use alloy_eips::eip2930::{AccessList, AccessListItem};
use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{Address, Bytes, TxKind, B256, U256};
use arb_executor_tests::helpers::{
    alice, alice_key, balance_of, fund_account, nonce_of, recover, sign_1559, sign_2930, ONE_ETH,
    ONE_GWEI, RECIPIENT,
};
use arb_evm::config::ArbEvmConfig;
use arb_test_utils::ArbosHarness;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::primitives::hardfork::SpecId;

fn execute_in_fresh_block(
    harness: &mut ArbosHarness,
    base_fee: u64,
    chain_id: u64,
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
    env.block_env.timestamp = U256::from(1_700_000_000u64);
    env.block_env.basefee = base_fee;
    env.block_env.gas_limit = 30_000_000;
    env.block_env.number = U256::from(1u64);

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
    executor
        .apply_pre_execution_changes()
        .map_err(|e| format!("pre-exec: {e}"))?;

    let recovered = recover(tx);
    let result = executor
        .execute_transaction_without_commit(recovered)
        .map_err(|e| format!("execute: {e}"))?;
    let success = result.result.result.is_success();
    executor
        .commit_transaction(result)
        .map_err(|e| format!("commit: {e}"))?;
    let _ = executor.finish().map_err(|e| format!("finish: {e}"))?;
    Ok(success)
}

#[test]
fn eip1559_transfer_credits_recipient_and_increments_nonce() {
    let mut s = arb_executor_tests::helpers::ExecutorScaffold::new();
    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));

    let send_value = U256::from(ONE_ETH);
    let tx = sign_1559(
        s.chain_id,
        0,
        2 * ONE_GWEI,
        ONE_GWEI,
        21_000,
        TxKind::Call(RECIPIENT),
        send_value,
        Bytes::new(),
        alice_key(),
    );
    let success =
        execute_in_fresh_block(&mut s.harness, s.base_fee, s.chain_id, tx).expect("execute");
    assert!(success);
    assert_eq!(balance_of(s.harness.state(), RECIPIENT), send_value);
    assert_eq!(nonce_of(s.harness.state(), alice()), 1);
}

#[test]
fn eip1559_with_priority_fee_drops_tip_per_arbitrum_semantics() {
    let mut s = arb_executor_tests::helpers::ExecutorScaffold::new();
    let initial = U256::from(10u128 * ONE_ETH);
    fund_account(s.harness.state(), alice(), initial);

    let send_value = U256::from(ONE_ETH);
    let max_fee = 5 * ONE_GWEI;
    let max_priority = 4 * ONE_GWEI;
    let tx = sign_1559(
        s.chain_id,
        0,
        max_fee,
        max_priority,
        21_000,
        TxKind::Call(RECIPIENT),
        send_value,
        Bytes::new(),
        alice_key(),
    );
    execute_in_fresh_block(&mut s.harness, s.base_fee, s.chain_id, tx).expect("execute");

    let alice_after = balance_of(s.harness.state(), alice());
    let max_paid = U256::from(21_000u64) * U256::from(max_fee);
    let min_paid = U256::from(21_000u64) * U256::from(s.base_fee);
    let alice_decrease = initial - send_value - alice_after;
    assert!(
        alice_decrease >= min_paid,
        "alice should have paid at least base_fee*gas: paid {alice_decrease}, min {min_paid}"
    );
    assert!(
        alice_decrease < max_paid,
        "Arbitrum drops tip — alice should not pay full max_fee*gas: paid {alice_decrease}, max {max_paid}"
    );
}

#[test]
fn eip2930_with_access_list_executes() {
    let mut s = arb_executor_tests::helpers::ExecutorScaffold::new();
    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));

    let access_list = AccessList(vec![AccessListItem {
        address: Address::repeat_byte(0xAB),
        storage_keys: vec![B256::repeat_byte(0xCD), B256::repeat_byte(0xEF)],
    }]);

    let send_value = U256::from(ONE_ETH / 2);
    let tx = sign_2930(
        s.chain_id,
        0,
        ONE_GWEI,
        100_000,
        TxKind::Call(RECIPIENT),
        send_value,
        Bytes::new(),
        access_list,
        alice_key(),
    );
    let success =
        execute_in_fresh_block(&mut s.harness, s.base_fee, s.chain_id, tx).expect("execute");
    assert!(success);
    assert_eq!(balance_of(s.harness.state(), RECIPIENT), send_value);
}

#[test]
fn eip2930_empty_access_list_executes_like_legacy() {
    let mut s = arb_executor_tests::helpers::ExecutorScaffold::new();
    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));

    let send_value = U256::from(ONE_ETH);
    let tx = sign_2930(
        s.chain_id,
        0,
        ONE_GWEI,
        21_000,
        TxKind::Call(RECIPIENT),
        send_value,
        Bytes::new(),
        AccessList::default(),
        alice_key(),
    );
    let success =
        execute_in_fresh_block(&mut s.harness, s.base_fee, s.chain_id, tx).expect("execute");
    assert!(success);
    assert_eq!(balance_of(s.harness.state(), RECIPIENT), send_value);
}

#[test]
fn eip1559_zero_value_with_calldata_executes() {
    let mut s = arb_executor_tests::helpers::ExecutorScaffold::new();
    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));

    let tx = sign_1559(
        s.chain_id,
        0,
        2 * ONE_GWEI,
        0,
        100_000,
        TxKind::Call(RECIPIENT),
        U256::ZERO,
        Bytes::from(vec![0xAA, 0xBB, 0xCC]),
        alice_key(),
    );
    let success =
        execute_in_fresh_block(&mut s.harness, s.base_fee, s.chain_id, tx).expect("execute");
    assert!(success);
    assert_eq!(nonce_of(s.harness.state(), alice()), 1);
}
