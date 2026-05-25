//! Per-selector gas pins for ArbDebug (0xff).
//!
//! ArbDebug is debug-only. When `allow_debug_precompiles=false` it burns the
//! caller's gas and reverts. Pinning that path catches any regression that
//! makes a debug-only selector accessible in production. The same selectors
//! are also exercised with `allow_debug_precompiles=true` so the dispatch
//! body's gas schedule is locked.

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, B256, U256};
use arb_precompiles::create_arbdebug_precompile;
use common::{calldata, word_u256, PrecompileTest};

const ARBOS_V30: u64 = 30;
const GAS_LIMIT: u64 = 1_000_000;

fn arbdebug(ctx: std::sync::Arc<arb_context::ArbPrecompileCtx>) -> DynPrecompile {
    create_arbdebug_precompile(ctx)
}

fn fixture(allow_debug: bool) -> PrecompileTest {
    PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .allow_debug_precompiles(allow_debug)
        .gas(GAS_LIMIT)
}

#[test]
fn become_chain_owner_disallowed_burns_all_gas_v30() {
    let run = fixture(false).call(arbdebug, &calldata("becomeChainOwner()", &[]));
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, GAS_LIMIT);
}

#[test]
fn events_view_disallowed_burns_all_gas_v30() {
    let run = fixture(false).call(arbdebug, &calldata("eventsView()", &[]));
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, GAS_LIMIT);
}

#[test]
fn events_allowed_gas_pin_v30() {
    let run = fixture(true)
        .caller(address!("00000000000000000000000000000000000000aa"))
        .value(U256::from(42u64))
        .call(
            arbdebug,
            &calldata(
                "events(bool,bytes32)",
                &[word_u256(U256::from(1u64)), word_u256(B256::ZERO.into())],
            ),
        );
    assert_eq!(run.gas_used(), 4580);
}

#[test]
fn events_view_allowed_gas_pin_v30() {
    let run = fixture(true).call(arbdebug, &calldata("eventsView()", &[]));
    assert_eq!(run.gas_used(), 800);
}

#[test]
fn legacy_error_allowed_reverts_v30() {
    let run = fixture(true).call(arbdebug, &calldata("legacyError()", &[]));
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 0);
}

#[test]
fn custom_revert_allowed_gas_pin_v30() {
    let run = fixture(true).call(
        arbdebug,
        &calldata("customRevert(uint64)", &[word_u256(U256::from(42u64))]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 24);
}
