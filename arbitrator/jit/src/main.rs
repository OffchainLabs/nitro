// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::machine::{Escape, WasmEnv};

use arbutil::{color, Color};
use structopt::StructOpt;
use wasmer::Value;

use std::path::PathBuf;

mod arbcompress;
mod gostack;
mod machine;
mod runtime;
mod socket;
mod syscall;
mod test;
mod wavmio;

#[derive(StructOpt)]
#[structopt(name = "jit-prover")]
pub struct Opts {
    #[structopt(short, long)]
    binary: PathBuf,
    #[structopt(long, default_value = "0")]
    inbox_position: u64,
    #[structopt(long, default_value = "0")]
    delayed_inbox_position: u64,
    #[structopt(long, default_value = "0")]
    position_within_message: u64,
    #[structopt(long)]
    last_block_hash: Option<String>,
    #[structopt(long)]
    last_send_root: Option<String>,
    #[structopt(long)]
    inbox: Vec<PathBuf>,
    #[structopt(long)]
    delayed_inbox: Vec<PathBuf>,
    #[structopt(long)]
    preimages: Option<PathBuf>,
    #[structopt(long)]
    cranelift: bool,
    #[structopt(long)]
    forks: bool,
    #[structopt(long)]
    debug: bool,
}

fn main() {
    let opts = Opts::from_args();

    let env = match WasmEnv::cli(&opts) {
        Ok(env) => env,
        Err(err) => panic!("{}", err),
    };

    let (instance, env, mut store) = machine::create(&opts, env);

    let memory = instance.exports.get_memory("mem").unwrap();
    let memory = memory.view(&store);

    // To pass in the program name argument, we need to put it in memory.
    // The Go linker guarantees a section of memory starting at byte 4096 is available for this purpose.
    // https://github.com/golang/go/blob/252324e879e32f948d885f787decf8af06f82be9/misc/wasm/wasm_exec.js#L520
    let free_memory_base: i32 = 4096;
    let name = free_memory_base;
    let argv = name + 8;

    memory.write(name as u64, b"js\0").unwrap(); // write "js\0" to the name ptr
    memory.write(argv as u64, &name.to_le_bytes()).unwrap(); // write the name ptr to the argv ptr
    let run_args = &[Value::I32(1), Value::I32(argv)]; // pass argv with our single name arg

    let main = instance.exports.get_function("run").unwrap();
    let outcome = main.call(&mut store, run_args);
    let escape = match outcome {
        Ok(outcome) => {
            println!("Go returned values {:?}", outcome);
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
        Some(Escape::SocketError(err)) => (false, format!("Socket failed with {err} in {time}.")),
        None => (false, "Machine exited prematurely".to_owned()),
    };

    if opts.debug {
        println!("{message}");
    }

    let error = match success {
        true => None,
        false => Some(message),
    };
    let memory_used = memory.size().0 as u64 * 65_536;

    env.send_results(error, memory_used);
}

// require a usize be at least 32 bits wide
#[cfg(not(any(target_pointer_width = "32", target_pointer_width = "64")))]
compile_error!(
    "Unsupported target pointer width (only 32 bit and 64 bit architectures are supported)"
);
