// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use anyhow::Result;
use arbutil::Bytes32;
use clap::{Args, Parser, ValueEnum};
use serde::{Deserialize, Serialize};
use std::fs::read_to_string;
use std::io;
use std::net::SocketAddr;
use std::path::PathBuf;
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

    #[clap(flatten)]
    module_root_config: ModuleRootConfig,

    #[clap(long)]
    workers: Option<usize>,
}

impl ServerConfig {
    pub fn load() -> Result<Self> {
        Self::load_from(std::env::args_os())
    }

    fn load_from<I, T>(args: I) -> Result<Self>
    where
        I: IntoIterator<Item = T>,
        T: Into<std::ffi::OsString> + Clone,
    {
        let mut config = Self::try_parse_from(args)
            .map_err(|e| io::Error::new(io::ErrorKind::InvalidInput, e))?;

        if config.workers.is_none() {
            let workers = match std::thread::available_parallelism() {
                Ok(count) => count.get(),
                Err(e) if e.kind() == std::io::ErrorKind::NotFound => {
                    warn!("Could not determine machine's available parallelism. Defaulting to {DEFAULT_NUM_WORKERS}.");
                    DEFAULT_NUM_WORKERS
                }
                Err(e) => return Err(e.into()),
            };
            config.workers = Some(workers);
        };

        Ok(config)
    }

    pub(crate) fn get_workers(&self) -> usize {
        self.workers
            .expect("Workers should have been set during ServerConfig::load()")
    }
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

#[derive(Copy, Clone, Eq, PartialEq, Debug, Default, ValueEnum, Deserialize, Serialize)]
#[serde(rename_all = "lowercase")]
pub enum LoggingFormat {
    #[default]
    Text,
    Json,
}

#[cfg(test)]
mod tests {
    use crate::config::ServerConfig;

    #[test]
    fn verify_cli() {
        use clap::CommandFactory;
        ServerConfig::command().debug_assert()
    }

    #[test]
    fn module_root_parsing() {
        assert!(
            ServerConfig::load_from([
                "server",
                "--module-root",
                "0x0000000000000000000000000000000000000000000000000000000000000000"
            ])
            .is_ok(),
            "Valid module root should parse correctly"
        );

        assert!(
            ServerConfig::load_from([
                "server",
                "--module-root",
                "0000000000000000000000000000000000000000000000000000000000000000"
            ])
            .is_ok(),
            "Valid module root (without 0x prefix) should parse correctly"
        );

        assert!(
            ServerConfig::load_from(["server", "--module-root", "0xinvalidhex"]).is_err(),
            "Invalid module root should fail to parse"
        );

        assert!(
            ServerConfig::load_from([
                "server",
                "--module-root-path",
                "/some/path/to/module/root.txt"
            ])
            .is_ok(),
            "Valid module root path should parse correctly"
        );

        assert!(
            ServerConfig::load_from([
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
            ServerConfig::load_from(["server"]).is_err(),
            "Not specifying either module root or module root path should fail"
        );
    }

    #[test]
    fn capacity_parsing() {
        assert!(
            ServerConfig::load_from([
                "server",
                "--module-root",
                "0x0000000000000000000000000000000000000000000000000000000000000000",
                "--workers",
                "5"
            ])
            .is_ok(),
            "Valid num of workers should parse correctly"
        );

        assert!(
            ServerConfig::load_from([
                "server",
                "--module-root",
                "0x0000000000000000000000000000000000000000000000000000000000000000",
                "--workers",
                "-5"
            ])
            .is_err(),
            "negative num of workers should fail"
        );

        assert!(
            ServerConfig::load_from([
                "server",
                "--module-root",
                "0x0000000000000000000000000000000000000000000000000000000000000000",
                "--workers",
                "abc"
            ])
            .is_err(),
            "non numeric value for workers should fail"
        );

        let server_config = ServerConfig::load_from([
            "server",
            "--module-root",
            "0x0000000000000000000000000000000000000000000000000000000000000000",
        ])
        .unwrap();

        assert!(server_config.workers.is_some());
        assert!(server_config.workers.unwrap() > 0);
    }
}
