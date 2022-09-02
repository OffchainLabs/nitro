// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::gostack::{Escape, WasmEnvArc};

use structopt::StructOpt;
use wasmer::{RuntimeError, Value};

use std::{path::PathBuf, time::Instant};

mod arbcompress;
mod gostack;
mod machine;
mod runtime;
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
}

fn main() {
    let opts = Opts::from_args();

    let env = match WasmEnvArc::cli(&opts) {
        Ok(env) => env,
        Err(err) => panic!("{}", err),
    };
    let (instance, env) = machine::create(&opts, env);

    let now = Instant::now();
    let main = instance.exports.get_function("run").unwrap();
    let resume = instance.exports.get_function("resume").unwrap();

    let mut escape;

    fn check_outcome(outcome: Result<Box<[Value]>, RuntimeError>) -> Option<Escape> {
        let outcome = match outcome {
            Ok(outcome) => {
                println!("Go returned values {:?}", outcome);
                return None;
            }
            Err(outcome) => outcome,
        };
        Some(match outcome.downcast() {
            Ok(escape) => escape,
            Err(outcome) => Escape::Failure(format!("unknown runtime error: {}", outcome)),
        })
    }

    let outcome = main.call(&[Value::I32(0), Value::I32(0)]);
    escape = check_outcome(outcome);

    if escape.is_none() {
        while let Some(event) = env.lock().js_future_events.pop_front() {
            if let Some(issue) = &env.lock().js_pending_event {
                println!("Go runtime overwriting pending event {:?}", issue);
            }
            env.lock().js_pending_event = Some(event);
            escape = check_outcome(resume.call(&[]));
            if escape.is_some() {
                break;
            }
        }
    }

    let block_hash = hex::encode(env.lock().large_globals[0]);
    let elapsed = now.elapsed().as_millis();
    match escape {
        Some(Escape::Exit(0)) => println!("Completed in {elapsed}ms with block hash {block_hash}"),
        Some(Escape::Exit(x)) => println!("Failed in {elapsed}ms with exit code {x}"),
        Some(Escape::Failure(err)) => println!("Jit failed with {err} in {elapsed}ms"),
        Some(Escape::HostIO(err)) => println!("Hostio failed with {err} in {elapsed}ms"),
        _ => println!("Execution ended prematurely"),
    }
}
