// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use activate::{util, wasm, Trial};
use arbutil::Color;
use eyre::Result;
use std::{fs::File, io::Write, path::PathBuf};

pub fn record(path: PathBuf, count: u64) -> Result<()> {
    let file = &mut File::options().create(true).append(true).open(path)?;
    util::set_cpu_affinity(&[1]);

    for _ in 0..count {
        let len = rand::random::<usize>() % 256 * 1024;
        let wasm = wasm::random_uniform(len)?;
        let trial = match Trial::new(&wasm) {
            Ok(trial) => trial,
            Err(error) => {
                if format!("{error:?}").contains("Out-of-bounds data memory init") {
                    println!("{}", error.red());
                    continue;
                }
                println!("{}", wasm::wat(&wasm)?);
                return Err(error);
            }
        };

        trial.print();

        writeln!(file, "{}", trial.to_hex()?)?;
    }
    Ok(())
}
