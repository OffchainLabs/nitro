// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use arbutil::evm::{api::Ink, EvmData};
use clap::{Parser, Subcommand};
use core::time::Duration;
use jit::machine::WasmEnv;
use jit::program::{
    exec_program, get_last_msg, pop_with_wasm_env, start_program_with_wasm_env, JitConfig,
};
use prover::programs::{config::CompileConfig, config::PricingParams, prelude::StylusConfig};
use std::fs::File;
use std::io::Write;
use std::path::PathBuf;
use std::str;
use stylus::native::compile;
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

#[derive(Debug, Parser)]
#[command(name = "stylus_benchmark")]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Debug, Subcommand)]
enum Commands {
    #[command(arg_required_else_help = true)]
    Benchmark { wat_path: PathBuf },
    GenerateWats {
        #[arg(value_name = "OUT_PATH")]
        out_path: PathBuf,
    },
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
    if msg.req_type < EVM_API_METHOD_REQ_OFFSET {
        let _ = pop_with_wasm_env(exec);

        let req_data = msg.req_data[8..].to_vec();
        check_result(msg.req_type, &req_data);
    } else {
        panic!("unsupported request type {:?}", msg.req_type);
    }

    let elapsed = msg.benchmark.unwrap().elapsed.expect("elapsed");
    let ink = msg.benchmark.unwrap().ink_total.expect("ink");
    (elapsed, ink)
}

fn benchmark(wat_path: &PathBuf) -> eyre::Result<()> {
    println!("Benchmarking {:?}", wat_path);

    let wat = match std::fs::read(wat_path) {
        Ok(wat) => wat,
        Err(err) => panic!("failed to read: {err}"),
    };
    let wasm = wasmer::wat2wasm(&wat)?;

    let compiled_module = compile(&wasm, 0, true, Target::default())?;

    let mut durations: Vec<Duration> = Vec::new();
    for i in 0..NUMBER_OF_BENCHMARK_RUNS {
        print!("Run {:?}, ", i);
        let (duration, ink) = run(compiled_module.clone());
        durations.push(duration);
        println!("duration: {:?}, ink: {:?}", duration, ink);
    }

    // discard top and bottom runs
    durations.sort();
    let l = NUMBER_OF_TOP_AND_BOTTOM_RUNS_TO_DISCARD as usize;
    let r = NUMBER_OF_BENCHMARK_RUNS as usize - NUMBER_OF_TOP_AND_BOTTOM_RUNS_TO_DISCARD as usize;
    durations = durations[l..r].to_vec();

    let sum = durations.iter().sum::<Duration>();
    println!(
        "Average duration after discarding top and bottom runs: {:?}",
        sum / (r - l) as u32
    );

    Ok(())
}

fn generate_add_i32_wat(mut out_path: PathBuf) -> eyre::Result<()> {
    let number_of_ops = 20_000_000;

    out_path.push("add_i32.wat");
    println!(
        "Generating {:?}, number_of_ops: {:?}",
        out_path, number_of_ops
    );

    let mut file = File::create(out_path)?;

    file.write_all(b"(module\n")?;
    file.write_all(b"    (import \"debug\" \"toggle_benchmark\" (func $toggle_benchmark))\n")?;
    file.write_all(b"    (memory (export \"memory\") 0 0)\n")?;
    file.write_all(b"    (func (export \"user_entrypoint\") (param i32) (result i32)\n")?;

    file.write_all(b"        call $toggle_benchmark\n")?;

    file.write_all(b"        i32.const 1\n")?;
    for _ in 0..number_of_ops {
        file.write_all(b"        i32.const 1\n")?;
        file.write_all(b"        i32.add\n")?;
    }

    file.write_all(b"        call $toggle_benchmark\n")?;

    file.write_all(b"        drop\n")?;
    file.write_all(b"        i32.const 0)\n")?;
    file.write_all(b")")?;

    Ok(())
}

fn generate_wats(out_path: PathBuf) -> eyre::Result<()> {
    return generate_add_i32_wat(out_path);
}

fn main() -> eyre::Result<()> {
    let args = Cli::parse();
    match args.command {
        Commands::Benchmark { wat_path } => {
            return benchmark(&wat_path);
        }
        Commands::GenerateWats { out_path } => {
            return generate_wats(out_path);
        }
    }
}
