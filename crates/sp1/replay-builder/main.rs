// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::{fs, path::PathBuf};

use anyhow::Context;
use clap::Parser;
use replay_builder::extract_function_names;
use sp1_sdk::{Elf, include_elf};

const REPLAY_ELF: Elf = include_elf!("replay-program");

#[derive(Parser)]
#[command(about = "Build the SP1 replay-program artifacts")]
struct Cli {
    /// Path to the replay.wasm binary whose execution the guest will host.
    #[arg(long)]
    replay_wasm: PathBuf,

    /// Output folder for generated artifacts.
    #[arg(long)]
    output_folder: PathBuf,
}

fn main() -> anyhow::Result<()> {
    let cli = Cli::parse();

    fs::create_dir_all(&cli.output_folder).context("create output folder")?;

    // Extract the original function names from replay.wasm. They are lost in
    // wasmer's compiled output, and the guest will consume this mapping to
    // register profiler symbols for debugging & profiling.
    let wasm = fs::read(&cli.replay_wasm)
        .with_context(|| format!("read replay.wasm from {}", cli.replay_wasm.display()))?;
    let names = extract_function_names(&wasm)?;
    let names_json = serde_json::to_string_pretty(&names).context("serialize function names")?;
    let names_output = cli.output_folder.join("function_names.json");
    fs::write(&names_output, &names_json).context("write function_names.json")?;
    println!("Wasm function names written to {}", names_output.display());

    let output = cli.output_folder.join("replay-program.elf");
    fs::write(&output, REPLAY_ELF.as_ref()).context("write replay-program.elf")?;
    println!("Replay program ELF written to {}", output.display());

    Ok(())
}
