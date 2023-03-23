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
    fn call_contract(
        contract: *const u8,
        calldata: *const u8,
        calldata_len: usize,
        value: *const u8,
        gas: u64,
        return_data_len: *mut usize,
    ) -> u8;

    fn read_return_data(dest: *mut u8);
}

pub fn call(contract: Bytes20, calldata: &[u8], value: Option<Bytes32>, gas: Option<u64>) -> Result<Vec<u8>, Vec<u8>> {
    let mut outs_len = 0;
    let value = value.unwrap_or_default();
    let gas = gas.unwrap_or(u64::MAX); // will be clamped by 63/64 rule
    let status = unsafe {
        call_contract(
            contract.ptr(),
            calldata.as_ptr(),
            calldata.len(),
            value.ptr(),
            gas,
            &mut outs_len as *mut _,
        )
    };
    let outs = unsafe {
        let mut outs = Vec::with_capacity(outs_len);
        read_return_data(outs.as_mut_ptr());
        outs.set_len(outs_len);
        outs
    };
    match status {
        0 => Ok(outs),
        _ => Err(outs),
    }
}
