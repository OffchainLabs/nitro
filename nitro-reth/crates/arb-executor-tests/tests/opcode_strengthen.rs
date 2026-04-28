use std::sync::Arc;

use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{Address, Bytes, TxKind, B256, U256};
use arb_evm::config::ArbEvmConfig;
use arb_executor_tests::helpers::{
    alice, alice_key, deploy_contract, fund_account, recover, sign_legacy, ExecutorScaffold,
    ONE_ETH, ONE_GWEI,
};
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
    l1_block_number: u64,
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
    executor.arb_ctx.l1_block_number = l1_block_number;
    executor
        .apply_pre_execution_changes()
        .map_err(|e| format!("pre: {e}"))?;

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
    executor
        .commit_transaction(result)
        .map_err(|e| format!("commit: {e}"))?;
    let _ = executor.finish().map_err(|e| format!("finish: {e}"))?;
    Ok((success, output))
}

fn one_op_returner(opcode: u8) -> Vec<u8> {
    vec![opcode, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xF3]
}

fn read_word(out: &Bytes) -> U256 {
    let mut buf = [0u8; 32];
    let copy_len = out.len().min(32);
    buf[32 - copy_len..].copy_from_slice(&out[..copy_len]);
    U256::from_be_bytes(buf)
}

/// Strong test: set L1 block number to a specific value, verify NUMBER returns
/// that value (not BlockEnv.number which is the L2 block).
#[test]
fn number_opcode_returns_specific_l1_block_not_l2() {
    let mut s = ExecutorScaffold::new();
    let target = Address::repeat_byte(0x43);
    fund_account(s.harness.state(), alice(), U256::from(10u128 * ONE_ETH));
    deploy_contract(s.harness.state(), target, one_op_returner(0x43), U256::ZERO);

    let (success, output) = execute_call(
        &mut s.harness,
        s.base_fee,
        s.chain_id,
        target,
        alice_key(),
        0,
        42_000_000,
    )
    .expect("call");
    assert!(success);
    let value = read_word(&output);
    assert_eq!(value, U256::from(42_000_000u64));
    assert_ne!(
        value,
        U256::from(1u64),
        "NUMBER must not be L2 block number"
    );
}

/// BALANCE on the tx sender subtracts the poster-fee correction.
#[test]
fn balance_opcode_on_sender_subtracts_poster_correction() {
    arb_precompiles::set_current_tx_sender(alice());
    let correction = U256::from(7u64) * U256::from(ONE_GWEI);
    arb_precompiles::set_poster_balance_correction(correction);

    let mut s = ExecutorScaffold::new();
    let alice_balance = U256::from(10u128) * U256::from(ONE_ETH);
    fund_account(s.harness.state(), alice(), alice_balance);

    let target = Address::repeat_byte(0xBB);
    let mut code = vec![0x73];
    code.extend_from_slice(alice().as_slice());
    code.extend_from_slice(&[0x31, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xF3]);
    deploy_contract(s.harness.state(), target, code, U256::ZERO);

    let (success, output) = execute_call(
        &mut s.harness,
        s.base_fee,
        s.chain_id,
        target,
        alice_key(),
        0,
        0,
    )
    .expect("call");
    assert!(success);
    let reported = read_word(&output);

    arb_precompiles::set_poster_balance_correction(U256::ZERO);
    arb_precompiles::set_current_tx_sender(Address::ZERO);

    assert!(
        reported < alice_balance,
        "BALANCE on sender must be reduced by poster correction: reported {reported}, balance {alice_balance}"
    );
    assert!(
        alice_balance - reported >= correction,
        "reduction must be at least the poster correction: {}, correction {correction}",
        alice_balance - reported
    );
}

/// BALANCE on a NON-sender address returns the full balance — no correction.
#[test]
fn balance_opcode_on_non_sender_returns_full_balance() {
    arb_precompiles::set_current_tx_sender(alice());
    arb_precompiles::set_poster_balance_correction(U256::from(1_000_000u64));

    let mut s = ExecutorScaffold::new();
    fund_account(
        s.harness.state(),
        alice(),
        U256::from(10u128) * U256::from(ONE_ETH),
    );

    let other = Address::repeat_byte(0x33);
    let other_balance = U256::from(5u128) * U256::from(ONE_ETH);
    fund_account(s.harness.state(), other, other_balance);

    let target = Address::repeat_byte(0xCC);
    let mut code = vec![0x73];
    code.extend_from_slice(other.as_slice());
    code.extend_from_slice(&[0x31, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xF3]);
    deploy_contract(s.harness.state(), target, code, U256::ZERO);

    let (success, output) = execute_call(
        &mut s.harness,
        s.base_fee,
        s.chain_id,
        target,
        alice_key(),
        0,
        0,
    )
    .expect("call");
    assert!(success);
    let reported = read_word(&output);

    arb_precompiles::set_poster_balance_correction(U256::ZERO);
    arb_precompiles::set_current_tx_sender(Address::ZERO);

    assert_eq!(reported, other_balance);
}

/// SELFBALANCE on a contract that is NOT the sender returns the full balance.
/// (EIP-3607 forbids txs from senders with code, so the SELFBALANCE==sender
/// correction path is exercised only by ArbRetryTx-style re-entry flows
/// where the executing contract equals the recorded tx sender.)
#[test]
fn selfbalance_opcode_on_non_sender_contract_returns_full_balance() {
    arb_precompiles::set_current_tx_sender(alice());
    arb_precompiles::set_poster_balance_correction(U256::from(999_999u64));

    let mut s = ExecutorScaffold::new();
    fund_account(
        s.harness.state(),
        alice(),
        U256::from(10u128) * U256::from(ONE_ETH),
    );

    let target = Address::repeat_byte(0xDD);
    let target_balance = U256::from(2u128) * U256::from(ONE_ETH);
    deploy_contract(
        s.harness.state(),
        target,
        vec![0x47, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xF3],
        target_balance,
    );

    let (success, output) = execute_call(
        &mut s.harness,
        s.base_fee,
        s.chain_id,
        target,
        alice_key(),
        0,
        0,
    )
    .expect("call");

    arb_precompiles::set_poster_balance_correction(U256::ZERO);
    arb_precompiles::set_current_tx_sender(Address::ZERO);

    assert!(success);
    assert_eq!(read_word(&output), target_balance);
}

/// Unit-level verification: the arb-stylus classifier correctly identifies
/// Stylus bytecode via the discriminant.
#[test]
fn stylus_discriminant_classifier_matches_arbitrum_spec() {
    assert!(arb_stylus::is_stylus_program(&[0xEF, 0xF0, 0x00, 0x00]));
    assert!(arb_stylus::is_stylus_program(&[
        0xEF, 0xF0, 0x00, 0x01, 0x42, 0x42
    ]));
    assert!(!arb_stylus::is_stylus_program(&[0xEF, 0xF0, 0x00]));
    assert!(!arb_stylus::is_stylus_program(&[0xEF, 0xF0, 0x01, 0x00]));
    assert!(!arb_stylus::is_stylus_program(&[0x60, 0x00, 0x52]));
    assert!(!arb_stylus::is_stylus_program(&[]));
    assert_eq!(arb_stylus::STYLUS_DISCRIMINANT, [0xEF, 0xF0, 0x00]);
}
