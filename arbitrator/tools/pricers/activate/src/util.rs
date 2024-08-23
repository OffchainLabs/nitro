// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use arbutil::evm::api::EvmApi;
use eyre::Result;
use rand::{distributions::Standard, prelude::Distribution, Rng};
use std::{
    fs::File,
    io::{Read, Write},
    path::Path,
    time::Duration,
};

pub fn random_vec<T>(len: usize) -> Vec<T>
where
    Standard: Distribution<T>,
{
    let mut rng = rand::thread_rng();
    let mut entropy = Vec::with_capacity(len);
    for _ in 0..len {
        entropy.push(rng.gen())
    }
    entropy
}

pub fn gas(time: Duration, fudge: f64) -> u64 {
    let sync = 2.;
    let block_time = 1e9 / sync;
    let speed_limit = 7e6;
    let ns = time.as_nanos() as f64;
    let gas = fudge * speed_limit * ns / block_time;
    gas.ceil() as u64
}

pub fn compress(data: &[u8], level: u32) -> Result<Vec<u8>> {
    let mut encoder = brotli2::write::BrotliEncoder::new(vec![], level);
    encoder.write_all(data)?;
    encoder.flush()?;
    Ok(encoder.finish().unwrap())
}

pub fn set_cpu_affinity(cpus: &[usize]) {
    affinity::set_thread_affinity(cpus).unwrap();
    let _core = affinity::get_thread_affinity().unwrap();
    //println!("Affinity {}: {core:?}", std::process::id());
}

pub fn file_bytes(path: &Path) -> Result<Vec<u8>> {
    let mut f = File::open(path)?;
    let mut buf = Vec::new();
    f.read_to_end(&mut buf)?;
    Ok(buf)
}

pub struct FakeEvm;

impl EvmApi for FakeEvm {
    fn account_balance(&mut self, _: arbutil::Bytes20) -> (arbutil::Bytes32, u64) {
        unimplemented!()
    }

    fn account_codehash(&mut self, _: arbutil::Bytes20) -> (arbutil::Bytes32, u64) {
        unimplemented!()
    }

    fn add_pages(&mut self, _: u16) -> u64 {
        unimplemented!()
    }

    fn capture_hostio(&self, _: &str, _: &[u8], _: &[u8], _: u64, _: u64) {
        unimplemented!()
    }

    fn contract_call(
        &mut self,
        _: arbutil::Bytes20,
        _: &[u8],
        _: u64,
        _: arbutil::Bytes32,
    ) -> (u32, u64, arbutil::evm::user::UserOutcomeKind) {
        unimplemented!()
    }

    fn create1(
        &mut self,
        _: Vec<u8>,
        _: arbutil::Bytes32,
        _: u64,
    ) -> (eyre::Result<arbutil::Bytes20>, u32, u64) {
        unimplemented!()
    }

    fn create2(
        &mut self,
        _: Vec<u8>,
        _: arbutil::Bytes32,
        _: arbutil::Bytes32,
        _: u64,
    ) -> (eyre::Result<arbutil::Bytes20>, u32, u64) {
        unimplemented!()
    }

    fn delegate_call(
        &mut self,
        _: arbutil::Bytes20,
        _: &[u8],
        _: u64,
    ) -> (u32, u64, arbutil::evm::user::UserOutcomeKind) {
        unimplemented!()
    }

    fn emit_log(&mut self, _: Vec<u8>, _: u32) -> eyre::Result<()> {
        unimplemented!()
    }

    fn get_bytes32(&mut self, _: arbutil::Bytes32) -> (arbutil::Bytes32, u64) {
        unimplemented!()
    }

    fn get_return_data(&mut self, _: u32, _: u32) -> Vec<u8> {
        unimplemented!()
    }

    fn set_bytes32(&mut self, _: arbutil::Bytes32, _: arbutil::Bytes32) -> eyre::Result<u64> {
        unimplemented!()
    }

    fn static_call(
        &mut self,
        _: arbutil::Bytes20,
        _: &[u8],
        _: u64,
    ) -> (u32, u64, arbutil::evm::user::UserOutcomeKind) {
        unimplemented!()
    }
}
