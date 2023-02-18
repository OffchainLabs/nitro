// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use wasmer::wasmparser::Operator;

mod native;
mod storage;
mod wavm;

fn expensive_add(op: &Operator) -> u64 {
    match op {
        Operator::I32Add => 100,
        _ => 0,
    }
}
