// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::Bytes20;

#[link(wasm_import_module = "forward")]
extern "C" {
    pub(crate) fn tx_origin(origin: *mut u8);
}

pub fn origin() -> Bytes20 {
    let mut data = [0; 20];
    unsafe { tx_origin(data.as_mut_ptr()) };
    Bytes20(data)
}
