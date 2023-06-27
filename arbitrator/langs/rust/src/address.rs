// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{Bytes20, Bytes32};

#[link(wasm_import_module = "forward")]
extern "C" {
    pub(crate) fn account_balance(address: *const u8, dest: *mut u8);
    pub(crate) fn account_codehash(address: *const u8, dest: *mut u8);
}

pub fn balance(address: Bytes20) -> Bytes32 {
    let mut data = [0; 32];
    unsafe { account_balance(address.ptr(), data.as_mut_ptr()) };
    data.into()
}

pub fn codehash(address: Bytes20) -> Option<Bytes32> {
    let mut data = [0; 32];
    unsafe { account_codehash(address.ptr(), data.as_mut_ptr()) };
    (data != [0; 32]).then_some(Bytes32(data))
}
