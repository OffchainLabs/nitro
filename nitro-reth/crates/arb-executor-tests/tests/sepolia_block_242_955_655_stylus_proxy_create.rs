//! Regression test for a Stylus-via-proxy-via-CREATE call chain.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use std::sync::Arc;

use alloy_evm::{
    block::{BlockExecutor, BlockExecutorFactory},
    eth::EthBlockExecutionCtx,
    EvmFactory,
};
use alloy_primitives::{address, b256, hex, keccak256, Address, Bytes, TxKind, B256, U256};
use arb_evm::config::ArbEvmConfig;
use arb_executor_tests::helpers::{
    alice, alice_key, deploy_contract, fund_account, recover, sign_1559,
};
use arb_storage::write_storage_at;
use arb_test_utils::ArbosHarness;
use arbos::programs::Program;
use reth_chainspec::ChainSpec;
use reth_evm::{ConfigureEvm, EvmEnv};
use revm::primitives::hardfork::SpecId;

const PROXY: Address = address!("b66432b88ca4495ed8621ae967a746df63b3e369");
const STYLUS_IMPL: Address = address!("417a531fd85ccbdcf41e47704de5a4a99d8a7e1a");
const FACTORY: Address = address!("be94dee64ff48e342cba90d176edf0a6300a6eac");
const STYLUS_INIT: Address = address!("806c95a7ba85e098c1aa269c487409c8e1b0a7f1");

const ERC1967_IMPL_SLOT: B256 =
    b256!("360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc");

const CHAIN_ID: u64 = 421614;
const TX_GAS_LIMIT: u64 = 30_000_000;
const TX_MAX_PRIO: u128 = 1;
const TX_MAX_FEE: u128 = 200_000_000;

const BLOCK_TIMESTAMP: u64 = 1_771_225_381;
const BLOCK_NUMBER: u64 = 242_955_655;

const CANON_GAS_USED: u64 = 383_313;
const RUNAWAY_GAS_FLOOR: u64 = 10_000_000;

const PROXY_HEX: &str = include_str!(
    "../../arb-spec-tests/fixtures/stylus/regression/sepolia_242955655_assets/proxy.hex"
);
const FACTORY_HEX: &str = include_str!(
    "../../arb-spec-tests/fixtures/stylus/regression/sepolia_242955655_assets/factory.hex"
);
const STYLUS_IMPL_HEX: &str = include_str!(
    "../../arb-spec-tests/fixtures/stylus/regression/sepolia_242955655_assets/stylus_impl.hex"
);
const STYLUS_INIT_HEX: &str = include_str!(
    "../../arb-spec-tests/fixtures/stylus/regression/sepolia_242955655_assets/stylus_init.hex"
);

fn decode_hex_file(s: &str) -> Vec<u8> {
    let trimmed = s.trim().trim_start_matches("0x");
    hex::decode(trimmed).expect("decode contract hex")
}

// ── Canonical input ─────────────────────────────────────────────────────────

const TX_INPUT_HEX: &str = "c2e1a90f00000000000000000000000048b3f901d040796f9cda37469fc5436fca7113660000000000000000000000005602a3f9b8a935df32871bb1c6289f24620233f70000000000000000000000000000000000000000000000000a688906bd8b00000000000000000000000000000000000000000000000000000000000005f5e10000000000000000000000000000000000000000000000000000470de4df820000000000000000000000000000000000000000000000000000016345785d8a00000000000000000000000000000000000000000000000000000b1a2bc2ec5000000000000000000000000000000000000000000000000000000d2f13f7789f00000000000000000000000000000000000000000000000000000de0b6b3a76400000000000000000000000000000000000000000000000000000bcbce7f1b15000000000000000000000000000000000000000000000000000000b1a2bc2ec50000";

fn build_tx() -> arb_primitives::signed_tx::ArbTransactionSigned {
    let input = hex::decode(TX_INPUT_HEX).expect("decode input");
    sign_1559(
        CHAIN_ID,
        /* nonce */ 0,
        TX_MAX_FEE,
        TX_MAX_PRIO,
        TX_GAS_LIMIT,
        TxKind::Call(PROXY),
        U256::ZERO,
        Bytes::from(input),
        alice_key(),
    )
}

fn activate_stylus(
    harness: &mut ArbosHarness,
    bytecode: &[u8],
    codehash: B256,
    arbos_version: u64,
    page_limit: u16,
    stylus_version: u16,
    activated_at_hours: u32,
) {
    let wasm = arb_stylus::decompress_wasm(bytecode).expect("decompress stylus WASM");
    let mut gas = u64::MAX;
    let activation = arb_stylus::activate_program(
        &wasm,
        codehash.as_ref(),
        stylus_version,
        arbos_version,
        page_limit,
        false,
        &mut gas,
    )
    .expect("activate stylus program");

    let estimate_kb = activation.asm_estimate.div_ceil(1024).min(0xFF_FFFF);

    let program = Program {
        version: stylus_version,
        init_cost: activation.init_gas,
        cached_cost: activation.cached_init_gas,
        footprint: activation.footprint,
        asm_estimate_kb: estimate_kb,
        activated_at: activated_at_hours,
        age_seconds: 0,
        cached: false,
    };

    let state_ptr = harness.state_ptr();
    let state = harness.arbos_state();
    // SAFETY: `state_ptr` aliases the harness's owned `State`; the
    // `ArbosState` returned above only borrows storage slots through the
    // backend, so the two handles do not race.
    let backend: &mut _ = unsafe { &mut *state_ptr };
    state
        .programs
        .set_module_hash(backend, codehash, activation.module_hash)
        .expect("set module hash");
    state
        .programs
        .set_program(backend, codehash, program)
        .expect("set program");
}

fn fresh_activated_at_hours() -> u32 {
    ((BLOCK_TIMESTAMP - 1_421_388_000) / 3600) as u32
}

#[test]
fn expiry_trap_burns_all_forwarded_gas() {
    let outcome = run_scenario(1);
    assert!(!outcome.success);
    assert!(
        outcome.gas_used >= RUNAWAY_GAS_FLOOR,
        "expected gas runaway (>={}) but got {}",
        RUNAWAY_GAS_FLOOR,
        outcome.gas_used,
    );
}

#[test]
fn fresh_activation_matches_canonical_gas() {
    let outcome = run_scenario(fresh_activated_at_hours());
    assert!(!outcome.success);
    assert!(outcome.gas_used < RUNAWAY_GAS_FLOOR);
    let drift = (outcome.gas_used as i128 - CANON_GAS_USED as i128).abs();
    assert!(
        drift < (CANON_GAS_USED / 4) as i128,
        "gas_used={} drifted >25% from canon={CANON_GAS_USED}",
        outcome.gas_used,
    );
}

struct ScenarioOutcome {
    success: bool,
    gas_used: u64,
}

fn run_scenario(activated_at_hours: u32) -> ScenarioOutcome {
    let mut harness = ArbosHarness::new()
        .with_arbos_version(51)
        .with_chain_id(CHAIN_ID)
        .initialize();

    fund_account(harness.state(), alice(), U256::from(1u128 << 100));

    let proxy_code = decode_hex_file(PROXY_HEX);
    let factory_code = decode_hex_file(FACTORY_HEX);
    let stylus_impl_code = decode_hex_file(STYLUS_IMPL_HEX);
    let stylus_init_code = decode_hex_file(STYLUS_INIT_HEX);

    let stylus_impl_codehash = keccak256(&stylus_impl_code);
    let stylus_init_codehash = keccak256(&stylus_init_code);

    assert_eq!(
        stylus_impl_codehash,
        b256!("d8781e55f826d516c6e60b6f5a87a9cc103e958d5bb8ba0f2cac07bb03496f7e"),
    );
    assert_eq!(
        stylus_init_codehash,
        b256!("3f9069516d5090a453a9f04057478422513da9620080abe828cb3ca441e5d6f8"),
    );

    deploy_contract(harness.state(), PROXY, proxy_code, U256::ZERO);
    deploy_contract(harness.state(), FACTORY, factory_code, U256::ZERO);
    deploy_contract(
        harness.state(),
        STYLUS_IMPL,
        stylus_impl_code.clone(),
        U256::ZERO,
    );
    deploy_contract(
        harness.state(),
        STYLUS_INIT,
        stylus_init_code.clone(),
        U256::ZERO,
    );

    write_storage_at(
        harness.state(),
        PROXY,
        U256::from_be_bytes(ERC1967_IMPL_SLOT.0),
        U256::from_be_bytes(STYLUS_IMPL.into_word().0),
    )
    .expect("write proxy impl slot");

    set_canonical_proxy_storage(harness.state());

    activate_stylus(
        &mut harness,
        &stylus_impl_code,
        stylus_impl_codehash,
        51,
        128,
        2,
        activated_at_hours,
    );
    activate_stylus(
        &mut harness,
        &stylus_init_code,
        stylus_init_codehash,
        51,
        128,
        2,
        activated_at_hours,
    );

    harness
        .state()
        .merge_transitions(revm::database::states::bundle_state::BundleRetention::Reverts);

    let chain_spec: Arc<ChainSpec> = Arc::new(ChainSpec::default());
    let cfg = ArbEvmConfig::new(chain_spec);

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
    env.block_env.prevrandao = Some(B256::from(U256::from(1u64)));
    env.block_env.difficulty = U256::from(1u64);

    let evm_factory = cfg.block_executor_factory().evm_factory();
    // Stage a per-block precompile ctx so the EVM and executor share an Arc
    // carrying the correct arbos_version. Without this, `create_evm` falls
    // back to a default ctx with arbos_version=0 and the Stylus dispatch
    // path never fires.
    let block_ctx =
        arb_context::BlockCtx::new(51, BLOCK_TIMESTAMP, BLOCK_NUMBER, BLOCK_NUMBER, false);
    let staged_ctx = std::sync::Arc::new(arb_context::ArbPrecompileCtx::with_block(
        std::sync::Arc::new(block_ctx),
    ));
    evm_factory.stage_ctx(staged_ctx);
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

    let tx = build_tx();
    let recovered = recover(tx);
    assert_eq!(recovered.signer(), alice());

    let exec_result = executor
        .execute_transaction_without_commit(recovered)
        .expect("execute tx");

    let gas_used = exec_result.result.result.gas_used();
    let success = exec_result.result.result.is_success();

    ScenarioOutcome { success, gas_used }
}

fn set_canonical_proxy_storage(state: &mut revm::database::State<arb_test_utils::EmptyDb>) {
    let mut slot0 = [0u8; 32];
    slot0[11] = 0x01;
    slot0[12..32].copy_from_slice(alice().as_slice());
    write_storage_at(state, PROXY, U256::ZERO, U256::from_be_bytes(slot0))
        .expect("write proxy slot 0");

    let canonical: &[(u64, &str)] = &[
        (1, "848cf71c933c9042e9110b0346a63c9874f8504"),
        (2, "32a2f9b33a595a69dff78b605336b97d98cfe9cf"),
        (3, "01017e1509e6ac180e4864bc46407a8bec70363d3"),
        (4, "fe405ce04fc81c54a693405b169818f092443ac5"),
        (5, "48b3f901d040796f9cda37469fc5436fca711366"),
        (6, "a550917c48ef95786e49da0505b2d5930a272768"),
        (7, "806c95a7ba85e098c1aa269c487409c8e1b0a7f1"),
        (8, "a1aafdd133c25679db1e289ab93ca53b77a31a9b"),
        (9, "b9f0d8d668b3685e8ec2981e0b706763cbfb14dd"),
        (10, "be94dee64ff48e342cba90d176edf0a6300a6eac"),
        (11, "84338e71eef83b688d385f25d3345565be5bdb7d"),
        (12, "295dda44d9c10300b5ef0fd1cdc41731aae9cef5"),
    ];
    for (slot, hex) in canonical {
        write_storage_at(
            state,
            PROXY,
            U256::from(*slot),
            U256::from_str_radix(hex, 16).unwrap(),
        )
        .expect("write proxy canonical slot");
    }

    let mapping_slots: &[(&str, &str)] = &[(
        "9de67d799112a2b477f424b9a8a907e80eb164ee299dff5ab6118a0730c440a5",
        "00000000000000000000000000000000000000000000000000000000000f4240",
    )];
    for (slot_hex, val_hex) in mapping_slots {
        write_storage_at(
            state,
            PROXY,
            U256::from_str_radix(slot_hex, 16).unwrap(),
            U256::from_str_radix(val_hex, 16).unwrap(),
        )
        .expect("write proxy mapping slot");
    }
}
