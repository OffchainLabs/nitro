//! `ArbWasmCache.cacheProgram` must not warm the program address (resolving its
//! code hash is a pure state read), so a later `CALL` still pays cold access.
//! Caching `P` then calling `P` must cost the same as caching `P` then calling
//! its identical-bytecode twin `Q`.

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
use arb_storage::{
    layout::{
        derive_subspace_key, map_slot_b256, programs::CACHE_MANAGERS_KEY, PROGRAMS_SUBSPACE,
        ROOT_STORAGE_KEY,
    },
    write_storage_at, ARBOS_STATE_ADDRESS,
};
use arb_test_utils::ArbosHarness;
use arbos::programs::Program;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::primitives::hardfork::SpecId;

const CHAIN_ID: u64 = 421614;
const ARBOS_VERSION: u64 = 60;
const ARBWASMCACHE: u8 = 0x72;
const BLOCK_TIMESTAMP: u64 = 1_771_225_381;
const ARBITRUM_GENESIS_SECS: u64 = 1_421_388_000;

const PROGRAM: Address = address!("1111111111111111111111111111111111111111");
const PROGRAM_TWIN: Address = address!("2222222222222222222222222222222222222222");
const CALLER: Address = address!("3333333333333333333333333333333333333333");

fn cache_manager_member_slot(addr: Address) -> U256 {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let cm_key = derive_subspace_key(programs_key.as_slice(), CACHE_MANAGERS_KEY);
    let by_addr_key = derive_subspace_key(cm_key.as_slice(), &[0]);
    let mut padded = [0u8; 32];
    padded[12..32].copy_from_slice(addr.as_slice());
    map_slot_b256(by_addr_key.as_slice(), &B256::from(padded))
}

fn run(call_target: Address) -> (bool, u64) {
    let selector = keccak256("cacheProgram(address)")[..4].try_into().unwrap();

    let mut harness = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION)
        .with_chain_id(CHAIN_ID)
        .initialize();

    fund_account(harness.state(), alice(), U256::from(1u128 << 100));
    // Identical bytecode (single STOP) -> shared code hash, so caching one
    // program activates the entry for both and a CALL costs the same apart
    // from access.
    deploy_contract(harness.state(), PROGRAM, vec![0x00], U256::ZERO);
    deploy_contract(harness.state(), PROGRAM_TWIN, vec![0x00], U256::ZERO);
    deploy_contract(
        harness.state(),
        CALLER,
        common::read_then_call(ARBWASMCACHE, selector, PROGRAM, call_target, false),
        U256::ZERO,
    );
    write_storage_at(
        harness.state(),
        ARBOS_STATE_ADDRESS,
        cache_manager_member_slot(CALLER),
        U256::from(1),
    )
    .expect("register cache manager");

    let codehash = keccak256([0x00u8]);
    let activated_at = ((BLOCK_TIMESTAMP - ARBITRUM_GENESIS_SECS) / 3600) as u32;
    {
        let state_ptr = harness.state_ptr();
        let state = harness.arbos_state();
        // SAFETY: `state_ptr` aliases the harness's owned `State`; `state` only
        // borrows storage slots through the backend, so the handles do not race.
        let backend: &mut _ = unsafe { &mut *state_ptr };
        let version = state.programs.params(backend).expect("params").version;
        let program = Program {
            version,
            init_cost: 0,
            cached_cost: 0,
            footprint: 0,
            asm_estimate_kb: 0,
            activated_at,
            age_seconds: 0,
            cached: false,
        };
        state
            .programs
            .set_program(backend, codehash, program)
            .expect("set program");
    }

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
    env.block_env.number = U256::from(1u64);

    let evm_factory = cfg.block_executor_factory().evm_factory();
    let block_ctx = arb_context::BlockCtx::new(ARBOS_VERSION, BLOCK_TIMESTAMP, 1, 1, false);
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
fn cache_program_does_not_warm_program_for_subsequent_call() {
    let (ok_cached, call_cached) = run(PROGRAM);
    let (ok_twin, call_twin) = run(PROGRAM_TWIN);
    assert!(ok_cached && ok_twin);
    assert_eq!(
        call_cached,
        call_twin,
        "calling the just-cached program cost {} less than calling its twin",
        call_twin as i64 - call_cached as i64,
    );
}
