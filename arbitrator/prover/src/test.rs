// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![cfg(test)]

use crate::binary;
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
            (import "forward" "some_import" (func (param i64) (result i64 i32)))
            (import "forward" "some_import" (func (param i64) (result i64 i32)))
        )"#,
    );
    let _ = binary::parse(&wasm, Path::new("")).unwrap();

    let wasm = as_wasm(
        r#"
        (module
            (import "forward" "some_import" (func (param i32) (result f64)))
            (import "forward" "some_import" (func (param i32) (result)))
        )"#,
    );
    let _ = binary::parse(&wasm, Path::new("")).unwrap_err();
}
