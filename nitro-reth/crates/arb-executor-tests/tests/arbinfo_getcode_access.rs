//! `ArbInfo.getCode` must not warm the queried address (it is a pure state
//! read), so a later `CALL` still pays cold access. Reading `X` then calling
//! `X` must cost the same as reading `X` then calling its twin `Y`.

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
use alloy_primitives::{address, keccak256, Address, Bytes, TxKind, B256, U256};
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
const ARBINFO: u8 = 0x65;
const TARGET: Address = address!("1111111111111111111111111111111111111111");
const TARGET_TWIN: Address = address!("2222222222222222222222222222222222222222");
const CALLER: Address = address!("3333333333333333333333333333333333333333");

fn run(call_target: Address) -> (bool, u64) {
    let selector = keccak256("getCode(address)")[..4].try_into().unwrap();

    let mut harness = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION)
        .with_chain_id(CHAIN_ID)
        .initialize();

    fund_account(harness.state(), alice(), U256::from(1u128 << 100));
    // Identical bytecode, so a CALL to either costs the same apart from access.
    deploy_contract(harness.state(), TARGET, vec![0x00], U256::ZERO);
    deploy_contract(harness.state(), TARGET_TWIN, vec![0x00], U256::ZERO);
    deploy_contract(
        harness.state(),
        CALLER,
        common::read_then_call(ARBINFO, selector, TARGET, call_target, false),
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
    env.block_env.basefee = 100_000_000;
    env.block_env.gas_limit = 1_125_899_906_842_624;
    env.block_env.number = U256::from(1u64);

    let evm_factory = cfg.block_executor_factory().evm_factory();
    let block_ctx = arb_context::BlockCtx::new(ARBOS_VERSION, 1_700_000_000, 1, 1, false);
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
        U256::ZERO,
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
fn get_code_does_not_warm_queried_address() {
    let (ok_read, call_read) = run(TARGET);
    let (ok_twin, call_twin) = run(TARGET_TWIN);
    assert!(ok_read && ok_twin);
    assert_eq!(
        call_read,
        call_twin,
        "calling the getCode-queried address cost {} less than calling its twin",
        call_twin as i64 - call_read as i64,
    );
}
