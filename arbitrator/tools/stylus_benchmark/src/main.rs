// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use clap::{Parser, Subcommand};
use std::path::PathBuf;

mod benchmark;
mod generate_wats;

#[derive(Debug, Parser)]
#[command(name = "stylus_benchmark")]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Debug, Subcommand)]
enum Commands {
    #[command(arg_required_else_help = true)]
    Benchmark { wat_path: PathBuf },
    GenerateWats {
        #[arg(value_name = "OUT_PATH")]
        out_path: PathBuf,
    },
}

fn main() -> eyre::Result<()> {
    let args = Cli::parse();
    match args.command {
        Commands::Benchmark { wat_path } => {
            return benchmark::benchmark(&wat_path);
        }
        Commands::GenerateWats { out_path } => {
            return generate_wats::generate_wats(out_path);
        }
    }
}
