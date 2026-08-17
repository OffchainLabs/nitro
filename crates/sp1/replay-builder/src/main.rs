// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::{
    fs,
    path::{Path, PathBuf},
    str::FromStr,
};

use anyhow::Context;
use bytes::Bytes;
use clap::Parser;
use replay_builder::extract_function_names;
use sp1_sdk::{Elf, include_elf};
use wasmer::{
    Module, Store,
    sys::{CpuFeature, EngineBuilder, LLVM, Target, Triple},
};

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
    let wasm = fs::read(&cli.replay_wasm)
        .with_context(|| format!("read replay.wasm from {}", cli.replay_wasm.display()))?;

    write_function_names(&wasm, &cli.output_folder)?;
    compile_wasmu(wasm, &cli.output_folder)?;
    write_artifact(&cli.output_folder, "replay-program.elf", REPLAY_ELF.as_ref())?;

    Ok(())
}

/// Writes an artifact to `output_folder/name` and reports the destination.
fn write_artifact(
    output_folder: &Path,
    name: &str,
    contents: impl AsRef<[u8]>,
) -> anyhow::Result<()> {
    let output = output_folder.join(name);
    fs::write(&output, contents).with_context(|| format!("write {name}"))?;
    println!("{name} written to {}", output.display());
    Ok(())
}

/// Writes replay.wasm's function names (lost in wasmer's compiled output) to
/// `function_names.json` for the guest's profiler symbols.
fn write_function_names(wasm: &[u8], output_folder: &Path) -> anyhow::Result<()> {
    let names = extract_function_names(wasm)?;
    let names_json = serde_json::to_string_pretty(&names).context("serialize function names")?;
    write_artifact(output_folder, "function_names.json", &names_json)
}

/// Compiles replay.wasm for the riscv64 target with wasmer's LLVM backend and
/// writes the serialized module to `replay.wasmu`; the bootloading step will
/// consume the returned bytes.
fn compile_wasmu(wasm: Vec<u8>, output_folder: &Path) -> anyhow::Result<Bytes> {
    let target = Target::new(
        Triple::from_str("riscv64").map_err(|e| anyhow::anyhow!("riscv64 triple: {e}"))?,
        CpuFeature::set(),
    );
    let store = Store::new(
        EngineBuilder::new(LLVM::new())
            .set_target(Some(target))
            .engine(),
    );
    let module = Module::new(&store, wasm).context("compile replay.wasm")?;
    let wasmu = module.serialize().context("serialize module")?;

    write_artifact(output_folder, "replay.wasmu", &wasmu)?;
    Ok(wasmu)
}
