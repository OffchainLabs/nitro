// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::{
    fs,
    path::{Path, PathBuf},
    str::FromStr,
    sync::Arc,
    time::SystemTime,
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
    sp1_sdk::utils::setup_logger();
    let cli = Cli::parse();

    let wasm = fs::read(&cli.replay_wasm)
        .with_context(|| format!("read replay.wasm from {}", cli.replay_wasm.display()))?;

    let artifacts = Artifacts::build(wasm)?;
    bootload(
        &artifacts.wasmu,
        &artifacts.function_names_json,
        &cli.output_folder,
    )?;
    artifacts.save(&cli.output_folder)
}

/// Bootloads the guest: executes it with the wasmu and name mapping loaded
/// up to its ELF dump point, producing `dumped_replay_wasm.elf`.
fn bootload(wasmu: &[u8], function_names_json: &str, output_folder: &Path) -> anyhow::Result<()> {
    fs::create_dir_all(output_folder).context("create output folder")?;
    let output = match std::env::var("DUMP_ELF_OUTPUT") {
        Ok(s) => s,
        Err(_) => {
            let output = output_folder.join("dumped_replay_wasm.elf");
            unsafe { std::env::set_var("DUMP_ELF_OUTPUT", &output) };
            output.display().to_string()
        }
    };
    let _ = fs::remove_file(&output);

    let program = Arc::new(
        Program::from(&REPLAY_ELF).map_err(|e| anyhow::anyhow!("parse replay ELF: {e:#}"))?,
    );
    let mut executor = MinimalExecutor::<UserMode>::simple(program);
    executor.with_input(wasmu);
    executor.with_input(function_names_json.as_bytes());
    // Bincode-encode the sentinel to match the runner's SP1Stdin wire format;
    // the guest recognizes it and halts cleanly after the ELF dump.
    let bootload_input = bincode::serialize(&SP1_BOOTLOAD_SENTINEL.to_vec())
        .context("serialize bootload sentinel")?;
    executor.with_input(&bootload_input);

    let t0 = SystemTime::now();
    let _ = executor.execute_chunk();
    let time_secs = t0.elapsed().context("measure bootload time")?.as_secs_f64();

    if !fs::exists(&output).context("check bootload output")? {
        bail!("SP1 bootloading failed: expected output at '{output}' was not produced");
    }

    tracing::info!(
        "[PROFILE] bootloading: cycles={}, time_secs={:.3}",
        executor.global_clk(),
        time_secs,
    );
    println!("Bootloaded program is written to {output}");
    Ok(())
}

struct Artifacts {
    /// Function names of replay.wasm (lost in wasmer's compiled output).
    function_names_json: String,
    /// replay.wasm compiled for riscv64.
    wasmu: Bytes,
}

impl Artifacts {
    fn build(wasm: Vec<u8>) -> anyhow::Result<Self> {
        let names = extract_function_names(&wasm)?;
        Ok(Self {
            function_names_json: serde_json::to_string_pretty(&names)
                .context("serialize function names")?,
            wasmu: compile_wasmu(wasm)?,
        })
    }

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
