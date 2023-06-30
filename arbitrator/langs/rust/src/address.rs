// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{hostio, Bytes20, Bytes32};

pub fn balance(address: Bytes20) -> Bytes32 {
    let mut data = [0; 32];
    unsafe { hostio::account_balance(address.ptr(), data.as_mut_ptr()) };
    data.into()
}

pub fn codehash(address: Bytes20) -> Option<Bytes32> {
    let mut data = [0; 32];
    unsafe { hostio::account_codehash(address.ptr(), data.as_mut_ptr()) };
    (data != [0; 32]).then_some(Bytes32(data))
}
