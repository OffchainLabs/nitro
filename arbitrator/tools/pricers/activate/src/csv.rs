// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::Trial;
use eyre::Result;
use std::{
    fs::File,
    io::{BufRead, BufReader},
    path::PathBuf,
};

pub fn csv(path: PathBuf) -> Result<()> {
    let file = BufReader::new(File::open(path)?);

    println!("parse,mod,hash,brotli,asm,init,wasm_len,mod_len,brotli_len,asm_len,funcs,code,data,mem_size");

    for line in file.lines() {
        let t: Trial = line?.parse()?;
        let i = t.info;
        
        println!(
            "{},{},{},{},{},{},{},{},{},{},{},{},{},{}",
            t.parse_time.as_micros(),
            t.module_time.as_micros(),
            t.hash_time.as_micros(),
            t.brotli_time.as_micros(),
            t.asm_time.as_micros(),
            t.init_time.as_micros(),
            t.wasm_len,
            t.module_len,
            t.brotli_len,
            t.asm_len,
            i.funcs,
            i.code,
            i.data,
            i.mem_size
        );
    }
    Ok(())
}
