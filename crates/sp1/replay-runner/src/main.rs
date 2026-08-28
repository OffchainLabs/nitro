// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::{
    ops::Deref,
    path::{Path, PathBuf},
    sync::Arc,
};

use crate::cli::Cli;
use crate::input::build_stdin;
use anyhow::{Context, Result, anyhow, bail};
use clap::{Parser, ValueEnum};
use sp1_core_executor::{MinimalExecutor, Program, UserMode};
use sp1_sdk::{
    Elf, ProvingKey, SP1Stdin,
    blocking::{ProveRequest, Prover, ProverClient},
};

mod cli;
mod input;

#[derive(Copy, Clone, Debug, PartialEq, Eq, ValueEnum)]
enum Mode {
    /// Fast execution of the replay ELF, without diagnostics nor proof generation.
    Fast,
    /// Full SP1 simulation: slower and more memory-hungry, but reports cycles, gas, etc.
    Simulate,
    /// Full execution with a validity proof, generated and verified. The most expensive mode.
    Prove,
}

fn main() -> Result<()> {
    sp1_sdk::utils::setup_logger();
    let cli = Cli::parse();

    let elf = cli.replay_elf()?;
    let stdin = build_stdin(&cli)?;

    match cli.mode {
        Mode::Fast => run_fast(elf, stdin),
        Mode::Simulate => run_simulation(elf, stdin),
        Mode::Prove => run_prove(elf, stdin),
    }
}

/// Executes the program directly in the minimal executor.
fn run_fast(elf: Elf, stdin: SP1Stdin) -> Result<()> {
    let program = Arc::new(Program::from(&elf).map_err(|e| anyhow!("parse ELF: {e:#}"))?);
    execute_minimal(program, stdin).map(|_| ())
}

/// Executes the program in the full executor and reports its diagnostics.
fn run_simulation(elf: Elf, stdin: SP1Stdin) -> Result<()> {
    let client = ProverClient::from_env();
    let (_output, report) = client
        .execute(elf, stdin)
        .run()
        .context("SP1 execution")?;

    tracing::info!("cycles: {}", report.total_instruction_count());
    tracing::info!("gas: {}", report.gas().unwrap_or(0));
    tracing::info!("syscalls:");
    for (code, count) in report.syscall_counts.iter() {
        if *count > 0 {
            tracing::info!("  {code}: {count}");
        }
    }
    tracing::info!("cycle trackers:");
    for (entry, cycles) in &report.cycle_tracker {
        tracing::info!("  {entry}: {cycles}");
    }

    if report.exit_code != 0 {
        bail!("program exited with non-zero code: {}", report.exit_code);
    }
    Ok(())
}

/// Executes the program in the full prover, then verifies the generated proof.
fn run_prove(elf: Elf, stdin: SP1Stdin) -> Result<()> {
    let client = ProverClient::from_env();
    let pk = client.setup(elf).context("setup ELF")?;
    let proof = client.prove(&pk, stdin).run().context("generate proof")?;
    client
        .verify(&proof, pk.verifying_key(), None)
        .context("verify proof")?;
    tracing::info!("proof generated and verified successfully");
    Ok(())
}

/// Runs a program in the minimal executor; fails unless it completes with a zero exit code.
fn execute_minimal(program: Arc<Program>, stdin: SP1Stdin) -> Result<Vec<u8>> {
    let mut executor = MinimalExecutor::<UserMode>::simple(program);
    replay_io::send::inject(&stdin, &mut executor);

    if executor.execute_chunk().is_some() {
        bail!("execution failed: executor returned a trace chunk unexpectedly");
    }
    let exit_code = executor.exit_code();
    if exit_code != 0 {
        bail!("program exited with non-zero code: {exit_code}");
    }
    Ok(executor.into_public_values_stream())
}
