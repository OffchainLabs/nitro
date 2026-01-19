// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use anyhow::Result;
use arbutil::Bytes32;
use clap::Parser;
use logging::init_logging;
use router::create_router;
use std::sync::Arc;
use tokio::net::TcpListener;
use tracing::info;

mod config;
mod logging;
mod router;
mod spawner_endpoints;

#[derive(Clone, Debug)]
pub struct ServerState {
    module_root: Bytes32,
}

#[tokio::main]
async fn main() -> Result<()> {
    let config = config::ServerConfig::parse();
    init_logging(config.logging_format)?;
    info!("Starting validator server with config: {:#?}", config);

    let state = Arc::new(ServerState {
        module_root: config.get_module_root()?,
    });

    let listener = TcpListener::bind(config.address).await?;
    axum::serve(listener, create_router().with_state(state))
        .await
        .map_err(Into::into)
}
