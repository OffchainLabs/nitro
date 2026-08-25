//! Standalone MEL runner for extracting messages from parent-chain blocks.

use std::process::ExitCode;

use clap::Parser;

use crate::{config::MelNodeConfig, node::MelNode};

mod config;
mod consumer;
mod database;
mod node;

fn main() -> ExitCode {
    let config = MelNodeConfig::parse();
    match run(&config) {
        Ok(()) => ExitCode::SUCCESS,
        Err(err) => {
            eprintln!("Error: {err:?}");
            ExitCode::FAILURE
        }
    }
}

#[tokio::main]
async fn run(config: &MelNodeConfig) -> eyre::Result<()> {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env().unwrap_or_else(|_| "info".into()),
        )
        .init();

    let node = MelNode::build(config).await?;
    node.run().await
}
