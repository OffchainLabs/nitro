//! Activating a Stylus program must not warm the program address (reading its
//! code is a pure state read), so a later `CALL` still pays cold access. Two
//! identical-bytecode programs share a code hash, so activating `P` then calling
//! `P` must cost the same as activating `P` then calling its twin `Q`.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

mod common;

use std::sync::Arc;

use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{address, hex, Address, Bytes, TxKind, B256, U256};
use arb_evm::config::ArbEvmConfig;
use arb_executor_tests::helpers::{
    alice, alice_key, deploy_contract, fund_account, recover, sign_1559,
};
use arb_test_utils::ArbosHarness;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::primitives::hardfork::SpecId;

const CHAIN_ID: u64 = 421614;
const ARBOS_VERSION: u64 = 60;
const ARBWASM: u8 = 0x71;
const ACTIVATE_PROGRAM_SELECTOR: [u8; 4] = [0x58, 0xc7, 0x80, 0xc2];
const BLOCK_NUMBER: u64 = 263_059_937;
const BLOCK_TIMESTAMP: u64 = 1_771_225_381;

const PROGRAM: Address = address!("1111111111111111111111111111111111111111");
const PROGRAM_TWIN: Address = address!("2222222222222222222222222222222222222222");
const CALLER: Address = address!("3333333333333333333333333333333333333333");

const ACTIVATION_VALUE: u128 = 1_000_000_000_000_000_000;

const STYLUS_RUNTIME_HEX: &str = include_str!("../../arb-fuzz/tests/factory_activate_runtime.hex");

fn stylus_runtime() -> Vec<u8> {
    hex::decode(STYLUS_RUNTIME_HEX.trim().trim_start_matches("0x")).expect("decode stylus runtime")
}

fn run(call_target: Address) -> (bool, u64) {
    let mut harness = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION)
        .with_chain_id(CHAIN_ID)
        .initialize();

    fund_account(harness.state(), alice(), U256::from(1u128 << 100));
    let runtime = stylus_runtime();
    deploy_contract(harness.state(), PROGRAM, runtime.clone(), U256::ZERO);
    deploy_contract(harness.state(), PROGRAM_TWIN, runtime, U256::ZERO);
    deploy_contract(
        harness.state(),
        CALLER,
        common::read_then_call(
            ARBWASM,
            ACTIVATE_PROGRAM_SELECTOR,
            PROGRAM,
            call_target,
            true,
        ),
        U256::ZERO,
    );

    harness
        .state()
        .merge_transitions(revm::database::states::bundle_state::BundleRetention::Reverts);

    let cfg = ArbEvmConfig::new(Arc::new(ChainSpec::default()));
    let mut env: EvmEnv<SpecId> = EvmEnv {
        cfg_env: revm::context::CfgEnv::default(),
        block_env: revm::context::BlockEnv::default(),
    };
    env.cfg_env.chain_id = CHAIN_ID;
    env.cfg_env.disable_base_fee = true;
    env.cfg_env.tx_gas_limit_cap = Some(u64::MAX);
    env.block_env.timestamp = U256::from(BLOCK_TIMESTAMP);
    env.block_env.basefee = 100_000_000;
    env.block_env.gas_limit = 1_125_899_906_842_624;
    env.block_env.number = U256::from(BLOCK_NUMBER);

    let evm_factory = cfg.block_executor_factory().evm_factory();
    let block_ctx = arb_context::BlockCtx::new(
        ARBOS_VERSION,
        BLOCK_TIMESTAMP,
        BLOCK_NUMBER,
        BLOCK_NUMBER,
        false,
    );
    evm_factory.stage_ctx(Arc::new(arb_context::ArbPrecompileCtx::with_block(
        Arc::new(block_ctx),
    )));
    let evm = evm_factory.create_evm(harness.state(), env);
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
        .create_arb_executor(evm, exec_ctx, CHAIN_ID);
    executor
        .apply_pre_execution_changes()
        .expect("pre-execution changes");

    let tx = sign_1559(
        CHAIN_ID,
        0,
        200_000_000,
        1,
        30_000_000,
        TxKind::Call(CALLER),
        U256::from(ACTIVATION_VALUE),
        Bytes::new(),
        alice_key(),
    );
    let exec_result = executor
        .execute_transaction_without_commit(recover(tx))
        .expect("execute tx");
    (
        exec_result.result.result.is_success(),
        exec_result.result.result.gas_used(),
    )
}

#[test]
fn activate_does_not_warm_program_for_subsequent_call() {
    let (ok_activated, call_activated) = run(PROGRAM);
    let (ok_twin, call_twin) = run(PROGRAM_TWIN);
    assert!(ok_activated && ok_twin);
    assert!(call_activated > 2_000_000, "activation did not run");
    assert_eq!(
        call_activated,
        call_twin,
        "calling the just-activated program cost {} less than calling its twin",
        call_twin as i64 - call_activated as i64,
    );
}
