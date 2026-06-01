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

mod commands;

use arb_node::{
    chainspec::ArbChainSpecParser, cli_components, launcher::ArbEngineLauncher, ArbNode,
};
use clap::Parser;
use reth::{cli::Cli, CliRunner};
use reth_engine_tree::tree::TreeConfig;
use reth_tracing::{RethTracer, Tracer};
use tracing::info;

fn main() {
    reth_cli_util::sigsegv_handler::install();

    if std::env::var_os("RUST_BACKTRACE").is_none() {
        // SAFETY: process startup, no other threads have been spawned, so
        // no concurrent readers of the environment exist (Rust 2024
        // unsafe set_var contract).
        unsafe { std::env::set_var("RUST_BACKTRACE", "1") };
    }

    // These offline commands run through Arbitrum-aware paths (see `commands`).
    // Every other subcommand is dispatched by reth.
    if let Some(sub @ ("re-execute" | "repair")) = std::env::args().nth(1).as_deref() {
        if let Err(err) = run_offline(sub) {
            eprintln!("Error: {err:?}");
            std::process::exit(1);
        }
        return;
    }

    if let Err(err) = Cli::<ArbChainSpecParser>::parse().run_with_components::<ArbNode>(
        cli_components,
        async move |builder, _| {
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
        },
    ) {
        eprintln!("Error: {err:?}");
        std::process::exit(1);
    }
}

fn run_offline(sub: &str) -> eyre::Result<()> {
    // Strip the subcommand token before clap parses the flags-only struct.
    let mut args = std::env::args_os();
    let bin = args.next().unwrap_or_default();
    let _ = args.next();
    let argv: Vec<_> = std::iter::once(bin).chain(args).collect();

    let _guard = RethTracer::new().init().ok().flatten();

    let runner = CliRunner::try_default_runtime()?;
    let runtime = runner.runtime();

    match sub {
        "re-execute" => {
            let cmd = commands::re_execute::Command::<ArbChainSpecParser>::parse_from(argv);
            runner.run_until_ctrl_c(cmd.execute::<ArbNode>(cli_components, runtime))
        }
        "repair" => {
            let cmd = commands::repair::Command::<ArbChainSpecParser>::parse_from(argv);
            runner.run_until_ctrl_c(cmd.execute::<ArbNode>(cli_components, runtime))
        }
        _ => unreachable!("dispatched only for known offline subcommands"),
    }
}
