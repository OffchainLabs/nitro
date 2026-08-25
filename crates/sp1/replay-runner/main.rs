// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::{
    fs,
    path::{Path, PathBuf},
    sync::Arc,
};

use anyhow::{Context, bail};
use clap::{ArgAction, Parser};
use sp1_core_executor::{MinimalExecutor, Program, UserMode};
use stylus_compiler_program::CompileInput;
use validation::{ValidationInput, ValidationRequest};

#[derive(Parser)]
#[command(about = "Validate an Arbitrum block in SP1")]
struct Cli {
    /// Path to the *dumped* SP1 replay program ELF, produced by replay-builder.
    #[arg(long)]
    program: PathBuf,

    /// Path to the SP1 stylus compiler ELF, produced by replay-builder.
    #[arg(long)]
    stylus_compiler_program: PathBuf,

    /// Path to the recorded block JSON (a `ValidationRequest`).
    #[arg(long)]
    block_file: PathBuf,

    /// Arbitrum version. Used by the stylus compiler.
    #[arg(long, default_value_t = 2)]
    version: u16,

    /// Debug flag, true by default; passing `--debug` makes it false. Used by the stylus compiler.
    #[arg(long, action = ArgAction::SetFalse, default_value_t = true)]
    debug: bool,
}

fn main() -> anyhow::Result<()> {
    let cli = Cli::parse();

    let program = build_program(&cli.program)?;
    let mut executor = MinimalExecutor::<UserMode>::simple(Arc::new(program));

    let payload = build_payload(&cli.block_file)?;
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
fn build_payload(block_file: &Path) -> anyhow::Result<Vec<u8>> {
    let block = fs::read(block_file)
        .with_context(|| format!("read block file from {}", block_file.display()))?;
    let request =
        serde_json::from_slice::<ValidationRequest>(&block).context("parse block file")?;

    // Missing rv64 binaries are expected: compiling Stylus programs arriving as wasm sources via
    // the SP1 stylus compiler is the next step to port from the feature branch.
    let input = ValidationInput::from_request_allowing_missing_binaries(&request, "rv64")
        .map_err(anyhow::Error::msg)
        .context("build validation input")?;

    Ok(rkyv::to_bytes::<rkyv::rancor::Error>(&input)
        .context("rkyv-serialize validation input")?
        .to_vec())
}
