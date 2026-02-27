#![allow(missing_docs)]

use arb_node::ArbNode;
use clap::Parser;
use reth::cli::Cli;
use reth_ethereum_cli::chainspec::EthereumChainSpecParser;
use tracing::info;

fn main() {
    reth_cli_util::sigsegv_handler::install();

    if std::env::var_os("RUST_BACKTRACE").is_none() {
        unsafe { std::env::set_var("RUST_BACKTRACE", "1") };
    }

    let _ = Cli::<EthereumChainSpecParser>::parse().run(async move |builder, _| {
        info!(target: "reth::cli", "Launching arb-reth node");
        let handle = builder
            .node(ArbNode::default())
            .launch()
            .await?;
        handle.wait_for_node_exit().await
    });
}
