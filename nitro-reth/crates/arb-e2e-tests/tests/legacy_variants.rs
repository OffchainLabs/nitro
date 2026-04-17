use std::sync::Arc;

use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{Address, B256, Bytes, TxKind, U256};
use arb_evm::config::ArbEvmConfig;
use arb_e2e_tests::helpers::{
    alice, alice_key, balance_of, bob, deploy_contract, fund_account, nonce_of, recover,
    sign_legacy, ExecutorScaffold, ONE_ETH, ONE_GWEI, RECIPIENT,
};
use arb_test_utils::ArbosHarness;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::primitives::hardfork::SpecId;

fn execute_in_fresh_block(
    harness: &mut ArbosHarness,
    base_fee: u64,
    chain_id: u64,
    tx_builder: impl FnOnce(u64) -> arb_primitives::ArbTransactionSigned,
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

    let tx = tx_builder(chain_id);
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
fn legacy_transfer_credits_recipient() {
    let mut scaffold = ExecutorScaffold::new();
    fund_account(scaffold.harness.state(), alice(), U256::from(10u128 * ONE_ETH));

    let send_value = U256::from(ONE_ETH);
    let result = execute_in_fresh_block(
        &mut scaffold.harness,
        scaffold.base_fee,
        scaffold.chain_id,
        |chain_id| {
            sign_legacy(
                chain_id,
                0,
                ONE_GWEI,
                21_000,
                TxKind::Call(RECIPIENT),
                send_value,
                Bytes::new(),
                alice_key(),
            )
        },
    );
    assert!(result.is_ok() && result.unwrap());
    assert_eq!(balance_of(scaffold.harness.state(), RECIPIENT), send_value);
    assert_eq!(nonce_of(scaffold.harness.state(), alice()), 1);
}

const MIN_RUNTIME: &[u8] = &[0x60, 0x00, 0x60, 0x00, 0xF3];
const MIN_INIT: &[u8] = &[
    0x6c, 0x60, 0x00, 0x60, 0x00, 0xF3, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x60, 0x00, 0x52,
    0x60, 0x05, 0x60, 0x1B, 0xF3,
];

#[test]
fn legacy_contract_deployment_creates_account_with_code() {
    let mut scaffold = ExecutorScaffold::new();
    fund_account(scaffold.harness.state(), alice(), U256::from(10u128 * ONE_ETH));

    let result = execute_in_fresh_block(
        &mut scaffold.harness,
        scaffold.base_fee,
        scaffold.chain_id,
        |chain_id| {
            sign_legacy(
                chain_id,
                0,
                ONE_GWEI,
                500_000,
                TxKind::Create,
                U256::ZERO,
                MIN_INIT.to_vec().into(),
                alice_key(),
            )
        },
    );
    assert!(result.is_ok() && result.unwrap());
    assert_eq!(nonce_of(scaffold.harness.state(), alice()), 1);
}

const REVERT_RUNTIME: &[u8] = &[0x60, 0x00, 0x60, 0x00, 0xFD];

#[test]
fn legacy_call_to_revert_contract_returns_failure_status() {
    let mut scaffold = ExecutorScaffold::new();
    let revert_addr = Address::repeat_byte(0xCA);
    fund_account(scaffold.harness.state(), alice(), U256::from(10u128 * ONE_ETH));
    deploy_contract(
        scaffold.harness.state(),
        revert_addr,
        REVERT_RUNTIME.to_vec(),
        U256::ZERO,
    );

    let result = execute_in_fresh_block(
        &mut scaffold.harness,
        scaffold.base_fee,
        scaffold.chain_id,
        |chain_id| {
            sign_legacy(
                chain_id,
                0,
                ONE_GWEI,
                100_000,
                TxKind::Call(revert_addr),
                U256::ZERO,
                Bytes::new(),
                alice_key(),
            )
        },
    );
    let success = result.expect("execute ok");
    assert!(!success, "tx should report revert as failure");
    assert_eq!(nonce_of(scaffold.harness.state(), alice()), 1);
}

#[test]
fn legacy_call_to_consume_loop_runs_out_of_gas() {
    let mut scaffold = ExecutorScaffold::new();
    let consume_addr = Address::repeat_byte(0xCB);
    let consume_runtime: Vec<u8> = vec![0x5B, 0x60, 0x00, 0x56];

    fund_account(scaffold.harness.state(), alice(), U256::from(10u128 * ONE_ETH));
    deploy_contract(
        scaffold.harness.state(),
        consume_addr,
        consume_runtime,
        U256::ZERO,
    );

    let result = execute_in_fresh_block(
        &mut scaffold.harness,
        scaffold.base_fee,
        scaffold.chain_id,
        |chain_id| {
            sign_legacy(
                chain_id,
                0,
                ONE_GWEI,
                30_000,
                TxKind::Call(consume_addr),
                U256::ZERO,
                Bytes::new(),
                alice_key(),
            )
        },
    );
    let success = result.expect("execute ok");
    assert!(!success, "infinite loop must hit gas limit");
}

#[test]
fn legacy_transfer_with_insufficient_balance_fails() {
    let mut scaffold = ExecutorScaffold::new();
    fund_account(scaffold.harness.state(), bob(), U256::from(100u128));

    let send_value = U256::from(ONE_ETH);
    let result = execute_in_fresh_block(
        &mut scaffold.harness,
        scaffold.base_fee,
        scaffold.chain_id,
        |chain_id| {
            sign_legacy(
                chain_id,
                0,
                ONE_GWEI,
                21_000,
                TxKind::Call(RECIPIENT),
                send_value,
                Bytes::new(),
                arb_e2e_tests::helpers::bob_key(),
            )
        },
    );
    assert!(
        result.is_err(),
        "tx with insufficient balance must be rejected before execution"
    );
}
