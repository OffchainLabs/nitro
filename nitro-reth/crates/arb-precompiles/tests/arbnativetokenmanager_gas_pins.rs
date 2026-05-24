//! Per-selector gas pins for ArbNativeTokenManager (0x73).
//!
//! Precompile is gated to ArbOS v41+. Below v41 the dispatch returns an
//! Ok(empty,0) — the canonical "no-op below activation" form — which is
//! what `check_precompile_version` produces.

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::U256;
use arb_precompiles::create_arbnativetokenmanager_precompile;
use common::{calldata, word_u256, PrecompileTest};

fn arbntm(ctx: std::sync::Arc<arb_context::ArbPrecompileCtx>) -> DynPrecompile {
    create_arbnativetokenmanager_precompile(ctx)
}

fn fixture(v: u64) -> PrecompileTest {
    PrecompileTest::new().arbos_version(v).arbos_state()
}

#[test]
fn below_v41_mint_returns_noop_zero_gas() {
    let run = fixture(40).gas(50_000).call(
        arbntm,
        &calldata("mintNativeToken(uint256)", &[word_u256(U256::from(1u64))]),
    );
    let out = run.assert_ok();
    assert!(!out.reverted);
    assert_eq!(out.gas_used, 0);
}

#[test]
fn mint_unauthorized_burns_all_gas_v41() {
    // Caller is not in NativeTokenOwners → unauthorized burn-out.
    let run = fixture(41).gas(50_000).call(
        arbntm,
        &calldata("mintNativeToken(uint256)", &[word_u256(U256::from(1u64))]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 50_000);
}

#[test]
fn burn_unauthorized_burns_all_gas_v41() {
    let run = fixture(41).gas(50_000).call(
        arbntm,
        &calldata("burnNativeToken(uint256)", &[word_u256(U256::from(1u64))]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 50_000);
}

#[test]
fn invalid_calldata_burns_all_gas_v41() {
    let run = fixture(41).gas(50_000).call(
        arbntm,
        &alloy_primitives::Bytes::from(vec![0xde, 0xad, 0xbe, 0xef]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 50_000);
}
