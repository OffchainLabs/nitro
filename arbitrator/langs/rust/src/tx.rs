// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{Bytes20, Bytes32};

#[link(wasm_import_module = "forward")]
extern "C" {
    pub(crate) fn tx_gas_price(gas_price: *mut u8);
    pub(crate) fn tx_ink_price() -> u64;
    pub(crate) fn tx_origin(origin: *mut u8);
}

pub fn gas_price() -> Bytes32 {
    let mut data = [0; 32];
    unsafe { tx_gas_price(data.as_mut_ptr()) };
    Bytes32(data)
}

pub fn ink_price() -> u64 {
    unsafe { tx_ink_price() }
}

pub fn origin() -> Bytes20 {
    let mut data = [0; 20];
    unsafe { tx_origin(data.as_mut_ptr()) };
    Bytes20(data)
}
