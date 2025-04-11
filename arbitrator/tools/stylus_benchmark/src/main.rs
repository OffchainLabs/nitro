// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

mod benchmark;
mod scenario;
mod scenarios;

use clap::{Parser, ValueEnum};
use scenario::Scenario;
use std::path::PathBuf;

#[derive(Parser, Debug)]
#[command(version, about, long_about = None)]
struct Args {
    #[arg(short, long)]
    output_wat_dir_path: Option<PathBuf>,

    #[arg(short, long)]
    scenario: Option<Scenario>,
}

fn handle_scenario(scenario: Scenario, output_wat_dir_path: Option<PathBuf>) -> eyre::Result<()> {
    println!("Benchmarking {:?}", scenario);
    let wat = scenario::generate_wat(scenario, output_wat_dir_path);
    benchmark::benchmark(wat)
}

fn main() -> eyre::Result<()> {
    let args = Args::parse();

    match args.scenario {
        Some(scenario) => handle_scenario(scenario, args.output_wat_dir_path),
        None => {
            println!("No scenario specified, benchmarking all scenarios\n");
            for scenario in Scenario::value_variants() {
                let benchmark_result = handle_scenario(*scenario, args.output_wat_dir_path.clone());
                if let Err(err) = benchmark_result {
                    return Err(err);
                }
            }
            Ok(())
        }
    }
}
