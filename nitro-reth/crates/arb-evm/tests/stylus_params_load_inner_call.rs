//! Verifies that the inner-call gas-cost path in `arb-evm` loads
//! `StylusParams` through the canonical `load` reader, including
//! the v40+ `max_wasm_size` field and the v60+ `max_fragment_count` field.
//!
//! Regression test for a latent consensus drift: an earlier local parser in
//! `arb-evm::evm::parse_stylus_params` silently zeroed those two fields,
//! which would mis-cost Stylus sub-calls once the executor reached ArbOS v40+
//! on a Sepolia-style upgrade path.

use arb_precompiles::storage_slot::{
    derive_subspace_key, ARBOS_STATE_ADDRESS, PROGRAMS_PARAMS_KEY, PROGRAMS_SUBSPACE,
    ROOT_STORAGE_KEY,
};
use arb_storage::{Detached, Storage};
use arb_test_utils::ArbosHarness;
use arbos::programs::params::{
    StylusParams, ARBOS_VERSION_40, ARBOS_VERSION_STYLUS_CONTRACT_LIMIT, INITIAL_PAGE_RAMP,
};

/// Build a baseline `StylusParams` value for `arbos_version`. The version-gated
/// fields are left as defaults; tests override the fields they assert on.
fn baseline_params(arbos_version: u64) -> StylusParams {
    StylusParams {
        arbos_version,
        version: 2,
        ink_price: 10_000,
        max_stack_depth: 22_000,
        free_pages: 2,
        page_gas: 1_000,
        page_ramp: INITIAL_PAGE_RAMP,
        page_limit: 128,
        min_init_gas: 69,
        min_cached_init_gas: 11,
        init_cost_scalar: 50,
        cached_cost_scalar: 50,
        expiry_days: 365,
        keepalive_days: 31,
        block_cache_size: 32,
        max_wasm_size: 0,
        max_fragment_count: 0,
    }
}

/// Detached params subspace handle that matches the one constructed by the
/// EVM inner-call dispatch path.
fn params_storage() -> Storage<Detached> {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let params_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_PARAMS_KEY);
    Storage::detached(ARBOS_STATE_ADDRESS, params_key)
}

#[test]
fn load_reads_max_wasm_size_at_v40() {
    let mut h = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION_40)
        .initialize();

    let mut want = baseline_params(ARBOS_VERSION_40);
    want.max_wasm_size = 100_000;
    // max_fragment_count stays 0: not written before v60.

    let sto = params_storage();
    {
        let state = h.state();
        want.save(&sto, state).unwrap();
    }

    let state = h.state();
    let got = StylusParams::load(ARBOS_VERSION_40, &sto, state).unwrap();

    assert_eq!(got.max_wasm_size, 100_000);
    assert_eq!(got.max_fragment_count, 0);
    assert_eq!(got.version, want.version);
    assert_eq!(got.ink_price, want.ink_price);
    assert_eq!(got.max_stack_depth, want.max_stack_depth);
    assert_eq!(got.free_pages, want.free_pages);
    assert_eq!(got.page_gas, want.page_gas);
    assert_eq!(got.page_limit, want.page_limit);
    assert_eq!(got.min_init_gas, want.min_init_gas);
    assert_eq!(got.min_cached_init_gas, want.min_cached_init_gas);
}

#[test]
fn load_reads_max_fragment_count_at_v60() {
    let mut h = ArbosHarness::new()
        .with_arbos_version(ARBOS_VERSION_STYLUS_CONTRACT_LIMIT)
        .initialize();

    let mut want = baseline_params(ARBOS_VERSION_STYLUS_CONTRACT_LIMIT);
    want.max_wasm_size = 256 * 1024;
    want.max_fragment_count = 7;

    let sto = params_storage();
    {
        let state = h.state();
        want.save(&sto, state).unwrap();
    }

    let state = h.state();
    let got = StylusParams::load(ARBOS_VERSION_STYLUS_CONTRACT_LIMIT, &sto, state).unwrap();

    assert_eq!(got.max_wasm_size, 256 * 1024);
    assert_eq!(got.max_fragment_count, 7);
}

#[test]
fn load_returns_initial_max_wasm_size_pre_v40() {
    // Below v40 the loader fills in the initial constant rather than reading
    // it from storage.
    let mut h = ArbosHarness::new().with_arbos_version(30).initialize();

    let sto = params_storage();
    let state = h.state();
    let got = StylusParams::load(30, &sto, state).unwrap();

    assert_eq!(got.max_wasm_size, 128 * 1024);
    assert_eq!(got.max_fragment_count, 0);
}
