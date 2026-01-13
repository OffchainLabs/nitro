// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::fs::read_to_string;
use anyhow::anyhow;
use arbutil::Bytes32;
use clap::{Args, Parser, ValueEnum};
use serde::{Deserialize, Serialize};
use std::net::SocketAddr;
use std::path::PathBuf;

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
}

#[derive(Clone, Debug, Args)]
#[group(required = true, multiple = false)]
struct ModuleRootConfig {
    /// Supported module root.
    #[clap(long)]
    #[arg(group = "module-root")]
    module_root: Option<Bytes32>,

    /// Path to the file containing the module root.
    #[clap(long)]
    #[arg(group = "module-root")]
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
                let root = content.trim().parse::<Bytes32>().map_err(|e| anyhow!(e))?;
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
