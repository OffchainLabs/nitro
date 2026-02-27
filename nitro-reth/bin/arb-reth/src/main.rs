use arb_node::{args::RollupArgs, ArbNode};
use clap::Parser;
use reth_ethereum_cli::{chainspec::EthereumChainSpecParser, Cli};
use tracing::info;

fn main() {
    reth_cli_util::sigsegv_handler::install();

    if std::env::var_os("RUST_BACKTRACE").is_none() {
        unsafe { std::env::set_var("RUST_BACKTRACE", "1") };
    }

    if let Err(err) =
        Cli::<EthereumChainSpecParser, RollupArgs>::parse().run(async move |builder, rollup_args| {
            info!(target: "reth::cli", "Launching arb-reth node");
            let handle = builder.node(ArbNode::new(rollup_args)).launch().await?;
            handle.wait_for_node_exit().await
        })
    {
        eprintln!("Error: {err:?}");
        std::process::exit(1);
    }
}
