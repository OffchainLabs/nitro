// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use anyhow::Result;
use clap::Parser;
use logging::init_logging;
use std::sync::Arc;
use tokio::net::TcpListener;
use tracing::info;

use crate::config::ServerState;
use crate::server::run_server;

mod config;
mod engine;
mod logging;
mod router;
mod server;
mod spawner_endpoints;

#[tokio::main]
async fn main() -> Result<()> {
    let config = config::ServerConfig::parse();
    init_logging(config.logging_format)?;
    info!("Starting validator server with config: {:#?}", config);

    let state = Arc::new(ServerState::new(&config)?);
    let listener = TcpListener::bind(config.address).await?;

    run_server(listener, state).await
}
