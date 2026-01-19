// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use anyhow::Result;
use arbutil::Bytes32;
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
    let config = config::ServerConfig::load()?;
    init_logging(config.logging_format)?;

    let workers = config.get_workers();

    let runtime = Builder::new_multi_thread()
        .worker_threads(workers)
        .enable_all()
        .build()
        .unwrap();

    runtime.block_on(async_main(config))?;

    Ok(())
}

async fn async_main(config: config::ServerConfig) -> Result<()> {
    info!("Starting validator server with config: {:#?}", config);

    let state = Arc::new(ServerState {
        module_root: config.get_module_root()?,
        available_workers: config.get_workers(),
    });

    let listener = TcpListener::bind(config.address).await?;
    axum::serve(listener, create_router().with_state(state))
        .await
        .map_err(Into::into)
}
