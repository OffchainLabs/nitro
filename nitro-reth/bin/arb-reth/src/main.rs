#![allow(missing_docs)]

/// Stack-probe shim for x86_64. wasmer's vm crate references the LLVM
/// `__rust_probestack` intrinsic that recent `compiler-builtins` no
/// longer exports; defining an empty function here satisfies the linker.
///
/// # Safety
///
/// Defined for the linker only; never called from Rust.
#[cfg(target_arch = "x86_64")]
#[no_mangle]
pub unsafe extern "C" fn __rust_probestack() {}

use arb_node::{chainspec::ArbChainSpecParser, launcher::ArbEngineLauncher, ArbNode};
use clap::Parser;
use reth::cli::Cli;
use reth_engine_tree::tree::TreeConfig;
use tracing::info;

fn main() {
    reth_cli_util::sigsegv_handler::install();

    if std::env::var_os("RUST_BACKTRACE").is_none() {
        // SAFETY: process startup, no other threads have been spawned, so
        // no concurrent readers of the environment exist (Rust 2024
        // unsafe set_var contract).
        unsafe { std::env::set_var("RUST_BACKTRACE", "1") };
    }

    if let Err(err) = Cli::<ArbChainSpecParser>::parse().run(async move |builder, _| {
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
