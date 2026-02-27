#![allow(missing_docs)]

use tracing::info;

fn main() {
    reth_cli_util::sigsegv_handler::install();

    if std::env::var_os("RUST_BACKTRACE").is_none() {
        unsafe { std::env::set_var("RUST_BACKTRACE", "1") };
    }

    // Node types, engine types, and all component builders (pool, network,
    // payload, executor, consensus) are implemented. Full node launch requires
    // ArbPrimitives to implement database codec traits (Compact, WrapperTypeEncode,
    // WrapperTypeDecode) so that reth's BlockchainProvider can store and retrieve
    // Arbitrum blocks. Once codecs are implemented, uncomment:
    //
    //   use clap::Parser;
    //   use reth::cli::Cli;
    //   use reth_ethereum_cli::chainspec::EthereumChainSpecParser;
    //   use arb_node::ArbNode;
    //
    //   Cli::<EthereumChainSpecParser>::parse().run(async move |builder, _| {
    //       info!(target: "reth::cli", "Launching arb-reth node");
    //       let handle = builder
    //           .node(ArbNode::default())
    //           .launch_with_debug_capabilities()
    //           .await?;
    //       handle.wait_for_node_exit().await
    //   })

    info!(target: "reth::cli", "arb-reth: Node types and component builders configured");
    info!(target: "reth::cli", "arb-reth: Full launch pending database codec impls for ArbPrimitives");
}
