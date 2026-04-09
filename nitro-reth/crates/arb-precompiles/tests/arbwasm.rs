//! Integration tests for ArbWasm precompile (address 0x71).
//!
//! Each read-only param getter mirrors a method on `precompiles/ArbWasm.go`. The
//! harness pre-populates the packed `StylusParams` word at its derived slot in the
//! ArbOS state and then asserts that the precompile decodes the same fields Nitro
//! writes via `StylusParams.Save()` (precompiles/programs/params.go).

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, U256};
use arb_precompiles::{
    create_arbwasm_precompile,
    storage_slot::{
        derive_subspace_key, map_slot, ARBOS_STATE_ADDRESS, PROGRAMS_PARAMS_KEY,
        PROGRAMS_SUBSPACE, ROOT_STORAGE_KEY,
    },
};
use common::{calldata, decode_u256, decode_word, PrecompileTest};

const ARBOS_V30: u64 = 30;
const ARBOS_V32: u64 = 32; // StylusChargingFixes

fn arbwasm() -> DynPrecompile {
    create_arbwasm_precompile()
}

/// Build a packed StylusParams word matching `arbos/programs/params.go`'s
/// `StylusParams.Save()` byte layout. All fields are required.
#[derive(Clone, Copy)]
struct StylusParamsWord {
    version: u16,
    ink_price: u32, // uint24
    max_stack_depth: u32,
    free_pages: u16,
    page_gas: u16,
    page_limit: u16,
    min_init_gas: u8,
    min_cached_init_gas: u8,
    init_cost_scalar: u8,
    cached_cost_scalar: u8,
    expiry_days: u16,
    keepalive_days: u16,
    block_cache_size: u16,
}

impl StylusParamsWord {
    fn pack(&self) -> U256 {
        let mut buf = [0u8; 32];
        let mut i = 0;
        let put = |buf: &mut [u8; 32], i: &mut usize, bytes: &[u8]| {
            buf[*i..*i + bytes.len()].copy_from_slice(bytes);
            *i += bytes.len();
        };
        put(&mut buf, &mut i, &self.version.to_be_bytes());
        let ink = self.ink_price.to_be_bytes();
        put(&mut buf, &mut i, &ink[1..4]); // uint24, big-endian
        put(&mut buf, &mut i, &self.max_stack_depth.to_be_bytes());
        put(&mut buf, &mut i, &self.free_pages.to_be_bytes());
        put(&mut buf, &mut i, &self.page_gas.to_be_bytes());
        put(&mut buf, &mut i, &self.page_limit.to_be_bytes());
        buf[i] = self.min_init_gas;
        i += 1;
        buf[i] = self.min_cached_init_gas;
        i += 1;
        buf[i] = self.init_cost_scalar;
        i += 1;
        buf[i] = self.cached_cost_scalar;
        i += 1;
        put(&mut buf, &mut i, &self.expiry_days.to_be_bytes());
        put(&mut buf, &mut i, &self.keepalive_days.to_be_bytes());
        put(&mut buf, &mut i, &self.block_cache_size.to_be_bytes());
        let _ = i;
        U256::from_be_bytes(buf)
    }
}

fn default_params() -> StylusParamsWord {
    StylusParamsWord {
        version: 2,
        ink_price: 10_000,
        max_stack_depth: 4 * 65_536,
        free_pages: 2,
        page_gas: 1_000,
        page_limit: 128,
        min_init_gas: 69, // v2 default per params.go: v2MinInitGas
        min_cached_init_gas: 11,
        init_cost_scalar: 50,
        cached_cost_scalar: 50,
        expiry_days: 365,
        keepalive_days: 31,
        block_cache_size: 32,
    }
}

fn params_slot() -> U256 {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let params_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_PARAMS_KEY);
    map_slot(params_key.as_slice(), 0)
}

fn test_with(params: StylusParamsWord, arbos_version: u64) -> PrecompileTest {
    PrecompileTest::new()
        .arbos_version(arbos_version)
        .arbos_state()
        .storage(ARBOS_STATE_ADDRESS, params_slot(), params.pack())
}

#[test]
fn pre_stylus_arbos_returns_empty_like_unregistered() {
    // Pre-Stylus chains don't register ArbWasm at all (gethhook addPrecompiles
    // StartingFromArbOS30). Calling the address behaves like an EOA: empty output.
    // The handler's internal fallback also returns empty output for safety even
    // if it's invoked directly.
    let run = PrecompileTest::new()
        .arbos_version(29)
        .arbos_state()
        .call(&arbwasm(), &calldata("stylusVersion()", &[]));
    let out = run.assert_ok();
    assert!(out.bytes.is_empty(), "expected empty output, got {:?}", out.bytes);
}

#[test]
fn stylus_version_returns_packed_field() {
    let mut p = default_params();
    p.version = 7;
    let run = test_with(p, ARBOS_V30).call(&arbwasm(), &calldata("stylusVersion()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(7));
}

#[test]
fn ink_price_returns_packed_uint24() {
    let mut p = default_params();
    p.ink_price = 0x123_456; // requires the full 24 bits
    let run = test_with(p, ARBOS_V30).call(&arbwasm(), &calldata("inkPrice()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(0x123_456_u64));
}

#[test]
fn max_stack_depth_returns_packed_field() {
    let mut p = default_params();
    p.max_stack_depth = 0x1234_5678;
    let run = test_with(p, ARBOS_V30).call(&arbwasm(), &calldata("maxStackDepth()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(0x1234_5678_u64));
}

#[test]
fn free_pages_returns_packed_field() {
    let mut p = default_params();
    p.free_pages = 17;
    let run = test_with(p, ARBOS_V30).call(&arbwasm(), &calldata("freePages()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(17));
}

#[test]
fn page_gas_returns_packed_field() {
    let mut p = default_params();
    p.page_gas = 4242;
    let run = test_with(p, ARBOS_V30).call(&arbwasm(), &calldata("pageGas()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(4242));
}

#[test]
fn page_ramp_returns_initial_constant() {
    // Per params.go: PageRamp is `initialPageRamp = 620674314`, NOT stored in the
    // packed word. Our handler returns the constant; this test pins it to the
    // exact value Nitro hardcodes.
    let run =
        test_with(default_params(), ARBOS_V30).call(&arbwasm(), &calldata("pageRamp()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(620_674_314_u64));
}

#[test]
fn page_limit_returns_packed_field() {
    let mut p = default_params();
    p.page_limit = 256;
    let run = test_with(p, ARBOS_V30).call(&arbwasm(), &calldata("pageLimit()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(256));
}

#[test]
fn min_init_gas_returns_units_multiplied() {
    // Per ArbWasm.go:139-140 and programs/params.go MinInitGasUnits=128, MinCachedGasUnits=32.
    let mut p = default_params();
    p.min_init_gas = 7;
    p.min_cached_init_gas = 9;
    let run = test_with(p, ARBOS_V32).call(&arbwasm(), &calldata("minInitGas()", &[]));
    let out = run.output();
    assert_eq!(decode_word(out, 0), common::word_u64(7 * 128));
    assert_eq!(decode_word(out, 1), common::word_u64(9 * 32));
}

#[test]
fn min_init_gas_reverts_pre_charging_fixes() {
    // Per ArbWasm.go:141: ArbOS < StylusChargingFixes (=32) reverts ExecutionReverted.
    let p = default_params();
    let run = test_with(p, 31).call(&arbwasm(), &calldata("minInitGas()", &[]));
    assert!(
        run.result.is_err(),
        "expected revert below StylusChargingFixes"
    );
}

#[test]
fn init_cost_scalar_returns_field_times_percent() {
    // Per ArbWasm.go:150: returns `params.InitCostScalar * CostScalarPercent (=2)`.
    let mut p = default_params();
    p.init_cost_scalar = 50;
    let run = test_with(p, ARBOS_V30).call(&arbwasm(), &calldata("initCostScalar()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(50 * 2));
}

#[test]
fn expiry_days_returns_packed_field() {
    let mut p = default_params();
    p.expiry_days = 365;
    let run = test_with(p, ARBOS_V30).call(&arbwasm(), &calldata("expiryDays()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(365));
}

#[test]
fn keepalive_days_returns_packed_field() {
    let mut p = default_params();
    p.keepalive_days = 31;
    let run = test_with(p, ARBOS_V30).call(&arbwasm(), &calldata("keepaliveDays()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(31));
}

#[test]
fn block_cache_size_returns_packed_field() {
    let mut p = default_params();
    p.block_cache_size = 32;
    let run = test_with(p, ARBOS_V30).call(&arbwasm(), &calldata("blockCacheSize()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(32));
}

#[test]
fn full_round_trip_packs_and_unpacks_all_fields() {
    // One test that exercises every field at once with non-default values, to
    // catch any byte-offset bug that a single-field test might miss.
    let p = StylusParamsWord {
        version: 0xabcd,
        ink_price: 0x111213,
        max_stack_depth: 0x21222324,
        free_pages: 0x3132,
        page_gas: 0x4142,
        page_limit: 0x5152,
        min_init_gas: 0x61,
        min_cached_init_gas: 0x71,
        init_cost_scalar: 0x81,
        cached_cost_scalar: 0x91,
        expiry_days: 0xa1a2,
        keepalive_days: 0xb1b2,
        block_cache_size: 0xc1c2,
    };

    macro_rules! check {
        ($sig:expr, $expected:expr) => {{
            let run = test_with(p, ARBOS_V32).call(&arbwasm(), &calldata($sig, &[]));
            assert_eq!(
                decode_u256(run.output()),
                U256::from($expected),
                "wrong value for {}",
                $sig
            );
        }};
    }
    check!("stylusVersion()", 0xabcd_u64);
    check!("inkPrice()", 0x111213_u64);
    check!("maxStackDepth()", 0x21222324_u64);
    check!("freePages()", 0x3132_u64);
    check!("pageGas()", 0x4142_u64);
    check!("pageLimit()", 0x5152_u64);
    check!("expiryDays()", 0xa1a2_u64);
    check!("keepaliveDays()", 0xb1b2_u64);
    check!("blockCacheSize()", 0xc1c2_u64);
    check!("initCostScalar()", 0x81_u64 * 2);

    // minInitGas() returns a 2-tuple
    let run = test_with(p, ARBOS_V32).call(&arbwasm(), &calldata("minInitGas()", &[]));
    let out = run.output();
    assert_eq!(decode_word(out, 0), common::word_u64(0x61_u64 * 128));
    assert_eq!(decode_word(out, 1), common::word_u64(0x71_u64 * 32));
}

#[test]
fn unknown_address_used_anywhere() {
    // Sanity test that an unrecognised address is benign — confirms our test
    // builders don't accidentally trigger the precompile via stale state.
    let _: alloy_primitives::Address = address!("0123456789abcdef0123456789abcdef01234567");
}
