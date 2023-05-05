// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{Bytes20, Bytes32};

#[link(wasm_import_module = "forward")]
extern "C" {
    pub(crate) fn msg_sender(sender: *mut u8);
    pub(crate) fn msg_value(value: *mut u8);
}

pub fn sender() -> Bytes20 {
    let mut data = [0; 20];
    unsafe { msg_sender(data.as_mut_ptr()) };
    Bytes20(data)
}

pub fn value() -> Bytes32 {
    let mut data = [0; 32];
    unsafe { msg_value(data.as_mut_ptr()) };
    Bytes32(data)
}
