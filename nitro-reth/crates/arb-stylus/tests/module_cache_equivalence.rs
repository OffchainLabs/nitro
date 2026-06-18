//! The process-wide module cache must be execution-equivalent to compiling a
//! program's WASM directly: a module served from the cache produces the same
//! result and consumes the same ink as one freshly compiled. This pins the
//! cache as a pure performance optimization with no influence on execution
//! outcomes (and therefore no effect on gas or state).

/// Stack-probe shim for x86_64 test binaries that link wasmer's vm crate.
///
/// # Safety
///
/// Defined for the linker only; never called from Rust.
#[cfg(target_arch = "x86_64")]
#[no_mangle]
pub unsafe extern "C" fn __rust_probestack() {}

use alloy_primitives::B256;
use arb_stylus::{cache::InitCache, compile_module, config::CompileConfig};
use wasmer::{imports, Instance, Module, Store, Value};

const VERSION: u16 = 1;
const DEBUG: bool = false;

// A deterministic compute with a memory store, so the ink-metering middlewares
// have non-trivial work to account for.
const WAT: &str = r#"
(module
  (memory (export "memory") 1)
  (func (export "user_entrypoint") (param $args_len i32) (result i32)
    (i32.store (i32.const 0) (local.get $args_len))
    local.get $args_len
    i32.const 7
    i32.mul
    i32.const 3
    i32.add))
"#;

fn ink_left(inst: &Instance, store: &mut Store) -> u64 {
    match inst
        .exports
        .get_global("stylus_ink_left")
        .unwrap()
        .get(store)
    {
        Value::I64(v) => v as u64,
        _ => unreachable!(),
    }
}

/// Instantiate `module`, seed the metering globals, run `compute(5)`, and return
/// `(result, ink_consumed)`.
fn run(module: &Module, store: &mut Store) -> (i32, u64) {
    let inst = Instance::new(store, module, &imports! {}).expect("instantiate");
    inst.exports
        .get_global("stylus_ink_left")
        .unwrap()
        .set(store, Value::I64(i64::MAX))
        .unwrap();
    inst.exports
        .get_global("stylus_ink_status")
        .unwrap()
        .set(store, Value::I32(0))
        .unwrap();
    inst.exports
        .get_global("stylus_stack_left")
        .unwrap()
        .set(store, Value::I32(i32::MAX))
        .unwrap();
    let before = ink_left(&inst, store);
    let func = inst
        .exports
        .get_function("user_entrypoint")
        .unwrap()
        .clone();
    let result = func.call(store, &[Value::I32(5)]).unwrap()[0]
        .i32()
        .unwrap();
    let consumed = before - ink_left(&inst, store);
    (result, consumed)
}

#[test]
fn cached_module_executes_identically_to_fresh_compile() {
    let wasm = wat::parse_bytes(WAT.as_bytes()).expect("wat2wasm");
    let key = B256::repeat_byte(0xab);

    // Fresh compile — the artifact a dispatch would build with no cache present.
    let compile = CompileConfig::version(VERSION, DEBUG).expect("supported version");
    let mut fresh_store = compile.store();
    let fresh_module = Module::new(&fresh_store, &wasm).expect("compile");
    let fresh = run(&fresh_module, &mut fresh_store);

    // Cache miss: compile + serialize + insert (the dispatch miss path).
    let serialized = compile_module(&wasm, VERSION, DEBUG).expect("compile_module");
    let (miss_module, mut miss_store) =
        InitCache::insert(key, &serialized, VERSION, 0, DEBUG).expect("insert");
    let miss = run(&miss_module, &mut miss_store);

    // Cache hit: a later dispatch reuses the stored module (proves insert populated it).
    let (hit_module, mut hit_store) =
        InitCache::get(key, VERSION, 0, DEBUG).expect("cached entry present after insert");
    let hit = run(&hit_module, &mut hit_store);

    assert_eq!(
        fresh, miss,
        "cache-miss execution diverged from a fresh compile"
    );
    assert_eq!(
        fresh, hit,
        "cache-hit execution diverged from a fresh compile"
    );
}
