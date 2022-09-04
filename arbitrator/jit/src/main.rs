// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::gostack::{Escape, WasmEnvArc};

use structopt::StructOpt;
use wasmer::{RuntimeError, Value};

use std::{path::PathBuf, time::Instant};

mod arbcompress;
mod color;
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
        while let Some(event) = env.lock().js_state.future_events.pop_front() {
            if let Some(issue) = &env.lock().js_state.pending_event {
                println!("Go runtime overwriting pending event {:?}", issue);
            }
            env.lock().js_state.pending_event = Some(event);
            escape = check_outcome(resume.call(&[]));
            if escape.is_some() {
                break;
            }
        }
    }

    let user = env.lock().process.socket.is_none();
    let time = format!("{}ms", now.elapsed().as_millis());
    let time = color::when(user, time, color::PINK);
    let hash = color::when(user, hex::encode(env.lock().large_globals[0]), color::PINK);
    let (success, message) = match escape {
        Some(Escape::Exit(0)) => (true, format!("Completed in {time} with hash {hash}.")),
        Some(Escape::Exit(x)) => (false, format!("Failed in {time} with exit code {x}.")),
        Some(Escape::Failure(err)) => (false, format!("Jit failed with {err} in {time}.")),
        Some(Escape::HostIO(err)) => (false, format!("Hostio failed with {err} in {time}.")),
        Some(Escape::SocketError(err)) => (false, format!("Socket failed with {err} in {time}.")),
        None => (false, format!("Machine exited prematurely")),
    };

    println!("{message}");

    let error = match success {
        true => None,
        false => Some(message),
    };

    env.send_results(error);
}
