// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use super::test_configs;
use crate::{
    env::{Escape, MaybeEscape},
    native::NativeInstance,
    test::{check_instrumentation, new_test_machine},
};
use eyre::Result;
use prover::programs::{prelude::*, start::STYLUS_START};
use wasmer::{imports, Function};

#[test]
fn test_bulk_memory() -> Result<()> {
    let (compile, config, ink) = test_configs();
    let mut store = compile.store();
    let filename = "../prover/test-cases/bulk-memory.wat";
    let imports = imports! {
        "env" => {
            "wavm_halt_and_set_finished" => Function::new_typed(&mut store, || -> MaybeEscape { Escape::logical("done") }),
        },
    };

    let mut native = NativeInstance::new_from_store(filename, store, imports)?;
    native.set_meter_data();

    let starter = native.get_start()?;
    native.set_stack(config.max_depth);
    native.set_ink(ink);
    starter.call(&mut native.store).unwrap_err();
    assert_ne!(native.ink_left(), MachineMeter::Exhausted);

    let expected = "0000080808050205000002020500020508000000000000000000000000000000";
    let data = native.read_slice("mem", 0x1000, 32)?;
    assert_eq!(expected, hex::encode(data));

    let mut machine = new_test_machine(filename, &compile)?;
    let module = machine.find_module("user")?;
    drop(machine.call_user_func("start", vec![], ink).unwrap_err()); // should halt
    let data = machine.read_memory(module, 0x1000, 32)?;
    assert_eq!(expected, hex::encode(data));

    check_instrumentation(native, machine)
}

#[test]
fn test_bulk_memory_oob() -> Result<()> {
    let filename = "tests/bulk-memory-oob.wat";
    let (compile, _, ink) = test_configs();

    let mut machine = new_test_machine(filename, &compile)?;
    let mut native = NativeInstance::new_test(filename, compile)?;
    let module = machine.find_module("user")?;

    let oobs = ["fill", "copy_left", "copy_right", "copy_same"];
    for oob in &oobs {
        drop(machine.call_user_func(oob, vec![], ink).unwrap_err());

        let exports = &native.instance.exports;
        let oob = exports.get_typed_function::<(), ()>(&native.store, oob)?;
        let err = format!("{}", native.call_func(oob, ink).unwrap_err());
        assert!(err.contains("out of bounds memory access"));
    }
    assert_eq!("0102", hex::encode(native.read_slice("memory", 0xfffe, 2)?));
    assert_eq!("0102", hex::encode(machine.read_memory(module, 0xfffe, 2)?));
    check_instrumentation(native, machine)
}

#[test]
fn test_console() -> Result<()> {
    let filename = "tests/console.wat";
    let (compile, config, ink) = test_configs();

    let mut native = NativeInstance::new_linked(filename, &compile, config)?;
    let starter = native.get_start()?;
    native.call_func(starter, ink)?;

    let mut machine = new_test_machine(filename, &compile)?;
    machine.call_user_func(STYLUS_START, vec![], ink)?;
    check_instrumentation(native, machine)
}
