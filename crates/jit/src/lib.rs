// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::machine::Escape;
use arbutil::{Bytes32, PreimageType};
use clap::{Args, Parser, Subcommand};
use std::collections::HashMap;
use std::io::BufWriter;
use std::net::TcpStream;
use std::path::PathBuf;
use std::time::Duration;
use validation::{BatchInfo, UserWasm};
use wasmer::{FrameInfo, Pages};

mod arbcompress;
mod arbkeccak;
mod caller_env;
pub mod machine;
mod prepare;
pub mod program;
pub mod socket;
pub mod stylus_backend;
mod test;
mod wasip1_stub;
mod wavmio;

#[derive(Clone, Debug, Parser)]
pub struct Opts {
    /// General validator configuration
    #[command(flatten)]
    pub validator: ValidatorOpts,
    /// How the validation inputs are provided
    #[command(subcommand)]
    pub input_mode: InputMode,
}

#[derive(Clone, Debug, Args)]
pub struct ValidatorOpts {
    /// Path to the `replay.wasm` binary
    #[clap(short, long)]
    pub binary: PathBuf,
    /// Use Cranelift backend
    #[clap(long, default_value_t = true)]
    pub cranelift: bool,
    /// Enable debug output
    #[clap(long)]
    pub debug: bool,
    /// Require that the validation succeeds
    #[clap(long)]
    pub require_success: bool,
}

#[derive(Clone, Debug, Subcommand)]
pub enum InputMode {
    /// Use a local JSON file containing the inputs
    Json {
        /// Path to the JSON input file
        #[clap(long)]
        inputs: PathBuf,
    },
    /// Use flag values and local files for inputs
    Local(LocalInput),
    /// Use direct Rust objects
    #[command(skip)]
    Native(NativeInput),
    /// Continuously read new inputs from TCP connections
    Continuous,
}

#[derive(Clone, Debug, Args)]
pub struct LocalInput {
    #[clap(flatten)]
    pub old_state: GlobalState,
    #[clap(long, default_value = "0")]
    pub delayed_inbox_position: u64,
    #[clap(long)]
    pub inbox: Vec<PathBuf>,
    #[clap(long)]
    pub delayed_inbox: Vec<PathBuf>,
    #[clap(long)]
    pub preimages: Option<PathBuf>,
}

#[derive(Clone, Debug, Args)]
pub struct GlobalState {
    #[clap(long, value_parser = cli_parsing::parse_hex)]
    pub last_block_hash: Bytes32,
    #[clap(long, value_parser = cli_parsing::parse_hex)]
    pub last_send_root: Bytes32,
    #[clap(long, default_value = "0")]
    pub inbox_position: u64,
    #[clap(long, default_value = "0")]
    pub position_within_message: u64,
}

impl From<validation::GoGlobalState> for GlobalState {
    fn from(state: validation::GoGlobalState) -> Self {
        Self {
            last_block_hash: state.block_hash,
            last_send_root: state.send_root,
            inbox_position: state.batch,
            position_within_message: state.pos_in_batch,
        }
    }
}

impl From<GlobalState> for validation::GoGlobalState {
    fn from(state: GlobalState) -> Self {
        Self {
            block_hash: state.last_block_hash,
            send_root: state.last_send_root,
            batch: state.inbox_position,
            pos_in_batch: state.position_within_message,
        }
    }
}

#[derive(Clone, Debug)]
pub struct NativeInput {
    pub old_state: GlobalState,
    pub inbox: Vec<BatchInfo>,
    pub delayed_inbox: Vec<BatchInfo>,
    pub preimages: HashMap<PreimageType, HashMap<Bytes32, Vec<u8>>>,
    pub programs: HashMap<Bytes32, UserWasm>,
}

/// Result of running the JIT validation.
pub struct RunResult {
    /// Amount of memory used by the Wasm instance.
    pub memory_used: Pages,
    /// Total runtime of the Wasm instance (measured from the first wavmio instruction to finish).
    pub runtime: Duration,

    /// New global state after running the Wasm instance. May be invalid, if `self.error` is `Some`.
    pub new_state: GlobalState,

    /// Error encountered during execution, if any.
    pub error: Option<Escape>,
    /// Stack trace of the error, if any.
    pub trace: Vec<FrameInfo>,
    /// Optional socket to report results back to the spawner, if `InputMode` was `Continuous`.
    pub socket: Option<BufWriter<TcpStream>>,
}

pub fn run(opts: &Opts) -> eyre::Result<RunResult> {
    let (instance, env, mut store) = machine::create(opts)?;
    let outcome = instance
        .exports
        .get_function("_start")?
        .call(&mut store, &[]);

    let memory_used = instance.exports.get_memory("memory")?.view(&store).size();
    let env = env.as_mut(&mut store);

    let mut result = RunResult {
        memory_used,
        runtime: env.process.timestamp.elapsed(),
        new_state: GlobalState {
            last_block_hash: env.large_globals[0],
            last_send_root: env.large_globals[1],
            inbox_position: env.small_globals[0],
            position_within_message: env.small_globals[1],
        },
        error: None,
        trace: vec![],
        // It is okay to take the socket ownership here because `env` will be dropped after this function returns.
        socket: env.process.socket.take().map(|(w, _)| w),
    };

    match outcome {
        Ok(value) => {
            // The proper way `_start` returns successfully is by trapping with an exit code `0`, not by returning a value.
            result.error = Some(Escape::UnexpectedReturn(value.to_vec()));
        }
        Err(err) => {
            result.trace = err.trace().to_vec();
            match Escape::from(err) {
                Escape::Exit(0) => {}
                escape => {
                    result.error = Some(escape);
                }
            }
        }
    }

    Ok(result)
}

mod cli_parsing {
    use arbutil::Bytes32;

    pub fn parse_hex(mut arg: &str) -> eyre::Result<Bytes32> {
        if arg.starts_with("0x") {
            arg = &arg[2..];
        }
        let mut bytes32 = [0u8; 32];
        hex::decode_to_slice(arg, &mut bytes32)?;
        Ok(bytes32.into())
    }
}
