// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use anyhow::Result;
use arbutil::Bytes32;
use clap::Parser;
use config::ServerConfig;
use logging::init_logging;
use router::create_router;
use std::sync::Arc;
use tokio::{net::TcpListener, runtime::Builder};
use tracing::info;

mod config;
mod logging;
mod router;
mod spawner_endpoints;

#[derive(Clone, Debug)]
pub struct ServerState {
    module_root: Bytes32,
    available_workers: usize,
}

fn main() -> Result<()> {
    let server_config = ServerConfig::parse();
    init_logging(server_config.logging_format)?;

    let available_workers = server_config.get_workers()?;
    let state = Arc::new(ServerState {
        module_root: server_config.get_module_root()?,
        available_workers,
    });

    let runtime = Builder::new_multi_thread()
        .worker_threads(available_workers)
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
    axum::serve(listener, create_router().with_state(state))
        .await
        .map_err(Into::into)
}
