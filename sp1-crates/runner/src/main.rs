use clap::{ArgAction, Parser, ValueEnum};
use prover::{
    binary_input::{Input, decompress_aligned},
};
use sp1_core_executor::{MinimalExecutor, Program};
use sp1_sdk::{Elf, Prover, ProverClient, SP1Stdin};
use std::collections::HashMap;
use std::ops::Deref;
use std::sync::Arc;
use std::time::SystemTime;
use validation::ValidationInput;

#[derive(Debug, Parser)]
#[command(version, about, long_about = None)]
struct Cli {
    /// Path to SP1 validation program, could be bootloaded one or the initial one.
    #[arg(long)]
    program: String,

    /// Path to SP1 stylus compiler program
    #[arg(long)]
    stylus_compiler_program: String,

    /// Arbitrum version. Used by the stylus compiler.
    #[arg(long, default_value_t = 2)]
    version: u16,

    /// Debug flag, by default it is true, when set in arguments this
    /// flag will become false. Used by the stylus compiler.
    #[arg(long, action = ArgAction::SetFalse, default_value_t = true)]
    debug: bool,

    /// Block file
    #[arg(long)]
    block_file: String,

    /// Execution mode, fast is, well, faster, but normal mode provides
    /// more diagnosis information, at the expense of longer running time
    /// and more memory.
    #[arg(value_enum, long, default_value_t = Mode::Fast)]
    mode: Mode,
}

#[derive(Copy, Clone, Debug, PartialEq, Eq, PartialOrd, Ord, ValueEnum)]
enum Mode {
    /// Fast mode without detailed opcode statistics
    Fast,

    /// Normal mode
    Normal,
}

#[tokio::main]
async fn main() {
    sp1_sdk::utils::setup_logger();

    let cli = Cli::parse();

    let input_data = build_input(&cli);
    let stdin: SP1Stdin = bincode::deserialize(&input_data).expect("deserializing stdin");

    let program_elf = Elf::from(std::fs::read(&cli.program).expect("read program"));

    let exit_code = match cli.mode {
        Mode::Fast => {
            let program = Arc::new(Program::from(&program_elf).expect("parse elf"));

            let mut executor = MinimalExecutor::simple(program);
            for buf in stdin.buffer {
                executor.with_input(&buf);
            }

            let a = SystemTime::now();
            assert!(executor.execute_chunk().is_none());
            let b = SystemTime::now();
            tracing::info!(
                "Exit code: {}, cycles: {}, execution time: {:?}",
                executor.exit_code(),
                executor.global_clk(),
                b.duration_since(a).unwrap(),
            );

            executor.exit_code() as i32
        }
        Mode::Normal => {
            let client = ProverClient::builder().cpu().build().await;

            let a = SystemTime::now();
            let (_output, report) = client.execute(program_elf, stdin).await.expect("run");
            let b = SystemTime::now();

            tracing::info!(
                "Completed execution, cycles: {}, execution time: {:?}",
                report.total_instruction_count(),
                b.duration_since(a).unwrap(),
            );

            tracing::info!("Syscalls:");
            for (code, count) in report.syscall_counts.iter() {
                if *count > 0 {
                    tracing::info!("  {}: {}", code, count);
                }
            }

            tracing::info!("Cycle trackers:");
            for (entry, cycles) in &report.cycle_tracker {
                tracing::info!("  {} consumed cycles: {}", entry, cycles);
            }

            report.exit_code as i32
        }
    };

    std::process::exit(exit_code);
}

// Build SP1 input from Arbitrum block. It is serialized to Vec<u8>, so
// we can easily inject debugging code to dump stdin when needed.
fn build_input(cli: &Cli) -> Vec<u8> {
    let file_data =
        serde_json::from_slice::<ValidationInput>(&std::fs::read(&cli.block_file).expect("read input block"))
            .expect("parse input block");

    let mut module_asms = HashMap::default();
    if let Some(binaries) = file_data.user_wasms.get("rv64") {
        for (module_hash, binary) in binaries.iter() {
            module_asms.insert(**module_hash, decompress_aligned(binary));
        }
    }
    if let Some(wasms) = file_data.user_wasms.get("wasm") {
        for (module_hash, wasm) in wasms.iter() {
            // rv64 binaries take precedence. This way when nitro introduces
            // caching for rv64 binaries, no changes will be needed for runner.
            if module_asms.contains_key(module_hash.deref()) {
                continue;
            }
            let decompressed = decompress_aligned(wasm);
            let binary = run_in_sp1(&cli, &decompressed);
            module_asms.insert(**module_hash, binary.into());
        }
    }

    let mut input = Input::from_file_data(file_data).expect("create input");
    input.module_asms = module_asms;

    let binary_input = rkyv::to_bytes::<rkyv::rancor::Error>(&input)
        .expect("to bytes")
        .to_vec();
    let mut stdin = SP1Stdin::new();
    stdin.write(&binary_input);
    bincode::serialize(&stdin).expect("serialize with bincode")
}

fn run_in_sp1(cli: &Cli, wasm: &[u8]) -> Vec<u8> {
    let mut stdin = SP1Stdin::new();
    stdin.write(&cli.version);
    stdin.write(&cli.debug);
    stdin.write(&wasm);

    let compiler_elf = std::fs::read(&cli.stylus_compiler_program).expect("read stylus program");
    let program = Arc::new(Program::from(&compiler_elf).expect("parse elf"));

    let mut executor = MinimalExecutor::simple(program);
    for buf in stdin.buffer {
        executor.with_input(&buf);
    }

    let a = SystemTime::now();
    assert!(executor.execute_chunk().is_none());
    let b = SystemTime::now();

    assert_eq!(executor.exit_code(), 0);
    tracing::info!(
        "Completed stylus compilation in SP1, cycles: {}, execution time: {:?}",
        executor.global_clk(),
        b.duration_since(a).unwrap(),
    );

    let public_value_stream = executor.into_public_values_stream();
    bincode::deserialize(&public_value_stream).expect("deserialize")
}
