// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use anyhow::Result;
use clap::Parser;
use logging::init_logging;
use router::create_router;
use tokio::net::TcpListener;
use tracing::info;

mod config;
mod logging;
mod router;
mod spawner_endpoints;

#[tokio::main]
async fn main() -> Result<()> {
    let config = config::ServerConfig::parse();
    init_logging(config.logging_format)?;
    info!("Starting validator server with config: {:#?}", config);

    let listener = TcpListener::bind(config.address).await?;
    axum::serve(listener, create_router())
        .await
        .map_err(Into::into)
}
