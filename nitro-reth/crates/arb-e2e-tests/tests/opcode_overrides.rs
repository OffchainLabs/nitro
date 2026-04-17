use std::sync::Arc;

use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{Address, B256, Bytes, TxKind, U256};
use arb_e2e_tests::helpers::{
    alice, alice_key, deploy_contract, fund_account, recover, sign_legacy, ExecutorScaffold,
    ONE_ETH, ONE_GWEI,
};
use arb_evm::config::ArbEvmConfig;
use arb_test_utils::ArbosHarness;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::{
    context::result::{ExecutionResult, Output},
    primitives::hardfork::SpecId,
};

fn execute_call(
    harness: &mut ArbosHarness,
    base_fee: u64,
    chain_id: u64,
    to: Address,
    sk: [u8; 32],
    nonce: u64,
) -> Result<(bool, Bytes), String> {
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
    executor.apply_pre_execution_changes().map_err(|e| format!("pre: {e}"))?;

    let tx = sign_legacy(
        chain_id,
        nonce,
        ONE_GWEI,
        100_000,
        TxKind::Call(to),
        U256::ZERO,
        Bytes::new(),
        sk,
    );
    let recovered = recover(tx);

    let result = executor
        .execute_transaction_without_commit(recovered)
        .map_err(|e| format!("exec: {e}"))?;
    let exec_result = &result.result.result;
    let success = exec_result.is_success();
    let output = match exec_result {
        ExecutionResult::Success {
            output: Output::Call(b),
            ..
        } => b.clone(),
        _ => Bytes::new(),
    };
    executor.commit_transaction(result).map_err(|e| format!("commit: {e}"))?;
    let _ = executor.finish().map_err(|e| format!("finish: {e}"))?;
    Ok((success, output))
}

fn store_and_return_top() -> Vec<u8> {
    vec![0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xF3]
}

fn one_op_returner(opcode: u8) -> Vec<u8> {
    let mut code = vec![opcode];
    code.extend_from_slice(&store_and_return_top());
    code
}

const NUMBER_OPCODE: u8 = 0x43;
const TIMESTAMP_OPCODE: u8 = 0x42;
const COINBASE_OPCODE: u8 = 0x41;
const CHAINID_OPCODE: u8 = 0x46;
const PREVRANDAO_OPCODE: u8 = 0x44;
const GASPRICE_OPCODE: u8 = 0x3A;
const SELFBALANCE_OPCODE: u8 = 0x47;
const BLOBBASEFEE_OPCODE: u8 = 0x4A;

fn read_word(out: &Bytes) -> U256 {
    let mut buf = [0u8; 32];
    let copy_len = out.len().min(32);
    buf[32 - copy_len..].copy_from_slice(&out[..copy_len]);
    U256::from_be_bytes(buf)
}

fn run_opcode_test(opcode: u8) -> Result<(bool, U256), String> {
    let mut s = ExecutorScaffold::new();
    let target = Address::repeat_byte(opcode);
    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));
    deploy_contract(s.harness.state(), target, one_op_returner(opcode), U256::ZERO);
    let (success, output) =
        execute_call(&mut s.harness, s.base_fee, s.chain_id, target, alice_key(), 0)?;
    Ok((success, read_word(&output)))
}

#[test]
fn number_opcode_returns_l1_block_number_not_l2() {
    let (success, value) = run_opcode_test(NUMBER_OPCODE).expect("opcode call");
    assert!(success);
    assert_eq!(
        value,
        U256::ZERO,
        "Arbitrum NUMBER returns L1 block number from arb_precompiles cache; \
         test setup leaves it at 0, distinct from BlockEnv.number=1"
    );
}

#[test]
fn timestamp_opcode_returns_block_timestamp() {
    let (success, value) = run_opcode_test(TIMESTAMP_OPCODE).expect("opcode call");
    assert!(success);
    assert_eq!(value, U256::from(1_700_000_000u64));
}

#[test]
fn chainid_opcode_returns_configured_chain_id() {
    let (success, value) = run_opcode_test(CHAINID_OPCODE).expect("opcode call");
    assert!(success);
    assert_eq!(value, U256::from(421614u64));
}

#[test]
fn coinbase_opcode_returns_block_coinbase() {
    let (success, value) = run_opcode_test(COINBASE_OPCODE).expect("opcode call");
    assert!(success);
    assert_eq!(value, U256::ZERO);
}

#[test]
fn gasprice_opcode_returns_base_fee_after_tip_drop() {
    let (success, value) = run_opcode_test(GASPRICE_OPCODE).expect("opcode call");
    assert!(success);
    assert_eq!(
        value,
        U256::from(100_000_000u64),
        "Arbitrum drops tip — GASPRICE returns block base_fee, not tx gas_price"
    );
}

#[test]
fn selfbalance_opcode_returns_account_balance() {
    let mut s = ExecutorScaffold::new();
    let target = Address::repeat_byte(SELFBALANCE_OPCODE);
    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));
    let target_balance = U256::from(7u128) * U256::from(ONE_ETH);
    deploy_contract(
        s.harness.state(),
        target,
        one_op_returner(SELFBALANCE_OPCODE),
        target_balance,
    );
    let (success, output) =
        execute_call(&mut s.harness, s.base_fee, s.chain_id, target, alice_key(), 0)
            .expect("call");
    assert!(success);
    assert_eq!(read_word(&output), target_balance);
}

#[test]
fn blobbasefee_opcode_halts_with_invalid_opcode() {
    let mut s = ExecutorScaffold::new();
    let target = Address::repeat_byte(BLOBBASEFEE_OPCODE);
    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));
    deploy_contract(
        s.harness.state(),
        target,
        one_op_returner(BLOBBASEFEE_OPCODE),
        U256::ZERO,
    );
    let (success, _) =
        execute_call(&mut s.harness, s.base_fee, s.chain_id, target, alice_key(), 0)
            .expect("call");
    assert!(!success, "BLOBBASEFEE must halt on Arbitrum");
}

#[test]
fn prevrandao_opcode_returns_arbitrum_constant() {
    let (success, value) = run_opcode_test(PREVRANDAO_OPCODE).expect("opcode call");
    assert!(success);
    assert_eq!(
        value,
        U256::from(1u64),
        "Arbitrum PREVRANDAO is BigToHash(difficulty=1) = 0x...0001"
    );
}

#[test]
fn blockhash_opcode_returns_zero_for_out_of_range() {
    let mut s = ExecutorScaffold::new();
    let target = Address::repeat_byte(0x40);
    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));
    let mut code = vec![0x60, 0x05, 0x40];
    code.extend_from_slice(&store_and_return_top());
    deploy_contract(s.harness.state(), target, code, U256::ZERO);
    let (success, output) =
        execute_call(&mut s.harness, s.base_fee, s.chain_id, target, alice_key(), 0)
            .expect("call");
    assert!(success);
    assert_eq!(
        read_word(&output),
        U256::ZERO,
        "BLOCKHASH(5) when L1 block is 0 must return zero (out of range)"
    );
}

#[test]
fn basefee_opcode_returns_block_basefee() {
    let mut s = ExecutorScaffold::new();
    let target = Address::repeat_byte(0x48);
    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));
    let mut code = vec![0x48];
    code.extend_from_slice(&store_and_return_top());
    deploy_contract(s.harness.state(), target, code, U256::ZERO);
    let (success, output) =
        execute_call(&mut s.harness, s.base_fee, s.chain_id, target, alice_key(), 0)
            .expect("call");
    assert!(success);
    assert_eq!(read_word(&output), U256::from(100_000_000u64));
}

#[test]
fn origin_opcode_returns_tx_signer() {
    let mut s = ExecutorScaffold::new();
    let target = Address::repeat_byte(0x32);
    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));
    let mut code = vec![0x32];
    code.extend_from_slice(&store_and_return_top());
    deploy_contract(s.harness.state(), target, code, U256::ZERO);
    let (success, output) =
        execute_call(&mut s.harness, s.base_fee, s.chain_id, target, alice_key(), 0)
            .expect("call");
    assert!(success);
    let mut expected = [0u8; 32];
    expected[12..].copy_from_slice(alice().as_slice());
    assert_eq!(read_word(&output), U256::from_be_bytes(expected));
}
