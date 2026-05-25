//! Per-selector gas pins for ArbWasmCache (0x72).
//!
//! Gated on ArbOS Stylus (v30). Method-level gating: `cacheCodehash` is
//! v30-only and `cacheProgram` is v31+ (StylusFixes). Tests cover at least
//! one path per selector.

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, B256};
use arb_precompiles::create_arbwasmcache_precompile;
use common::{calldata, word_address, PrecompileTest};

const SLOAD: u64 = 800;
const COPY: u64 = 3;

fn arbwasmcache(ctx: std::sync::Arc<arb_context::ArbPrecompileCtx>) -> DynPrecompile {
    create_arbwasmcache_precompile(ctx)
}

fn fixture(v: u64) -> PrecompileTest {
    PrecompileTest::new().arbos_version(v).arbos_state()
}

fn codehash_calldata(name: &str, hash: B256) -> alloy_primitives::Bytes {
    let mut buf = Vec::with_capacity(36);
    buf.extend_from_slice(&common::selector(name));
    buf.extend_from_slice(hash.as_slice());
    alloy_primitives::Bytes::from(buf)
}

#[test]
fn below_stylus_returns_noop_zero_gas() {
    // Precompile-level gating: ArbOS < v30 → Ok(empty, 0).
    let run = fixture(29)
        .gas(50_000)
        .call(arbwasmcache, &calldata("allCacheManagers()", &[]));
    let out = run.assert_ok();
    assert!(!out.reverted);
    assert_eq!(out.gas_used, 0);
}

#[test]
fn all_cache_managers_empty_v30_gas_pin() {
    // 1 OpenArbosState + 1 size SLOAD = 2 SLOAD body; result is 2 head words
    // (offset + count). argsCost = 0.
    // Total = 800 (init) + 800 (size) + (0 + 2)*COPY = 1606.
    let run = fixture(30).call(arbwasmcache, &calldata("allCacheManagers()", &[]));
    assert_eq!(run.gas_used(), 2 * SLOAD + 2 * COPY);
}

#[test]
fn is_cache_manager_unknown_v30_gas_pin() {
    let addr: Address = address!("00000000000000000000000000000000000000aa");
    let run = fixture(30).call(
        arbwasmcache,
        &calldata("isCacheManager(address)", &[word_address(addr)]),
    );
    // SLOAD (init) + SLOAD (cache_managers.is_member) + argsCost(1) + resultCost(1) = 1606.
    assert_eq!(run.gas_used(), 2 * SLOAD + 2 * COPY);
}

#[test]
fn codehash_is_cached_unknown_v30_gas_pin() {
    let hash = B256::repeat_byte(0xab);
    let run = fixture(30).call(
        arbwasmcache,
        &codehash_calldata("codehashIsCached(bytes32)", hash),
    );
    // 2 * SLOAD + 1 arg word + 1 result word = 1606.
    assert_eq!(run.gas_used(), 2 * SLOAD + 2 * COPY);
}

#[test]
fn cache_codehash_unauthorized_burns_all_v30() {
    // set_program_cached path: caller is not in cache_managers nor chain_owners.
    // Returns burn_all_revert(input.gas).
    let hash = B256::repeat_byte(0xcd);
    let run = fixture(30).gas(50_000).call(
        arbwasmcache,
        &codehash_calldata("cacheCodehash(bytes32)", hash),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 50_000);
}

#[test]
fn cache_codehash_above_v30_reverts_with_full_gas() {
    // Method-level gating: cacheCodehash is v30-ONLY (min=v30, max=v30).
    let hash = B256::repeat_byte(0xab);
    let run = fixture(31).gas(50_000).call(
        arbwasmcache,
        &codehash_calldata("cacheCodehash(bytes32)", hash),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 50_000);
}

#[test]
fn cache_program_below_v31_reverts_with_full_gas() {
    // cacheProgram requires v31+ (StylusFixes).
    let addr: Address = address!("00000000000000000000000000000000000000bb");
    let run = fixture(30).gas(50_000).call(
        arbwasmcache,
        &calldata("cacheProgram(address)", &[word_address(addr)]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 50_000);
}

#[test]
fn cache_program_unauthorized_burns_all_v31() {
    let addr: Address = address!("00000000000000000000000000000000000000cc");
    let run = fixture(31).gas(50_000).call(
        arbwasmcache,
        &calldata("cacheProgram(address)", &[word_address(addr)]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 50_000);
}

#[test]
fn evict_codehash_unauthorized_burns_all_v30() {
    let hash = B256::repeat_byte(0xef);
    let run = fixture(30).gas(50_000).call(
        arbwasmcache,
        &codehash_calldata("evictCodehash(bytes32)", hash),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 50_000);
}
