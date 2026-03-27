use clap::{Parser, Subcommand};
use sp1_sdk::{
    blocking::ProveRequest,
    blocking::{Prover, ProverClient},
    include_elf, Elf, ProvingKey, SP1Stdin,
};
use std::path::PathBuf;
use stylus_compiler_program::{compile, CompileInput};

const COMPILER_ELF: Elf = include_elf!("stylus-compiler-program");

#[derive(Parser)]
#[command(about = "Run the Stylus WASM compiler in various execution modes")]
struct Cli {
    #[command(subcommand)]
    command: Command,

    /// Path to the Stylus WASM binary to compile.
    #[arg(default_value = "crates/sp1/stylus-compiler-runner/testdata/memory.wasm")]
    wasm: PathBuf,

    /// Arbitrum version passed to the Stylus compiler.
    #[arg(long, default_value_t = 2)]
    version: u16,

    /// Enable debug compilation mode.
    #[arg(long)]
    debug: bool,
}

#[derive(Subcommand)]
enum Command {
    /// Compile the WASM natively using the host Stylus compiler.
    ///
    /// Uses the same wasmer singlepass pipeline as the SP1 program, but runs
    /// directly on the host without any zkVM overhead. Useful for quick
    /// iteration and as a reference output for comparison.
    Native,

    /// Compile the WASM inside SP1 in fast execution mode (no proof).
    ///
    /// Runs the stylus-compiler-program ELF through SP1's MinimalExecutor.
    /// Fast and lightweight — validates that the SP1 program executes correctly
    /// without the cost of proof generation.
    Execute,

    /// Compile the WASM inside SP1 and generate a validity proof.
    ///
    /// Runs the stylus-compiler-program ELF through the full SP1 prover,
    /// producing a proof that the compilation was executed correctly.
    /// Slower than `execute` but provides cryptographic guarantees.
    Prove,

    /// Compile natively and via SP1, then assert the outputs match.
    ///
    /// Runs both `native` and `execute` and compares the resulting rv64
    /// binaries byte-for-byte. Useful for verifying that the SP1 program
    /// faithfully reproduces the native compilation result.
    Compare,
}

fn main() {
    sp1_sdk::utils::setup_logger();

    let cli = Cli::parse();
    let wasm = std::fs::read(&cli.wasm).expect("failed to read wasm file");
    let input = CompileInput {
        version: cli.version,
        debug: cli.debug,
        wasm,
    };

    match cli.command {
        Command::Native => {
            let binary = compile(&input).expect("native compilation failed");
            tracing::info!("compiled successfully, output size: {} bytes", binary.len());
        }
        Command::Execute => {
            let binary = sp1_execute(&input);
            tracing::info!(
                "SP1 execution completed, output size: {} bytes",
                binary.len()
            );
        }
        Command::Prove => sp1_prove(&input),
        Command::Compare => {
            let native = compile(&input).expect("native compilation failed");
            let sp1 = sp1_execute(&input);
            assert_eq!(native, sp1, "native and SP1 outputs differ");
            tracing::info!("outputs match ({} bytes)", native.len());
        }
    }
}

fn build_stdin(input: &CompileInput) -> SP1Stdin {
    let mut stdin = SP1Stdin::new();
    stdin.write(input);
    stdin
}

fn sp1_execute(input: &CompileInput) -> Vec<u8> {
    let client = ProverClient::from_env();
    let stdin = build_stdin(input);
    let (output, report) = client
        .execute(COMPILER_ELF, stdin)
        .run()
        .expect("SP1 execution failed");
    tracing::info!("cycles: {}", report.total_instruction_count());
    bincode::deserialize(output.as_slice()).expect("deserialize output")
}

fn sp1_prove(input: &CompileInput) {
    let client = ProverClient::from_env();
    let stdin = build_stdin(input);
    let pk = client.setup(COMPILER_ELF).expect("failed to setup ELF");
    let proof = client
        .prove(&pk, stdin)
        .run()
        .expect("failed to generate proof");
    client
        .verify(&proof, pk.verifying_key(), None)
        .expect("failed to verify proof");
    tracing::info!("proof generated and verified successfully");
}
