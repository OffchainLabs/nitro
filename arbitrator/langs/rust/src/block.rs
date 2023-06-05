// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{Bytes20, Bytes32};

#[link(wasm_import_module = "forward")]
extern "C" {
    pub(crate) fn block_basefee(basefee: *mut u8);
    pub(crate) fn block_chainid(chainid: *mut u8);
    pub(crate) fn block_coinbase(coinbase: *mut u8);
    pub(crate) fn block_difficulty(difficulty: *mut u8);
    pub(crate) fn block_gas_limit() -> u64;
    pub(crate) fn block_number(number: *mut u8);
    pub(crate) fn block_timestamp() -> u64;
}

pub fn basefee() -> Bytes32 {
    let mut data = [0; 32];
    unsafe { block_basefee(data.as_mut_ptr()) };
    Bytes32(data)
}

pub fn chainid() -> Bytes32 {
    let mut data = [0; 32];
    unsafe { block_chainid(data.as_mut_ptr()) };
    Bytes32(data)
}

pub fn coinbase() -> Bytes20 {
    let mut data = [0; 20];
    unsafe { block_coinbase(data.as_mut_ptr()) };
    Bytes20(data)
}

pub fn difficulty() -> Bytes32 {
    let mut data = [0; 32];
    unsafe { block_difficulty(data.as_mut_ptr()) };
    Bytes32(data)
}

pub fn gas_limit() -> u64 {
    unsafe { block_gas_limit() }
}

pub fn number() -> Bytes32 {
    let mut data = [0; 32];
    unsafe { block_number(data.as_mut_ptr()) };
    Bytes32(data)
}

pub fn timestamp() -> u64 {
    unsafe { block_timestamp() }
}
