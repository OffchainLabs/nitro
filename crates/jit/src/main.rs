// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use arbutil::{color, Color};
use clap::Parser;
use eyre::{eyre, Result};
use jit::machine::{Escape, WasmEnv};
use jit::{machine, run, GlobalState, RunError};
use jit::{report_error, report_success, Opts};
use sha2::digest::typenum::op;
use wasmer::RuntimeError;

fn main() -> Result<()> {
    let opts = Opts::parse();
    let result = run(&opts)?;

    let runtime = format!("{}ms", result.stats.runtime.as_millis());

    if let Some(state) = result.new_state {
        if opts.validator.debug {
            println!(
                "Completed in {runtime} with hash {}.",
                state.last_block_hash
            )
        }
        if let Some(mut socket) = result.socket {
            report_success(&mut socket, &state, &result.stats.memory_used);
        }
    } else if let Some(error) = result.error {
        print_trace(&error);
        let message = match Escape::from(error) {
            Escape::Exit(x) => format!("Failed in {runtime} with exit code {x}."),
            Escape::Failure(err) => format!("Jit failed with {err} in {runtime}."),
            Escape::HostIO(err) => format!("Hostio failed with {err} in {runtime}."),
            Escape::Child(err) => format!("Child failed with {err} in {runtime}."),
            Escape::SocketError(err) => format!("Socket failed with {err} in {runtime}."),
        };
        if opts.validator.debug {
            println!("{message}")
        }
        if let Some(mut socket) = result.socket {
            report_error(&mut socket, message);
        }
        if opts.validator.require_success {
            std::process::exit(1);
        }
    }
    Ok(())
}

fn print_trace(error: &RuntimeError) {
    let trace = error.trace();
    if !trace.is_empty() {
        println!("backtrace:");
    }
    for frame in trace {
        let module = frame.module_name();
        let name = frame.function_name().unwrap_or("??");
        println!("  in {} of {}", name.red(), module.red());
    }
}

// require an usize be at least 32 bits wide
#[cfg(not(any(target_pointer_width = "32", target_pointer_width = "64")))]
compile_error!(
    "Unsupported target pointer width (only 32 bit and 64 bit architectures are supported)"
);
