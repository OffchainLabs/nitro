// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{host::Escape, runtime::Runtime, wasm};
use arbutil::{format, Bytes32};
use eyre::{bail, Result};
use prover::{
    binary::WasmBinary,
    programs::{prelude::CompileConfig, start::StartlessMachine},
};
use std::fs;
use wabt::wat2wasm;
use wasmer::Target;
use stylus::target_cache::target_from_string;

pub fn benchmark(target :String) -> Result<()> {
    
    #[cfg(feature = "affinity")]
    {
        affinity::set_thread_affinity([1]).unwrap();
        let core = affinity::get_thread_affinity().unwrap();
        println!("Affinity {}: {core:?}", std::process::id());
    }
    
    let target=target_from_string(target)?;

    let add64 = excute("i64-add", 20_000, 0., &target)?;
    let add32 = excute("i32-add", 30_000, 0., &target)?;
    excute("i64-mul", 20_000, 0., &target)?;
    excute("i32-mul", 30_000, 0., &target)?;
    excute("i64-div", 10_000, add64, &target)?;
    excute("i32-div", 15_000, add32, &target)?;
    excute("i64-load", 10_000, add64, &target)?;
    excute("i32-load", 10_000, add32, &target)?;
    excute("i64-store", 12_000, 0., &target)?;
    excute("i32-store", 12_000, 0., &target)?;

    excute("i64-xor", 20_000, 0., &target)?;
    excute("i32-xor", 30_000, 0., &target)?;
    let eq64 = excute("i64-eq", 20_000, 0., &target)?;
    let _q32 = excute("i32-eq", 30_000, 0., &target)?;

    excute("i64-clz", 20_000, 0., &target)?;
    excute("i32-clz", 20_000, 0., &target)?;
    excute("i64-popcnt", 20_000, 0., &target)?;
    excute("i32-popcnt", 30_000, 0., &target)?;
    excute("select", 10_000, 0., &target)?;

    excute("global-get", 10_000, add64, &target)?;
    let global = excute("global-set", 6_000, 0., &target)?;

    let get = excute("local-get", 10_000, 0., &target)?;
    let set = excute("local-set", 10_000, 0., &target)?;
    let scramble = excute("local-scramble", 5_000, add64 + 2. * get + 2. * set, &target)?;
    excute("local-locomotion", 2048, get + set, &target)?;
    assert!(scramble < 0.);
    //assert!(locomotion < 0.);

    excute("call", 512, global, &target)?;
    excute("call-indirect", 512, global, &target)?;
    excute(
        "call-indirect-recursive",
        250 * 100,
        125.5 * get + global / 250.,
        &target,
    )?;
    excute("br-table", 100, (50. / 2.) * global + add32, &target)?;
    excute(
        "if",
        300,
        get + add64 + eq64 + 0.4 * (2. * get + add64 + set),
        &target,
    )?;

    excute("memory-size", 20_000, add32, &target)?;

    excute("ink-check", 140, 0., &target)?;

    Ok(())
}

fn excute(file: &str, count: usize, discount: f64, target: &Target) -> Result<f64> {
    
    print!("executing {file}..");
    let wasm = fs::read(format!("../benchmarks/wasm-benchmarks/{file}.wat"))?;
    let wasm = wat2wasm(&wasm)?;
    wasm::validate(&wasm)?;

    // ensure wasm is a reasonable size
    let len = wasm.len() as f64 / 1024. / 128.;
    if len < 1. || len > 2. {
        //bail!("wasm wrong size: {}", len);
        println!(" wrong size: {len}");
    } else {
        println!("")
    }

    let mut compile = CompileConfig::version(2, true);
    compile.debug.count_ops = true;

    // ensure the wasm passes onchain instrumentation
    WasmBinary::parse_user(&wasm, 128, &compile, &Bytes32::default())?;
    _ = Runtime::new(&wasm, compile.clone(), Target::default())?;

    let trials = 8;
    let mut op_min: f64 = f64::MAX;

    for _ in 0..trials {
        let mut runtime = Runtime::new_simple(&wasm, target.clone())?;

        let start = runtime.get_start()?;

        match start.call(&mut runtime.store) {
            Ok(_) => Escape::Incomplete,
            Err(outcome) => match outcome.downcast() {
                Ok(escape) => escape,
                Err(error) => bail!("error: {}", error),
            },
        };

        let time = runtime.time().0;
        let op = (time.as_nanos() as f64 / count as f64) - discount;
        op_min = op_min.min(op);

        let fudge = 2.;
        let sync = 2.;
        let block_time = 1e9 / sync;
        let speed_limit = 7e6 * 10_000.;
        let cost = (fudge * speed_limit * op / block_time).max(0.);

        println!(
            "{file:25}: {}\t=>\t{}",
            format::time(time),
            cost.ceil(),
        );
    }
    Ok(op_min)
}
