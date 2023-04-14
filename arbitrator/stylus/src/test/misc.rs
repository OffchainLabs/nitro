// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    env::{Escape, MaybeEscape},
    native::NativeInstance,
    test::{check_instrumentation, new_test_machine, uniform_cost_config},
};
use eyre::Result;
use prover::programs::{prelude::*, start::STYLUS_START};
use wasmer::{imports, Function, Instance, Module};

#[test]
fn test_bulk_memory() -> Result<()> {
    let config = uniform_cost_config();
    let mut store = config.store();
    let filename = "../prover/test-cases/bulk-memory.wat";
    let wat = std::fs::read(filename)?;
    let wasm = wasmer::wat2wasm(&wat)?;
    let module = Module::new(&store, &wasm)?;
    let imports = imports! {
        "env" => {
            "wavm_halt_and_set_finished" => Function::new_typed(&mut store, || -> MaybeEscape { Escape::logical("done") }),
        },
    };

    let instance = Instance::new(&mut store, &module, &imports)?;
    let mut native = NativeInstance::new_sans_env(instance, store);
    let starter = native.get_start()?;
    starter.call(&mut native.store).unwrap_err();
    assert_ne!(native.ink_left(), MachineMeter::Exhausted);

    let expected = "0000080808050205000002020500020508000000000000000000000000000000";
    let data = native.read_slice("mem", 0x1000, 32)?;
    assert_eq!(expected, hex::encode(data));

    let mut machine = new_test_machine(filename, &config)?;
    let module = machine.find_module("user")?;
    drop(machine.call_function("user", "start", vec![]).unwrap_err()); // should halt
    let data = machine.read_memory(module, 0x1000, 32)?;
    assert_eq!(expected, hex::encode(data));

    check_instrumentation(native, machine)
}

#[test]
fn test_bulk_memory_oob() -> Result<()> {
    let filename = "tests/bulk-memory-oob.wat";
    let config = uniform_cost_config();

    let mut machine = new_test_machine(filename, &config)?;
    let module = machine.find_module("user")?;

    let mut native = NativeInstance::from_path(filename, &config)?;
    let store = &mut native.store;
    let exports = &native.instance.exports;

    let oobs = ["fill", "copy_left", "copy_right", "copy_same"];
    for oob in &oobs {
        drop(machine.call_function("user", oob, vec![]).unwrap_err());

        let oob = exports.get_typed_function::<(), ()>(store, oob)?;
        let err = format!("{}", oob.call(store).unwrap_err());
        assert!(err.contains("out of bounds memory access"));
    }
    assert_eq!("0102", hex::encode(native.read_slice("memory", 0xfffe, 2)?));
    assert_eq!("0102", hex::encode(machine.read_memory(module, 0xfffe, 2)?));
    check_instrumentation(native, machine)
}

#[test]
fn test_console() -> Result<()> {
    let filename = "tests/console.wat";
    let config = uniform_cost_config();

    let mut machine = new_test_machine(filename, &config)?;
    machine.call_function("user", STYLUS_START, vec![])?;

    let mut native = NativeInstance::from_path(filename, &config)?;
    let starter = native.get_start()?;
    starter.call(&mut native.store)?;
    Ok(())
}
