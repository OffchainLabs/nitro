// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use super::test_configs;
use crate::{
    env::{Escape, MaybeEscape},
    native::NativeInstance,
    test::{check_instrumentation, new_test_machine, run_machine_read_memory},
};
use eyre::Result;
use prover::programs::{prelude::*, start::StartMover};
use wasmer::{imports, Function, Target};

#[test]
fn test_bulk_memory() -> Result<()> {
    let (compile, config, ink) = test_configs();
    let mut store = compile.store(Target::default(), false);
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
    let data = native.read_slice("memory", 0x1000, 32)?;
    assert_eq!(expected, hex::encode(data));

    let mut machine = new_test_machine(filename, &compile)?;
    let module = machine.find_module("user")?;
    drop(machine.call_user_func("start", vec![], ink).unwrap_err()); // should halt
    let data = machine.read_memory(module, 0x1000, 32)?;
    assert_eq!(expected, hex::encode(data));

    check_instrumentation(native, machine)
}

#[test]
fn test_memory_fill_value_overflow() -> Result<()> {
    // Verifies the version gate for the memory.fill fix (version >= 3).
    // value=0x100: low 8 bits = 0x00, so correct output is all zeros.
    let filename = "../prover/test-cases/memory-fill-overflow.wat";
    let (_, _, ink) = test_configs();

    // V2 is the last buggy version: upper bits of value leak into the fill pattern.
    // value=0x100 produces a fill pattern of 0x0101_0101_0101_0100 (little-endian i64.store),
    // so the first 3 bytes written are 0x00 and the remaining 7 are 0x01.
    let compile_v2 = CompileConfig::version(2, true);
    let machine_v2_data = run_machine_read_memory(filename, &compile_v2, "run", ink, 0xaaa, 10)?;
    assert_eq!(
        machine_v2_data,
        [0x0, 0x0, 0x0, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1, 0x1]
    );

    // V3 is the first fixed version: value correctly masked to 8 bits
    let compile_v3 = CompileConfig::version(3, true);
    let machine_v3_data = run_machine_read_memory(filename, &compile_v3, "run", ink, 0xaaa, 10)?;
    assert_eq!(machine_v3_data, vec![0u8; 10]);

    Ok(())
}

#[test]
fn test_memory_fill_value_overflow_nonzero() -> Result<()> {
    // Verifies that V3 preserves non-zero low bits, not just zero-fills everything.
    // Uses value 0x1ab (low 8 bits = 0xab); all filled bytes must be 0xab.
    let filename = "../prover/test-cases/memory-fill-overflow.wat";
    let (_, _, ink) = test_configs();
    let compile_v3 = CompileConfig::version(3, true);

    let machine_data =
        run_machine_read_memory(filename, &compile_v3, "run_nonzero", ink, 0xbbb, 10)?;
    assert_eq!(machine_data, vec![0xab_u8; 10]);

    Ok(())
}

#[test]
fn test_memory_fill_overflow_native_trap() -> Result<()> {
    let filename = "tests/memory-fill-overflow.wat";
    let (compile, _, ink) = test_configs();

    let mut native = NativeInstance::new_test(filename, compile)?;

    let exports = &native.instance.exports;
    let fill = exports.get_typed_function::<(), ()>(&native.store, "fill_overflow")?;
    let err = format!("{}", native.call_func(fill, ink).unwrap_err());
    assert!(
        err.contains("memory.fill value exceeds 8 bits"),
        "expected overflow error, got: {}",
        err,
    );

    Ok(())
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
    machine.call_user_func(StartMover::NAME, vec![], ink)?;
    check_instrumentation(native, machine)
}
