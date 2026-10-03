// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::sync::Arc;

use anyhow::Result;
use clap::Parser;
use config::ServerConfig;
use logging::init_logging;
use tokio::{net::TcpListener, runtime::Builder};
use tracing::info;

use crate::{config::ServerState, server::run_server};

mod config;
mod engine;
mod jwt;
mod logging;
mod server;
mod spawner_endpoints;
fn main() -> Result<()> {
    let server_config = ServerConfig::parse();
    init_logging(server_config.logging_format)?;

    let available_workers = server_config.get_workers()?;

    let runtime = Builder::new_multi_thread()
        .worker_threads(available_workers)
        .enable_all()
        .build()?;

    runtime.block_on(async_main(server_config, available_workers))?;

    Ok(())
}

async fn async_main(server_config: ServerConfig, available_workers: usize) -> Result<()> {
    info!(
        "Starting validator server with config: {:#?}",
        server_config
    );

    let state = Arc::new(ServerState::new(&server_config, available_workers)?);
    let listener = TcpListener::bind(server_config.address).await?;
    let local_addr = listener.local_addr()?;
    info!("Listening on {local_addr}");
    run_server(listener, state).await
}
