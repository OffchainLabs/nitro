// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::Bytes20;
use crate::Bytes32;

#[link(wasm_import_module = "forward")]
extern "C" {
    pub(crate) fn emit_log(data: *const u8, len: usize, topics: usize);
}

pub fn log(topics: &[Bytes32], data: &[u8]) -> Result<(), &'static str> {
    if topics.len() > 4 {
        return Err("too many topics");
    }
    let mut bytes: Vec<u8> = vec![];
    bytes.extend(topics.iter().map(|x| x.0.iter()).flatten());
    bytes.extend(data);
    unsafe { emit_log(bytes.as_ptr(), bytes.len(), topics.len()) }
    Ok(())
}

#[link(wasm_import_module = "forward")]
extern "C" {
    pub(crate) fn evm_address_balance(address: *const u8, dest: *mut u8);
    pub(crate) fn evm_address_codehash(address: *const u8, dest: *mut u8);
    pub(crate) fn evm_blockhash(key: *const u8, dest: *mut u8);
    pub(crate) fn evm_gas_left() -> u64;
    pub(crate) fn evm_ink_left() -> u64;
}

pub fn address_balance(address: Bytes20) -> Option<Bytes32> {
    let mut data = [0; 32];
    unsafe { evm_address_balance(address.ptr(), data.as_mut_ptr()) };
    (data != [0; 32]).then_some(Bytes32(data))
}

pub fn address_code_hash(address: Bytes20) -> Option<Bytes32> {
    let mut data = [0; 32];
    unsafe { evm_address_codehash(address.ptr(), data.as_mut_ptr()) };
    (data != [0; 32]).then_some(Bytes32(data))
}

pub fn blockhash(key: Bytes32) -> Option<Bytes32> {
    let mut data = [0; 32];
    unsafe { evm_blockhash(key.ptr(), data.as_mut_ptr()) };
    (data != [0; 32]).then_some(Bytes32(data))
}

pub fn gas_left() -> u64 {
    unsafe { evm_gas_left() }
}

pub fn ink_left() -> u64 {
    unsafe { evm_ink_left() }
}
