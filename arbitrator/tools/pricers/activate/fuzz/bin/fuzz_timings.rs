// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![no_main]

use activate::{Trial, util, wasm};
use arbutil::color::Color;
use eyre::{Result, bail};
use libfuzzer_sys::fuzz_target;
use std::{fs::File, io::Write};

fuzz_target!(|data: &[u8]| {
    util::set_cpu_affinity(&[1, 2, 3, 4, 5, 6, 7, 8]);
    if let Err(error) = fuzz(data) {
        println!("{}", error);
    }
});

fn fuzz(data: &[u8]) -> Result<()> {
    let wasm = wasm::random(data)?;
    if wasm.len() > 128 * 1024 {
        bail!("too big");
    }
    
    let trial = Trial::new(&wasm)?;
    let margin = 10.;

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
