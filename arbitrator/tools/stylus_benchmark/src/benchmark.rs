// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use arbutil::evm::{api::Ink, EvmData};
use core::time::Duration;
use jit::machine::WasmEnv;
use jit::program::JitConfig;
use prover::programs::{config::CompileConfig, config::PricingParams, prelude::StylusConfig};
use std::str;
use stylus::native;
use wasmer::Target;

const EVM_API_METHOD_REQ_OFFSET: u32 = 0x10000000;

const NUMBER_OF_BENCHMARK_RUNS: u32 = 7;
const NUMBER_OF_TOP_AND_BOTTOM_RUNS_TO_DISCARD: u32 = 2;

fn check_result(req_type: u32, req_data: &Vec<u8>) {
    let _ = match str::from_utf8(req_data) {
        Ok(v) => v,
        Err(e) => panic!("Invalid UTF-8 sequence: {}", e),
    };

    match req_type {
        0 => return,
        1 => panic!("ErrExecutionReverted user revert"),
        2 => panic!("ErrExecutionReverted user failure"),
        3 => panic!("ErrOutOfGas user out of ink"),
        4 => panic!("ErrDepth user out of stack"),
        _ => panic!("ErrExecutionReverted user unknown"),
    }
}

fn run(compiled_module: Vec<u8>) -> (Duration, Ink) {
    let calldata = Vec::from([0u8; 32]);
    let evm_data = EvmData::default();
    let config = JitConfig {
        stylus: StylusConfig {
            version: 2,
            max_depth: 10000,
            pricing: PricingParams { ink_price: 1 },
        },
        compile: CompileConfig::version(2, true),
    };

    let exec = &mut WasmEnv::default();

    let module = jit::program::launch_program_thread(
        exec,
        compiled_module.into(),
        calldata,
        config,
        evm_data,
        u64::MAX,
    )
    .unwrap();

    let req_id = jit::program::start_program_with_wasm_env(exec, module).unwrap();
    let msg = jit::program::get_last_msg(exec, req_id).unwrap();
    if msg.req_type < EVM_API_METHOD_REQ_OFFSET {
        let _ = jit::program::pop_with_wasm_env(exec);

        let req_data = msg.req_data[8..].to_vec();
        check_result(msg.req_type, &req_data);
    } else {
        panic!("unsupported request type {:?}", msg.req_type);
    }

    (msg.benchmark.elapsed_total, msg.benchmark.ink_total)
}

pub fn benchmark(wat: Vec<u8>) -> eyre::Result<()> {
    let wasm = wasmer::wat2wasm(&wat)?;

    let compiled_module = native::compile(&wasm, 2, true, Target::default(), false)?;

    let mut durations: Vec<Duration> = Vec::new();
    let mut ink_spent = Ink(0);
    for i in 0..NUMBER_OF_BENCHMARK_RUNS {
        print!("Run {:?}, ", i);
        let (duration_run, ink_spent_run) = run(compiled_module.clone());
        durations.push(duration_run);
        ink_spent = ink_spent_run;
        println!(
            "duration: {:?}, ink_spent: {:?}",
            duration_run, ink_spent_run
        );
    }

    // discard top and bottom runs
    durations.sort();
    let l = NUMBER_OF_TOP_AND_BOTTOM_RUNS_TO_DISCARD as usize;
    let r = NUMBER_OF_BENCHMARK_RUNS as usize - NUMBER_OF_TOP_AND_BOTTOM_RUNS_TO_DISCARD as usize;
    durations = durations[l..r].to_vec();

    let avg_duration = durations.iter().sum::<Duration>() / (r - l) as u32;
    let avg_ink_spent_per_nano_second = ink_spent.0 / avg_duration.as_nanos() as u64;
    println!("After discarding top and bottom runs: ");
    println!(
        "avg_duration: {:?}, avg_ink_spent_per_nano_second: {:?}",
        avg_duration, avg_ink_spent_per_nano_second
    );

    Ok(())
}
