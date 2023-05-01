// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(
    clippy::field_reassign_with_default,
    clippy::inconsistent_digit_grouping
)]

use crate::{
    run::RunProgram,
    test::{
        check_instrumentation, new_test_machine, random_bytes20, random_bytes32, random_ink,
        run_machine, run_native, test_compile_config, test_configs, TestInstance,
    },
};
use arbutil::{crypto, evm::user::UserOutcome, format, Bytes20, Bytes32, Color};
use eyre::{bail, Result};
use p256::ecdsa::{
    signature::{Signer, Verifier},
    Signature, SigningKey, VerifyingKey,
};
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
use wasmer::{CompilerConfig, ExportIndex, Imports, MemoryType, Pages, Store};
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

    let starter = StartMover::default();
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

    fn check(path: &str, both: bool) -> Result<()> {
        if both {
            let compile = test_compile_config();
            assert!(TestInstance::new_test(path, compile).is_err());
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
    // test wasms
    //     memory.wat   there's a 2-page memory with an upper limit of 4
    //     memory2.wat  there's a 2-page memory with no upper limit

    let mut compile = CompileConfig::default();
    compile.bounds.heap_bound = Pages(1).into();
    assert!(TestInstance::new_test("tests/memory.wat", compile.clone()).is_err());
    assert!(TestInstance::new_test("tests/memory2.wat", compile).is_err());

    let check = |start: u32, bound: u32, expected: u32, file: &str| -> Result<()> {
        let mut compile = CompileConfig::default();
        compile.bounds.heap_bound = Pages(bound).into();

        let instance = TestInstance::new_test(file, compile.clone())?;
        let machine = new_test_machine(file, &compile)?;

        let ty = MemoryType::new(start, Some(expected), false);
        let memory = instance.exports.get_memory("mem")?;
        assert_eq!(ty, memory.ty(&instance.store));

        let memory = machine.main_module_memory();
        assert_eq!(expected as u64, memory.max_size);
        Ok(())
    };

    check(2, 2, 2, "tests/memory.wat")?;
    check(2, 2, 2, "tests/memory2.wat")?;
    check(2, 3, 3, "tests/memory.wat")?;
    check(2, 3, 3, "tests/memory2.wat")?;
    check(2, 5, 4, "tests/memory.wat")?; // the upper limit of 4 is stricter
    check(2, 5, 5, "tests/memory2.wat")
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

    /*let mut machine = Machine::from_user_path(Path::new(filename), &compile)?;
    let output = run_machine(&mut machine, &args, config, ink)?;
    assert_eq!(hex::encode(output), hash);

    check_instrumentation(native, machine)*/
    Ok(())
}

#[test]
fn test_c() -> Result<()> {
    // in siphash.c
    //     the inputs are a hash, key, and plaintext
    //     the output is whether the hash was valid

    let filename = "tests/siphash/siphash.wasm";
    let (compile, config, ink) = test_configs();

    let text: Vec<u8> = (0..63).collect();
    let key: Vec<u8> = (0..16).collect();
    let key: [u8; 16] = key.try_into().unwrap();
    let hash = crypto::siphash(&text, &key);

    let mut args = hash.to_le_bytes().to_vec();
    args.extend(key);
    args.extend(text);
    let args_string = hex::encode(&args);

    let mut native = TestInstance::new_linked(filename, &compile, config)?;
    let output = run_native(&mut native, &args, ink)?;
    assert_eq!(hex::encode(output), args_string);

    /*let mut machine = Machine::from_user_path(Path::new(filename), &compile)?;
    let output = run_machine(&mut machine, &args, config, ink)?;
    assert_eq!(hex::encode(output), args_string);

    check_instrumentation(native, machine)*/
    Ok(())
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

    /*let mut machine = Machine::from_user_path(Path::new(filename), &compile)?;
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
    assert_eq!(native.stack_left(), machine.stack_left());*/
    Ok(())
}

/*#[test]
fn test_storage() -> Result<()> {
    // in storage.rs
    //     an input starting with 0x00 will induce a storage read
    //     all other inputs induce a storage write

    let filename = "tests/storage/target/wasm32-unknown-unknown/release/storage.wasm";
    let (compile, config, ink) = test_configs();

    let key = crypto::keccak(filename.as_bytes());
    let value = crypto::keccak("value".as_bytes());

    let mut args = vec![0x01];
    args.extend(key);
    args.extend(value);

    let address = Bytes20::default();
    let mut native = TestInstance::new_linked(filename, &compile, config)?;
    let api = native.set_test_evm_api(
        address,
        TestEvmStorage::default(),
        TestEvmContracts::new(compile, config),
    );

    run_native(&mut native, &args, ink)?;
    assert_eq!(api.get_bytes32(address, Bytes32(key)), Some(Bytes32(value)));

    args[0] = 0x00; // load the value
    let output = run_native(&mut native, &args, ink)?;
    assert_eq!(output, value);
    Ok(())
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

    let (mut native, mut contracts, storage) =
        TestInstance::new_with_evm(&filename, compile, config)?;
    contracts.insert(calls_addr, "multicall")?;
    contracts.insert(store_addr, "storage")?;

    run_native(&mut native, &args, ink)?;

    for (key, value) in slots {
        assert_eq!(storage.get_bytes32(store_addr, key), Some(value));
    }
    Ok(())
}
*/
