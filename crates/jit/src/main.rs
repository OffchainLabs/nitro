// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use arbutil::Color;
use clap::Parser;
use eyre::Result;
use jit::{machine::Escape, run, Opts};
use validation::transfer::{send_failure_response, send_successful_response};
use wasmer::FrameInfo;

fn main() -> Result<()> {
    let opts = Opts::parse();
    let result = run(&opts)?;

    let runtime = format!("{}ms", result.runtime.as_millis());

    if let Some(error) = result.error {
        print_trace(&result.trace);
        let message = match error {
            Escape::Exit(x) => format!("Failed in {runtime} with exit code {x}."),
            Escape::Failure(err) => format!("Jit failed with {err} in {runtime}."),
            Escape::HostIO(err) => format!("Hostio failed with {err} in {runtime}."),
            Escape::Child(err) => format!("Child failed with {err} in {runtime}."),
            Escape::SocketError(err) => format!("Socket failed with {err} in {runtime}."),
            Escape::UnexpectedReturn(values) => {
                format!("Jit unexpectedly returned values {values:?} in {runtime}.")
            }
        };
        if opts.validator.debug {
            println!("{message}")
        }
        if let Some(mut socket) = result.socket {
            send_failure_response(&mut socket, &message)?;
        }
        if opts.validator.require_success {
            std::process::exit(1);
        }
    } else {
        if opts.validator.debug {
            println!(
                "Completed in {runtime} with hash {}.",
                result.new_state.last_block_hash
            )
        }
        if let Some(mut socket) = result.socket {
            send_successful_response(
                &mut socket,
                &result.new_state.into(),
                result.memory_used.bytes().0 as u64,
            )?;
        }
    }
    Ok(())
}

fn print_trace(trace: &[FrameInfo]) {
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
