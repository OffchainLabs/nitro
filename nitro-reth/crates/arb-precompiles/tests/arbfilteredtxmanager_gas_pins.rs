//! Per-selector gas pins for ArbFilteredTransactionsManager (0x74).
//!
//! Precompile-level gated to ArbOS v60+. Below that, dispatch returns the
//! "no-op" Ok(empty, 0). The dispatcher itself wraps the inner method in a
//! FreeAccessPrecompile that overrides gas to either 0 (caller is a
//! transaction filterer) or 1600 (the wrapper's 2-SLOAD overhead).

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::B256;
use arb_precompiles::create_arbfilteredtxmanager_precompile;
use common::PrecompileTest;

const SLOAD: u64 = 800;

fn arbftm(ctx: std::sync::Arc<arb_context::ArbPrecompileCtx>) -> DynPrecompile {
    create_arbfilteredtxmanager_precompile(ctx)
}

fn fixture(v: u64) -> PrecompileTest {
    PrecompileTest::new().arbos_version(v).arbos_state()
}

fn b32_calldata(name: &str, hash: B256) -> alloy_primitives::Bytes {
    let mut buf = Vec::with_capacity(36);
    buf.extend_from_slice(&common::selector(name));
    buf.extend_from_slice(hash.as_slice());
    alloy_primitives::Bytes::from(buf)
}

#[test]
fn below_v60_dispatch_returns_noop_zero_gas() {
    let run = fixture(59).gas(50_000).call(
        arbftm,
        &b32_calldata("isTransactionFiltered(bytes32)", B256::ZERO),
    );
    let out = run.assert_ok();
    assert!(!out.reverted);
    assert_eq!(out.gas_used, 0);
}

#[test]
fn is_transaction_filtered_non_filterer_caller_pays_wrapper_v60_gas_pin() {
    // Caller is not a transaction filterer: wrapper overrides gas to
    // 2 * SLOAD (its is-filterer probe) = 1600. Inner output is preserved.
    let run = fixture(60).gas(50_000).call(
        arbftm,
        &b32_calldata("isTransactionFiltered(bytes32)", B256::ZERO),
    );
    let out = run.assert_ok();
    assert!(!out.reverted);
    assert_eq!(out.gas_used, 2 * SLOAD);
}

#[test]
fn add_filtered_transaction_non_filterer_caller_v60_gas_pin() {
    // Inner reverts (not a filterer) → wrapper converts that to
    // Ok(reverted, wrapper_gas).
    let run = fixture(60).gas(50_000).call(
        arbftm,
        &b32_calldata("addFilteredTransaction(bytes32)", B256::ZERO),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 2 * SLOAD);
}

#[test]
fn delete_filtered_transaction_non_filterer_caller_v60_gas_pin() {
    let run = fixture(60).gas(50_000).call(
        arbftm,
        &b32_calldata("deleteFilteredTransaction(bytes32)", B256::ZERO),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 2 * SLOAD);
}

#[test]
fn invalid_calldata_reverts_with_wrapper_gas_v60() {
    let run = fixture(60).gas(50_000).call(
        arbftm,
        &alloy_primitives::Bytes::from(vec![0xde, 0xad, 0xbe, 0xef]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    // The free-access wrapper charges only its own reads (OpenArbosState +
    // membership = 1600) and discards the inner precompile's gas, so a bad
    // selector reverts with 1600 rather than burning all gas.
    assert_eq!(out.gas_used, 1600);
}
