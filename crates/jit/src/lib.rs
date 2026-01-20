// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::machine::Escape;
use arbutil::{Bytes32, PreimageType};
use clap::{Args, Parser, Subcommand};
use eyre::Report;
use std::collections::HashMap;
use std::io::{BufWriter, Write};
use std::net::TcpStream;
use std::path::PathBuf;
use std::time::Duration;
use wasmer::{ExportError, FrameInfo, Pages, RuntimeError};

mod arbcompress;
mod caller_env;
pub mod machine;
mod prepare;
pub mod program;
mod socket;
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

#[derive(Clone, Debug)]
pub struct SequencerMessage {
    pub number: u64,
    pub data: Vec<u8>,
}

#[derive(Clone, Debug)]
pub struct NativeInput {
    pub old_state: GlobalState,
    pub inbox: Vec<SequencerMessage>,
    pub delayed_inbox: Vec<SequencerMessage>,
    pub preimages: HashMap<PreimageType, HashMap<Bytes32, Vec<u8>>>,
    pub programs: HashMap<Bytes32, Vec<u8>>,
}

pub struct RunResult {
    pub memory_used: Pages,
    pub runtime: Duration,

    pub new_state: GlobalState,

    pub error: Option<Escape>,
    pub trace: Vec<FrameInfo>,
    pub socket: Option<BufWriter<TcpStream>>,
}

pub fn run(opts: &Opts) -> eyre::Result<RunResult> {
    let (instance, env, mut store) = machine::create(&opts)?;
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
            result.error = Some(Escape::from(err));
        }
    }

    Ok(result)
}

macro_rules! check {
    ($expr:expr) => {{
        if let Err(comms_error) = $expr {
            eprintln!("Failed to send results to Go: {comms_error}");
            panic!("Communication failure");
        }
    }};
}

pub fn report_success(
    writer: &mut BufWriter<TcpStream>,
    new_state: &GlobalState,
    memory_used: &Pages,
) {
    check!(socket::write_u8(writer, socket::SUCCESS));
    check!(socket::write_u64(writer, new_state.inbox_position));
    check!(socket::write_u64(writer, new_state.position_within_message));
    check!(socket::write_bytes32(writer, &new_state.last_block_hash));
    check!(socket::write_bytes32(writer, &new_state.last_send_root));
    check!(socket::write_u64(writer, memory_used.bytes().0 as u64));
    check!(writer.flush());
}

pub fn report_error(writer: &mut BufWriter<TcpStream>, error: String) {
    check!(socket::write_u8(writer, socket::FAILURE));
    check!(socket::write_bytes(writer, &error.into_bytes()));
    check!(writer.flush());
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
