// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{hostio, Bytes20, Bytes32};

pub fn sender() -> Bytes20 {
    let mut data = [0; 20];
    unsafe { hostio::msg_sender(data.as_mut_ptr()) };
    Bytes20(data)
}

pub fn value() -> Bytes32 {
    let mut data = [0; 32];
    unsafe { hostio::msg_value(data.as_mut_ptr()) };
    Bytes32(data)
}
