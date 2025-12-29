// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use clap::{Parser, ValueEnum};
use serde::{Deserialize, Serialize};

#[derive(Clone, Debug, Parser)]
pub struct ServerConfig {
    /// Host where the server should be run.
    #[clap(long, default_value = "0.0.0.0")]
    pub host: String,

    /// Port where the server should be run.
    #[clap(long, default_value_t = 4141)]
    pub port: u16,

    /// Logging format configuration.
    #[clap(long, value_enum, default_value_t = LoggingFormat::Text)]
    pub logging_format: LoggingFormat,
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
}
