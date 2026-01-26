// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use anyhow::Result;
use clap::Parser;
use logging::init_logging;
use router::create_router;
use std::sync::Arc;
use tokio::net::TcpListener;
use tracing::info;

use crate::{config::InputMode, server_jit::machine_locator::MachineLocator};

mod config;
mod endpoints;
mod logging;
mod router;
mod server_jit;

#[derive(Clone, Debug)]
pub struct ServerState {
    mode: InputMode,
    locator: MachineLocator,
}

#[tokio::main]
async fn main() -> Result<()> {
    let config = config::ServerConfig::parse();
    init_logging(config.logging_format)?;
    info!("Starting validator server with config: {:#?}", config);

    let locator = MachineLocator::new(&config.module_root_config)?;

    let state = Arc::new(ServerState {
        mode: config.mode,
        locator,
    });

    let listener = TcpListener::bind(config.address).await?;
    axum::serve(listener, create_router().with_state(state))
        .await
        .map_err(Into::into)
}
