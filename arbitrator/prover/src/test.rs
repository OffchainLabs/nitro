// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![cfg(test)]

use crate::binary;
use brotli::Dictionary;
use eyre::Result;
use std::path::Path;

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
    let data = std::fs::read("test-cases/block.wat")?;
    let dict = Dictionary::Empty;

    let deflate = &mut Vec::with_capacity(data.len());
    assert!(brotli::compress(&data, deflate, 0, 22).is_ok());
    assert!(!deflate.is_empty());

    let inflate = &mut Vec::with_capacity(data.len());
    assert!(brotli::decompress(deflate, inflate, dict, false).is_ok());
    assert_eq!(hex::encode(inflate), hex::encode(&data));

    let inflate = &mut vec![];
    assert!(brotli::decompress(deflate, inflate, dict, false).is_err());
    assert!(brotli::decompress(deflate, inflate, dict, true).is_ok());
    assert_eq!(hex::encode(inflate), hex::encode(&data));
    Ok(())
}
