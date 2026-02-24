// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! Server Configuration and CLI Argument Parsing.
//!
//! This module defines the command-line interface (CLI) and configuration structures
//! for the validation server. It utilizes `clap` to parse arguments and environment variables
//! into strongly-typed configuration objects used throughout the application.

use anyhow::Result;
use clap::{Parser, ValueEnum};
use std::collections::HashMap;
use std::net::SocketAddr;
use std::path::PathBuf;
use tracing::info;

use crate::engine::machine::JitProcessManager;
use crate::engine::machine_locator::MachineLocator;
use crate::engine::{replay_binary, ModuleRoot, DEFAULT_JIT_CRANELIFT};

pub struct ServerState {
    pub mode: InputMode,
    /// Machine locator is responsible for locating replay.wasm binary and building
    /// a map of module roots to their respective location + binary
    pub locator: MachineLocator,
    /// Jit manager is responsible for computing next GlobalState. Not wrapped
    /// in Arc<> since the caller of ServerState is wrapped in Arc<>.
    pub jit_manager: JitProcessManager,
    pub available_workers: usize,
    /// Cache of pre-compiled WASM modules keyed by module root.
    /// Populated at startup to avoid re-compiling on every native validation request.
    pub module_cache: HashMap<ModuleRoot, jit::CompiledModule>,
}

impl ServerState {
    pub fn new(config: &ServerConfig, available_workers: usize) -> Result<Self> {
        let locator = MachineLocator::new(&config.root_path)?;

        let jit_manager = match config.mode {
            InputMode::Continuous => JitProcessManager::new(&locator)?,
            InputMode::Native => JitProcessManager::new_empty(),
        };

        let mut module_cache = HashMap::new();
        if matches!(config.mode, InputMode::Native) {
            for meta in locator.module_roots() {
                let binary = replay_binary(meta.path.clone());
                let validator_opts = jit::ValidatorOpts {
                    binary: binary.clone(),
                    cranelift: DEFAULT_JIT_CRANELIFT,
                    debug: false,
                    require_success: false,
                };
                match jit::machine::compile_module(&validator_opts) {
                    Ok(compiled) => {
                        info!(
                            "Pre-compiled module for root 0x{} from {binary:?}",
                            meta.module_root
                        );
                        module_cache.insert(meta.module_root, compiled);
                    }
                    Err(err) => {
                        warn!(
                            "Failed to pre-compile module for root 0x{}: {err}",
                            meta.module_root
                        );
                    }
                }
            }
        }

        Ok(ServerState {
            mode: config.mode,
            locator,
            jit_manager,
            available_workers,
            module_cache,
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

    /// Root path to where 0x1234.../replay.wasm machines are located
    #[clap(long)]
    pub root_path: Option<PathBuf>,
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
