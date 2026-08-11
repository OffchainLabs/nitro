// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::{fs, path::PathBuf};

use anyhow::Context;
use clap::Parser;
use sp1_sdk::{Elf, include_elf};

const REPLAY_ELF: Elf = include_elf!("replay-program");

#[derive(Parser)]
#[command(about = "Build the SP1 replay-program artifacts")]
struct Cli {
    /// Output folder for generated artifacts.
    #[arg(long)]
    output_folder: PathBuf,
}

/// Mocked builder: the real one will compile replay.wasm with wasmer's LLVM
/// backend and bootload the guest for faster startup; for now it only
/// materializes the guest ELF so the runner has something to execute.
fn main() -> anyhow::Result<()> {
    let cli = Cli::parse();

    fs::create_dir_all(&cli.output_folder).context("create output folder")?;
    let output = cli.output_folder.join("replay-program.elf");
    fs::write(&output, REPLAY_ELF.as_ref()).context("write replay-program.elf")?;

    println!("Replay program ELF written to {}", output.display());
    Ok(())
}
