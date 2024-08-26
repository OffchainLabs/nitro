// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//use crate::trial::Trial;
use eyre::Result;
/*use humantime::Duration;
//use model::Model;
use prover::programs::prelude::*;
use std::{
    fs::OpenOptions,
    io::Write,
    path::{Path, PathBuf},
    time::Instant,
};*/
use structopt::StructOpt;
use wasmer::Target;
//use trial::Feed;

mod fees;
mod activate;
mod benchmark;
mod host;
mod model;
mod runtime;
mod trial;
mod util;
mod wasm;

#[derive(StructOpt)]
#[structopt(name = "pricer")]
struct Opts {
    #[structopt(subcommand)]
    pub cmd: Command,
}

#[derive(StructOpt)]
enum Command {
    /*#[structopt(name = "record")]
    Record {
        #[structopt(short, long)]
        path: PathBuf,
        #[structopt(short, long)]
        limit: Duration,
        #[structopt(short, long)]
        filter: Duration,
},*/
    #[structopt(name = "activate")]
    Activate,

    #[structopt(name = "benchmark")]
    Benchmark {
        #[structopt(long, default_value = "")]
        target: String,
    },

    #[structopt(name = "fees")]
    Fees,


    
    /*#[structopt(name = "model")]
    Model {
        #[structopt(short, long)]
        path: PathBuf,
        #[structopt(short, long)]
        output: PathBuf,
    },*/
}

fn main() -> Result<()> {
    let opts = Opts::from_args();

    match opts.cmd {
        Command::Activate => activate::activate(Target::default()),
        Command::Benchmark{target} => benchmark::benchmark(target),
        Command::Fees => fees::fees(),
    }
}

/*fn record(path: &Path, limit: Duration, filter: Duration) -> Result<()> {
    let file = &mut OpenOptions::new()
        .create(true)
        .write(true)
        .append(true)
        .open(path)?;

    affinity::set_thread_affinity([1]).unwrap();
    let core = affinity::get_thread_affinity().unwrap();
    println!("Affinity {}: {core:?}", std::process::id());

    let start = Instant::now();
    let config = StylusConfig::new(1, 4096, 10_000);
    let ink = 1_000_000;

    while start.elapsed() < limit.into() {
        let trial = Trial::sample(config, ink)?;
        if start.elapsed() > filter.into() {
            writeln!(file, "{}", trial)?;
        }
    }
    Ok(())
}*/

/*fn model(path: &Path, _output: &Path) -> Result<()> {
    let mut file = Feed::new(path)?;
    let mut best = Model::new();
    let mut trials = 0;

    let mut winners = vec![];

    while let Ok(trial) = file.next() {
        //trial.long_stats();
        let error = best.eval(&trial);

        let other = best.clone().tweak(&trial);
        let other_err = other.eval(&trial);
        if other_err.abs() < error.abs() {
            best = other;
        }

        if error.abs() < 0.05 {
            winners.push(best.clone());
            //println!("Best: {error}");
            //best.print(&trial);
        }
        if trials % 50_000_000 == 0 {
            println!("\n\nBest {:.2}, {}", error, winners.len());
            let avg = Model::avg(&winners);
            avg.print(&trial);
        }
        trials += 1;
    }

    Ok(())
}
*/
