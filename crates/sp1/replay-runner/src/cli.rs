use std::{
    fs,
    path::{Path, PathBuf},
};

use anyhow::{Context, Result, anyhow};
use clap::Parser;
use sp1_core_executor::Program;
use sp1_sdk::Elf;
use validation::ValidationRequest;

use crate::Mode;

#[derive(Parser)]
#[command(about = "Validate an Arbitrum block in SP1")]
pub struct Cli {
    /// Path to the *dumped* (bootloaded) SP1 replay program ELF, produced by `replay-builder`.
    #[arg(long)]
    program: PathBuf,

    /// Path to the recorded block JSON (a `ValidationRequest`).
    #[arg(long)]
    block_file: PathBuf,

    /// Execution mode.
    #[arg(value_enum, long, default_value_t = Mode::Fast)]
    pub mode: Mode,

    /// Path to the SP1 stylus compiler ELF, produced by `replay-builder`.
    #[arg(long)]
    stylus_compiler_program: PathBuf,

    /// Stylus version, used by the Stylus compiler.
    #[arg(long, default_value_t = 2)]
    pub stylus_version: u16,
}

impl Cli {
    pub fn replay_elf(&self) -> Result<Elf> {
        self.elf(&self.program)
    }

    pub fn stylus_compiler_program(&self) -> Result<Program> {
        let elf = self.elf(&self.stylus_compiler_program)?;
        Program::from(&elf).map_err(|e| anyhow!("parse ELF: {e:#}"))
    }

    fn elf(&self, path: &Path) -> Result<Elf> {
        Ok(fs::read(path)
            .with_context(|| format!("read ELF from {}", path.display()))?
            .into())
    }

    pub fn validation_request(&self) -> Result<ValidationRequest> {
        let block = fs::read(&self.block_file)
            .with_context(|| format!("read block file from {}", self.block_file.display()))?;
        serde_json::from_slice::<ValidationRequest>(&block).context("parse block file")
    }
}
