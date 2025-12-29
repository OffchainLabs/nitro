use crate::config::LoggingFormat;
use anyhow::anyhow;
use std::{env, io};
use tracing_subscriber::EnvFilter;

/// Initialize `tracing` logging based on the specified format. By default, the logging level is set
/// to "info" unless overridden by the `RUST_LOG` environment variable.
pub fn init(format: LoggingFormat) -> anyhow::Result<()> {
    const LOG_CONFIGURATION_ENVVAR: &str = "RUST_LOG";

    let filter = EnvFilter::new(
        env::var(LOG_CONFIGURATION_ENVVAR)
            .as_deref()
            .unwrap_or("info"),
    );

    let subscriber = tracing_subscriber::fmt()
        .with_writer(io::stdout)
        .with_env_filter(filter);

    match format {
        LoggingFormat::Json => subscriber.json().try_init(),
        LoggingFormat::Text => subscriber.try_init(),
    }
    .map_err(|err| anyhow!(err))
}
