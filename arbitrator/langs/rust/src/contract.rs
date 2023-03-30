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

    fn delegate_call_contract(
        contract: *const u8,
        calldata: *const u8,
        calldata_len: usize,
        gas: u64,
        return_data_len: *mut usize,
    ) -> u8;

    fn static_call_contract(
        contract: *const u8,
        calldata: *const u8,
        calldata_len: usize,
        gas: u64,
        return_data_len: *mut usize,
    ) -> u8;

    /// A noop when there's never been a call
    fn read_return_data(dest: *mut u8);
}

/// Calls the contract at the given address, with options for passing value and to limit the amount of gas supplied.
/// On failure, the output consists of the call's revert data.
pub fn call(
    contract: Bytes20,
    calldata: &[u8],
    value: Option<Bytes32>,
    gas: Option<u64>,
) -> Result<Vec<u8>, Vec<u8>> {
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

/// Delegate calls the contract at the given address, with the option to limit the amount of gas supplied.
/// On failure, the output consists of the call's revert data.
pub fn delegate_call(
    contract: Bytes20,
    calldata: &[u8],
    gas: Option<u64>,
) -> Result<Vec<u8>, Vec<u8>> {
    let mut outs_len = 0;
    let gas = gas.unwrap_or(u64::MAX); // will be clamped by 63/64 rule
    let status = unsafe {
        delegate_call_contract(
            contract.ptr(),
            calldata.as_ptr(),
            calldata.len(),
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

/// Static calls the contract at the given address, with the option to limit the amount of gas supplied.
/// On failure, the output consists of the call's revert data.
pub fn static_call(
    contract: Bytes20,
    calldata: &[u8],
    gas: Option<u64>,
) -> Result<Vec<u8>, Vec<u8>> {
    let mut outs_len = 0;
    let gas = gas.unwrap_or(u64::MAX); // will be clamped by 63/64 rule
    let status = unsafe {
        static_call_contract(
            contract.ptr(),
            calldata.as_ptr(),
            calldata.len(),
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
