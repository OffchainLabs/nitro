// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::{fs::File, path::PathBuf};

use clap::Parser;
use eyre::Result;
use forward::{forward, forward_stub};

#[derive(Parser)]
#[command(name = "forward")]
struct Opts {
    #[arg(long)]
    path: PathBuf,
    #[arg(long)]
    stub: bool,
}

fn main() -> Result<()> {
    let opts = Opts::parse();
    let file = &mut File::options()
        .create(true)
        .write(true)
        .truncate(true)
        .open(opts.path)?;

    match opts.stub {
        true => forward_stub(file),
        false => forward(file),
    }
}
