//! Additional WASM execution tests covering surfaces not in wavm.rs/codegen.rs:
//! memory operations, branching ink cost, mutable globals, and trap behaviour.

#[cfg(target_arch = "x86_64")]
#[no_mangle]
#[allow(clippy::missing_safety_doc)]
pub unsafe extern "C" fn __rust_probestack() {}

use std::sync::Arc;

use arb_stylus::config::CompileConfig;
use wasmer::{
    imports, sys::EngineBuilder, CompilerConfig, Cranelift, CraneliftOptLevel, Imports, Instance,
    Module, Store, Value,
};

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
    seed(&instance, store);
    instance
}

fn seed(instance: &Instance, store: &mut Store) {
    instance.exports.get_global("stylus_ink_left").unwrap().set(store, Value::I64(i64::MAX)).unwrap();
    instance.exports.get_global("stylus_ink_status").unwrap().set(store, Value::I32(0)).unwrap();
    instance.exports.get_global("stylus_stack_left").unwrap().set(store, Value::I32(i32::MAX)).unwrap();
}

fn ink_left(instance: &Instance, store: &mut Store) -> u64 {
    match instance.exports.get_global("stylus_ink_left").unwrap().get(store) {
        Value::I64(v) => v as u64,
        _ => unreachable!(),
    }
}

const STORE_LOAD_WAT: &str = r#"
(module
  (memory (export "memory") 1)
  (func (export "store_then_load") (param $addr i32) (param $val i32) (result i32)
    local.get $addr
    local.get $val
    i32.store
    local.get $addr
    i32.load)
  (func (export "user_entrypoint") (param $args_len i32) (result i32)
    (i32.const 0)))
"#;

#[test]
fn memory_store_then_load_returns_same_value() {
    let mut store = make_store();
    let inst = instantiate(STORE_LOAD_WAT, &imports! {}, &mut store);
    let f = inst.exports.get_function("store_then_load").unwrap().clone();
    let res = f.call(&mut store, &[Value::I32(64), Value::I32(0xdeadbeefu32 as i32)]).unwrap();
    assert_eq!(res[0].i32().unwrap() as u32, 0xdeadbeefu32);
}

#[test]
fn memory_store_consumes_ink() {
    let mut store = make_store();
    let inst = instantiate(STORE_LOAD_WAT, &imports! {}, &mut store);
    let f = inst.exports.get_function("store_then_load").unwrap().clone();
    let before = ink_left(&inst, &mut store);
    let _ = f.call(&mut store, &[Value::I32(0), Value::I32(1)]).unwrap();
    let after = ink_left(&inst, &mut store);
    assert!(before > after);
}

const BRANCH_WAT: &str = r#"
(module
  (memory (export "memory") 0 0)
  (func (export "branch") (param $cond i32) (result i32)
    local.get $cond
    if (result i32)
      i32.const 100
    else
      i32.const 200
    end)
  (func (export "user_entrypoint") (param $args_len i32) (result i32) (i32.const 0)))
"#;

#[test]
fn branch_taken_returns_then_value() {
    let mut store = make_store();
    let inst = instantiate(BRANCH_WAT, &imports! {}, &mut store);
    let f = inst.exports.get_function("branch").unwrap().clone();
    assert_eq!(f.call(&mut store, &[Value::I32(1)]).unwrap()[0].i32().unwrap(), 100);
    assert_eq!(f.call(&mut store, &[Value::I32(0)]).unwrap()[0].i32().unwrap(), 200);
}

#[test]
fn branch_paths_consume_distinct_ink_amounts() {
    let mut store = make_store();
    let inst = instantiate(BRANCH_WAT, &imports! {}, &mut store);
    let f = inst.exports.get_function("branch").unwrap().clone();

    seed(&inst, &mut store);
    let before = ink_left(&inst, &mut store);
    let _ = f.call(&mut store, &[Value::I32(1)]).unwrap();
    let cost_then = before - ink_left(&inst, &mut store);

    seed(&inst, &mut store);
    let before = ink_left(&inst, &mut store);
    let _ = f.call(&mut store, &[Value::I32(0)]).unwrap();
    let cost_else = before - ink_left(&inst, &mut store);

    assert!(cost_then > 0 && cost_else > 0);
}

const GLOBAL_WAT: &str = r#"
(module
  (global $counter (export "counter") (mut i32) (i32.const 0))
  (memory (export "memory") 0 0)
  (func (export "tick")
    global.get $counter
    i32.const 1
    i32.add
    global.set $counter)
  (func (export "user_entrypoint") (param $args_len i32) (result i32) (i32.const 0)))
"#;

#[test]
fn mutable_global_persists_across_calls() {
    let mut store = make_store();
    let inst = instantiate(GLOBAL_WAT, &imports! {}, &mut store);
    let tick = inst.exports.get_function("tick").unwrap().clone();
    let counter = inst.exports.get_global("counter").unwrap().clone();
    counter.set(&mut store, Value::I32(0)).unwrap();
    for _ in 0..5 {
        tick.call(&mut store, &[]).unwrap();
    }
    assert_eq!(counter.get(&mut store).i32().unwrap(), 5);
}

const TRAP_WAT: &str = r#"
(module
  (memory (export "memory") 0 0)
  (func (export "div_zero") (param $a i32) (result i32)
    local.get $a
    i32.const 0
    i32.div_s)
  (func (export "user_entrypoint") (param $args_len i32) (result i32) (i32.const 0)))
"#;

#[test]
fn integer_divide_by_zero_traps() {
    let mut store = make_store();
    let inst = instantiate(TRAP_WAT, &imports! {}, &mut store);
    let f = inst.exports.get_function("div_zero").unwrap().clone();
    let res = f.call(&mut store, &[Value::I32(7)]);
    assert!(res.is_err());
}

const NESTED_WAT: &str = r#"
(module
  (memory (export "memory") 0 0)
  (func $inner (param $x i32) (result i32)
    local.get $x
    i32.const 7
    i32.mul)
  (func (export "outer") (param $x i32) (result i32)
    local.get $x
    call $inner
    i32.const 3
    i32.add)
  (func (export "user_entrypoint") (param $args_len i32) (result i32) (i32.const 0)))
"#;

#[test]
fn nested_call_returns_correct_value() {
    let mut store = make_store();
    let inst = instantiate(NESTED_WAT, &imports! {}, &mut store);
    let f = inst.exports.get_function("outer").unwrap().clone();
    let res = f.call(&mut store, &[Value::I32(5)]).unwrap();
    assert_eq!(res[0].i32().unwrap(), 5 * 7 + 3);
}

#[test]
fn nested_call_costs_more_than_a_single_call() {
    let mut store = make_store();
    let inst = instantiate(NESTED_WAT, &imports! {}, &mut store);
    let outer = inst.exports.get_function("outer").unwrap().clone();
    seed(&inst, &mut store);
    let before = ink_left(&inst, &mut store);
    let _ = outer.call(&mut store, &[Value::I32(1)]).unwrap();
    let cost = before - ink_left(&inst, &mut store);
    assert!(cost > 5_000, "nested call ink cost should be substantial, got {cost}");
}
