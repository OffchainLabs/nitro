// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use super::util::{Bytes20, Bytes32};

#[derive(Copy, Clone)]
#[repr(C)]
struct RustVec {
    ptr: *mut u8,
    len: usize,
    cap: usize,
}

impl Default for RustVec {
    fn default() -> Self {
        Self {
            ptr: std::ptr::null_mut(),
            len: 0,
            cap: 0,
        }
    }
}

#[link(wasm_import_module = "forward")]
extern "C" {
    fn util_move_vec(source: *const RustVec, dest: *mut u8);

    fn call_contract(
        contract: *const u8,
        calldata: *const u8,
        calldata_len: usize,
        value: *const u8,
        output: *mut RustVec,
    ) -> u8;
}

pub fn call(contract: Bytes20, calldata: &[u8], value: Bytes32) -> Result<Vec<u8>, Vec<u8>> {
    let mut outs = RustVec::default();
    let status = unsafe {
        call_contract(
            contract.ptr(),
            calldata.as_ptr(),
            calldata.len(),
            value.ptr(),
            &mut outs as *mut _,
        )
    };
    let outs = unsafe {
        let mut data = Vec::with_capacity(outs.len);
        util_move_vec(&outs as *const _, data.as_mut_ptr());
        data.set_len(outs.len);
        data
    };
    match status {
        0 => Ok(outs),
        _ => Err(outs),
    }
}
