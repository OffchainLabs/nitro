// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use std::path::PathBuf;
use structopt::StructOpt;

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
    pub debug: bool,
    #[structopt(long)]
    pub require_success: bool,
    // JSON inputs supercede any of the command-line inputs which could
    // be specified in the JSON file.
    #[structopt(long)]
    json_inputs: Option<PathBuf>,
}
