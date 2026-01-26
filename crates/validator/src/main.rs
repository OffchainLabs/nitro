// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use anyhow::Result;
use clap::Parser;
use config::ServerConfig;
use logging::init_logging;
use std::sync::Arc;
use tokio::{net::TcpListener, runtime::Builder};
use tracing::info;

use crate::config::ServerState;
use crate::server::run_server;

mod config;
mod engine;
mod logging;
mod router;
mod server;
mod spawner_endpoints;

fn main() -> Result<()> {
    let server_config = ServerConfig::parse();
    init_logging(server_config.logging_format)?;

    let state = Arc::new(ServerState::new(&server_config)?);

    let runtime = Builder::new_multi_thread()
        .worker_threads(state.available_workers)
        .enable_all()
        .build()?;

    runtime.block_on(async_main(server_config, state))?;

    Ok(())
}

async fn async_main(server_config: ServerConfig, state: Arc<ServerState>) -> Result<()> {
    info!(
        "Starting validator server with config: {:#?}",
        server_config
    );

    let listener = TcpListener::bind(server_config.address).await?;
    run_server(listener, state).await
}