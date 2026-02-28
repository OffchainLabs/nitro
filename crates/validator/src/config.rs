// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! Server Configuration and CLI Argument Parsing.
//!
//! This module defines the command-line interface (CLI) and configuration structures
//! for the validation server. It utilizes `clap` to parse arguments and environment variables
//! into strongly-typed configuration objects used throughout the application.

use anyhow::{anyhow, Context, Result};
use clap::{Parser, ValueEnum};
use std::collections::HashMap;
use std::env;
use std::net::SocketAddr;
use std::path::{Path, PathBuf};
use tracing::info;

use crate::engine::machine::JitProcessManager;
use crate::engine::machine_locator::MachineLocator;
use crate::engine::{replay_binary, ModuleRoot, DEFAULT_JIT_CRANELIFT};

/// Mode-specific execution state, built at startup.
pub enum ExecutionMode {
    Native {
        module_cache: HashMap<ModuleRoot, CompiledModule>,
    },
    Continuous {
        // Not wrapped in Arc<> since the caller of ServerState is already wrapped.
        jit_manager: JitProcessManager,
    },
}

pub struct ServerState {
    /// Machine locator is responsible for locating replay.wasm binary and building
    /// a map of module roots to their respective location + binary
    pub locator: MachineLocator,
    pub available_workers: usize,
    pub execution: ExecutionMode,
}

impl ServerState {
    pub fn new(config: &ServerConfig, available_workers: usize) -> Result<Self> {
        let locator = MachineLocator::new(&config.root_path)?;

        let execution = match config.mode {
            InputMode::Continuous => ExecutionMode::Continuous {
                jit_manager: JitProcessManager::new(&locator)?,
            },
            InputMode::Native => {
                let mut module_cache = HashMap::new();
                for meta in locator.module_roots() {
                    let binary = replay_binary(&meta.path);
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
                ExecutionMode::Native { module_cache }
            }
        };

        Ok(ServerState {
            locator,
            available_workers,
            execution,
        })
    }

    /// Gracefully shuts down mode-specific resources.
    pub async fn shutdown(&self) -> Result<()> {
        match &self.execution {
            ExecutionMode::Continuous { jit_manager } => jit_manager.complete_machines().await,
            ExecutionMode::Native { .. } => Ok(()),
        }
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
use jit::CompiledModule;
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

pub fn get_jit_path(config_path: &Option<String>) -> Result<PathBuf> {
    // 1. If a custom path is provided, use it directly
    if let Some(jit_path) = config_path {
        let path = Path::new(&jit_path);
        if path.exists() {
            return Ok(path.to_path_buf());
        }
        return Err(anyhow!(
            "Custom JIT path provided but not found: {jit_path}",
        ));
    }

    // 2. Fall back to auto-detection
    let current_exe = env::current_exe().context("failed to get path of current executable")?;

    let exe_string = current_exe.to_string_lossy();
    let is_test_env = exe_string.contains("test")
        || exe_string.contains("deps")
        || exe_string.contains("system_tests");

    let candidate = if is_test_env {
        // CARGO_MANIFEST_DIR points to crates/validator, therefore we need to look for the grandparent
        let manifest_dir = Path::new(env!("CARGO_MANIFEST_DIR"));
        if let Some(grandparent) = manifest_dir.parent().and_then(|p| p.parent()) {
            grandparent.join("target").join("bin").join("jit")
        } else {
            return Err(anyhow!(
                "Custom JIT path not found for test env: {manifest_dir:?}",
            ));
        }
    } else {
        current_exe
            .parent()
            .ok_or_else(|| anyhow!("failed to resolve parent directory of executable"))?
            .join("jit")
    };

    if candidate.exists() {
        return Ok(candidate);
    }

    // 3. Fallback: Search system PATH
    // We treat a missing PATH var as "just continue" rather than a hard error
    if let Ok(path_var) = env::var("PATH") {
        for split_path in env::split_paths(&path_var) {
            let joined = split_path.join("jit");
            if joined.exists() {
                return Ok(joined);
            }
        }
    }

    Err(anyhow!(
        "jit binary not found in local paths or system PATH"
    ))
}

#[cfg(test)]
mod tests {
    use clap::Parser;

    use crate::config::{get_jit_path, ServerConfig};

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

    #[test]
    fn test_get_jit_path() {
        let jit_path = get_jit_path(&None).unwrap();

        assert!(jit_path.exists(), "JIT binary does not exist");
        assert!(
            jit_path.is_file(),
            "JIT path points to a directory, expected a file"
        );

        let path_str = jit_path.to_str().expect("path contains invalid utf-8");

        assert!(
            path_str.contains("nitro/target/bin/jit"),
            "Path {:?} did not contain expected substring 'nitro/target/bin/jit'",
            jit_path
        );
    }

    // TODO: Add tests for get_jit_path(Some("path"))
}
