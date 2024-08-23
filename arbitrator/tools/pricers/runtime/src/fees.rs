use crate::{wasm, util};
use brotli2;
use eyre::Result;
use prover::{
    binary::WasmBinary,
    programs::prelude::*, machine::Module,
};
use std::{time::{Instant, Duration}, fs, io::{Read, Write}};
use stylus::native;

pub fn fees() -> Result<()> {
    let len = 1000 * 1024;
    //let unit = util::random_vec(len);
    /*let mut full = vec![];
    for _ in 0..1000 {
        full.extend(&unit);
}*/

    let mut best = 1.;

    loop {
        let mut unit = util::random_vec(len);
        for i in 0..unit.len() {
            unit[i] /= 224;
            if i % 2 == 0 || i % 3 == 0 || i % 5 == 0 {
                unit[i] = 0;
            }
        }
        
        let b0 = compress(&unit, 0)?;
        let b11 = compress(&unit, 11)?;
        let diff = b11 as f64 / b0 as f64;

        if diff < best {
            best = diff;
            println!("{}", hex::encode(&unit));
            println!("{} {} {}", diff, b0, b11);
        }
    }
}

fn compress(data: &[u8], level: u32) -> Result<usize> {
    let mut encoder = brotli2::write::BrotliEncoder::new(vec![], level);
    encoder.write_all(data)?;
    encoder.flush()?;
    Ok(encoder.finish().unwrap().len())
}

