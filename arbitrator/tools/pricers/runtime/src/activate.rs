// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::wasm;
use arbutil::{format, Bytes32};
use brotli2;
use eyre::Result;
use wasmer::Target;
use prover::{
    binary::WasmBinary,
    programs::prelude::*, machine::Module,
};
use std::{time::{Instant, Duration}, fs, io::{Read, Write}};
use stylus::native;

pub fn activate(target: Target) -> Result<()> {

    let wasm = fs::read(format!("../../../stylus/tests/erc20/target/wasm32-unknown-unknown/release/erc20.wasm"))?;
    //let wasm = wat2wasm(&wasm)?;
    wasm::validate(&wasm)?;

    let mut compile = CompileConfig::version(1, false);
    compile.debug.count_ops = false;

    let timer = Instant::now();
    let (bin, stylus_data) = WasmBinary::parse_user(&wasm, 128, &compile, &Bytes32::default())?;
    let parse_time = timer.elapsed();

    let timer = Instant::now();
    let module = Module::from_user_binary(&bin, false, Some(stylus_data))?;
    let wavm_time = timer.elapsed();

    let timer = Instant::now();
    module.hash();
    let hash_time = timer.elapsed();

    let timer = Instant::now();
    let asm = native::module(&wasm, compile, target)?;
    let asm_time = timer.elapsed();

    let timer = Instant::now();
    let b2_asm = compress(&asm)?;
    let b2_asm_time = timer.elapsed();
    
    let module = module.into_bytes();

    let timer = Instant::now();
    let b2_mod = compress(&module)?;
    let b2_mod_time = timer.elapsed();

    let timer = Instant::now();
    decompress(&b2_mod)?;
    let inflate_time = timer.elapsed();

    let fudge = 3.;
    let sync = 2.;
    let block_time = 1e9 / sync;
    let speed_limit = 7e6;

    let compute = |time: Duration| fudge * speed_limit * time.as_nanos() as f64 / block_time;
    let storage = |len| (len * 200) as f64 * 0.7;

    println!("compute");
    println!("  parse wasm {} => {:.1}k", format::time(parse_time), compute(parse_time) / 1e3);
    println!("  make mod   {} => {:.1}m", format::time(wavm_time), compute(wavm_time) / 1e6);
    println!("  shrink mod {} => {:.2}k", format::time(b2_mod_time), compute(b2_mod_time) / 1e3);
    println!("  grow mod   {} => {:.2}k", format::time(inflate_time), compute(inflate_time) / 1e3);
    println!("  hash mod   {} => {:.1}k", format::time(hash_time), compute(hash_time) / 1e3);
    println!("  make asm   {} => {:.1}m", format::time(asm_time), compute(asm_time) / 1e6);
    println!("  grow asm   {} => {:.2}k", format::time(b2_asm_time), compute(b2_asm_time) / 1e3);
    println!();

    println!("storage");
    println!("  wasm   {}kb", wasm.len() / 1024);
    println!("  mod    {}kb => {:.1}m", module.len() / 1024, storage(module.len()) / 1e6);
    println!("  b2 mod {}kb => {:.1}m", b2_mod.len() / 1024, storage(b2_mod.len()) / 1e6);
    println!("  asm    {}kb => {:.1}m", asm.len() / 1024, storage(asm.len()) / 1e6);
    println!("  b2 asm {}kb => {:.1}m", b2_asm.len() / 1024, storage(b2_asm.len()) / 1e6);

    Ok(())
}

fn compress(data: &[u8]) -> Result<Vec<u8>> {
    let mut encoder = brotli2::write::BrotliEncoder::new(vec![], 2);
    encoder.write_all(data)?;
    encoder.flush()?;
    Ok(encoder.finish().unwrap())
}

fn decompress(data: &[u8]) -> Result<Vec<u8>> {
    let mut decoder = brotli2::read::BrotliDecoder::new(data);
    let mut out = vec![];
    decoder.read_to_end(&mut out)?;
    Ok(out)
}
