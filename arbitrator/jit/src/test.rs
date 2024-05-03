// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![cfg(test)]

use eyre::Result;
use wasmer::{imports, Instance, Module, Store, Value};

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
