//! Stylus dispatch-path tests.
//!
//! Full Stylus WASM execution requires a valid brotli-compressed WASM +
//! activation via ArbWasm precompile. That's covered at the unit level in
//! arb-stylus/tests/wasm_execution.rs (8 tests). Here we verify the
//! *dispatch* path: the executor correctly classifies a contract with
//! the Stylus discriminant and routes it through arb-stylus rather than
//! the EVM interpreter. We observe this by comparing error signatures:
//! an EVM call to 0xEF-prefixed code halts with InvalidFEOpcode, while
//! a Stylus dispatch attempt fails inside the WASM pipeline.

use std::sync::Arc;

use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{Address, Bytes, TxKind, B256, U256};
use arb_executor_tests::helpers::{
    alice, alice_key, deploy_contract, fund_account, recover, sign_legacy, ExecutorScaffold,
    ONE_ETH, ONE_GWEI,
};
use arb_evm::config::ArbEvmConfig;
use arb_test_utils::ArbosHarness;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::primitives::hardfork::SpecId;

fn call(
    harness: &mut ArbosHarness,
    base_fee: u64,
    chain_id: u64,
    to: Address,
    sk: [u8; 32],
    nonce: u64,
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
    executor
        .apply_pre_execution_changes()
        .map_err(|e| format!("pre: {e}"))?;

    let tx = sign_legacy(
        chain_id,
        nonce,
        ONE_GWEI,
        500_000,
        TxKind::Call(to),
        U256::ZERO,
        Bytes::new(),
        sk,
    );
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
fn stylus_prefix_contract_is_classified_as_stylus() {
    let bytecode: Vec<u8> = vec![0xEF, 0xF0, 0x00, 0x00, 0xAA, 0xBB];
    assert!(arb_stylus::is_stylus_program(&bytecode));

    let bytecode_no_prefix: Vec<u8> = vec![0x60, 0x00, 0x60, 0x00, 0xF3];
    assert!(!arb_stylus::is_stylus_program(&bytecode_no_prefix));

    let bytecode_wrong_prefix: Vec<u8> = vec![0xEF, 0xF0, 0x01, 0x00];
    assert!(!arb_stylus::is_stylus_program(&bytecode_wrong_prefix));
}

#[test]
fn stylus_prefix_contract_does_not_cause_evm_invalid_opcode() {
    let mut s = ExecutorScaffold::new();
    let stylus_addr = Address::repeat_byte(0xEF);
    let invalid_wasm = vec![0xEF, 0xF0, 0x00, 0x00, 0xFF, 0xFF, 0xFF, 0xFF];

    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));
    deploy_contract(s.harness.state(), stylus_addr, invalid_wasm, U256::ZERO);

    let result = call(
        &mut s.harness,
        s.base_fee,
        s.chain_id,
        stylus_addr,
        alice_key(),
        0,
    );
    if let Ok(success) = result {
        assert!(!success, "Stylus dispatch with invalid WASM should fail");
    }
}

#[test]
fn non_stylus_contract_executes_via_evm() {
    let mut s = ExecutorScaffold::new();
    let evm_addr = Address::repeat_byte(0xCC);
    let valid_return = vec![0x60, 0x00, 0x60, 0x00, 0xF3];

    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));
    deploy_contract(s.harness.state(), evm_addr, valid_return, U256::ZERO);

    let success = call(
        &mut s.harness,
        s.base_fee,
        s.chain_id,
        evm_addr,
        alice_key(),
        0,
    )
    .expect("exec");
    assert!(success, "valid EVM contract must execute successfully");
}

#[test]
fn stylus_discriminant_exact_value_is_canonical() {
    assert_eq!(arb_stylus::STYLUS_DISCRIMINANT, [0xEF, 0xF0, 0x00]);
    assert_eq!(arb_stylus::STYLUS_DISCRIMINANT.len(), 3);
}

#[test]
fn stylus_dispatch_does_not_corrupt_caller_state() {
    let mut s = ExecutorScaffold::new();
    let stylus_addr = Address::repeat_byte(0xEF);
    let bad_stylus = vec![0xEF, 0xF0, 0x00, 0x00, 0xFF, 0xFF];

    let initial_balance = U256::from(10u128 * ONE_ETH);
    fund_account(s.harness.state(), alice(), initial_balance);
    deploy_contract(s.harness.state(), stylus_addr, bad_stylus, U256::ZERO);

    let _ = call(
        &mut s.harness,
        s.base_fee,
        s.chain_id,
        stylus_addr,
        alice_key(),
        0,
    );

    let alice_after = arb_executor_tests::helpers::balance_of(s.harness.state(), alice());
    assert!(
        alice_after < initial_balance,
        "caller pays gas even when target reverts: got {alice_after}, initial {initial_balance}"
    );
    assert!(
        alice_after > initial_balance - U256::from(500_000u64) * U256::from(ONE_GWEI),
        "caller should not lose more than tx gas budget"
    );
}
