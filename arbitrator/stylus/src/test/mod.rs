// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use prover::utils::{Bytes20, Bytes32};
use rand::prelude::*;
use wasmer::wasmparser::Operator;

mod api;
mod native;
mod wavm;

fn expensive_add(op: &Operator) -> u64 {
    match op {
        Operator::I32Add => 100,
        _ => 0,
    }
}

pub fn random_bytes20() -> Bytes20 {
    let mut data = [0; 20];
    rand::thread_rng().fill_bytes(&mut data);
    data.into()
}

fn random_bytes32() -> Bytes32 {
    let mut data = [0; 32];
    rand::thread_rng().fill_bytes(&mut data);
    data.into()
}
