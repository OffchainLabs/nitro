// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::{
    fs,
    path::{Path, PathBuf},
    str::FromStr,
    sync::Arc,
};

use anyhow::{Context, bail};
use bytes::Bytes;
use clap::Parser;
use replay_builder::extract_function_names;
use sp1_core_executor::{MinimalExecutor, Program, UserMode};
use sp1_sdk::{Elf, include_elf};
use validation::SP1_BOOTLOAD_SENTINEL;
use wasmer::{
    Module, Store,
    sys::{CompilerConfig, CpuFeature, EngineBuilder, LLVM, Target, Triple},
};

const REPLAY_ELF: Elf = include_elf!("replay-program");

/// Env var the SP1 executor dumps the bootloaded ELF to.
const DUMP_ELF_OUTPUT: &str = "DUMP_ELF_OUTPUT";

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

    Artifacts::prepare(&cli.output_folder)?;
    Artifacts::build(wasm)?.save(&cli.output_folder)
}

struct Artifacts {
    /// Function names of replay.wasm (lost in wasmer's compiled output).
    function_names_json: String,
    /// replay.wasm compiled for riscv64.
    wasmu: Bytes,
}

impl Artifacts {
    /// Prepares `output_folder`: creates it and points DUMP_ELF_OUTPUT at the bootload dump target,
    /// removing any stale dump so `save` can assert the fresh one was written. Must run before `build`.
    fn prepare(output_folder: &Path) -> anyhow::Result<()> {
        fs::create_dir_all(output_folder).context("create output folder")?;
        if std::env::var(DUMP_ELF_OUTPUT).is_err() {
            let dump = output_folder.join("dumped_replay_wasm.elf");
            unsafe { std::env::set_var(DUMP_ELF_OUTPUT, dump) };
        }
        let dump = std::env::var(DUMP_ELF_OUTPUT).context("read dump target")?;
        let _ = fs::remove_file(dump);
        Ok(())
    }

    fn build(wasm: Vec<u8>) -> anyhow::Result<Self> {
        let names = extract_function_names(&wasm)?;
        let artifacts = Self {
            function_names_json: serde_json::to_string_pretty(&names)
                .context("serialize function names")?,
            wasmu: compile_wasmu(wasm)?,
        };
        artifacts.bootload()?;
        Ok(artifacts)
    }

    /// Bootloads the guest: executes it with the wasmu and name mapping loaded up to its ELF dump
    /// point (the DUMP_ELF_OUTPUT target).
    fn bootload(&self) -> anyhow::Result<()> {
        let program = Arc::new(
            Program::from(&REPLAY_ELF).map_err(|e| anyhow::anyhow!("parse replay ELF: {e:#}"))?,
        );
        let mut executor = MinimalExecutor::<UserMode>::simple(program);
        executor.with_input(self.wasmu.as_ref());
        executor.with_input(self.function_names_json.as_bytes());
        // Bincode-encode the sentinel to match the runner's SP1Stdin wire
        // format; the guest recognizes it and halts cleanly after the dump.
        let bootload_input = bincode::serialize(&SP1_BOOTLOAD_SENTINEL.to_vec())
            .context("serialize bootload sentinel")?;
        executor.with_input(&bootload_input);

        let _ = executor.execute_chunk();
        Ok(())
    }

    /// Writes all artifacts and ensures bootloading produced its dump.
    fn save(&self, output_folder: &Path) -> anyhow::Result<()> {
        for (name, contents) in [
            ("function_names.json", self.function_names_json.as_bytes()),
            ("replay.wasmu", self.wasmu.as_ref()),
            ("replay-program.elf", REPLAY_ELF.as_ref()),
        ] {
            let output = output_folder.join(name);
            fs::write(&output, contents).with_context(|| format!("write {name}"))?;
            println!("{name} written to {}", output.display());
        }

        let dump = std::env::var(DUMP_ELF_OUTPUT).context("read dump target")?;
        if !fs::exists(&dump).context("check bootload output")? {
            bail!("SP1 bootloading failed: expected output at '{dump}' was not produced");
        }
        println!("Bootloaded program is written to {dump}");
        Ok(())
    }
}

/// Compiles replay.wasm for the riscv64 target with wasmer's LLVM backend into a serialized module.
fn compile_wasmu(wasm: Vec<u8>) -> anyhow::Result<Bytes> {
    let target = Target::new(
        Triple::from_str("riscv64").map_err(|e| anyhow::anyhow!("riscv64 triple: {e}"))?,
        CpuFeature::set(),
    );
    let mut compiler = LLVM::new();
    compiler.canonicalize_nans(true);
    compiler.enable_verifier();
    let store = Store::new(
        EngineBuilder::new(compiler)
            .set_target(Some(target))
            .engine(),
    );
    let module = Module::new(&store, wasm).context("compile replay.wasm")?;
    module.serialize().context("serialize module")
}
