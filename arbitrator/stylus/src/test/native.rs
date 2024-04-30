// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(
    clippy::field_reassign_with_default,
    clippy::inconsistent_digit_grouping
)]

use crate::{
    run::RunProgram,
    test::{
        check_instrumentation, random_bytes20, random_bytes32, random_ink, run_machine, run_native,
        test_compile_config, test_configs, TestInstance,
    },
};
use arbutil::{
    crypto,
    evm::{
        api::EvmApi,
        user::{UserOutcome, UserOutcomeKind},
    },
    format, Bytes20, Bytes32, Color,
};
use eyre::{bail, ensure, Result};
use prover::{
    binary,
    programs::{
        counter::{Counter, CountingMachine},
        prelude::*,
        start::StartMover,
        MiddlewareWrapper, ModuleMod,
    },
    Machine,
};
use std::{collections::HashMap, path::Path, sync::Arc, time::Instant};
use wasmer::wasmparser::Operator;
use wasmer::{CompilerConfig, ExportIndex, Imports, Pages, Store};
use wasmer_compiler_singlepass::Singlepass;

#[test]
fn test_ink() -> Result<()> {
    let mut compile = test_compile_config();
    compile.pricing.costs = super::expensive_add;

    let mut native = TestInstance::new_test("tests/add.wat", compile)?;
    let exports = &native.exports;
    let add_one = exports.get_typed_function::<i32, i32>(&native.store, "add_one")?;

    macro_rules! exhaust {
        ($ink:expr) => {
            native.set_ink($ink);
            assert_eq!(native.ink_left(), MachineMeter::Ready($ink));
            assert!(add_one.call(&mut native.store, 32).is_err());
            assert_eq!(native.ink_left(), MachineMeter::Exhausted);
        };
    }

    exhaust!(0);
    exhaust!(50);
    exhaust!(99);

    let mut ink_left = 500;
    native.set_ink(ink_left);
    while ink_left > 0 {
        assert_eq!(native.ink_left(), MachineMeter::Ready(ink_left));
        assert_eq!(add_one.call(&mut native.store, 64)?, 65);
        ink_left -= 100;
    }
    assert!(add_one.call(&mut native.store, 32).is_err());
    assert_eq!(native.ink_left(), MachineMeter::Exhausted);
    Ok(())
}

#[test]
fn test_depth() -> Result<()> {
    // in depth.wat
    //    the `depth` global equals the number of times `recurse` is called
    //    the `recurse` function calls itself
    //    the `recurse` function has 1 parameter and 2 locals
    //    comments show that the max depth is 3 words

    let mut native = TestInstance::new_test("tests/depth.wat", test_compile_config())?;
    let exports = &native.exports;
    let recurse = exports.get_typed_function::<i64, ()>(&native.store, "recurse")?;

    let program_depth: u32 = native.get_global("depth")?;
    assert_eq!(program_depth, 0);

    let mut check = |space: u32, expected: u32| -> Result<()> {
        native.set_global("depth", 0)?;
        native.set_stack(space);
        assert_eq!(native.stack_left(), space);

        assert!(recurse.call(&mut native.store, 0).is_err());
        assert_eq!(native.stack_left(), 0);

        let program_depth: u32 = native.get_global("depth")?;
        assert_eq!(program_depth, expected);
        Ok(())
    };

    let locals = 2;
    let depth = 3;
    let fixed = 4;

    let frame_size = locals + depth + fixed;

    check(frame_size, 0)?; // should immediately exhaust (space left <= frame)
    check(frame_size + 1, 1)?;
    check(2 * frame_size, 1)?;
    check(2 * frame_size + 1, 2)?;
    check(4 * frame_size, 3)?;
    check(4 * frame_size + frame_size / 2, 4)
}

#[test]
fn test_start() -> Result<()> {
    // in start.wat
    //     the `status` global equals 10 at initialization
    //     the `start` function increments `status`
    //     by the spec, `start` must run at initialization

    fn check(native: &mut TestInstance, value: i32) -> Result<()> {
        let status: i32 = native.get_global("status")?;
        assert_eq!(status, value);
        Ok(())
    }

    let mut native = TestInstance::new_vanilla("tests/start.wat")?;
    check(&mut native, 11)?;

    let mut native = TestInstance::new_test("tests/start.wat", test_compile_config())?;
    check(&mut native, 10)?;

    let exports = &native.exports;
    let move_me = exports.get_typed_function::<(), ()>(&native.store, "move_me")?;
    let starter = native.get_start()?;
    let ink = random_ink(100_000);

    native.call_func(move_me, ink)?;
    native.call_func(starter, ink)?;
    check(&mut native, 12)?;
    Ok(())
}

#[test]
fn test_count() -> Result<()> {
    let mut compiler = Singlepass::new();
    compiler.canonicalize_nans(true);
    compiler.enable_verifier();

    let starter = StartMover::new(true);
    let counter = Counter::new();
    compiler.push_middleware(Arc::new(MiddlewareWrapper::new(starter)));
    compiler.push_middleware(Arc::new(MiddlewareWrapper::new(counter)));

    let mut instance =
        TestInstance::new_from_store("tests/clz.wat", Store::new(compiler), Imports::new())?;

    let starter = instance.get_start()?;
    starter.call(&mut instance.store)?;

    let counts = instance.operator_counts()?;
    let check = |value: Operator<'_>| counts.get(&value.into());

    use Operator::*;
    assert_eq!(check(Unreachable), None);
    assert_eq!(check(Drop), Some(&1));
    assert_eq!(check(I64Clz), Some(&1));

    // test the instrumentation's contribution
    assert_eq!(check(GlobalGet { global_index: 0 }), Some(&8)); // one in clz.wat
    assert_eq!(check(GlobalSet { global_index: 0 }), Some(&7));
    assert_eq!(check(I64Add), Some(&7));
    assert_eq!(check(I64Const { value: 0 }), Some(&7));
    Ok(())
}

#[test]
fn test_import_export_safety() -> Result<()> {
    // test wasms
    //     bad-export.wat   there's a global named `stylus_ink_left`
    //     bad-export2.wat  there's a func named `stylus_global_with_random_name`
    //     bad-import.wat   there's an import named `stylus_global_with_random_name`

    fn check(file: &str, both: bool, instrument: bool) -> Result<()> {
        let path = &Path::new(file);
        let wat = std::fs::read(path)?;
        let wasm = wasmer::wat2wasm(&wat)?;
        let bin = binary::parse(&wasm, path);
        if !instrument {
            assert!(bin.is_err());
            return Ok(());
        }

        let mut compile = test_compile_config();
        let mut bin = bin?;
        assert!(bin.clone().instrument(&compile).is_err());
        compile.debug.debug_info = false;
        assert!(bin.instrument(&compile).is_err());

        if both {
            assert!(TestInstance::new_test(file, compile).is_err());
        }
        Ok(())
    }

    // TODO: perform all the same checks in instances
    check("tests/bad-mods/bad-export.wat", true, false)?;
    check("tests/bad-mods/bad-export2.wat", true, false)?;
    check("tests/bad-mods/bad-export3.wat", true, true)?;
    check("tests/bad-mods/bad-export4.wat", false, true)?;
    check("tests/bad-mods/bad-import.wat", true, false)
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
    let binary = binary::parse(&wasm, Path::new(file))?;

    let native = TestInstance::new_test(file, test_compile_config())?;
    let module = native.module().info();

    assert_eq!(module.all_functions()?, binary.all_functions()?);
    assert_eq!(module.all_signatures()?, binary.all_signatures()?);

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
    // in memory.wat
    //     the input is the target size and amount to step each `memory.grow`
    //     the output is the memory size in pages

    let (mut compile, config, _) = test_configs();
    compile.bounds.heap_bound = Pages(128);
    compile.pricing.costs = |_, _| 0;

    let extra: u8 = rand::random::<u8>() % 128;

    for step in 1..128 {
        let (mut native, _) = TestInstance::new_with_evm("tests/memory.wat", &compile, config)?;
        let ink = random_ink(32_000_000);
        let args = vec![128, step];

        let pages = run_native(&mut native, &args, ink)?[0];
        assert_eq!(pages, 128);

        let used = config.pricing.ink_to_gas(ink - native.ink_ready()?);
        ensure!((used as i64 - 32_000_000).abs() < 3_000, "wrong ink");
        assert_eq!(native.memory_size(), Pages(128));

        if step == extra {
            let mut machine = Machine::from_user_path(Path::new("tests/memory.wat"), &compile)?;
            run_machine(&mut machine, &args, config, ink)?;
            check_instrumentation(native, machine)?;
        }
    }

    // in memory2.wat
    //     the user program calls pay_for_memory_grow directly with malicious arguments
    //     the cost should exceed a maximum u32, consuming more gas than can ever be bought

    let (mut native, _) = TestInstance::new_with_evm("tests/memory2.wat", &compile, config)?;
    let outcome = native.run_main(&[], config, config.pricing.ink_to_gas(u32::MAX.into()))?;
    assert_eq!(outcome.kind(), UserOutcomeKind::OutOfInk);

    // ensure we reject programs with excessive footprints
    compile.bounds.heap_bound = Pages(0);
    _ = TestInstance::new_with_evm("tests/memory.wat", &compile, config).unwrap_err();
    _ = Machine::from_user_path(Path::new("tests/memory.wat"), &compile).unwrap_err();
    Ok(())
}

#[test]
fn test_rust() -> Result<()> {
    // in keccak.rs
    //     the input is the # of hashings followed by a preimage
    //     the output is the iterated hash of the preimage

    let filename = "tests/keccak/target/wasm32-unknown-unknown/release/keccak.wasm";
    let preimage = "°º¤ø,¸,ø¤°º¤ø,¸,ø¤°º¤ø,¸ nyan nyan ~=[,,_,,]:3 nyan nyan";
    let preimage = preimage.as_bytes().to_vec();
    let hash = hex::encode(crypto::keccak(&preimage));
    let (compile, config, ink) = test_configs();

    let mut args = vec![0x01];
    args.extend(preimage);

    let mut native = TestInstance::new_linked(filename, &compile, config)?;
    let start = Instant::now();
    let output = run_native(&mut native, &args, ink)?;
    println!("Exec {}", format::time(start.elapsed()));
    assert_eq!(hex::encode(output), hash);

    let mut machine = Machine::from_user_path(Path::new(filename), &compile)?;
    let start = Instant::now();
    let output = run_machine(&mut machine, &args, config, ink)?;
    assert_eq!(hex::encode(output), hash);
    println!("Exec {}", format::time(start.elapsed()));

    check_instrumentation(native, machine)
}

#[test]
fn test_fallible() -> Result<()> {
    // in fallible.rs
    //     an input starting with 0x00 will execute an unreachable
    //     an empty input induces a panic

    let filename = "tests/fallible/target/wasm32-unknown-unknown/release/fallible.wasm";
    let (compile, config, ink) = test_configs();

    let mut native = TestInstance::new_linked(filename, &compile, config)?;
    match native.run_main(&[0x00], config, ink)? {
        UserOutcome::Failure(err) => println!("{}", format!("{err:?}").grey()),
        err => bail!("expected hard error: {}", err.red()),
    }
    match native.run_main(&[], config, ink)? {
        UserOutcome::Failure(err) => println!("{}", format!("{err:?}").grey()),
        err => bail!("expected hard error: {}", err.red()),
    }

    let mut machine = Machine::from_user_path(Path::new(filename), &compile)?;
    match machine.run_main(&[0x00], config, ink)? {
        UserOutcome::Failure(err) => println!("{}", format!("{err:?}").grey()),
        err => bail!("expected hard error: {}", err.red()),
    }
    match machine.run_main(&[], config, ink)? {
        UserOutcome::Failure(err) => println!("{}", format!("{err:?}").grey()),
        err => bail!("expected hard error: {}", err.red()),
    }

    let native_counts = native.operator_counts()?;
    let machine_counts = machine.operator_counts()?;
    assert_eq!(native_counts, machine_counts);
    assert_eq!(native.ink_left(), machine.ink_left());
    assert_eq!(native.stack_left(), machine.stack_left());
    Ok(())
}

#[test]
fn test_storage() -> Result<()> {
    // in storage.rs
    //     an input starting with 0x00 will induce a storage read
    //     all other inputs induce a storage write

    let filename = "tests/storage/target/wasm32-unknown-unknown/release/storage.wasm";
    let (compile, config, ink) = test_configs();

    let key = crypto::keccak(filename.as_bytes());
    let value = crypto::keccak("value".as_bytes());

    let mut store_args = vec![0x01];
    store_args.extend(key);
    store_args.extend(value);

    let mut load_args = vec![0x00];
    load_args.extend(key);

    let (mut native, mut evm) = TestInstance::new_with_evm(filename, &compile, config)?;
    run_native(&mut native, &store_args, ink)?;
    assert_eq!(evm.get_bytes32(key.into()).0, Bytes32(value));
    assert_eq!(run_native(&mut native, &load_args, ink)?, value);

    let mut machine = Machine::from_user_path(Path::new(filename), &compile)?;
    run_machine(&mut machine, &store_args, config, ink)?;
    assert_eq!(run_machine(&mut machine, &load_args, config, ink)?, value);

    check_instrumentation(native, machine)
}

#[test]
fn test_calls() -> Result<()> {
    // in call.rs
    //     the first bytes determines the number of calls to make
    //     each call starts with a length specifying how many input bytes it constitutes
    //     the first byte determines the kind of call to be made (normal, delegate, or static)
    //     the next 20 bytes select the address you want to call, with the rest being calldata
    //
    // in storage.rs
    //     an input starting with 0x00 will induce a storage read
    //     all other inputs induce a storage write

    let calls_addr = random_bytes20();
    let store_addr = random_bytes20();
    println!("calls.wasm {}", calls_addr);
    println!("store.wasm {}", store_addr);

    let mut slots = HashMap::new();

    /// Forms a 2ary call tree where each leaf writes a random storage cell.
    fn nest(
        level: usize,
        calls: Bytes20,
        store: Bytes20,
        slots: &mut HashMap<Bytes32, Bytes32>,
    ) -> Vec<u8> {
        let mut args = vec![];

        if level == 0 {
            // call storage.wasm
            args.push(0x00);
            args.extend(Bytes32::default());
            args.extend(store);

            let key = random_bytes32();
            let value = random_bytes32();
            slots.insert(key, value);

            // insert value @ key
            args.push(0x01);
            args.extend(key);
            args.extend(value);
            return args;
        }

        // do the two following calls
        args.push(0x00);
        args.extend(Bytes32::default());
        args.extend(calls);
        args.push(2);

        for _ in 0..2 {
            let inner = nest(level - 1, calls, store, slots);
            args.extend(u32::to_be_bytes(inner.len() as u32));
            args.extend(inner);
        }
        args
    }

    // drop the first address to start the call tree
    let tree = nest(3, calls_addr, store_addr, &mut slots);
    let args = tree[53..].to_vec();
    println!("ARGS {}", hex::encode(&args));

    let filename = "tests/multicall/target/wasm32-unknown-unknown/release/multicall.wasm";
    let (compile, config, ink) = test_configs();

    let (mut native, mut evm) = TestInstance::new_with_evm(filename, &compile, config)?;
    evm.deploy(calls_addr, config, "multicall")?;
    evm.deploy(store_addr, config, "storage")?;

    run_native(&mut native, &args, ink)?;

    for (key, value) in slots {
        assert_eq!(evm.get_bytes32(key).0, value);
    }
    Ok(())
}

#[test]
fn test_exit_early() -> Result<()> {
    // in exit-early.wat
    //     the input is returned as the output
    //     the status code is the first byte
    //
    // in panic-after-write.wat
    //     the program writes a result but then panics

    let file = |f: &str| format!("tests/exit-early/{f}.wat");
    let (compile, config, ink) = test_configs();
    let args = &[0x01; 32];

    let mut native = TestInstance::new_linked(file("exit-early"), &compile, config)?;
    let output = match native.run_main(args, config, ink)? {
        UserOutcome::Revert(output) => output,
        err => bail!("expected revert: {}", err.red()),
    };
    assert_eq!(hex::encode(output), hex::encode(args));

    let mut native = TestInstance::new_linked(file("panic-after-write"), &compile, config)?;
    match native.run_main(args, config, ink)? {
        UserOutcome::Failure(error) => println!("{error:?}"),
        err => bail!("expected hard error: {}", err.red()),
    }
    Ok(())
}
