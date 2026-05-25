//! Per-selector gas pins for ArbWasm (0x71).
//!
//! Locks the exact `PrecompileOutput::gas_used` returned for each pure read
//! selector. The two state-mutating selectors (`activateProgram`,
//! `codehashKeepalive`) take value/code inputs and aren't covered here —
//! they have richer integration tests in `arbwasm.rs`.

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::B256;
use arb_precompiles::create_arbwasm_precompile;
use common::{calldata, word_u256, PrecompileTest};

const ARBOS_V30: u64 = 30;
const ARBOS_V32: u64 = 32;
const ARBOS_V59: u64 = 59;

fn arbwasm(ctx: std::sync::Arc<arb_context::ArbPrecompileCtx>) -> DynPrecompile {
    create_arbwasm_precompile(ctx)
}

fn fixture(v: u64) -> PrecompileTest {
    PrecompileTest::new().arbos_version(v).arbos_state()
}

#[test]
fn stylus_version_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbwasm, &calldata("stylusVersion()", &[]));
    assert_eq!(run.gas_used(), 903);
}

#[test]
fn ink_price_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbwasm, &calldata("inkPrice()", &[]));
    assert_eq!(run.gas_used(), 903);
}

#[test]
fn max_stack_depth_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbwasm, &calldata("maxStackDepth()", &[]));
    assert_eq!(run.gas_used(), 903);
}

#[test]
fn free_pages_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbwasm, &calldata("freePages()", &[]));
    assert_eq!(run.gas_used(), 903);
}

#[test]
fn page_gas_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbwasm, &calldata("pageGas()", &[]));
    assert_eq!(run.gas_used(), 903);
}

#[test]
fn page_ramp_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbwasm, &calldata("pageRamp()", &[]));
    assert_eq!(run.gas_used(), 903);
}

#[test]
fn page_limit_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbwasm, &calldata("pageLimit()", &[]));
    assert_eq!(run.gas_used(), 903);
}

#[test]
fn min_init_gas_v32_gas_pin() {
    let run = fixture(ARBOS_V32).call(arbwasm, &calldata("minInitGas()", &[]));
    assert_eq!(run.gas_used(), 906);
}

#[test]
fn init_cost_scalar_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbwasm, &calldata("initCostScalar()", &[]));
    assert_eq!(run.gas_used(), 903);
}

#[test]
fn expiry_days_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbwasm, &calldata("expiryDays()", &[]));
    assert_eq!(run.gas_used(), 903);
}

#[test]
fn keepalive_days_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbwasm, &calldata("keepaliveDays()", &[]));
    assert_eq!(run.gas_used(), 903);
}

#[test]
fn block_cache_size_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbwasm, &calldata("blockCacheSize()", &[]));
    assert_eq!(run.gas_used(), 903);
}

#[test]
fn activation_gas_v59_gas_pin() {
    let run = fixture(ARBOS_V59).call(arbwasm, &calldata("activationGas()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn codehash_version_unknown_v30_gas_pin() {
    let codehash = B256::repeat_byte(0x42);
    let run = fixture(ARBOS_V30).call(
        arbwasm,
        &calldata("codehashVersion(bytes32)", &[word_u256(codehash.into())]),
    );
    // No active program → ProgramNotActivated revert at LOOKUP_GAS.
    assert!(run.assert_ok().reverted);
}

#[test]
fn codehash_asm_size_unknown_v30_gas_pin() {
    let codehash = B256::repeat_byte(0x43);
    let run = fixture(ARBOS_V30).call(
        arbwasm,
        &calldata("codehashAsmSize(bytes32)", &[word_u256(codehash.into())]),
    );
    assert!(run.assert_ok().reverted);
}
