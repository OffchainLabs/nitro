// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use std::path::Path;

use arbutil::Color;
use eyre::{bail, Result};
use prover::{
    binary,
    programs::{
        config::PolyglotConfig,
        meter::{MachineMeter, MeteredMachine},
        start::StartlessMachine,
        GlobalMod, ModuleMod,
    },
};
use wasmer::{
    imports, wasmparser::Operator, CompilerConfig, ExportIndex, Function, Imports, Instance,
    MemoryType, Module, Pages, Store,
};
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
    let imports = imports! {
        "test" => {
            "noop" => Function::new_typed(&mut store, || {}),
        }
    }; // TODO: add polyhost imports in a future PR
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
    let starter = instance.get_start(&store)?;

    move_me.call(&mut store)?;
    starter.call(&mut store)?;
    check(&mut store, &instance, 12);
    Ok(())
}

#[test]
fn test_import_export_safety() -> Result<()> {
    // test wasms
    //     bad-export.wat   there's a global named `polyglot_gas_left`
    //     bad-export2.wat  there's a func named `polyglot_global_with_random_name`
    //     bad-import.wat   there's an import named `polyglot_global_with_random_name`

    fn check(path: &str, both: bool) -> Result<()> {
        if both {
            let config = PolyglotConfig::default();
            assert!(new_test_instance(path, config).is_err());
        }
        let path = &Path::new(path);
        let wat = std::fs::read(path)?;
        let wasm = wasmer::wat2wasm(&wat)?;
        assert!(binary::parse(&wasm, path).is_err());
        Ok(())
    }

    // TODO: perform all the same checks in instances
    check("tests/bad-export.wat", true)?;
    check("tests/bad-export2.wat", false)?;
    check("tests/bad-import.wat", false)
}

#[test]
fn test_module_mod() -> Result<()> {
    // in module-mod.wat
    //     the func `void` has the signature λ()
    //     the func `more` has the signature λ(i32, i64) -> f32
    //     the func `noop` is imported

    let file = "tests/module-mod.wat";
    let wat = std::fs::read(file)?;
    let wasm = wasmer::wat2wasm(&wat)?;
    let binary = binary::parse(&wasm, &Path::new(file))?;

    let config = PolyglotConfig::default();
    let (instance, _) = new_test_instance(file, config)?;
    let module = instance.module().info();

    let check = |name: &str| {
        let Some(ExportIndex::Function(func)) = module.exports.get(name) else {
            bail!("no func named {}", name.red())
        };
        let wasmer_ty = module.get_function(*func)?;
        let binary_ty = binary.get_function(*func)?;
        assert_eq!(wasmer_ty, binary_ty);
        println!("{} {}", func.as_u32(), binary_ty.blue());
        Ok(())
    };

    check("void")?;
    check("more")
}

#[test]
fn test_heap() -> Result<()> {
    // test wasms
    //     memory.wat   there's a 2-page memory with an upper limit of 4
    //     memory2.wat  there's a 2-page memory with no upper limit

    let mut config = PolyglotConfig::default();
    config.heap_bound = Pages(1).into();
    assert!(new_test_instance("tests/memory.wat", config.clone()).is_err());
    assert!(new_test_instance("tests/memory2.wat", config.clone()).is_err());

    let check = |start: u32, bound: u32, expected: u32, file: &str| -> Result<()> {
        let mut config = PolyglotConfig::default();
        config.heap_bound = Pages(bound).into();

        let (instance, store) = new_test_instance(file, config.clone())?;

        let ty = MemoryType::new(start, Some(expected), false);
        let memory = instance.exports.get_memory("mem")?;
        assert_eq!(ty, memory.ty(&store));
        Ok(())
    };

    check(2, 2, 2, "tests/memory.wat")?;
    check(2, 2, 2, "tests/memory2.wat")?;
    check(2, 3, 3, "tests/memory.wat")?;
    check(2, 3, 3, "tests/memory2.wat")?;
    check(2, 5, 4, "tests/memory.wat")?; // the upper limit of 4 is stricter
    check(2, 5, 5, "tests/memory2.wat")
}
