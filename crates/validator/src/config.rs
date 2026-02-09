// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! Server Configuration and CLI Argument Parsing.
//!
//! This module defines the command-line interface (CLI) and configuration structures
//! for the validation server. It utilizes `clap` to parse arguments and environment variables
//! into strongly-typed configuration objects used throughout the application.

use anyhow::Result;
use clap::{Args, Parser, ValueEnum};
use std::net::SocketAddr;
use std::path::PathBuf;

use crate::engine::config::{JitManagerConfig, ModuleRoot};
use crate::engine::machine::JitProcessManager;
use crate::engine::machine_locator::MachineLocator;

#[derive(Debug)]
pub struct ServerState {
    pub mode: InputMode,
    /// Machine locator is responsible for locating replay.wasm binary and building
    /// a map of module roots to their respective location + binary
    pub locator: MachineLocator,
    /// Jit manager is responsible for computing next GlobalState. Not wrapped
    /// in Arc<> since the caller of ServerState is wrapped in Arc<>. This field
    /// is optional because it's only available in continuous InputMode
    pub jit_manager: JitProcessManager,
    pub available_workers: usize,
}

impl ServerState {
    pub fn new(config: &ServerConfig, available_workers: usize) -> Result<Self> {
        let manager_config = JitManagerConfig::default();
        let locator = MachineLocator::new()?;

        let jit_manager = match config.mode {
            InputMode::Continuous => JitProcessManager::new(&manager_config, &locator)?,
            InputMode::Native => JitProcessManager::new_empty(&manager_config),
        };
        Ok(ServerState {
            mode: config.mode,
            locator,
            jit_manager,
            available_workers,
        })
    }
}

#[derive(Copy, Clone, Debug, ValueEnum)]
pub enum InputMode {
    Native,
    Continuous,
}

#[derive(Copy, Clone, Debug, ValueEnum, Default)]
pub enum LoggingFormat {
    #[default]
    Text,
    Json,
}
use tracing::warn;

const DEFAULT_NUM_WORKERS: usize = 4;

#[derive(Clone, Debug, Parser)]
pub struct ServerConfig {
    /// Socket address where the server should be run.
    #[clap(long, default_value = "0.0.0.0:4141")]
    pub address: SocketAddr,

    /// Logging format configuration.
    #[clap(long, value_enum, default_value_t = LoggingFormat::Text)]
    pub logging_format: LoggingFormat,

    /// Defines how Validator consumes input
    #[clap(long, value_enum, default_value_t = InputMode::Native)]
    pub mode: InputMode,

    #[clap(long)]
    workers: Option<usize>,
}

#[derive(Clone, Debug, Args)]
#[group(required = true, multiple = false)]
struct ModuleRootConfig {
    /// Supported module root.
    #[clap(long)]
    module_root: Option<ModuleRoot>,

    /// Path to the file containing the module root.
    #[clap(long)]
    module_root_path: Option<PathBuf>,
}

impl ServerConfig {
    pub fn get_workers(&self) -> Result<usize> {
        if let Some(workers) = self.workers {
            Ok(workers)
        } else {
            let workers = match std::thread::available_parallelism() {
                Ok(count) => count.get(),
                Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
                    warn!("Could not determine machine's available parallelism. Defaulting to {DEFAULT_NUM_WORKERS}.");
                    DEFAULT_NUM_WORKERS
                }
                Err(e) => return Err(e.into()),
            };
            Ok(workers)
        }
    }
}

#[cfg(test)]
mod tests {
    use clap::Parser;

    use crate::config::ServerConfig;

    #[test]
    fn verify_cli() {
        use clap::CommandFactory;
        ServerConfig::command().debug_assert()
    }

    #[test]
    fn capacity_parsing() {
        let server_config = ServerConfig::try_parse_from(["server"]).unwrap();

        assert!(server_config.workers.is_none());
        let workers = server_config.get_workers().unwrap();
        assert!(workers > 0);
    }
}
