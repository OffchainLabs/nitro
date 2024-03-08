// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(clippy::missing_safety_doc)]

use crate::{Program, ARGS, EVER_PAGES, KEYS, LOGS, OPEN_PAGES, OUTS};
use arbutil::{
    crypto, evm,
    pricing::{EVM_API_INK, HOSTIO_INK, PTR_INK},
    Bytes32,
};
use caller_env::{static_caller::STATIC_MEM, MemAccess, GuestPtr};
use prover::programs::{
    memory::MemoryModel,
    prelude::{GasMeteredMachine, MeteredMachine},
};

unsafe fn read_bytes32(ptr: GuestPtr) -> Bytes32 {
    STATIC_MEM.read_fixed(ptr).into()
}

#[no_mangle]
pub unsafe extern "C" fn vm_hooks__read_args(ptr: GuestPtr) {
    let mut program = Program::start(0);
    program.pay_for_write(ARGS.len() as u32).unwrap();
    STATIC_MEM.write_slice(ptr, &ARGS);
}

#[no_mangle]
pub unsafe extern "C" fn vm_hooks__write_result(ptr: GuestPtr, len: u32) {
    let mut program = Program::start(0);
    program.pay_for_read(len).unwrap();
    program.pay_for_geth_bytes(len).unwrap();
    OUTS = STATIC_MEM.read_slice(ptr, len as usize);
}

#[no_mangle]
pub unsafe extern "C" fn vm_hooks__storage_load_bytes32(key: GuestPtr, dest: GuestPtr) {
    let mut program = Program::start(2 * PTR_INK + EVM_API_INK);
    let key = read_bytes32(key);

    let value = KEYS.lock().get(&key).cloned().unwrap_or_default();
    program.buy_gas(2100).unwrap(); // pretend it was cold
    STATIC_MEM.write_slice(dest, &value.0);
}

#[no_mangle]
pub unsafe extern "C" fn vm_hooks__storage_store_bytes32(key: GuestPtr, value: GuestPtr) {
    let mut program = Program::start(2 * PTR_INK + EVM_API_INK);
    program.require_gas(evm::SSTORE_SENTRY_GAS).unwrap();
    program.buy_gas(22100).unwrap(); // pretend the worst case

    let key = read_bytes32(key);
    let value = read_bytes32(value);
    KEYS.lock().insert(key, value);
}

#[no_mangle]
pub unsafe extern "C" fn vm_hooks__emit_log(data: GuestPtr, len: u32, topics: u32) {
    let mut program = Program::start(EVM_API_INK);
    if topics > 4 || len < topics * 32 {
        panic!("bad topic data");
    }
    program.pay_for_read(len).unwrap();
    program.pay_for_evm_log(topics, len - topics * 32).unwrap();

    let data = STATIC_MEM.read_slice(data, len as usize);
    LOGS.push(data)
}

#[no_mangle]
pub unsafe extern "C" fn vm_hooks__pay_for_memory_grow(pages: u16) {
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
pub unsafe extern "C" fn vm_hooks__native_keccak256(bytes: GuestPtr, len: u32, output: GuestPtr) {
    let mut program = Program::start(0);
    program.pay_for_keccak(len).unwrap();

    let preimage = STATIC_MEM.read_slice(bytes, len as usize);
    let digest = crypto::keccak(preimage);
    STATIC_MEM.write_slice(output, &digest);
}

#[no_mangle]
pub unsafe extern "C" fn vm_hooks__msg_reentrant() -> u32 {
    let _ = Program::start(0);
    0
}
