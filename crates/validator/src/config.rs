// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! Server Configuration and CLI Argument Parsing.
//!
//! This module defines the command-line interface (CLI) and configuration structures
//! for the validation server. It utilizes `clap` to parse arguments and environment variables
//! into strongly-typed configuration objects used throughout the application.

use anyhow::Result;
use arbutil::Bytes32;
use clap::{Args, Parser, ValueEnum};
use std::fs::read_to_string;
use std::net::SocketAddr;
use std::path::PathBuf;
use tokio::sync::Mutex;

use crate::engine::config::JitMachineConfig;
use crate::engine::machine::JitMachine;

#[derive(Debug)]
pub struct ServerState {
    pub mode: InputMode,
    pub module_root: Bytes32,
    pub jit_machine: Option<Mutex<JitMachine>>,
}

impl ServerState {
    pub fn new(config: &ServerConfig) -> Result<Self> {
        let module_root = config.get_module_root()?;
        let jit_machine = match config.mode {
            InputMode::Continuous => {
                let config = JitMachineConfig::default();

                let jit_machine = JitMachine::new(&config, Some(module_root))?;

                Some(Mutex::new(jit_machine))
            }
            InputMode::Native => None,
        };
        Ok(ServerState {
            mode: config.mode,
            module_root,
            jit_machine,
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

    #[clap(flatten)]
    module_root_config: ModuleRootConfig,
}

#[derive(Clone, Debug, Args)]
#[group(required = true, multiple = false)]
struct ModuleRootConfig {
    /// Supported module root.
    #[clap(long)]
    module_root: Option<Bytes32>,

    /// Path to the file containing the module root.
    #[clap(long)]
    module_root_path: Option<PathBuf>,
}

impl ServerConfig {
    pub fn get_module_root(&self) -> anyhow::Result<Bytes32> {
        match (
            self.module_root_config.module_root,
            &self.module_root_config.module_root_path,
        ) {
            (Some(root), None) => Ok(root),
            (None, Some(ref path)) => {
                let content = read_to_string(path)?;
                let root = content
                    .trim()
                    .parse::<Bytes32>()
                    .map_err(|e| anyhow::anyhow!(e))?;
                Ok(root)
            }
            _ => Err(anyhow::anyhow!(
                "Either module_root or module_root_path must be specified"
            )),
        }
    }
}

#[cfg(test)]
mod tests {
    use crate::config::ServerConfig;
    use clap::Parser;

    #[test]
    fn verify_cli() {
        use clap::CommandFactory;
        ServerConfig::command().debug_assert()
    }

    #[test]
    fn module_root_parsing() {
        assert!(
            ServerConfig::try_parse_from([
                "server",
                "--module-root",
                "0x0000000000000000000000000000000000000000000000000000000000000000"
            ])
            .is_ok(),
            "Valid module root should parse correctly"
        );

        assert!(
            ServerConfig::try_parse_from([
                "server",
                "--module-root",
                "0000000000000000000000000000000000000000000000000000000000000000"
            ])
            .is_ok(),
            "Valid module root (without 0x prefix) should parse correctly"
        );

        assert!(
            ServerConfig::try_parse_from(["server", "--module-root", "0xinvalidhex"]).is_err(),
            "Invalid module root should fail to parse"
        );

        assert!(
            ServerConfig::try_parse_from([
                "server",
                "--module-root-path",
                "/some/path/to/module/root.txt"
            ])
            .is_ok(),
            "Valid module root path should parse correctly"
        );

        assert!(
            ServerConfig::try_parse_from([
                "server",
                "--module-root",
                "0x0000000000000000000000000000000000000000000000000000000000000000",
                "--module-root-path",
                "/some/path/to/module/root.txt"
            ])
            .is_err(),
            "Specifying both module root and module root path should fail"
        );

        assert!(
            ServerConfig::try_parse_from(["server"]).is_err(),
            "Not specifying either module root or module root path should fail"
        );
    }
}
