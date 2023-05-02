// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(clippy::missing_safety_doc)]

use crate::{Program, ARGS, KEYS, LOGS, OUTS};
use arbutil::{evm, wavm, Bytes32};
use prover::programs::prelude::GasMeteredMachine;

#[no_mangle]
pub unsafe extern "C" fn forward__read_args(ptr: usize) {
    let mut program = Program::start();
    program.pay_for_evm_copy(ARGS.len() as u64).unwrap();
    wavm::write_slice_usize(&ARGS, ptr);
}

#[no_mangle]
pub unsafe extern "C" fn forward__return_data(ptr: usize, len: usize) {
    let mut program = Program::start();
    program.pay_for_evm_copy(len as u64).unwrap();
    OUTS = wavm::read_slice_usize(ptr, len);
}

#[no_mangle]
pub unsafe extern "C" fn forward__account_load_bytes32(key: usize, dest: usize) {
    let mut program = Program::start();
    let key = Bytes32(wavm::read_bytes32(key));

    let value = KEYS.lock().get(&key).cloned().unwrap_or_default();
    program.buy_gas(2100).unwrap(); // pretend it was cold
    wavm::write_slice_usize(&value.0, dest);
}

#[no_mangle]
pub unsafe extern "C" fn forward__account_store_bytes32(key: usize, value: usize) {
    let mut program = Program::start();
    program.require_gas(evm::SSTORE_SENTRY_GAS).unwrap();
    program.buy_gas(22100).unwrap(); // pretend the worst case

    let key = wavm::read_bytes32(key);
    let value = wavm::read_bytes32(value);
    KEYS.lock().insert(key.into(), value.into());
}

#[no_mangle]
pub unsafe extern "C" fn forward__emit_log(data: usize, len: u32, topics: u32) {
    let mut program = Program::start();
    if topics > 4 || len < topics * 32 {
        panic!("bad topic data");
    }
    program.pay_for_evm_log(topics, len - topics * 32).unwrap();
    let data = wavm::read_slice_usize(data, len as usize);
    LOGS.push(data)
}
