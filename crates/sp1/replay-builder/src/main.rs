// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! Builds the artifacts from replay.wasm: the riscv64 `wasmu` module and the function-name mapping.
//! Then, it generates `dumped_replay_wasm.elf` by bootloading the guest (fed with the artifacts).

use std::{
    fs,
    path::{Path, PathBuf},
    str::FromStr,
    sync::Arc,
};

use anyhow::{Context, Result, anyhow, bail};
use bytes::Bytes;
use clap::Parser;
use replay_builder::extract_function_names;
use sp1_core_executor::{MinimalExecutor, Program, UserMode};
use sp1_sdk::{Elf, include_elf};
use wasmer::{
    Module, Store,
    sys::{CompilerConfig, CpuFeature, EngineBuilder, LLVM, Target, Triple},
};

/// The ELF of the replay guest program.
const REPLAY_ELF: Elf = include_elf!("replay-program");
/// The SP1 executor reads the dump destination from this env var.
const SP1_DUMP_TARGET_ENV: &str = "DUMP_ELF_OUTPUT";

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

fn main() -> Result<()> {
    let cli = Cli::parse();

    let wasm = fs::read(&cli.replay_wasm)
        .with_context(|| format!("read replay.wasm from {}", cli.replay_wasm.display()))?;

    let artifacts = Artifacts::build(&wasm)?;
    artifacts.save(&cli.output_folder)?;

    bootload(
        &artifacts,
        &cli.output_folder.join("dumped_replay_wasm.elf"),
    )
}

/// Artifacts generated from the original `replay.wasm`.
struct Artifacts {
    /// Function names of replay.wasm (lost in wasmer's compiled output).
    function_names_json: String,
    /// replay.wasm compiled for riscv64.
    wasmu: Bytes,
}

impl Artifacts {
    fn build(wasm: &[u8]) -> Result<Self> {
        let names = extract_function_names(wasm)?;
        Ok(Self {
            function_names_json: serde_json::to_string_pretty(&names)
                .context("serialize function names")?,
            wasmu: compile_wasmu(wasm)?,
        })
    }

    fn save(&self, output_folder: &Path) -> Result<()> {
        fs::create_dir_all(output_folder).context("create output folder")?;
        for (name, contents) in [
            ("function_names.json", self.function_names_json.as_bytes()),
            ("replay.wasmu", self.wasmu.as_ref()),
        ] {
            let output = output_folder.join(name);
            fs::write(&output, contents).with_context(|| format!("write {name}"))?;
            println!("{name} written to {}", output.display());
        }
        Ok(())
    }
}

/// Compiles replay.wasm for the riscv64 target with wasmer's LLVM backend into a serialized module.
fn compile_wasmu(wasm: &[u8]) -> Result<Bytes> {
    let target = Target::new(
        Triple::from_str("riscv64").map_err(|e| anyhow!("riscv64 triple: {e}"))?,
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

/// Executes the guest with the wasmu and name mapping loaded up to its ELF dump point. Ensures that
/// the dumped state is saved in `dump_target`.
fn bootload(artifacts: &Artifacts, dump_target: &Path) -> Result<()> {
    prepare_bootload(dump_target);

    let program = Program::from(&REPLAY_ELF).map_err(|e| anyhow!("parse replay ELF: {e:#}"))?;
    let mut executor = MinimalExecutor::<UserMode>::simple(Arc::new(program));
    replay_io::send::bootload_mode(&mut executor, &artifacts.wasmu, &artifacts.function_names_json);

    let _ = executor.execute_chunk();
    let exit_code = executor.exit_code();
    if exit_code != 0 {
        bail!("bootload execution exited with code {exit_code}");
    }

    check_bootload_output(dump_target)
}

fn prepare_bootload(dump_target: &Path) {
    unsafe { std::env::set_var(SP1_DUMP_TARGET_ENV, dump_target) };
    let _ = fs::remove_file(dump_target);
}

fn check_bootload_output(dump_target: &Path) -> Result<()> {
    if !fs::exists(dump_target).context("check bootload output")? {
        bail!(
            "SP1 bootloading failed: expected output at '{}' was not produced",
            dump_target.display()
        );
    }
    println!(
        "dumped_replay_wasm.elf written to {}",
        dump_target.display()
    );
    Ok(())
}
