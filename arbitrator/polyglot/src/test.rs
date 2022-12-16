// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use std::path::Path;

use eyre::Result;
use prover::programs::{
    config::PolyglotConfig,
    meter::{MachineMeter, MeteredMachine},
    GlobalMod,
};
use wasmer::{imports, wasmparser::Operator, CompilerConfig, Imports, Instance, Module, Store};
use wasmer_compiler_singlepass::Singlepass;

fn expensive_add(op: &Operator) -> u64 {
    match op {
        Operator::I32Add => 100,
        _ => 0,
    }
}

fn new_test_instance(path: &str, config: PolyglotConfig) -> Result<(Instance, Store)> {
    let mut store = config.store();
    let wat = std::fs::read(path)?;
    let module = Module::new(&store, &wat)?;
    let imports = imports! {}; // TODO: add polyhost imports in a future PR
    let instance = Instance::new(&mut store, &module, &imports)?;
    Ok((instance, store))
}

fn new_vanilla_instance(path: &str) -> Result<(Instance, Store)> {
    let mut compiler = Singlepass::new();
    compiler.canonicalize_nans(true);
    compiler.enable_verifier();

    let mut store = Store::new(compiler);
    let wat = std::fs::read(path)?;
    let module = Module::new(&mut store, &wat)?;
    let instance = Instance::new(&mut store, &module, &Imports::new())?;
    Ok((instance, store))
}

#[test]
fn test_gas() -> Result<()> {
    let mut config = PolyglotConfig::default();
    config.costs = expensive_add;

    let (mut instance, mut store) = new_test_instance("tests/add.wat", config)?;
    let exports = &instance.exports;
    let add_one = exports.get_typed_function::<i32, i32>(&store, "add_one")?;
    let store = &mut store;

    assert_eq!(instance.gas_left(store), MachineMeter::Ready(0));

    macro_rules! exhaust {
        ($gas:expr) => {
            instance.set_gas(store, $gas);
            assert_eq!(instance.gas_left(store), MachineMeter::Ready($gas));
            assert!(add_one.call(store, 32).is_err());
            assert_eq!(instance.gas_left(store), MachineMeter::Exhausted);
        };
    }

    exhaust!(0);
    exhaust!(50);
    exhaust!(99);

    let mut gas_left = 500;
    instance.set_gas(store, gas_left);
    while gas_left > 0 {
        assert_eq!(instance.gas_left(store), MachineMeter::Ready(gas_left));
        assert_eq!(add_one.call(store, 64)?, 65);
        gas_left -= 100;
    }
    assert!(add_one.call(store, 32).is_err());
    assert_eq!(instance.gas_left(store), MachineMeter::Exhausted);
    Ok(())
}

#[test]
fn test_start() -> Result<()> {
    // in start.wat
    //     the `status` global equals 10 at initialization
    //     the `start` function increments `status`
    //     by the spec, `start` must run at initialization

    fn check(store: &mut Store, instance: &Instance, value: i32) {
        let status: i32 = instance.get_global(store, "status");
        assert_eq!(status, value);
    }

    let (instance, mut store) = new_vanilla_instance("tests/start.wat")?;
    check(&mut store, &instance, 11);

    let config = PolyglotConfig::default();
    let (instance, mut store) = new_test_instance("tests/start.wat", config)?;
    check(&mut store, &instance, 10);

    let exports = &instance.exports;
    let move_me = exports.get_typed_function::<(), ()>(&store, "move_me")?;
    let starter = exports.get_typed_function::<(), ()>(&store, "polyglot_start")?;

    move_me.call(&mut store)?;
    starter.call(&mut store)?;
    check(&mut store, &instance, 12);
    Ok(())
}

#[test]
fn test_import_export_safety() -> Result<()> {
    // in bad-export.wat
    //    there's a global named `polyglot_gas_left`

    fn check(path: &str) -> Result<()> {
        let config = PolyglotConfig::default();
        assert!(new_test_instance(path, config).is_err());

        let path = &Path::new(path);
        let wat = std::fs::read(path)?;
        let wasm = wasmer::wat2wasm(&wat)?;
        assert!(prover::binary::parse(&wasm, path).is_err());
        prover::binary::parse(&wasm, path)?;
        Ok(())
    }

    check("tests/bad-export.wat")?;
    check("tests/bad-export2.wat")?;
    Ok(())
}
