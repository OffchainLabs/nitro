// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::{path::PathBuf, sync::Arc};

use anyhow::{Context, bail};
use clap::Parser;
use sp1_core_executor::{MinimalExecutor, Program, UserMode};

#[derive(Parser)]
#[command(about = "Validate an Arbitrum block in SP1")]
struct Cli {
    /// Path to the SP1 replay program ELF, produced by replay-builder.
    #[arg(long)]
    program: PathBuf,
}

fn main() -> anyhow::Result<()> {
    let cli = Cli::parse();

    let elf = std::fs::read(&cli.program)
        .with_context(|| format!("read program ELF from {}", cli.program.display()))?;
    let program = Program::from(&elf).map_err(|e| anyhow::anyhow!("parse program ELF: {e:#}"))?;

    let mut executor = MinimalExecutor::<UserMode>::simple(Arc::new(program));
    if executor.execute_chunk().is_some() {
        bail!("execution failed: executor returned a trace chunk unexpectedly");
    }

    let exit_code = executor.exit_code();
    if exit_code != 0 {
        bail!("program exited with non-zero code: {exit_code}");
    }
    Ok(())
}
