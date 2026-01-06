// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use arbutil::Bytes32;
use clap::{Args, Parser, Subcommand};
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

#[derive(Clone, Debug, Parser)]
pub struct Opts {
    #[command(flatten)]
    pub validator: ValidatorOpts,
    #[command(subcommand)]
    pub input_mode: InputMode,
}

#[derive(Clone, Debug, Args)]
pub struct ValidatorOpts {
    #[clap(short, long)]
    pub binary: PathBuf,
    #[clap(long)]
    pub cranelift: bool,
    #[clap(long)]
    pub debug: bool,
    #[clap(long)]
    pub require_success: bool,
}

#[derive(Clone, Debug, Subcommand)]
pub enum InputMode {
    Json {
        #[clap(long)]
        inputs: PathBuf,
    },
    Local(LocalInput),
    Continuous,
}

#[derive(Clone, Debug, Args)]
pub struct LocalInput {
    #[clap(long, default_value = "0")]
    inbox_position: u64,
    #[clap(long, default_value = "0")]
    delayed_inbox_position: u64,
    #[clap(long, default_value = "0")]
    position_within_message: u64,
    #[clap(long, value_parser = cli_parsing::parse_hex)]
    last_block_hash: Bytes32,
    #[clap(long, value_parser = cli_parsing::parse_hex)]
    last_send_root: Bytes32,
    #[clap(long)]
    inbox: Vec<PathBuf>,
    #[clap(long)]
    delayed_inbox: Vec<PathBuf>,
    #[clap(long)]
    preimages: Option<PathBuf>,
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
