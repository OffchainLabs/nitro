// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![cfg(test)]

use eyre::{Result, bail};
use wasmer::{Engine, Instance, Module, Store, Value, imports};
use wasmer_compiler_cranelift::Cranelift;

#[test]
fn test_crate() -> Result<()> {
    // Adapted from https://docs.rs/wasmer/3.1.0/wasmer/index.html

    let source = std::fs::read("programs/pure/main.wat")?;

    let mut store = Store::default();
    let module = Module::new(&store, source)?;
    let imports = imports! {};
    let instance = Instance::new(&mut store, &module, &imports)?;

    let add_one = instance.exports.get_function("add_one")?;
    let result = add_one.call(&mut store, &[Value::I32(42)])?;
    assert_eq!(result[0], Value::I32(43));
    Ok(())
}

/// Loads soft-float.wasm into a plain Wasmer store (no NaN canonicalization),
/// mirroring how WAVM calls into the soft-float library via cross-module calls.
fn load_soft_float() -> Result<(Store, Instance)> {
    let wasm = std::fs::read("../../target/machines/latest/soft-float.wasm")?;
    let mut store = Store::default();
    let module = Module::new(&store, &wasm)?;
    let instance = Instance::new(&mut store, &module, &imports! {})?;
    Ok((store, instance))
}

/// Creates a Wasmer store backed by Cranelift with NaN canonicalization enabled,
/// matching the JIT configuration in `crates/jit/src/machine.rs`.
fn make_jit_store() -> Store {
    let mut compiler = Cranelift::new();
    compiler.canonicalize_nans(true);
    Store::new(Engine::from(compiler))
}

/// Compiles a tiny WASM module in the JIT store that wraps the given float binary
/// operation. Arguments and return value are passed as reinterpreted integers so
/// the caller can inspect raw NaN bit patterns.
fn jit_float_binop(op: &str) -> Result<(Store, Instance)> {
    let wat = format!(
        r#"(module
               (func (export "op") (param i32 i32) (result i32)
                   (f32.reinterpret_i32 (local.get 0))
                   (f32.reinterpret_i32 (local.get 1))
                   ({op})
                   (i32.reinterpret_f32)))"#
    );
    let wasm = wasmer::wat2wasm(wat.as_bytes())?.to_vec();
    let mut store = make_jit_store();
    let module = Module::new(&store, &wasm)?;
    let instance = Instance::new(&mut store, &module, &imports! {})?;
    Ok((store, instance))
}

fn jit_float64_binop(op: &str) -> Result<(Store, Instance)> {
    let wat = format!(
        r#"(module
               (func (export "op") (param i64 i64) (result i64)
                   (f64.reinterpret_i64 (local.get 0))
                   (f64.reinterpret_i64 (local.get 1))
                   ({op})
                   (i64.reinterpret_f64)))"#
    );
    let wasm = wasmer::wat2wasm(wat.as_bytes())?.to_vec();
    let mut store = make_jit_store();
    let module = Module::new(&store, &wasm)?;
    let instance = Instance::new(&mut store, &module, &imports! {})?;
    Ok((store, instance))
}

fn call_i32_binop(
    store: &mut Store,
    instance: &Instance,
    fname: &str,
    a: u32,
    b: u32,
) -> Result<u32> {
    let f = instance.exports.get_function(fname)?;
    let res = f.call(store, &[Value::I32(a as i32), Value::I32(b as i32)])?;
    match res[0] {
        Value::I32(x) => Ok(x as u32),
        _ => bail!("expected i32 result"),
    }
}

fn jit_float_unop(op: &str) -> Result<(Store, Instance)> {
    let wat = format!(
        r#"(module
               (func (export "op") (param i32) (result i32)
                   (f32.reinterpret_i32 (local.get 0))
                   ({op})
                   (i32.reinterpret_f32)))"#
    );
    let wasm = wasmer::wat2wasm(wat.as_bytes())?.to_vec();
    let mut store = make_jit_store();
    let module = Module::new(&store, &wasm)?;
    let instance = Instance::new(&mut store, &module, &imports! {})?;
    Ok((store, instance))
}

fn jit_float64_unop(op: &str) -> Result<(Store, Instance)> {
    let wat = format!(
        r#"(module
               (func (export "op") (param i64) (result i64)
                   (f64.reinterpret_i64 (local.get 0))
                   ({op})
                   (i64.reinterpret_f64)))"#
    );
    let wasm = wasmer::wat2wasm(wat.as_bytes())?.to_vec();
    let mut store = make_jit_store();
    let module = Module::new(&store, &wasm)?;
    let instance = Instance::new(&mut store, &module, &imports! {})?;
    Ok((store, instance))
}

fn call_i32_unop(store: &mut Store, instance: &Instance, fname: &str, a: u32) -> Result<u32> {
    let f = instance.exports.get_function(fname)?;
    let res = f.call(store, &[Value::I32(a as i32)])?;
    match res[0] {
        Value::I32(x) => Ok(x as u32),
        _ => bail!("expected i32 result"),
    }
}

fn call_i64_unop(store: &mut Store, instance: &Instance, fname: &str, a: u64) -> Result<u64> {
    let f = instance.exports.get_function(fname)?;
    let res = f.call(store, &[Value::I64(a as i64)])?;
    match res[0] {
        Value::I64(x) => Ok(x as u64),
        _ => bail!("expected i64 result"),
    }
}

fn jit_f32_from_f64_unop(op: &str) -> Result<(Store, Instance)> {
    let wat = format!(
        r#"(module
               (func (export "op") (param i64) (result i32)
                   (f64.reinterpret_i64 (local.get 0))
                   ({op})
                   (i32.reinterpret_f32)))"#
    );
    let wasm = wasmer::wat2wasm(wat.as_bytes())?.to_vec();
    let mut store = make_jit_store();
    let module = Module::new(&store, &wasm)?;
    let instance = Instance::new(&mut store, &module, &imports! {})?;
    Ok((store, instance))
}

fn jit_f64_from_f32_unop(op: &str) -> Result<(Store, Instance)> {
    let wat = format!(
        r#"(module
               (func (export "op") (param i32) (result i64)
                   (f32.reinterpret_i32 (local.get 0))
                   ({op})
                   (i64.reinterpret_f64)))"#
    );
    let wasm = wasmer::wat2wasm(wat.as_bytes())?.to_vec();
    let mut store = make_jit_store();
    let module = Module::new(&store, &wasm)?;
    let instance = Instance::new(&mut store, &module, &imports! {})?;
    Ok((store, instance))
}

fn call_i32_from_i64(store: &mut Store, instance: &Instance, fname: &str, a: u64) -> Result<u32> {
    let f = instance.exports.get_function(fname)?;
    let res = f.call(store, &[Value::I64(a as i64)])?;
    match res[0] {
        Value::I32(x) => Ok(x as u32),
        _ => bail!("expected i32 result"),
    }
}

fn call_i64_from_i32(store: &mut Store, instance: &Instance, fname: &str, a: u32) -> Result<u64> {
    let f = instance.exports.get_function(fname)?;
    let res = f.call(store, &[Value::I32(a as i32)])?;
    match res[0] {
        Value::I64(x) => Ok(x as u64),
        _ => bail!("expected i64 result"),
    }
}

fn call_i64_binop(
    store: &mut Store,
    instance: &Instance,
    fname: &str,
    a: u64,
    b: u64,
) -> Result<u64> {
    let f = instance.exports.get_function(fname)?;
    let res = f.call(store, &[Value::I64(a as i64), Value::I64(b as i64)])?;
    match res[0] {
        Value::I64(x) => Ok(x as u64),
        _ => bail!("expected i64 result"),
    }
}

/// Both soft-float (WAVM) and JIT must return the canonical quiet NaN (0x7FC00000)
/// from f32.min when either operand is a NaN.
#[test]
fn test_f32_min_nan_canonicalization() -> Result<()> {
    // 0x7F800001: signaling NaN (sNaN) — exponent=0xFF, mantissa=1, sign=0
    let snan: u32 = 0x7F800001;
    let one: u32 = 0x3F800000; // 1.0
    let canonical_nan: u32 = 0x7FC00000;

    let (mut sf_store, sf) = load_soft_float()?;
    let soft_bits = call_i32_binop(&mut sf_store, &sf, "wavm__f32_min", snan, one)?;

    let (mut jit_store, jit) = jit_float_binop("f32.min")?;
    let jit_bits = call_i32_binop(&mut jit_store, &jit, "op", snan, one)?;

    assert_eq!(
        jit_bits, canonical_nan,
        "JIT f32.min should return canonical NaN"
    );
    assert_eq!(
        soft_bits, canonical_nan,
        "soft-float f32.min with sNaN operand: got 0x{soft_bits:08x}, want canonical NaN 0x{canonical_nan:08x}"
    );
    Ok(())
}

/// Both soft-float (WAVM) and JIT must return the canonical quiet NaN (0x7FC00000)
/// from f32.max when either operand is a NaN.
#[test]
fn test_f32_max_nan_canonicalization() -> Result<()> {
    let snan: u32 = 0x7F800001;
    let one: u32 = 0x3F800000;
    let canonical_nan: u32 = 0x7FC00000;

    let (mut sf_store, sf) = load_soft_float()?;
    let soft_bits = call_i32_binop(&mut sf_store, &sf, "wavm__f32_max", snan, one)?;

    let (mut jit_store, jit) = jit_float_binop("f32.max")?;
    let jit_bits = call_i32_binop(&mut jit_store, &jit, "op", snan, one)?;

    assert_eq!(
        jit_bits, canonical_nan,
        "JIT f32.max should return canonical NaN"
    );
    assert_eq!(
        soft_bits, canonical_nan,
        "soft-float f32.max with sNaN operand: got 0x{soft_bits:08x}, want canonical NaN 0x{canonical_nan:08x}"
    );
    Ok(())
}

/// Both soft-float (WAVM) and JIT must return the canonical quiet NaN (0x7FF8000000000000)
/// from f64.min when either operand is a NaN.
#[test]
fn test_f64_min_nan_canonicalization() -> Result<()> {
    // 0x7FF0000000000001: signaling NaN for f64
    let snan: u64 = 0x7FF0000000000001;
    let one: u64 = 0x3FF0000000000000; // 1.0
    let canonical_nan: u64 = 0x7FF8000000000000;

    let (mut sf_store, sf) = load_soft_float()?;
    let soft_bits = call_i64_binop(&mut sf_store, &sf, "wavm__f64_min", snan, one)?;

    let (mut jit_store, jit) = jit_float64_binop("f64.min")?;
    let jit_bits = call_i64_binop(&mut jit_store, &jit, "op", snan, one)?;

    assert_eq!(
        jit_bits, canonical_nan,
        "JIT f64.min should return canonical NaN"
    );
    assert_eq!(
        soft_bits, canonical_nan,
        "soft-float f64.min with sNaN operand: got 0x{soft_bits:016x}, want canonical NaN 0x{canonical_nan:016x}"
    );
    Ok(())
}

/// Both soft-float (WAVM) and JIT must return the canonical quiet NaN (0x7FF8000000000000)
/// from f64.max when either operand is a NaN.
#[test]
fn test_f64_max_nan_canonicalization() -> Result<()> {
    let snan: u64 = 0x7FF0000000000001;
    let one: u64 = 0x3FF0000000000000;
    let canonical_nan: u64 = 0x7FF8000000000000;

    let (mut sf_store, sf) = load_soft_float()?;
    let soft_bits = call_i64_binop(&mut sf_store, &sf, "wavm__f64_max", snan, one)?;

    let (mut jit_store, jit) = jit_float64_binop("f64.max")?;
    let jit_bits = call_i64_binop(&mut jit_store, &jit, "op", snan, one)?;

    assert_eq!(
        jit_bits, canonical_nan,
        "JIT f64.max should return canonical NaN"
    );
    assert_eq!(
        soft_bits, canonical_nan,
        "soft-float f64.max with sNaN operand: got 0x{soft_bits:016x}, want canonical NaN 0x{canonical_nan:016x}"
    );
    Ok(())
}

/// Both soft-float (WAVM) and JIT must return the canonical quiet NaN (0x7FC00000)
/// from f32 operations that generate NaN from scratch (e.g. +∞ + -∞).
#[test]
fn test_f32_add_generated_nan_canonicalization() -> Result<()> {
    let pos_inf: u32 = 0x7F800000; // +∞
    let neg_inf: u32 = 0xFF800000; // -∞
    let canonical_nan: u32 = 0x7FC00000;

    let (mut sf_store, sf) = load_soft_float()?;
    let soft_bits = call_i32_binop(&mut sf_store, &sf, "wavm__f32_add", pos_inf, neg_inf)?;

    let (mut jit_store, jit) = jit_float_binop("f32.add")?;
    let jit_bits = call_i32_binop(&mut jit_store, &jit, "op", pos_inf, neg_inf)?;

    assert_eq!(
        jit_bits, canonical_nan,
        "JIT f32.add(+∞, -∞) should return canonical NaN"
    );
    assert_eq!(
        soft_bits, canonical_nan,
        "soft-float f32.add(+∞, -∞): got 0x{soft_bits:08x}, want canonical NaN 0x{canonical_nan:08x}"
    );
    Ok(())
}

/// Both soft-float (WAVM) and JIT must return the canonical f32 NaN (0x7FC00000)
/// when demoting a NaN f64 to f32.
#[test]
fn test_f32_demote_f64_nan_canonicalization() -> Result<()> {
    let snan_f64: u64 = 0x7FF0000000000001; // sNaN: exponent=0x7FF, mantissa=1
    let canonical_nan_f32: u32 = 0x7FC00000;

    let (mut sf_store, sf) = load_soft_float()?;
    let soft_bits = call_i32_from_i64(&mut sf_store, &sf, "wavm__f32_demote_f64", snan_f64)?;

    let (mut jit_store, jit) = jit_f32_from_f64_unop("f32.demote_f64")?;
    let jit_bits = call_i32_from_i64(&mut jit_store, &jit, "op", snan_f64)?;

    assert_eq!(
        jit_bits, canonical_nan_f32,
        "JIT f32.demote_f64 should return canonical f32 NaN"
    );
    assert_eq!(
        soft_bits, canonical_nan_f32,
        "soft-float f32.demote_f64: got 0x{soft_bits:08x}, want canonical f32 NaN 0x{canonical_nan_f32:08x}"
    );
    Ok(())
}

/// Both soft-float (WAVM) and JIT must return the canonical f64 NaN (0x7FF8000000000000)
/// when promoting a NaN f32 to f64.
#[test]
fn test_f64_promote_f32_nan_canonicalization() -> Result<()> {
    let snan_f32: u32 = 0x7F800001;
    let canonical_nan_f64: u64 = 0x7FF8000000000000;

    let (mut sf_store, sf) = load_soft_float()?;
    let soft_bits = call_i64_from_i32(&mut sf_store, &sf, "wavm__f64_promote_f32", snan_f32)?;

    let (mut jit_store, jit) = jit_f64_from_f32_unop("f64.promote_f32")?;
    let jit_bits = call_i64_from_i32(&mut jit_store, &jit, "op", snan_f32)?;

    assert_eq!(
        jit_bits, canonical_nan_f64,
        "JIT f64.promote_f32 should return canonical f64 NaN"
    );
    assert_eq!(
        soft_bits, canonical_nan_f64,
        "soft-float f64.promote_f32: got 0x{soft_bits:016x}, want canonical f64 NaN 0x{canonical_nan_f64:016x}"
    );
    Ok(())
}

/// Both soft-float (WAVM) and JIT must return the canonical quiet NaN (0x7FF8000000000000)
/// from f64 operations that generate NaN from scratch (e.g. +∞ + -∞).
#[test]
fn test_f64_add_generated_nan_canonicalization() -> Result<()> {
    let pos_inf: u64 = 0x7FF0000000000000; // +∞
    let neg_inf: u64 = 0xFFF0000000000000; // -∞
    let canonical_nan: u64 = 0x7FF8000000000000;

    let (mut sf_store, sf) = load_soft_float()?;
    let soft_bits = call_i64_binop(&mut sf_store, &sf, "wavm__f64_add", pos_inf, neg_inf)?;

    let (mut jit_store, jit) = jit_float64_binop("f64.add")?;
    let jit_bits = call_i64_binop(&mut jit_store, &jit, "op", pos_inf, neg_inf)?;

    assert_eq!(
        jit_bits, canonical_nan,
        "JIT f64.add(+∞, -∞) should return canonical NaN"
    );
    assert_eq!(
        soft_bits, canonical_nan,
        "soft-float f64.add(+∞, -∞): got 0x{soft_bits:016x}, want canonical NaN 0x{canonical_nan:016x}"
    );
    Ok(())
}

/// Both soft-float (WAVM) and JIT must return the canonical quiet NaN (0x7FC00000)
/// from f32 unary operations (ceil, floor, trunc, nearest, sqrt) when the operand is a NaN.
#[test]
fn test_f32_unary_nan_canonicalization() -> Result<()> {
    let snan: u32 = 0x7F800001;
    let canonical_nan: u32 = 0x7FC00000;

    for (jit_op, sf_name) in &[
        ("f32.ceil", "wavm__f32_ceil"),
        ("f32.floor", "wavm__f32_floor"),
        ("f32.trunc", "wavm__f32_trunc"),
        ("f32.nearest", "wavm__f32_nearest"),
        ("f32.sqrt", "wavm__f32_sqrt"),
    ] {
        let (mut sf_store, sf) = load_soft_float()?;
        let soft_bits = call_i32_unop(&mut sf_store, &sf, sf_name, snan)?;
        let (mut jit_store, jit) = jit_float_unop(jit_op)?;
        let jit_bits = call_i32_unop(&mut jit_store, &jit, "op", snan)?;
        assert_eq!(
            jit_bits, canonical_nan,
            "JIT {jit_op}(sNaN) should return canonical NaN"
        );
        assert_eq!(
            soft_bits, canonical_nan,
            "soft-float {sf_name}(sNaN): got 0x{soft_bits:08x}, want 0x{canonical_nan:08x}"
        );
    }
    Ok(())
}

/// Both soft-float (WAVM) and JIT must return the canonical quiet NaN (0x7FC00000)
/// from f32 binary operations (sub, mul, div) when one operand is a NaN.
#[test]
fn test_f32_binary_nan_canonicalization() -> Result<()> {
    let snan: u32 = 0x7F800001;
    let one: u32 = 0x3F800000;
    let canonical_nan: u32 = 0x7FC00000;

    for (jit_op, sf_name) in &[
        ("f32.sub", "wavm__f32_sub"),
        ("f32.mul", "wavm__f32_mul"),
        ("f32.div", "wavm__f32_div"),
    ] {
        let (mut sf_store, sf) = load_soft_float()?;
        let soft_bits = call_i32_binop(&mut sf_store, &sf, sf_name, snan, one)?;
        let (mut jit_store, jit) = jit_float_binop(jit_op)?;
        let jit_bits = call_i32_binop(&mut jit_store, &jit, "op", snan, one)?;
        assert_eq!(
            jit_bits, canonical_nan,
            "JIT {jit_op}(sNaN, 1.0) should return canonical NaN"
        );
        assert_eq!(
            soft_bits, canonical_nan,
            "soft-float {sf_name}(sNaN, 1.0): got 0x{soft_bits:08x}, want 0x{canonical_nan:08x}"
        );
    }
    Ok(())
}

/// Both soft-float (WAVM) and JIT must return the canonical quiet NaN (0x7FF8000000000000)
/// from f64 unary operations (ceil, floor, trunc, nearest, sqrt) when the operand is a NaN.
#[test]
fn test_f64_unary_nan_canonicalization() -> Result<()> {
    let snan: u64 = 0x7FF0000000000001;
    let canonical_nan: u64 = 0x7FF8000000000000;

    for (jit_op, sf_name) in &[
        ("f64.ceil", "wavm__f64_ceil"),
        ("f64.floor", "wavm__f64_floor"),
        ("f64.trunc", "wavm__f64_trunc"),
        ("f64.nearest", "wavm__f64_nearest"),
        ("f64.sqrt", "wavm__f64_sqrt"),
    ] {
        let (mut sf_store, sf) = load_soft_float()?;
        let soft_bits = call_i64_unop(&mut sf_store, &sf, sf_name, snan)?;
        let (mut jit_store, jit) = jit_float64_unop(jit_op)?;
        let jit_bits = call_i64_unop(&mut jit_store, &jit, "op", snan)?;
        assert_eq!(
            jit_bits, canonical_nan,
            "JIT {jit_op}(sNaN) should return canonical NaN"
        );
        assert_eq!(
            soft_bits, canonical_nan,
            "soft-float {sf_name}(sNaN): got 0x{soft_bits:016x}, want 0x{canonical_nan:016x}"
        );
    }
    Ok(())
}

/// Both soft-float (WAVM) and JIT must return the canonical quiet NaN (0x7FF8000000000000)
/// from f64 binary operations (sub, mul, div) when one operand is a NaN.
#[test]
fn test_f64_binary_nan_canonicalization() -> Result<()> {
    let snan: u64 = 0x7FF0000000000001;
    let one: u64 = 0x3FF0000000000000;
    let canonical_nan: u64 = 0x7FF8000000000000;

    for (jit_op, sf_name) in &[
        ("f64.sub", "wavm__f64_sub"),
        ("f64.mul", "wavm__f64_mul"),
        ("f64.div", "wavm__f64_div"),
    ] {
        let (mut sf_store, sf) = load_soft_float()?;
        let soft_bits = call_i64_binop(&mut sf_store, &sf, sf_name, snan, one)?;
        let (mut jit_store, jit) = jit_float64_binop(jit_op)?;
        let jit_bits = call_i64_binop(&mut jit_store, &jit, "op", snan, one)?;
        assert_eq!(
            jit_bits, canonical_nan,
            "JIT {jit_op}(sNaN, 1.0) should return canonical NaN"
        );
        assert_eq!(
            soft_bits, canonical_nan,
            "soft-float {sf_name}(sNaN, 1.0): got 0x{soft_bits:016x}, want 0x{canonical_nan:016x}"
        );
    }
    Ok(())
}
