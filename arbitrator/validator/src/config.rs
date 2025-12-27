use clap::{Parser, ValueEnum};
use std::str::FromStr;

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

#[derive(Copy, Clone, Eq, PartialEq, Debug, Default, ValueEnum)]
pub enum LoggingFormat {
    #[default]
    Text,
    Json,
}

impl FromStr for LoggingFormat {
    type Err = ();

    fn from_str(s: &str) -> Result<Self, Self::Err> {
        match s.to_lowercase().as_str() {
            "text" => Ok(Self::Text),
            "json" => Ok(Self::Json),
            _ => Err(()),
        }
    }
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
