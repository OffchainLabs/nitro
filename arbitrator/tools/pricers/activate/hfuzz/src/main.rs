// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use activate::{util, wasm, Trial};
use eyre::{bail, Result};
use honggfuzz::fuzz;
use std::{fs::File, io::Write};

fn main() {
    util::set_cpu_affinity(&[0, 1, 2, 3, 4, 5, 6, 7]);

    loop {
        fuzz!(|data: &[u8]| {
            if let Err(error) = fuzz_impl(data) {
                println!("{}", error);
            }
        });
    }
}

fn fuzz_impl(data: &[u8]) -> Result<()> {
    let wasm = wasm::random(data)?;
    if wasm.len() > 128 * 1024 {
        bail!("too big");
    }

    for _ in 0..4 {
        let trial = Trial::new(&wasm)?;
        if trial.check_model(100.).is_ok() {
            return Ok(());
        }
    }

    let trial = Trial::new(&wasm)?;
    let margin = 100.;

    if let Err(error) = trial.check_model(margin) {
        let wat = wasm::wat(&wasm).unwrap_or("???".into());
        println!("{}", &wat);
        println!("{}", wasm.len());
        trial.print();

        let mut file = File::create("crash.wat")?;
        writeln!(file, "{}", wat)?;
        panic!("{error:?}");
    }
    Ok(())
}
