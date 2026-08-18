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

    let wasm = fs::read(&cli.replay_wasm)
        .with_context(|| format!("read replay.wasm from {}", cli.replay_wasm.display()))?;

    let names = extract_function_names(&wasm)?;
    let artifacts = Artifacts {
        function_names_json: serde_json::to_string_pretty(&names)
            .context("serialize function names")?,
        wasmu: compile_wasmu(wasm)?,
    };
    artifacts.save(&cli.output_folder)
}

struct Artifacts {
    /// Function names of replay.wasm (lost in wasmer's compiled output).
    function_names_json: String,
    /// replay.wasm compiled for riscv64.
    wasmu: Bytes,
}

impl Artifacts {
    fn save(&self, output_folder: &Path) -> anyhow::Result<()> {
        fs::create_dir_all(output_folder).context("create output folder")?;
        for (name, contents) in [
            ("function_names.json", self.function_names_json.as_bytes()),
            ("replay.wasmu", self.wasmu.as_ref()),
            ("replay-program.elf", REPLAY_ELF.as_ref()),
        ] {
            let output = output_folder.join(name);
            fs::write(&output, contents).with_context(|| format!("write {name}"))?;
            println!("{name} written to {}", output.display());
        }
        Ok(())
    }
}

/// Compiles replay.wasm for the riscv64 target with wasmer's LLVM backend
/// into a serialized module.
fn compile_wasmu(wasm: Vec<u8>) -> anyhow::Result<Bytes> {
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
    module.serialize().context("serialize module")
}
