use crate::config::LoggingFormat;
use anyhow::{anyhow, Result};
use axum::{
    routing::{get, post},
    Router,
};
use clap::Parser;
use std::{env, io};
use tokio::net::TcpListener;
use tracing::info;
use tracing_subscriber::EnvFilter;

mod config;
mod spawner_endpoints;

const BASE_NAMESPACE: &str = "/validation";

#[tokio::main]
async fn main() -> Result<()> {
    let config = config::ServerConfig::parse();
    init_logging(config.logging_format)?;
    info!("Starting validator server with config: {:#?}", config);

    let app = Router::new()
        .route(
            &format!("{BASE_NAMESPACE}_capacity"),
            get(spawner_endpoints::capacity),
        )
        .route(
            &format!("{BASE_NAMESPACE}_name"),
            get(spawner_endpoints::name),
        )
        .route(
            &format!("{BASE_NAMESPACE}_stylusArchs"),
            get(spawner_endpoints::stylus_archs),
        )
        .route(
            &format!("{BASE_NAMESPACE}_validate"),
            post(spawner_endpoints::validate),
        )
        .route(
            &format!("{BASE_NAMESPACE}_wasmModuleRoots"),
            get(spawner_endpoints::wasm_module_roots),
        );

    let listener = TcpListener::bind(format!("{}:{}", config.host, config.port)).await?;
    axum::serve(listener, app).await.map_err(Into::into)
}

/// Initialize `tracing` logging based on the specified format. By default, the logging level is set
/// to "info" unless overridden by the `RUST_LOG` environment variable.
fn init_logging(format: LoggingFormat) -> Result<()> {
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
