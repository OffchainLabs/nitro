// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{hostio, Bytes20, Bytes32};

pub fn basefee() -> Bytes32 {
    let mut data = [0; 32];
    unsafe { hostio::block_basefee(data.as_mut_ptr()) };
    Bytes32(data)
}

pub fn chainid() -> Bytes32 {
    let mut data = [0; 32];
    unsafe { hostio::block_chainid(data.as_mut_ptr()) };
    Bytes32(data)
}

pub fn coinbase() -> Bytes20 {
    let mut data = [0; 20];
    unsafe { hostio::block_coinbase(data.as_mut_ptr()) };
    Bytes20(data)
}

pub fn difficulty() -> Bytes32 {
    let mut data = [0; 32];
    unsafe { hostio::block_difficulty(data.as_mut_ptr()) };
    Bytes32(data)
}

pub fn gas_limit() -> u64 {
    unsafe { hostio::block_gas_limit() }
}

pub fn number() -> Bytes32 {
    let mut data = [0; 32];
    unsafe { hostio::block_number(data.as_mut_ptr()) };
    Bytes32(data)
}

pub fn timestamp() -> u64 {
    unsafe { hostio::block_timestamp() }
}
