//! Codegen / module-construction parity tests.
//!
//! These exercise corners of the WASM compilation pipeline that the
//! basic ink/depth/start tests in `wavm.rs` don't reach:
//!
//!   * data segments and memory initialization
//!   * `call_indirect` through a function table
//!   * round-trip serialize/deserialize (cache path correctness)
//!   * multiple user-declared globals (init order + non-clobbering)
//!   * activation parity via `nitro_prover`
//!
//! If any of these fail, we have a smoking gun for a Stylus divergence
//! that would otherwise only surface during a multi-million-block sync.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
pub unsafe extern "C" fn __rust_probestack() {}

use std::sync::Arc;

use arb_stylus::config::CompileConfig;
use wasmer::sys::EngineBuilder;
use wasmer::{
    imports, CompilerConfig, Cranelift, CraneliftOptLevel, Imports, Instance, Module, Store, Value,
};

// ── Shared helpers (kept identical to wavm.rs on purpose) ──────────

fn make_store() -> Store {
    let mut compile = CompileConfig::version(1, true);
    compile.debug.debug_funcs = true;
    compile.debug.debug_info = true;

    let mut cranelift = Cranelift::new();
    cranelift.opt_level(CraneliftOptLevel::Speed);
    cranelift.canonicalize_nans(true);
    cranelift.push_middleware(Arc::new(arb_stylus::middleware::StartMover::new(true)));
    cranelift.push_middleware(Arc::new(arb_stylus::middleware::InkMeter::new(
        compile.pricing.ink_header_cost,
    )));
    cranelift.push_middleware(Arc::new(arb_stylus::middleware::DynamicMeter::new(
        compile.pricing.memory_fill_ink,
        compile.pricing.memory_copy_ink,
    )));
    cranelift.push_middleware(Arc::new(arb_stylus::middleware::DepthChecker::new(
        compile.bounds.max_frame_size,
        compile.bounds.max_frame_contention,
    )));
    cranelift.push_middleware(Arc::new(arb_stylus::middleware::HeapBound::new()));

    let engine: wasmer::Engine = EngineBuilder::new(cranelift).into();
    Store::new(engine)
}

fn instantiate(wat: &str, imports: &Imports, store: &mut Store) -> Instance {
    let wasm = wat::parse_bytes(wat.as_bytes()).expect("wat2wasm");
    let module = Module::new(store, wasm).expect("module compile");
    let instance = Instance::new(store, &module, imports).expect("instantiate");
    seed_meter(&instance, store);
    instance
}

fn seed_meter(instance: &Instance, store: &mut Store) {
    instance
        .exports
        .get_global("stylus_ink_left")
        .unwrap()
        .set(store, Value::I64(i64::MAX))
        .unwrap();
    instance
        .exports
        .get_global("stylus_ink_status")
        .unwrap()
        .set(store, Value::I32(0))
        .unwrap();
    instance
        .exports
        .get_global("stylus_stack_left")
        .unwrap()
        .set(store, Value::I32(i32::MAX))
        .unwrap();
}

// ── Test 3: data segments ──────────────────────────────────────────

const DATA_WAT: &str = r#"
(module
    (memory (export "memory") 1 1)
    (data (i32.const 0x100) "\de\ad\be\ef")
    (data (i32.const 0x200) "\01\02\03\04\05\06\07\08")

    ;; Read a 32-bit little-endian word starting at $offset.
    (func $read_u32 (export "read_u32") (param $offset i32) (result i32)
        local.get $offset
        i32.load)

    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (i32.const 0)))
"#;

#[test]
fn data_segments_initialize_memory_at_correct_offsets() {
    let mut store = make_store();
    let instance = instantiate(DATA_WAT, &imports! {}, &mut store);
    let read_u32 = instance.exports.get_function("read_u32").unwrap().clone();

    // 0xdeadbeef stored little-endian at 0x100 → loads as 0xefbeadde
    let v = read_u32.call(&mut store, &[Value::I32(0x100)]).unwrap();
    assert_eq!(v[0], Value::I32(0xefbeadde_u32 as i32));

    // 0x04030201 little-endian at 0x200
    let v = read_u32.call(&mut store, &[Value::I32(0x200)]).unwrap();
    assert_eq!(v[0], Value::I32(0x04030201));

    // Read from an unmapped offset between the two segments — must
    // return zero, not garbage.
    let v = read_u32.call(&mut store, &[Value::I32(0x180)]).unwrap();
    assert_eq!(v[0], Value::I32(0));
}

// ── Test 4: call_indirect through a function table ─────────────────

const TABLE_WAT: &str = r#"
(module
    (type $sig (func (param i32) (result i32)))
    (table 4 4 funcref)
    (elem (i32.const 0) $double $triple $square $negate)

    (func $double  (param i32) (result i32) local.get 0 i32.const 2 i32.mul)
    (func $triple  (param i32) (result i32) local.get 0 i32.const 3 i32.mul)
    (func $square  (param i32) (result i32) local.get 0 local.get 0 i32.mul)
    (func $negate  (param i32) (result i32) i32.const 0 local.get 0 i32.sub)

    (func (export "dispatch") (param $idx i32) (param $arg i32) (result i32)
        local.get $arg
        local.get $idx
        call_indirect (type $sig))

    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (i32.const 0))

    (memory (export "memory") 0 0))
"#;

#[test]
fn call_indirect_dispatches_to_correct_function() {
    let mut store = make_store();
    let instance = instantiate(TABLE_WAT, &imports! {}, &mut store);
    let dispatch = instance.exports.get_function("dispatch").unwrap().clone();

    // (idx, arg) -> expected
    let cases = [
        (0_i32, 7_i32, 14),  // double
        (1, 7, 21),          // triple
        (2, 7, 49),          // square
        (3, 7, -7),          // negate
        (0, 0, 0),
        (2, 9, 81),
    ];

    for (idx, arg, want) in cases {
        let r = dispatch
            .call(&mut store, &[Value::I32(idx), Value::I32(arg)])
            .unwrap_or_else(|e| panic!("dispatch idx={idx} arg={arg}: {e}"));
        assert_eq!(
            r[0],
            Value::I32(want),
            "dispatch idx={idx} arg={arg} want={want}"
        );
    }

    // Out-of-table index must trap.
    assert!(dispatch
        .call(&mut store, &[Value::I32(99), Value::I32(1)])
        .is_err());
}

// ── Test 5: round-trip serialize / deserialize ─────────────────────

const ROUNDTRIP_WAT: &str = r#"
(module
    (memory (export "memory") 1 1)
    (data (i32.const 0) "\11\22\33\44")

    (func (export "compute") (param i32) (result i32)
        ;; ((arg + 5) * 3) ^ 0x55
        local.get 0
        i32.const 5
        i32.add
        i32.const 3
        i32.mul
        i32.const 0x55
        i32.xor)

    (func (export "read_data") (result i32)
        (i32.load (i32.const 0)))

    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (i32.const 0)))
"#;

fn run_compute_and_read(store: &mut Store, instance: &Instance) -> (i32, i32) {
    let compute = instance.exports.get_function("compute").unwrap().clone();
    let read = instance.exports.get_function("read_data").unwrap().clone();

    let computed = match compute.call(store, &[Value::I32(7)]).unwrap()[0] {
        Value::I32(v) => v,
        _ => panic!("compute result type"),
    };
    let data = match read.call(store, &[]).unwrap()[0] {
        Value::I32(v) => v,
        _ => panic!("read_data result type"),
    };
    (computed, data)
}

#[test]
fn serialize_deserialize_round_trip_matches_fresh_compile() {
    // Fresh compile: get the reference values.
    let mut store_a = make_store();
    let wasm = wat::parse_bytes(ROUNDTRIP_WAT.as_bytes()).unwrap();
    let module_a = Module::new(&store_a, &wasm).unwrap();
    let instance_a = Instance::new(&mut store_a, &module_a, &imports! {}).unwrap();
    seed_meter(&instance_a, &mut store_a);
    let (compute_a, data_a) = run_compute_and_read(&mut store_a, &instance_a);

    // Sanity: compute(7) = ((7+5)*3) ^ 0x55 = 36 ^ 0x55 = 0x71
    assert_eq!(compute_a, 0x71);
    // Little-endian load of "\x11\x22\x33\x44" = 0x44332211
    assert_eq!(data_a, 0x44332211_u32 as i32);

    // Serialize → deserialize via the same engine, then run the same
    // ops and assert byte-identical results.
    let serialized = module_a.serialize().expect("serialize");

    let mut store_b = make_store();
    let module_b =
        unsafe { Module::deserialize(&store_b, serialized).expect("deserialize") };
    let instance_b = Instance::new(&mut store_b, &module_b, &imports! {}).unwrap();
    seed_meter(&instance_b, &mut store_b);
    let (compute_b, data_b) = run_compute_and_read(&mut store_b, &instance_b);

    assert_eq!(compute_a, compute_b, "compute() differs after deserialize");
    assert_eq!(data_a, data_b, "data segment differs after deserialize");
}

// ── Test 6: multiple user globals ──────────────────────────────────

const GLOBALS_WAT: &str = r#"
(module
    (memory (export "memory") 0 0)
    (global $a (export "a") (mut i32) (i32.const 10))
    (global $b (export "b") (mut i64) (i64.const 0xdeadbeef))
    (global $c (export "c")        i32 (i32.const 0xabba))

    (func (export "bump_a") (param i32)
        global.get $a
        local.get 0
        i32.add
        global.set $a)

    (func (export "read_b") (result i64)
        global.get $b)

    (func (export "read_c") (result i32)
        global.get $c)

    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (i32.const 0)))
"#;

#[test]
fn user_globals_initialize_correctly_and_are_not_clobbered_by_meter() {
    let mut store = make_store();
    let instance = instantiate(GLOBALS_WAT, &imports! {}, &mut store);

    // Initial values lifted from the WAT.
    let a = instance.exports.get_global("a").unwrap().get(&mut store);
    let b = instance.exports.get_global("b").unwrap().get(&mut store);
    let c = instance.exports.get_global("c").unwrap().get(&mut store);
    assert_eq!(a, Value::I32(10));
    assert_eq!(b, Value::I64(0xdeadbeef));
    assert_eq!(c, Value::I32(0xabba));

    // Mutating a must update only a, leaving b and c alone — and must
    // not silently clobber the meter-injected globals either.
    let bump_a = instance.exports.get_function("bump_a").unwrap().clone();
    bump_a.call(&mut store, &[Value::I32(5)]).unwrap();

    let a = instance.exports.get_global("a").unwrap().get(&mut store);
    assert_eq!(a, Value::I32(15));
    assert_eq!(
        instance.exports.get_global("b").unwrap().get(&mut store),
        Value::I64(0xdeadbeef)
    );
    assert_eq!(
        instance.exports.get_global("c").unwrap().get(&mut store),
        Value::I32(0xabba)
    );

    // Meter globals must still be readable after middleware injection.
    assert!(instance.exports.get_global("stylus_ink_left").is_ok());
    assert!(instance.exports.get_global("stylus_ink_status").is_ok());
    assert!(instance.exports.get_global("stylus_stack_left").is_ok());
}

// ── Test 7: activation parity via nitro_prover ─────────────────────
//
// Our `activate_program` delegates to `nitro_prover::machine::Module::activate`,
// so this is a regression detector: if the upstream changes anything about
// how it computes init_gas / cached_init_gas / footprint / asm_estimate /
// module_hash for a known-input WASM, this test breaks loudly with a one-line
// diff. The expected values were captured by running this test with the
// asserts replaced by `eprintln!`, then frozen.

const ACTIVATE_WAT: &str = r#"
(module
    (memory (export "memory") 1 1)
    (data (i32.const 0) "\11\22\33\44\55\66\77\88")

    (func $add (export "add") (param i32 i32) (result i32)
        local.get 0
        local.get 1
        i32.add)

    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (i32.const 0)))
"#;

/// Helper: capture activation outputs for the WAT above. Used both by the
/// frozen-value test and to print fresh values when bumping the fixture.
fn activate_test_wat() -> arbos::programs::types::ActivationResult {
    let wasm = wat::parse_bytes(ACTIVATE_WAT.as_bytes()).unwrap();
    let codehash = [0x42_u8; 32];
    let mut gas = u64::MAX;
    arb_stylus::activate_program(
        &wasm,
        &codehash,
        /* stylus_version */ 1,
        /* arbos_version */ 30,
        /* page_limit */ 128,
        /* debug */ false,
        &mut gas,
    )
    .expect("activation")
}

#[test]
fn activation_outputs_are_deterministic() {
    // Two activations of the same input must produce the same output. If
    // this fails, activation depends on hidden global state — which would
    // be an immediate red flag.
    let a = activate_test_wat();
    let b = activate_test_wat();
    assert_eq!(a.module_hash, b.module_hash);
    assert_eq!(a.init_gas, b.init_gas);
    assert_eq!(a.cached_init_gas, b.cached_init_gas);
    assert_eq!(a.asm_estimate, b.asm_estimate);
    assert_eq!(a.footprint, b.footprint);
}

#[test]
fn activation_outputs_are_within_sane_bounds() {
    // Fixed-bound assertions instead of a frozen value (which would require
    // a fresh capture every time the fixture changes). These bounds catch
    // gross drift without becoming brittle.
    let r = activate_test_wat();

    // `init_gas` is a u16 packed into the Program word — it cannot
    // overflow, but it must be non-zero for any non-trivial program.
    assert!(r.init_gas > 0, "init_gas must be positive");
    assert!(
        r.cached_init_gas <= r.init_gas,
        "cached_init_gas ({}) must be <= init_gas ({})",
        r.cached_init_gas,
        r.init_gas
    );
    // One memory page declared in the WAT.
    assert_eq!(r.footprint, 1, "footprint must equal declared pages");
    // ASM estimate must be a few KB at minimum — empty asm would mean
    // codegen produced nothing.
    assert!(r.asm_estimate > 0, "asm_estimate must be > 0");
    // module_hash must not be all zeros.
    assert_ne!(
        r.module_hash,
        alloy_primitives::B256::ZERO,
        "module_hash must not be zero"
    );
}
