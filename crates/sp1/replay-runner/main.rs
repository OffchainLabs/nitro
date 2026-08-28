// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::{
    fs,
    ops::Deref,
    path::{Path, PathBuf},
    sync::Arc,
};

use anyhow::{Context, bail};
use clap::{Parser, ValueEnum};
use sp1_core_executor::{MinimalExecutor, Program, UserMode};
use sp1_sdk::{
    Elf, ProvingKey, SP1Stdin,
    blocking::{ProveRequest, Prover, ProverClient},
};
use stylus_compiler_program::CompileInput;
use validation::{ValidationInput, ValidationRequest};

#[derive(Parser)]
#[command(about = "Validate an Arbitrum block in SP1")]
struct Cli {
    /// Path to the *dumped* SP1 replay program ELF, produced by replay-builder.
    #[arg(long)]
    program: PathBuf,

    /// Path to the recorded block JSON (a `ValidationRequest`).
    #[arg(long)]
    block_file: PathBuf,

    /// Execution mode.
    #[arg(value_enum, long, default_value_t = Mode::Fast)]
    mode: Mode,

    /// Path to the SP1 stylus compiler ELF, produced by replay-builder.
    #[arg(long)]
    stylus_compiler_program: PathBuf,

    /// Stylus version, used by the Stylus compiler.
    #[arg(long, default_value_t = 2)]
    stylus_version: u16,
}

#[derive(Copy, Clone, Debug, PartialEq, Eq, ValueEnum)]
enum Mode {
    /// Direct execution, without diagnostics.
    Fast,
    /// Full execution: slower and more memory-hungry, but reports cycles, gas, syscall counts,
    /// and cycle trackers.
    Normal,
    /// Full execution with a validity proof, generated and verified. The most expensive mode.
    Prove,
}

fn main() -> anyhow::Result<()> {
    sp1_sdk::utils::setup_logger();
    let cli = Cli::parse();

    let payload = build_payload(&cli)?;
    let stdin = replay_io::send::validation_stdin(&payload);

    match cli.mode {
        Mode::Fast => run_fast(&cli.program, stdin),
        Mode::Normal => run_normal(&cli.program, stdin),
        Mode::Prove => run_prove(&cli.program, stdin),
    }
}

/// Executes the program directly in the minimal executor.
fn run_fast(program_file: &Path, stdin: SP1Stdin) -> anyhow::Result<()> {
    let program = Arc::new(build_program(program_file)?);
    execute_minimal(program, stdin)?;
    Ok(())
}

/// Executes the program in the full executor and reports its diagnostics.
fn run_normal(program_file: &Path, stdin: SP1Stdin) -> anyhow::Result<()> {
    let elf = read_elf(program_file)?;

    let client = ProverClient::from_env();
    let (_output, report) = client
        .execute(elf, stdin)
        .run()
        .context("normal-mode execution")?;

    tracing::info!("cycles: {}", report.total_instruction_count());
    tracing::info!("gas: {}", report.gas().unwrap_or(0));
    tracing::info!("syscalls:");
    for (code, count) in report.syscall_counts.iter() {
        if *count > 0 {
            tracing::info!("  {code}: {count}");
        }
    }
    tracing::info!("cycle trackers:");
    for (entry, cycles) in &report.cycle_tracker {
        tracing::info!("  {entry}: {cycles}");
    }

    if report.exit_code != 0 {
        bail!("program exited with non-zero code: {}", report.exit_code);
    }
    Ok(())
}

/// Executes the program in the full prover, then verifies the generated proof.
fn run_prove(program_file: &Path, stdin: SP1Stdin) -> anyhow::Result<()> {
    let elf = read_elf(program_file)?;

    let client = ProverClient::from_env();
    let pk = client.setup(elf).context("setup ELF")?;
    let proof = client.prove(&pk, stdin).run().context("generate proof")?;
    client
        .verify(&proof, pk.verifying_key(), None)
        .context("verify proof")?;
    tracing::info!("proof generated and verified successfully");
    Ok(())
}

fn read_elf(program_file: &Path) -> anyhow::Result<Elf> {
    let elf = fs::read(program_file)
        .with_context(|| format!("read program ELF from {}", program_file.display()))?;
    Ok(Elf::from(elf))
}

fn load_program(program_file: &Path) -> anyhow::Result<Program> {
    let elf = read_elf(program_file)?;
    Program::from(&elf).map_err(|e| anyhow::anyhow!("parse program ELF: {e:#}"))
}

/// Runs a program in the minimal executor; fails unless it completes with a zero exit code.
fn execute_minimal(
    program: Arc<Program>,
    stdin: SP1Stdin,
) -> anyhow::Result<MinimalExecutor<UserMode>> {
    let mut executor = MinimalExecutor::<UserMode>::simple(program);
    replay_io::send::inject(stdin, &mut executor);

    if executor.execute_chunk().is_some() {
        bail!("execution failed: executor returned a trace chunk unexpectedly");
    }
    let exit_code = executor.exit_code();
    if exit_code != 0 {
        bail!("program exited with non-zero code: {exit_code}");
    }
    Ok(executor)
}

/// Builds the validation payload from a recorded block: the rkyv-serialized `ValidationInput`.
fn build_payload(cli: &Cli) -> anyhow::Result<Vec<u8>> {
    let block = fs::read(&cli.block_file)
        .with_context(|| format!("read block file from {}", cli.block_file.display()))?;
    let request =
        serde_json::from_slice::<ValidationRequest>(&block).context("parse block file")?;

    // Missing rv64 binaries are expected: Stylus programs arriving as wasm sources are compiled
    // via the SP1 stylus compiler right below.
    let mut input = ValidationInput::from_request_allowing_missing_binaries(&request, "rv64")
        .map_err(anyhow::Error::msg)
        .context("build validation input")?;

    if let Some(wasms) = request.user_wasms.get("wasm") {
        let compiler = Arc::new(load_program(&cli.stylus_compiler_program)?);
        for (module_hash, wasm) in wasms.iter() {
            if input.module_asms.contains_key(module_hash.deref()) {
                continue;
            }
            let binary = compile_in_sp1(
                compiler.clone(),
                wasm.as_ref(),
                cli.stylus_version,
                request.debug_chain,
            )?;
            input.module_asms.insert(**module_hash, binary);
        }
    }

    Ok(rkyv::to_bytes::<rkyv::rancor::Error>(&input)
        .context("rkyv-serialize validation input")?
        .to_vec())
}

/// Compiles a Stylus wasm to a rv64 binary by running the stylus compiler inside SP1.
fn compile_in_sp1(
    compiler: Arc<Program>,
    wasm: &[u8],
    version: u16,
    debug: bool,
) -> anyhow::Result<Vec<u8>> {
    let compile_input = CompileInput {
        version,
        debug,
        wasm: wasm.to_vec(),
    };

    let mut stdin = SP1Stdin::new();
    stdin.write(&compile_input);

    let executor = execute_minimal(compiler, stdin).context("stylus compilation in SP1")?;
    bincode::deserialize(&executor.into_public_values_stream())
        .context("deserialize compiled binary")
}
