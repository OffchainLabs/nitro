// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::Bytes20;

#[link(wasm_import_module = "forward")]
extern "C" {
    pub(crate) fn tx_origin(origin: *mut u8);
    pub(crate) fn tx_gas_price(gas_price: *mut u64);
}

pub fn origin() -> Bytes20 {
    let mut data = [0; 20];
    unsafe { tx_origin(data.as_mut_ptr()) };
    Bytes20(data)
}

pub fn gas_price() -> u64 {
    let mut gas_price: u64 = 0;
    unsafe { tx_gas_price(&mut gas_price as *mut _) };
    gas_price
}
