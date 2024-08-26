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

pub fn benchmark() -> Result<()> {
    
    #[cfg(feature = "affinity")]
    {
        affinity::set_thread_affinity([1]).unwrap();
        let core = affinity::get_thread_affinity().unwrap();
        println!("Affinity {}: {core:?}", std::process::id());
    }
    
    let add64 = excute("i64-add", 20_000, 0., 200.)?;
    let add32 = excute("i32-add", 30_000, 0., 200.)?;
    excute("i64-mul", 20_000, 0., 550.)?;
    excute("i32-mul", 30_000, 0., 550.)?;
    excute("i64-div", 10_000, add64, 2900.)?;
    excute("i32-div", 15_000, add32, 2500.)?;
    excute("i64-load", 10_000, add64, 2750.)?;
    excute("i32-load", 10_000, add32, 2200.)?;
    excute("i64-store", 12_000, 0., 3100.)?;
    excute("i32-store", 12_000, 0., 2400.)?;

    excute("i64-xor", 20_000, 0., 200.)?;
    excute("i32-xor", 30_000, 0., 200.)?;
    let eq64 = excute("i64-eq", 20_000, 0., 760.)?;
    let _q32 = excute("i32-eq", 30_000, 0., 570.)?;

    excute("i64-clz", 20_000, 0., 750.)?;
    excute("i32-clz", 20_000, 0., 750.)?;
    excute("i64-popcnt", 20_000, 0., 750.)?;
    excute("i32-popcnt", 30_000, 0., 500.)?;
    excute("select", 10_000, 0., 4000.)?;

    excute("global-get", 10_000, add64, 300.)?;
    let global = excute("global-set", 6_000, 0., 990.)?;

    let get = excute("local-get", 10_000, 0., 200.)?;
    let set = excute("local-set", 10_000, 0., 375.)?;
    let scramble = excute("local-scramble", 5_000, add64 + 2. * get + 2. * set, 0.)?;
    let locomotion = excute("local-locomotion", 2048, get + set, 0.)?;
    assert!(scramble < 0.);
    //assert!(locomotion < 0.);

    excute("call", 512, global, 13750.)?;
    excute("call-indirect", 512, global, 13610.)?;
    excute(
        "call-indirect-recursive",
        250 * 100,
        125.5 * get + global / 250.,
        13610., // wrong
    )?;
    excute("br-table", 100, (50. / 2.) * global + add32, 2400.)?;
    excute(
        "if",
        300,
        get + add64 + eq64 + 0.4 * (2. * get + add64 + set),
        2400.,
    )?;

    excute("memory-size", 20_000, add32, 13500.)?;

    excute("ink-check", 140, 0., 2695.)?;

    Ok(())
}

fn excute(file: &str, count: usize, discount: f64, curr: f64) -> Result<f64> {
    println!("executing {file}");
    let wasm = fs::read(format!("../benchmarks/wasm-benchmarks/{file}.wat"))?;
    let wasm = wat2wasm(&wasm)?;
    wasm::validate(&wasm)?;

    // ensure wasm is a reasonable size
    let len = wasm.len() as f64 / 1024. / 128.;
    if len < 1. || len > 2. {
        //bail!("wasm wrong size: {}", len);
        println!("wrong size: {len}");
    }

    let mut compile = CompileConfig::version(1, true);
    compile.debug.count_ops = true;

    // ensure the wasm passes onchain instrumentation
    WasmBinary::parse_user(&wasm, 128, &compile, &Bytes32::default())?;
    _ = Runtime::new(&wasm, compile, Target::default())?;

    let trials = 8;
    let mut op_min: f64 = f64::MAX;
    let mut new_max: f64 = 0.;

    for _ in 0..trials {
        let mut runtime = Runtime::new_simple(&wasm)?;

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

        let old = op / 11.39 * 10_000.;

        let fudge = 2.;
        let sync = 2.;
        let block_time = 1e9 / sync;
        let speed_limit = 7e6 * 10_000.;
        let new = fudge * speed_limit * op / block_time;
        new_max = new_max.max(new);

        let better = 100. * (new - curr) / curr;

        println!(
            "{} => {file}\tis {:.5} {:4} => {:4} ({:.1}% vs {curr})",
            format::time(time),
            op,
            old.ceil(),
            new.ceil(),
            better,
        );
    }

    println!("{file}: {}", new_max.ceil());
    Ok(op_min)
}
