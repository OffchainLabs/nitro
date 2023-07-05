// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{hostio, Bytes20, Bytes32};

pub fn gas_price() -> Bytes32 {
    let mut data = [0; 32];
    unsafe { hostio::tx_gas_price(data.as_mut_ptr()) };
    Bytes32(data)
}

pub fn ink_price() -> u64 {
    unsafe { hostio::tx_ink_price() }
}

pub fn origin() -> Bytes20 {
    let mut data = [0; 20];
    unsafe { hostio::tx_origin(data.as_mut_ptr()) };
    Bytes20(data)
}
