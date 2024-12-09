// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use arbutil::evm::EvmData;
use clap::Parser;
use core::time::Duration;
use jit::machine::WasmEnv;
use jit::program::{
    exec_program, get_last_msg, pop_with_wasm_env, start_program_with_wasm_env, JitConfig,
};
use prover::programs::{config::CompileConfig, config::PricingParams, prelude::StylusConfig};
use std::path::PathBuf;
use std::str;
use stylus::native::compile;
use wasmer::Target;

const EVM_API_METHOD_REQ_OFFSET: u32 = 0x10000000;

const NUMBER_OF_BENCHMARK_RUNS: u32 = 7;
const NUMBER_OF_TOP_AND_BOTTOM_RUNS_TO_DISCARD: u32 = 2;

fn to_result(req_type: u32, req_data: &Vec<u8>) -> (&str, &str) {
    let msg = match str::from_utf8(req_data) {
        Ok(v) => v,
        Err(e) => panic!("Invalid UTF-8 sequence: {}", e),
    };

    match req_type {
        0 => return ("", ""),                      // userSuccess
        1 => return (msg, "ErrExecutionReverted"), // userRevert
        2 => return (msg, "ErrExecutionReverted"), // userFailure
        3 => return ("", "ErrOutOfGas"),           // userOutOfInk
        4 => return ("", "ErrDepth"),              // userOutOfStack
        _ => return ("", "ErrExecutionReverted"),  // userUnknown
    }
}

#[derive(Parser, Debug)]
#[command(author, version, about, long_about = None)]
struct Args {
    // Path to the wat file to be benchmarked
    #[arg(short, long)]
    wat_path: PathBuf,
}

fn run(compiled_module: Vec<u8>) -> Duration {
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

    let module = exec_program(
        exec,
        compiled_module.into(),
        calldata,
        config,
        evm_data,
        160000000,
    )
    .unwrap();

    let req_id = start_program_with_wasm_env(exec, module).unwrap();
    let msg = get_last_msg(exec, req_id).unwrap();
    println!(
        "req_id: {:?}, msg.req_type: {:?}, msg.req_data: {:?}, msg.benchmark.elapsed: {:?}",
        req_id,
        msg.req_type,
        msg.req_data,
        msg.benchmark.unwrap().elapsed,
    );

    if msg.req_type < EVM_API_METHOD_REQ_OFFSET {
        let _ = pop_with_wasm_env(exec);

        let gas_left = u64::from_be_bytes(msg.req_data[..8].try_into().unwrap());
        let req_data = msg.req_data[8..].to_vec();
        let (msg, err) = to_result(msg.req_type, &req_data);
        println!(
            "gas_left: {:?}, msg: {:?}, err: {:?}, req_data: {:?}",
            gas_left, msg, err, req_data
        );
        if err != "" {
            panic!("error: {:?}", err);
        }
    } else {
        panic!("unsupported request");
    }

    msg.benchmark.unwrap().elapsed.expect("timer_elapsed")
}

fn benchmark(wat_path: &PathBuf) -> eyre::Result<()> {
    let mut durations: Vec<Duration> = Vec::new();

    let wat = match std::fs::read(wat_path) {
        Ok(wat) => wat,
        Err(err) => panic!("failed to read: {err}"),
    };
    let wasm = wasmer::wat2wasm(&wat).unwrap();

    let compiled_module = compile(&wasm, 0, true, Target::default()).unwrap();

    for i in 0..NUMBER_OF_BENCHMARK_RUNS {
        println!("Benchmark run {:?} {:?}", i, wat_path);
        let duration = run(compiled_module.clone());
        durations.push(duration);
    }

    durations.sort();
    println!("durations: {:?}", durations);

    let l = NUMBER_OF_TOP_AND_BOTTOM_RUNS_TO_DISCARD as usize;
    let r = NUMBER_OF_BENCHMARK_RUNS as usize - NUMBER_OF_TOP_AND_BOTTOM_RUNS_TO_DISCARD as usize;
    let sum = durations[l..r].to_vec().iter().sum::<Duration>();
    println!(
        "sum {:?}, average duration: {:?}",
        sum,
        sum / (r - l) as u32
    );

    Ok(())
}

fn main() -> eyre::Result<()> {
    let args = Args::parse();
    return benchmark(&args.wat_path);
}
