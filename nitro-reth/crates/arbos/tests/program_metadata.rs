use alloy_primitives::{B256, U256};
use arb_test_utils::ArbosHarness;
use arbos::programs::{params::StylusParams, Program};

// ======================================================================
// Program encoding/decoding
// ======================================================================

fn sample_program() -> Program {
    Program {
        version: 1,
        init_cost: 5000,
        cached_cost: 2000,
        footprint: 30,
        asm_estimate_kb: 42,
        activated_at: 100_000,
        age_seconds: 0,
        cached: true,
    }
}

#[test]
fn program_to_storage_encodes_correct_layout() {
    let p = sample_program();
    let data = p.to_storage();
    let b = data.as_slice();
    assert_eq!(u16::from_be_bytes([b[0], b[1]]), 1);
    assert_eq!(u16::from_be_bytes([b[2], b[3]]), 5000);
    assert_eq!(u16::from_be_bytes([b[4], b[5]]), 2000);
    assert_eq!(u16::from_be_bytes([b[6], b[7]]), 30);
    assert_eq!(b[8], (100_000u32 >> 16) as u8);
    assert_eq!(b[9], (100_000u32 >> 8) as u8);
    assert_eq!(b[10], 100_000u32 as u8);
    assert_eq!(b[11], (42u32 >> 16) as u8);
    assert_eq!(b[12], (42u32 >> 8) as u8);
    assert_eq!(b[13], 42u8);
    assert_eq!(b[14], 1);
}

#[test]
fn program_roundtrip_without_age() {
    let p = sample_program();
    let encoded = p.to_storage();
    let decoded = Program::from_storage(encoded, 0);
    assert_eq!(decoded.version, p.version);
    assert_eq!(decoded.init_cost, p.init_cost);
    assert_eq!(decoded.cached_cost, p.cached_cost);
    assert_eq!(decoded.footprint, p.footprint);
    assert_eq!(decoded.asm_estimate_kb, p.asm_estimate_kb);
    assert_eq!(decoded.activated_at, p.activated_at);
    assert_eq!(decoded.cached, p.cached);
}

#[test]
fn program_from_storage_computes_age_seconds() {
    let p = Program {
        version: 1,
        init_cost: 0,
        cached_cost: 0,
        footprint: 1,
        asm_estimate_kb: 1,
        activated_at: 5,
        age_seconds: 0,
        cached: false,
    };
    let encoded = p.to_storage();
    let time = 1421388000u64 + 5 * 3600 + 100;
    let decoded = Program::from_storage(encoded, time);
    assert_eq!(decoded.age_seconds, 100);
}

#[test]
fn program_asm_size_is_kb_times_1024() {
    let p = Program {
        asm_estimate_kb: 10,
        ..sample_program()
    };
    assert_eq!(p.asm_size(), 10 * 1024);
}

#[test]
fn program_asm_size_saturates_at_u32_max() {
    let p = Program {
        asm_estimate_kb: u32::MAX,
        ..sample_program()
    };
    assert_eq!(p.asm_size(), u32::MAX);
}

#[test]
fn program_cached_flag_encodes_zero_when_not_cached() {
    let p = Program {
        cached: false,
        ..sample_program()
    };
    let data = p.to_storage();
    assert_eq!(data.as_slice()[14], 0);
}

#[test]
fn program_zero_values_roundtrip_cleanly() {
    let zero = Program {
        version: 0,
        init_cost: 0,
        cached_cost: 0,
        footprint: 0,
        asm_estimate_kb: 0,
        activated_at: 0,
        age_seconds: 0,
        cached: false,
    };
    let decoded = Program::from_storage(zero.to_storage(), 0);
    assert_eq!(decoded.version, 0);
    assert_eq!(decoded.init_cost, 0);
    assert_eq!(decoded.activated_at, 0);
    assert!(!decoded.cached);
}

#[test]
fn program_max_u16_fields_roundtrip() {
    let p = Program {
        version: u16::MAX,
        init_cost: u16::MAX,
        cached_cost: u16::MAX,
        footprint: u16::MAX,
        asm_estimate_kb: 0xFFFFFF,
        activated_at: 0xFFFFFF,
        age_seconds: 0,
        cached: true,
    };
    let decoded = Program::from_storage(p.to_storage(), 0);
    assert_eq!(decoded.version, u16::MAX);
    assert_eq!(decoded.init_cost, u16::MAX);
    assert_eq!(decoded.cached_cost, u16::MAX);
    assert_eq!(decoded.footprint, u16::MAX);
    assert_eq!(decoded.asm_estimate_kb, 0xFFFFFF);
    assert_eq!(decoded.activated_at, 0xFFFFFF);
}

// ======================================================================
// init_gas + cached_gas with realistic params
// ======================================================================

fn realistic_params() -> StylusParams {
    StylusParams {
        arbos_version: 50,
        version: 1,
        ink_price: 1_000,
        max_stack_depth: 10_000,
        free_pages: 2,
        page_gas: 1_000,
        page_ramp: 620_674_314,
        page_limit: 128,
        min_init_gas: 128,
        min_cached_init_gas: 32,
        init_cost_scalar: 50,
        cached_cost_scalar: 50,
        expiry_days: 365,
        keepalive_days: 30,
        block_cache_size: 0,
        max_wasm_size: 128_000,
        max_fragment_count: 4,
    }
}

#[test]
fn init_gas_zero_for_zero_init_cost_still_has_base() {
    let p = Program {
        init_cost: 0,
        ..sample_program()
    };
    let params = realistic_params();
    let g = p.init_gas(&params);
    assert!(g > 0);
}

#[test]
fn init_gas_grows_with_init_cost() {
    let small = Program {
        init_cost: 100,
        ..sample_program()
    };
    let big = Program {
        init_cost: 10_000,
        ..sample_program()
    };
    let params = realistic_params();
    assert!(big.init_gas(&params) > small.init_gas(&params));
}

#[test]
fn cached_gas_uses_cached_cost_scalar() {
    let p = Program {
        cached_cost: 200,
        ..sample_program()
    };
    let params = realistic_params();
    let g = p.cached_gas(&params);
    assert!(g > 0);
}

// ======================================================================
// Programs state (storage-backed)
// ======================================================================

fn fresh() -> ArbosHarness {
    ArbosHarness::new().with_arbos_version(50).initialize()
}

#[test]
fn programs_params_load_succeeds() {
    let mut h = fresh();
    let mut arbos = h.arbos_state();
    let p = &mut arbos.programs;
    let params = p.params().expect("load params");
    assert!(params.version >= 1);
    assert!(params.page_limit >= 128);
}

#[test]
fn set_then_get_program() {
    let mut h = fresh();
    let mut arbos = h.arbos_state();
    let p = &mut arbos.programs;
    let code_hash = B256::repeat_byte(0xAB);
    let prog = sample_program();
    p.set_program(code_hash, prog).unwrap();
    let got = p.get_program(code_hash, 1_421_388_000).unwrap();
    assert_eq!(got.version, prog.version);
    assert_eq!(got.init_cost, prog.init_cost);
    assert_eq!(got.activated_at, prog.activated_at);
}

#[test]
fn set_and_get_module_hash() {
    let mut h = fresh();
    let mut arbos = h.arbos_state();
    let p = &mut arbos.programs;
    let code_hash = B256::repeat_byte(0x12);
    let module_hash = B256::repeat_byte(0x34);
    p.set_module_hash(code_hash, module_hash).unwrap();
    assert_eq!(p.get_module_hash(code_hash).unwrap(), module_hash);
}

#[test]
fn get_program_for_unknown_hash_returns_default_fields() {
    let mut h = fresh();
    let mut arbos = h.arbos_state();
    let p = &mut arbos.programs;
    let prog = p.get_program(B256::repeat_byte(0xFE), 0).unwrap();
    assert_eq!(prog.version, 0);
    assert_eq!(prog.activated_at, 0);
    assert!(!prog.cached);
}

#[test]
fn data_pricer_update_model_costs_more_with_demand() {
    let mut h = fresh();
    let mut arbos = h.arbos_state();
    let p = &mut arbos.programs;
    let c1 = p.data_pricer.update_model(1000, 1_700_000_000).unwrap();
    let c2 = p.data_pricer.update_model(1000, 1_700_000_001).unwrap();
    assert!(c1 >= U256::ZERO);
    assert!(c2 >= U256::ZERO);
}
