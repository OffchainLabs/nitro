// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(clippy::missing_safety_doc)]

use crate::{Program, ARGS, EVER_PAGES, KEYS, LOGS, OPEN_PAGES, OUTS};
use arbutil::{
    crypto, evm,
    pricing::{EVM_API_INK, HOSTIO_INK, PTR_INK},
    wavm,
};
use prover::programs::{
    memory::MemoryModel,
    prelude::{GasMeteredMachine, MeteredMachine},
};

#[no_mangle]
pub unsafe extern "C" fn vm_hooks__read_args(ptr: usize) {
    let mut program = Program::start(0);
    program.pay_for_write(ARGS.len() as u32).unwrap();
    wavm::write_slice_usize(&ARGS, ptr);
}

#[no_mangle]
pub unsafe extern "C" fn vm_hooks__write_result(ptr: usize, len: usize) {
    let mut program = Program::start(0);
    program.pay_for_read(len as u32).unwrap();
    OUTS = wavm::read_slice_usize(ptr, len);
}

#[no_mangle]
pub unsafe extern "C" fn vm_hooks__storage_load_bytes32(key: usize, dest: usize) {
    let mut program = Program::start(2 * PTR_INK + EVM_API_INK);
    let key = wavm::read_bytes32(key);

    let value = KEYS.lock().get(&key).cloned().unwrap_or_default();
    program.buy_gas(2100).unwrap(); // pretend it was cold
    wavm::write_bytes32(dest, value);
}

#[no_mangle]
pub unsafe extern "C" fn vm_hooks__storage_store_bytes32(key: usize, value: usize) {
    let mut program = Program::start(2 * PTR_INK + EVM_API_INK);
    program.require_gas(evm::SSTORE_SENTRY_GAS).unwrap();
    program.buy_gas(22100).unwrap(); // pretend the worst case

    let key = wavm::read_bytes32(key);
    let value = wavm::read_bytes32(value);
    KEYS.lock().insert(key, value);
}

#[no_mangle]
pub unsafe extern "C" fn vm_hooks__emit_log(data: usize, len: u32, topics: u32) {
    let mut program = Program::start(EVM_API_INK);
    if topics > 4 || len < topics * 32 {
        panic!("bad topic data");
    }
    program.pay_for_read(len.into()).unwrap();
    program.pay_for_evm_log(topics, len - topics * 32).unwrap();

    let data = wavm::read_slice_usize(data, len as usize);
    LOGS.push(data)
}

#[no_mangle]
pub unsafe extern "C" fn vm_hooks__memory_grow(pages: u16) {
    let mut program = Program::start_free();
    if pages == 0 {
        return program.buy_ink(HOSTIO_INK).unwrap();
    }
    let model = MemoryModel::new(2, 1000);

    let (open, ever) = (OPEN_PAGES, EVER_PAGES);
    OPEN_PAGES = OPEN_PAGES.saturating_add(pages);
    EVER_PAGES = EVER_PAGES.max(OPEN_PAGES);
    program.buy_gas(model.gas_cost(pages, open, ever)).unwrap();
}

#[no_mangle]
pub unsafe extern "C" fn vm_hooks__native_keccak256(bytes: usize, len: usize, output: usize) {
    let mut program = Program::start(0);
    program.pay_for_keccak(len as u32).unwrap();

    let preimage = wavm::read_slice_usize(bytes, len);
    let digest = crypto::keccak(preimage);
    wavm::write_slice_usize(&digest, output);
}

#[no_mangle]
pub unsafe extern "C" fn vm_hooks__msg_reentrant() -> u32 {
    let _ = Program::start(0);
    0
}
