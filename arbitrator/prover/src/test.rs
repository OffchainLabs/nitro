// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::binary;
use brotli::Dictionary;
use eyre::Result;
use std::path::Path;

fn multi_value_wasm() -> Vec<u8> {
    as_wasm(
        r#"(module
            (memory (export "memory") 1 1)
            (func (result i32 i32)
                i32.const 1
                i32.const 2
            )
            (func (export "user_entrypoint") (param i32) (result i32)
                i32.const 0
            )
        )"#,
    )
}

#[test]
fn test_multi_value_rejected_at_stylus_v3() {
    // New activations at Stylus V3 must reject multi-value wasm.
    let wasm = multi_value_wasm();
    binary::parse_with_stylus_version(&wasm, Path::new("test"), 3).unwrap_err();
}

#[test]
fn test_multi_value_allowed_below_stylus_v3() {
    // V1 and V2 activations must still accept multi-value wasm.
    let wasm = multi_value_wasm();
    binary::parse_with_stylus_version(&wasm, Path::new("test"), 1).unwrap();
    binary::parse_with_stylus_version(&wasm, Path::new("test"), 2).unwrap();
}

#[test]
fn test_multi_value_allowed_on_recompile() {
    // Version 0 is the recompilation path: multi-value must be accepted even at V3,
    // because the contract was legitimately activated pre-V3 and its wasm cannot change.
    let wasm = multi_value_wasm();
    binary::parse_with_stylus_version(&wasm, Path::new("test"), 0).unwrap();
}

fn as_wasm(wat: &str) -> Vec<u8> {
    let wasm = wasmer::wat2wasm(wat.as_bytes());
    wasm.unwrap().to_vec()
}

#[test]
pub fn reject_reexports() {
    let wasm = as_wasm(
        r#"
        (module
            (import "env" "some_hostio_func" (func (param) (result)))
            (func $should_reject (export "some_hostio_func") (param) (result))
        )"#,
    );
    let _ = binary::parse(&wasm, Path::new("")).unwrap_err();

    let wasm = as_wasm(
        r#"
        (module
            (import "env" "some_hostio_func" (func (param) (result)))
            (global $should_reject (export "some_hostio_func") f32 (f32.const 0))
        )"#,
    );
    let _ = binary::parse(&wasm, Path::new("")).unwrap_err();
}

#[test]
pub fn reject_ambiguous_imports() {
    let wasm = as_wasm(
        r#"
        (module
            (import "vm_hooks" "some_import" (func (param i64) (result i64 i32)))
            (import "vm_hooks" "some_import" (func (param i64) (result i64 i32)))
        )"#,
    );
    let _ = binary::parse(&wasm, Path::new("")).unwrap();

    let wasm = as_wasm(
        r#"
        (module
            (import "vm_hooks" "some_import" (func (param i32) (result f64)))
            (import "vm_hooks" "some_import" (func (param i32) (result)))
        )"#,
    );
    let _ = binary::parse(&wasm, Path::new("")).unwrap_err();
}

#[test]
pub fn test_compress() -> Result<()> {
    let data = include_bytes!("../../../target/machines/latest/forward_stub.wasm");
    let mut last = vec![];

    for dict in [Dictionary::Empty, Dictionary::StylusProgram] {
        let deflate = brotli::compress(data, 11, 22, dict).unwrap();
        let inflate = brotli::decompress(&deflate, dict).unwrap();
        assert_eq!(hex::encode(inflate), hex::encode(data));
        assert!(deflate != last);
        last = deflate;
    }
    Ok(())
}
