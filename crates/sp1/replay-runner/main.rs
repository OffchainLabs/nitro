// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::{
    fs,
    ops::Deref,
    path::{Path, PathBuf},
    sync::Arc,
};

use anyhow::{Context, bail};
use clap::{ArgAction, Parser};
use sp1_core_executor::{MinimalExecutor, Program, UserMode};
use sp1_sdk::SP1Stdin;
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

    /// Path to the SP1 stylus compiler ELF, produced by replay-builder.
    #[arg(long)]
    stylus_compiler_program: PathBuf,

    /// Stylus version, used by the Stylus compiler.
    #[arg(long, default_value_t = 2)]
    stylus_version: u16,

    /// Turns the debug mode for Stylus compilation off.
    #[arg(long)]
    stylus_debug_off: bool,
}

fn main() -> anyhow::Result<()> {
    let cli = Cli::parse();

    let program = build_program(&cli.program)?;
    let mut executor = MinimalExecutor::<UserMode>::simple(Arc::new(program));

    let payload = build_payload(&cli)?;
    replay_io::send::validation_mode(&mut executor, &payload);

    if executor.execute_chunk().is_some() {
        bail!("execution failed: executor returned a trace chunk unexpectedly");
    }

    let exit_code = executor.exit_code();
    if exit_code != 0 {
        bail!("program exited with non-zero code: {exit_code}");
    }
    Ok(())
}

fn build_program(program_file: &Path) -> anyhow::Result<Program> {
    let elf = fs::read(program_file)
        .with_context(|| format!("read program ELF from {}", program_file.display()))?;
    Program::from(&elf).map_err(|e| anyhow::anyhow!("parse program ELF: {e:#}"))
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
        for (module_hash, wasm) in wasms.iter() {
            if input.module_asms.contains_key(module_hash.deref()) {
                continue;
            }
            let binary = compile_in_sp1(cli, wasm.as_ref())?;
            input.module_asms.insert(**module_hash, binary);
        }
    }

    Ok(rkyv::to_bytes::<rkyv::rancor::Error>(&input)
        .context("rkyv-serialize validation input")?
        .to_vec())
}

/// Compiles a Stylus wasm to a rv64 binary by running the stylus compiler inside SP1.
fn compile_in_sp1(cli: &Cli, wasm: &[u8]) -> anyhow::Result<Vec<u8>> {
    let compile_input = CompileInput {
        version: cli.stylus_version,
        debug: !cli.stylus_debug_off,
        wasm: wasm.to_vec(),
    };

    let mut stdin = SP1Stdin::new();
    stdin.write(&compile_input);

    let program = build_program(&cli.stylus_compiler_program)?;
    let mut executor = MinimalExecutor::<UserMode>::simple(Arc::new(program));
    for input in &stdin.buffer {
        executor.with_input(input);
    }

    if executor.execute_chunk().is_some() {
        bail!("stylus compilation in SP1 failed: executor returned a trace chunk unexpectedly");
    }
    let exit_code = executor.exit_code();
    if exit_code != 0 {
        bail!("stylus compiler exited with non-zero code: {exit_code}");
    }

    bincode::deserialize(&executor.into_public_values_stream())
        .context("deserialize compiled binary")
}
