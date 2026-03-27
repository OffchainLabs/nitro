use clap::{Parser, Subcommand};
use std::path::PathBuf;

#[derive(Parser)]
#[command(version, about = "Run the Stylus WASM compiler in various execution modes")]
struct Cli {
    #[command(subcommand)]
    command: Command,

    /// Path to the Stylus WASM binary to compile.
    wasm: PathBuf,
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
    let _cli = Cli::parse();
}
