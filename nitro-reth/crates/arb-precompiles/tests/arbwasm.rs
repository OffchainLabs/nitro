mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, B256, U256};
use arb_precompiles::create_arbwasm_precompile;
use arb_storage::{
    layout::{
        derive_subspace_key, map_slot, map_slot_b256,
        programs::{PARAMS_KEY, PROGRAM_DATA_KEY},
        PROGRAMS_SUBSPACE, ROOT_STORAGE_KEY,
    },
    ARBOS_STATE_ADDRESS,
};
use common::{calldata, decode_u256, decode_word, word_address, PrecompileTest};
use revm::state::AccountInfo;

const ARBOS_V30: u64 = 30;
const ARBOS_V32: u64 = 32; // StylusChargingFixes

fn arbwasm(ctx: std::sync::Arc<arb_context::ArbPrecompileCtx>) -> DynPrecompile {
    create_arbwasm_precompile(ctx)
}

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
        put(&mut buf, &mut i, &ink[1..4]);
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
        min_init_gas: 69,
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
    let params_key = derive_subspace_key(programs_key.as_slice(), PARAMS_KEY);
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
    let run = PrecompileTest::new()
        .arbos_version(29)
        .arbos_state()
        .call(arbwasm, &calldata("stylusVersion()", &[]));
    let out = run.assert_ok();
    assert!(out.bytes.is_empty());
}

#[test]
fn stylus_version_returns_packed_field() {
    let mut p = default_params();
    p.version = 7;
    let run = test_with(p, ARBOS_V30).call(arbwasm, &calldata("stylusVersion()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(7));
}

#[test]
fn ink_price_returns_packed_uint24() {
    let mut p = default_params();
    p.ink_price = 0x123_456; // requires the full 24 bits
    let run = test_with(p, ARBOS_V30).call(arbwasm, &calldata("inkPrice()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(0x123_456_u64));
}

#[test]
fn max_stack_depth_returns_packed_field() {
    let mut p = default_params();
    p.max_stack_depth = 0x1234_5678;
    let run = test_with(p, ARBOS_V30).call(arbwasm, &calldata("maxStackDepth()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(0x1234_5678_u64));
}

#[test]
fn free_pages_returns_packed_field() {
    let mut p = default_params();
    p.free_pages = 17;
    let run = test_with(p, ARBOS_V30).call(arbwasm, &calldata("freePages()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(17));
}

#[test]
fn page_gas_returns_packed_field() {
    let mut p = default_params();
    p.page_gas = 4242;
    let run = test_with(p, ARBOS_V30).call(arbwasm, &calldata("pageGas()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(4242));
}

#[test]
fn page_ramp_returns_initial_constant() {
    let run = test_with(default_params(), ARBOS_V30).call(arbwasm, &calldata("pageRamp()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(620_674_314_u64));
}

#[test]
fn page_limit_returns_packed_field() {
    let mut p = default_params();
    p.page_limit = 256;
    let run = test_with(p, ARBOS_V30).call(arbwasm, &calldata("pageLimit()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(256));
}

#[test]
fn min_init_gas_returns_units_multiplied() {
    let mut p = default_params();
    p.min_init_gas = 7;
    p.min_cached_init_gas = 9;
    let run = test_with(p, ARBOS_V32).call(arbwasm, &calldata("minInitGas()", &[]));
    let out = run.output();
    assert_eq!(decode_word(out, 0), common::word_u64(7 * 128));
    assert_eq!(decode_word(out, 1), common::word_u64(9 * 32));
}

#[test]
fn min_init_gas_reverts_pre_charging_fixes() {
    let p = default_params();
    let gas = 100_000_u64;
    let run = test_with(p, 31)
        .gas(gas)
        .call(arbwasm, &calldata("minInitGas()", &[]));
    let out = run.assert_ok();
    assert!(out.reverted);
    // OpenArbosState SLOAD (800) + warm Params SLOAD (100) = 900.
    assert_eq!(out.gas_used, 900);
}

#[test]
fn init_cost_scalar_returns_field_times_percent() {
    let mut p = default_params();
    p.init_cost_scalar = 50;
    let run = test_with(p, ARBOS_V30).call(arbwasm, &calldata("initCostScalar()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(50 * 2));
}

#[test]
fn expiry_days_returns_packed_field() {
    let mut p = default_params();
    p.expiry_days = 365;
    let run = test_with(p, ARBOS_V30).call(arbwasm, &calldata("expiryDays()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(365));
}

#[test]
fn keepalive_days_returns_packed_field() {
    let mut p = default_params();
    p.keepalive_days = 31;
    let run = test_with(p, ARBOS_V30).call(arbwasm, &calldata("keepaliveDays()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(31));
}

#[test]
fn block_cache_size_returns_packed_field() {
    let mut p = default_params();
    p.block_cache_size = 32;
    let run = test_with(p, ARBOS_V30).call(arbwasm, &calldata("blockCacheSize()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(32));
}

#[test]
fn full_round_trip_packs_and_unpacks_all_fields() {
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
            let run = test_with(p, ARBOS_V32).call(arbwasm, &calldata($sig, &[]));
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

    let run = test_with(p, ARBOS_V32).call(arbwasm, &calldata("minInitGas()", &[]));
    let out = run.output();
    assert_eq!(decode_word(out, 0), common::word_u64(0x61_u64 * 128));
    assert_eq!(decode_word(out, 1), common::word_u64(0x71_u64 * 32));
}

// ── validate_active_program error paths (Nitro programs.go::getActiveProgram) ──

const ARBITRUM_START_TIME: u64 = 1_421_388_000;

fn hours_since_start(time: u64) -> u32 {
    ((time.saturating_sub(ARBITRUM_START_TIME)) / 3600) as u32
}

/// Pack a Program word matching `programs.go::setProgram`:
///   [0..2]  version       uint16
///   [2..4]  init_cost     uint16
///   [4..6]  cached_cost   uint16
///   [6..8]  footprint     uint16
///   [8..11] activated_at  uint24 (hours since Arbitrum epoch)
///   [11..14] asm_estimate_kb uint24
///   [14]    cached        bool
fn pack_program(version: u16, footprint: u16, activated_at_hours: u32) -> U256 {
    let mut buf = [0u8; 32];
    buf[0..2].copy_from_slice(&version.to_be_bytes());
    buf[6..8].copy_from_slice(&footprint.to_be_bytes());
    buf[8] = (activated_at_hours >> 16) as u8;
    buf[9] = (activated_at_hours >> 8) as u8;
    buf[10] = activated_at_hours as u8;
    U256::from_be_bytes(buf)
}

fn program_data_slot(codehash: B256) -> U256 {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let data_key = derive_subspace_key(programs_key.as_slice(), PROGRAM_DATA_KEY);
    map_slot_b256(data_key.as_slice(), &codehash)
}

#[test]
fn codehash_version_reverts_program_not_activated_for_unset_program() {
    let codehash = B256::from_slice(&[0x42u8; 32]);
    let run = test_with(default_params(), ARBOS_V32)
        .call(arbwasm, &calldata("codehashVersion(bytes32)", &[codehash]));
    let out = run.assert_ok();
    assert!(out.reverted);
    let sel = alloy_primitives::keccak256(b"ProgramNotActivated()");
    assert_eq!(&out.bytes[..4], &sel[..4]);
    // lookup(1703) + 1-word error(3) = 1706.
    assert_eq!(out.gas_used, 1706);
}

#[test]
fn codehash_version_reverts_program_needs_upgrade_for_stale_version() {
    // Params at v=2, program at v=1 -> ProgramNeedsUpgrade(1, 2).
    let codehash = B256::from_slice(&[0x33u8; 32]);
    let now = 1_700_000_000;
    let prog_word = pack_program(1, 3, hours_since_start(now));
    let test = test_with(default_params(), ARBOS_V32)
        .block_timestamp(now)
        .storage(ARBOS_STATE_ADDRESS, program_data_slot(codehash), prog_word);
    let run = test.call(arbwasm, &calldata("codehashVersion(bytes32)", &[codehash]));
    let out = run.assert_ok();
    assert!(out.reverted, "must revert");
    let sel = alloy_primitives::keccak256(b"ProgramNeedsUpgrade(uint16,uint16)");
    assert_eq!(&out.bytes[..4], &sel[..4]);
    let prog_v = U256::from_be_slice(&out.bytes[4..36]);
    let params_v = U256::from_be_slice(&out.bytes[36..68]);
    assert_eq!(prog_v, U256::from(1u64));
    assert_eq!(params_v, U256::from(default_params().version));
    // lookup(1703) + 3-word error(9) = 1712.
    assert_eq!(out.gas_used, 1712);
}

#[test]
fn codehash_version_reverts_program_expired_after_expiry() {
    // expiry_days default 365 -> 31_536_000 seconds.
    // Activate at hour 0 since the Arbitrum epoch and run 366 days later.
    let codehash = B256::from_slice(&[0x77u8; 32]);
    let now = ARBITRUM_START_TIME + 366 * 86_400;
    let prog_word = pack_program(default_params().version, 3, 0);
    let test = test_with(default_params(), ARBOS_V32)
        .block_timestamp(now)
        .storage(ARBOS_STATE_ADDRESS, program_data_slot(codehash), prog_word);
    let run = test.call(arbwasm, &calldata("codehashVersion(bytes32)", &[codehash]));
    let out = run.assert_ok();
    assert!(out.reverted);
    let sel = alloy_primitives::keccak256(b"ProgramExpired(uint64)");
    assert_eq!(&out.bytes[..4], &sel[..4]);
    let age = U256::from_be_slice(&out.bytes[4..36]);
    assert_eq!(age, U256::from(366u64 * 86_400));
    // lookup(1703) + 2-word error(6) = 1709.
    assert_eq!(out.gas_used, 1709);
}

#[test]
fn codehash_version_returns_active_version_for_fresh_program() {
    let codehash = B256::from_slice(&[0xa1u8; 32]);
    let now = 1_700_000_000;
    let prog_word = pack_program(default_params().version, 5, hours_since_start(now - 86_400));
    let test = test_with(default_params(), ARBOS_V32)
        .block_timestamp(now)
        .storage(ARBOS_STATE_ADDRESS, program_data_slot(codehash), prog_word);
    let run = test.call(arbwasm, &calldata("codehashVersion(bytes32)", &[codehash]));
    assert_eq!(
        decode_u256(run.output()),
        U256::from(default_params().version)
    );
}

#[test]
fn program_memory_footprint_returns_packed_value() {
    let codehash = B256::from_slice(&[0xb2u8; 32]);
    let now = 1_700_000_000;
    let prog_word = pack_program(default_params().version, 7, hours_since_start(now));
    let prog_addr = address!("00000000000000000000000000000000000000aa");
    let test = test_with(default_params(), ARBOS_V32)
        .block_timestamp(now)
        .account(
            prog_addr,
            AccountInfo {
                code_hash: codehash,
                ..Default::default()
            },
        )
        .storage(ARBOS_STATE_ADDRESS, program_data_slot(codehash), prog_word);
    let run = test.call(
        arbwasm,
        &calldata(
            "programMemoryFootprint(address)",
            &[word_address(prog_addr)],
        ),
    );
    assert_eq!(decode_u256(run.output()), U256::from(7u64));
}

// ── More query coverage ──────────────────────────────────────────────

/// Pack a Program word with explicit init_cost and cached_cost fields.
fn pack_program_full(
    version: u16,
    init_cost: u16,
    cached_cost: u16,
    footprint: u16,
    activated_at_hours: u32,
    asm_estimate_kb: u32,
) -> U256 {
    let mut buf = [0u8; 32];
    buf[0..2].copy_from_slice(&version.to_be_bytes());
    buf[2..4].copy_from_slice(&init_cost.to_be_bytes());
    buf[4..6].copy_from_slice(&cached_cost.to_be_bytes());
    buf[6..8].copy_from_slice(&footprint.to_be_bytes());
    buf[8] = (activated_at_hours >> 16) as u8;
    buf[9] = (activated_at_hours >> 8) as u8;
    buf[10] = activated_at_hours as u8;
    buf[11] = (asm_estimate_kb >> 16) as u8;
    buf[12] = (asm_estimate_kb >> 8) as u8;
    buf[13] = asm_estimate_kb as u8;
    U256::from_be_bytes(buf)
}

#[test]
fn program_init_gas_returns_init_and_cached_costs() {
    // Mirrors Nitro Programs.ProgramInitGas:
    //   init   = init_cost * scalar * 2 / 100 + min_init_gas * 128
    //   cached = cached_cost * cached_scalar * 2 / 100 + min_cached * 32
    //   if params.Version > 1 { init += cached }
    let codehash = B256::from_slice(&[0xc1u8; 32]);
    let now = 1_700_000_000;
    let prog_word = pack_program_full(
        default_params().version, // 2
        100,                      // init_cost
        50,                       // cached_cost
        3,                        // footprint
        hours_since_start(now),
        0,
    );
    let prog_addr = address!("00000000000000000000000000000000000000bc");
    let test = test_with(default_params(), ARBOS_V32)
        .block_timestamp(now)
        .account(
            prog_addr,
            AccountInfo {
                code_hash: codehash,
                ..Default::default()
            },
        )
        .storage(ARBOS_STATE_ADDRESS, program_data_slot(codehash), prog_word);
    let run = test.call(
        arbwasm,
        &calldata("programInitGas(address)", &[word_address(prog_addr)]),
    );
    let out = run.output();

    // default_params(): min_init_gas=69, min_cached_init_gas=11,
    // init_cost_scalar=50, cached_cost_scalar=50.
    let init_base = 69u64 * 128;
    let init_dyno = 100u64 * 50 * 2; // = 10_000
    let init_dyno_div_ceil = init_dyno.div_ceil(100); // = 100
    let cached_base = 11u64 * 32;
    let cached_dyno = 50u64 * 50 * 2; // = 5_000
    let cached_dyno_div_ceil = cached_dyno.div_ceil(100); // = 50
    let cached = cached_base + cached_dyno_div_ceil;
    let init = init_base + init_dyno_div_ceil + cached; // version > 1, so + cached

    assert_eq!(decode_word(out, 0), common::word_u64(init));
    assert_eq!(decode_word(out, 1), common::word_u64(cached));
}

#[test]
fn codehash_asm_size_returns_kb_times_1024() {
    let codehash = B256::from_slice(&[0xc2u8; 32]);
    let now = 1_700_000_000;
    let prog_word = pack_program_full(
        default_params().version,
        0,
        0,
        0,
        hours_since_start(now),
        7, // asm_estimate_kb
    );
    let test = test_with(default_params(), ARBOS_V32)
        .block_timestamp(now)
        .storage(ARBOS_STATE_ADDRESS, program_data_slot(codehash), prog_word);
    let run = test.call(arbwasm, &calldata("codehashAsmSize(bytes32)", &[codehash]));
    assert_eq!(decode_u256(run.output()), U256::from(7u64 * 1024));
}

#[test]
fn program_time_left_returns_expiry_seconds_minus_age() {
    // Use hour-aligned timestamps so hours_to_age math is lossless, then we can
    // compute the exact expected time_left.
    let codehash = B256::from_slice(&[0xc3u8; 32]);
    let activated_hours: u32 = 1_000_000;
    let activated_unix = ARBITRUM_START_TIME + (activated_hours as u64) * 3600;
    let now = activated_unix + 24 * 3600; // exactly 1 day later
    let prog_word = pack_program(default_params().version, 3, activated_hours);
    let prog_addr = address!("00000000000000000000000000000000000000c3");
    let test = test_with(default_params(), ARBOS_V32)
        .block_timestamp(now)
        .account(
            prog_addr,
            AccountInfo {
                code_hash: codehash,
                ..Default::default()
            },
        )
        .storage(ARBOS_STATE_ADDRESS, program_data_slot(codehash), prog_word);
    let run = test.call(
        arbwasm,
        &calldata("programTimeLeft(address)", &[word_address(prog_addr)]),
    );
    let expected = 365u64 * 86_400 - 86_400;
    assert_eq!(decode_u256(run.output()), U256::from(expected));
}

#[test]
fn codehash_asm_size_revert_charges_canonical_gas() {
    // Pin against the canonical receipt for tx 0x08b6a928 at Sepolia block
    // 109,336,195: revert must cost SLOAD + WARM + SLOAD + 2*COPY = 1706.
    let codehash = B256::from_slice(&[0xeeu8; 32]);
    let run = test_with(default_params(), ARBOS_V32)
        .call(arbwasm, &calldata("codehashAsmSize(bytes32)", &[codehash]));
    let out = run.assert_ok();
    assert!(out.reverted);
    let sel = alloy_primitives::keccak256(b"ProgramNotActivated()");
    assert_eq!(&out.bytes[..4], &sel[..4]);
    assert_eq!(out.gas_used, 1706);
}

#[test]
fn program_version_revert_charges_canonical_gas() {
    let codehash = B256::from_slice(&[0xefu8; 32]);
    let prog_addr = address!("00000000000000000000000000000000000000ef");
    let run = test_with(default_params(), ARBOS_V32)
        .account(
            prog_addr,
            AccountInfo {
                code_hash: codehash,
                ..Default::default()
            },
        )
        .call(
            arbwasm,
            &calldata("programVersion(address)", &[word_address(prog_addr)]),
        );
    let out = run.assert_ok();
    assert!(out.reverted);
    let sel = alloy_primitives::keccak256(b"ProgramNotActivated()");
    assert_eq!(&out.bytes[..4], &sel[..4]);
    // Open(800) + argsCost(3) + Params warm(100) + GetCodeHash(2600) + getProgram(800) + result(3)
    assert_eq!(out.gas_used, 4306);
}

#[test]
fn program_version_returns_program_version_for_fresh_program() {
    let codehash = B256::from_slice(&[0xc4u8; 32]);
    let now = 1_700_000_000;
    let prog_word = pack_program(default_params().version, 1, hours_since_start(now));
    let prog_addr = address!("00000000000000000000000000000000000000c4");
    let test = test_with(default_params(), ARBOS_V32)
        .block_timestamp(now)
        .account(
            prog_addr,
            AccountInfo {
                code_hash: codehash,
                ..Default::default()
            },
        )
        .storage(ARBOS_STATE_ADDRESS, program_data_slot(codehash), prog_word);
    let run = test.call(
        arbwasm,
        &calldata("programVersion(address)", &[word_address(prog_addr)]),
    );
    assert_eq!(
        decode_u256(run.output()),
        U256::from(default_params().version)
    );
}

// ── codehashKeepalive value-source regression ─────────────────────────
//
// Pre-fix, the handler read the activation value from STYLUS_CALL_VALUE — a
// thread-local that `build.rs` only sets when the outer tx target IS 0x71.
// When a Solidity contract makes a nested call into ArbWasm.codehashKeepalive
// (the same shape that breaks ArbWasm.activateProgram in Sepolia block
// 169,854,826), the thread-local stays at 0 and the precompile reverts with
// `ProgramInsufficientValue(0, data_fee)` even though revm has already
// transferred the value into 0x71 and exposed it as `input.value`.
//
// The fix prefers `input.value` (call-frame value) over the thread-local,
// then mirrors Nitro's `payActivationDataFee` for the success path. These
// tests pin the symptom in both directions.

fn data_pricer_slot(offset: u64) -> U256 {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let pricer_key = derive_subspace_key(programs_key.as_slice(), arbos::programs::DATA_PRICER_KEY);
    map_slot(pricer_key.as_slice(), offset)
}

fn keepalive_test_setup_at(codehash: B256, now: u64, arbos_version: u64) -> PrecompileTest {
    let mut params = default_params();
    params.version = 1;
    params.keepalive_days = 30;
    params.expiry_days = 365;

    let activated_at_hours = hours_since_start(now - 40 * 86_400);
    let prog_word = pack_program_full(1, 0, 0, 0, activated_at_hours, 5);

    test_with(params, arbos_version)
        .block_timestamp(now)
        .storage(ARBOS_STATE_ADDRESS, program_data_slot(codehash), prog_word)
        .storage(ARBOS_STATE_ADDRESS, data_pricer_slot(0), U256::ZERO)
        .storage(ARBOS_STATE_ADDRESS, data_pricer_slot(1), U256::ZERO)
        .storage(ARBOS_STATE_ADDRESS, data_pricer_slot(2), U256::from(now))
        .storage(
            ARBOS_STATE_ADDRESS,
            data_pricer_slot(3),
            U256::from(82_928_201_u64),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            data_pricer_slot(4),
            U256::from(21_360_419_u64),
        )
}

fn keepalive_test_setup(codehash: B256, now: u64) -> PrecompileTest {
    keepalive_test_setup_at(codehash, now, ARBOS_V32)
}

#[test]
fn keepalive_inner_call_with_value_passes() {
    // Reset the thread-local that `build.rs` would normally manage; in
    // unit tests it stays set across runs in the same process.

    let codehash = B256::from_slice(&[0xdeu8; 32]);
    let now = 1_700_000_000_u64;
    // input.value = 1 ETH, far above the ~0.0004 ETH data_fee for a 5KB
    // program. Simulates revm's value-transfer-into-0x71 for an inner CALL.
    let test = keepalive_test_setup(codehash, now).value(U256::from(10u128).pow(U256::from(18u64)));

    let run = test.call(
        arbwasm,
        &calldata("codehashKeepalive(bytes32)", &[codehash]),
    );

    let out = run.assert_ok();
    if out.reverted {
        let sel = &out.bytes[..4];
        let insufficient =
            alloy_primitives::keccak256(b"ProgramInsufficientValue(uint256,uint256)");
        if sel == &insufficient[..4] {
            panic!(
                "BUG: keepalive reverted with ProgramInsufficientValue even though \
                 input.value = 1 ETH was passed via inner CALL. Pre-fix arbwasm.rs \
                 read STYLUS_CALL_VALUE (thread-local, only set for outer-EOA→0x71); \
                 fix should read input.value instead."
            );
        }
        panic!(
            "keepalive reverted unexpectedly (selector {:02x?}); inner-call value path broken",
            sel
        );
    }
}

#[test]
fn keepalive_inner_call_with_zero_value_still_reverts() {
    let codehash = B256::from_slice(&[0xdfu8; 32]);
    let now = 1_700_000_000_u64;
    let test = keepalive_test_setup(codehash, now).value(U256::ZERO);

    let run = test.call(
        arbwasm,
        &calldata("codehashKeepalive(bytes32)", &[codehash]),
    );

    let out = run.assert_ok();
    assert!(out.reverted, "expected revert with value=0");
    let sel = alloy_primitives::keccak256(b"ProgramInsufficientValue(uint256,uint256)");
    assert_eq!(
        &out.bytes[..4],
        &sel[..4],
        "expected ProgramInsufficientValue revert selector"
    );
}

#[test]
fn keepalive_inner_call_value_passes_at_arbos_v40() {
    // Sepolia at block 169,854,826 is at ArbOS v40 (Prague-era), same as
    // 149,889,087 and 152,429,039. Run the same check at that version to
    // confirm the fix is not version-gated.

    let codehash = B256::from_slice(&[0xe0u8; 32]);
    let now = 1_700_000_000_u64;
    let test =
        keepalive_test_setup_at(codehash, now, 40).value(U256::from(10u128).pow(U256::from(18u64)));

    let run = test.call(
        arbwasm,
        &calldata("codehashKeepalive(bytes32)", &[codehash]),
    );

    let out = run.assert_ok();
    if out.reverted {
        let sel = &out.bytes[..4];
        let insufficient =
            alloy_primitives::keccak256(b"ProgramInsufficientValue(uint256,uint256)");
        if sel == &insufficient[..4] {
            panic!(
                "BUG: keepalive at ArbOS v40 reverted with ProgramInsufficientValue \
                 even though input.value = 1 ETH was passed via inner CALL"
            );
        }
        panic!("unexpected revert at v40: selector {:02x?}", sel);
    }
}

#[test]
fn keepalive_inner_call_value_passes_at_arbos_v50() {
    // Belt-and-braces: also exercise the Dia-era v50 (next upgrade after
    // v40). Same expectation: the fix is structural, not version-gated.

    let codehash = B256::from_slice(&[0xe1u8; 32]);
    let now = 1_700_000_000_u64;
    let test =
        keepalive_test_setup_at(codehash, now, 50).value(U256::from(10u128).pow(U256::from(18u64)));

    let run = test.call(
        arbwasm,
        &calldata("codehashKeepalive(bytes32)", &[codehash]),
    );

    let out = run.assert_ok();
    if out.reverted {
        let sel = &out.bytes[..4];
        let insufficient =
            alloy_primitives::keccak256(b"ProgramInsufficientValue(uint256,uint256)");
        if sel == &insufficient[..4] {
            panic!(
                "BUG: keepalive at ArbOS v50 reverted with ProgramInsufficientValue \
                 even though input.value = 1 ETH was passed via inner CALL"
            );
        }
        panic!("unexpected revert at v50: selector {:02x?}", sel);
    }
}
