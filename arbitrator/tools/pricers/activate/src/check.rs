// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use activate::{util, Trial};
use arbutil::format;
use eyre::Result;
use std::path::PathBuf;

pub fn check(path: PathBuf) -> Result<()> {
    util::set_cpu_affinity(&[0]);
    let wat = util::file_bytes(&path)?;
    let wasm = wasmer::wat2wasm(&wat)?;
    let trial = Trial::new(&wasm)?;
    trial.print();
    println!("size: {}", format::bytes(wasm.len()));
    trial.check_model(1.)
}
