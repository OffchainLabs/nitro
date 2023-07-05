// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{hostio, Bytes32};

pub fn log(topics: &[Bytes32], data: &[u8]) -> Result<(), &'static str> {
    if topics.len() > 4 {
        return Err("too many topics");
    }
    let mut bytes: Vec<u8> = vec![];
    bytes.extend(topics.iter().flat_map(|x| x.0.iter()));
    bytes.extend(data);
    unsafe { hostio::emit_log(bytes.as_ptr(), bytes.len(), topics.len()) }
    Ok(())
}

pub fn blockhash(number: Bytes32) -> Option<Bytes32> {
    let mut dest = [0; 32];
    unsafe { hostio::evm_blockhash(number.ptr(), dest.as_mut_ptr()) };
    (dest != [0; 32]).then_some(Bytes32(dest))
}

pub fn gas_left() -> u64 {
    unsafe { hostio::evm_gas_left() }
}

pub fn ink_left() -> u64 {
    unsafe { hostio::evm_ink_left() }
}
