// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use clap::Parser;
use std::path::PathBuf;

mod benchmark;
mod scenario;

#[derive(Parser, Debug)]
#[command(version, about, long_about = None)]
struct Args {
    #[arg(short, long)]
    output_wat_dir_path: Option<PathBuf>,

    #[arg(short, long)]
    scenario: Option<scenario::Scenario>,
}

fn main() -> eyre::Result<()> {
    let args = Args::parse();

    match args.scenario {
        Some(scenario) => {
            println!("Benchmarking {:?}", scenario);
            let wat = scenario::generate_wat(scenario, args.output_wat_dir_path);
            benchmark::benchmark(wat)
        }
        None => {
            println!("No scenario specified, benchmarking all scenarios");
            Ok(())
        }
    }
}
