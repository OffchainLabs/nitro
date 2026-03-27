#![allow(missing_docs)]
#![allow(clippy::missing_safety_doc)]

#[cfg(target_arch = "x86_64")]
#[no_mangle]
pub unsafe extern "C" fn __rust_probestack() {}

use arb_node::{launcher::ArbEngineLauncher, ArbNode};
use clap::Parser;
use reth::cli::Cli;
use reth_engine_tree::tree::TreeConfig;
use reth_ethereum_cli::chainspec::EthereumChainSpecParser;
use tracing::info;

fn main() {
    reth_cli_util::sigsegv_handler::install();

    if std::env::var_os("RUST_BACKTRACE").is_none() {
        unsafe { std::env::set_var("RUST_BACKTRACE", "1") };
    }

    if let Err(err) = Cli::<EthereumChainSpecParser>::parse().run(async move |builder, _| {
        info!(target: "reth::cli", "Launching arb-reth node");
        let node = builder.node(ArbNode::default());
        let engine_tree_config = TreeConfig::default();
        let launcher = ArbEngineLauncher::new(
            node.task_executor().clone(),
            node.config().datadir(),
            engine_tree_config,
        );
        let handle = node.launch_with(launcher).await?;
        handle.wait_for_node_exit().await
    }) {
        eprintln!("Error: {err:?}");
        std::process::exit(1);
    }
}
