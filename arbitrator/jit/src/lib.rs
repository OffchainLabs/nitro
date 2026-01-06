// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use clap::Parser;
use std::path::PathBuf;

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

#[derive(Parser)]
pub struct Opts {
    #[clap(short, long)]
    binary: PathBuf,
    #[clap(long, default_value = "0")]
    inbox_position: u64,
    #[clap(long, default_value = "0")]
    delayed_inbox_position: u64,
    #[clap(long, default_value = "0")]
    position_within_message: u64,
    #[clap(long)]
    last_block_hash: Option<String>,
    #[clap(long)]
    last_send_root: Option<String>,
    #[clap(long)]
    inbox: Vec<PathBuf>,
    #[clap(long)]
    delayed_inbox: Vec<PathBuf>,
    #[clap(long)]
    preimages: Option<PathBuf>,
    #[clap(long)]
    cranelift: bool,
    #[clap(long)]
    forks: bool,
    #[clap(long)]
    pub debug: bool,
    #[clap(long)]
    pub require_success: bool,
    // JSON inputs supercede any of the command-line inputs which could
    // be specified in the JSON file.
    #[clap(long)]
    json_inputs: Option<PathBuf>,
}
