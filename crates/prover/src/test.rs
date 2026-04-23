// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::path::Path;

use brotli::Dictionary;
use eyre::Result;

use crate::binary;

#[test]
fn test_multi_value_gate() {
    // Multi-value wasm is allowed for Stylus V1/V2 and rejected from V3 onward.
    // During recompilation Go passes the stored per-contract version, so a V2 contract
    // recompiled after the V3 upgrade still arrives here as stylus_version=2.
    let wasm = as_wasm(
        r#"(module
            (memory (export "memory") 1 1)
            (func (result i32 i32) i32.const 1 i32.const 2)
            (func (export "user_entrypoint") (param i32) (result i32) i32.const 0)
        )"#,
    );
    for (version, allowed) in [(1, true), (2, true), (3, false), (4, false)] {
        let result = binary::parse_with_stylus_version(&wasm, Path::new("test"), version);
        assert_eq!(result.is_ok(), allowed, "stylus_version={version}");
    }
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
    let data = include_bytes!("forward_stub.wat");
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
