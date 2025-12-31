// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use arbutil::{color, Color};
use eyre::Result;
use jit::machine;
use jit::machine::{Escape, WasmEnv};
use jit::Opts;
use structopt::StructOpt;

fn main() -> Result<()> {
    let opts = Opts::from_args();
    let env = match WasmEnv::cli(&opts) {
        Ok(env) => env,
        Err(err) => panic!("{err}"),
    };

    let (instance, env, mut store) = machine::create(&opts, env);

    let main = instance.exports.get_function("_start").unwrap();
    let outcome = main.call(&mut store, &[]);
    let escape = match outcome {
        Ok(outcome) => {
            println!("Go returned values {outcome:?}");
            None
        }
        Err(outcome) => {
            let trace = outcome.trace();
            if !trace.is_empty() {
                println!("backtrace:");
            }
            for frame in trace {
                let module = frame.module_name();
                let name = frame.function_name().unwrap_or("??");
                println!("  in {} of {}", name.red(), module.red());
            }
            Some(Escape::from(outcome))
        }
    };

    let memory_used = instance
        .exports
        .get_memory("memory")
        .unwrap()
        .view(&store)
        .size();

    let env = env.as_mut(&mut store);
    let user = env.process.socket.is_none();
    let time = format!("{}ms", env.process.timestamp.elapsed().as_millis());
    let time = color::when(user, time, color::PINK);
    let hash = color::when(user, hex::encode(env.large_globals[0]), color::PINK);
    let (success, message) = match escape {
        Some(Escape::Exit(0)) => (true, format!("Completed in {time} with hash {hash}.")),
        Some(Escape::Exit(x)) => (false, format!("Failed in {time} with exit code {x}.")),
        Some(Escape::Failure(err)) => (false, format!("Jit failed with {err} in {time}.")),
        Some(Escape::HostIO(err)) => (false, format!("Hostio failed with {err} in {time}.")),
        Some(Escape::Child(err)) => (false, format!("Child failed with {err} in {time}.")),
        Some(Escape::SocketError(err)) => (false, format!("Socket failed with {err} in {time}.")),
        None => (false, "Machine exited prematurely".to_owned()),
    };

    if opts.debug || !success {
        println!("{message}");
    }

    let error = match success {
        true => None,
        false => Some(message),
    };

    env.send_results(error, memory_used);

    if !success && opts.require_success {
        std::process::exit(1);
    }
    Ok(())
}

// require a usize be at least 32 bits wide
#[cfg(not(any(target_pointer_width = "32", target_pointer_width = "64")))]
compile_error!(
    "Unsupported target pointer width (only 32 bit and 64 bit architectures are supported)"
);
