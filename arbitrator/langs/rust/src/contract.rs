// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{address as addr, Bytes20, Bytes32};

#[derive(Clone, Default)]
#[must_use]
pub struct Call {
    kind: CallKind,
    value: Bytes32,
    ink: Option<u64>,
}

#[derive(Clone, PartialEq)]
enum CallKind {
    Basic,
    Delegate,
    Static,
}

impl Default for CallKind {
    fn default() -> Self {
        CallKind::Basic
    }
}

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
        ink: u64,
        return_data_len: *mut usize,
    ) -> u8;

    fn delegate_call_contract(
        contract: *const u8,
        calldata: *const u8,
        calldata_len: usize,
        ink: u64,
        return_data_len: *mut usize,
    ) -> u8;

    fn static_call_contract(
        contract: *const u8,
        calldata: *const u8,
        calldata_len: usize,
        ink: u64,
        return_data_len: *mut usize,
    ) -> u8;

    /// A noop when there's never been a call
    fn read_return_data(dest: *mut u8);
}

impl Call {
    pub fn new() -> Self {
        Default::default()
    }

    pub fn new_delegate() -> Self {
        Self {
            kind: CallKind::Delegate,
            ..Default::default()
        }
    }

    pub fn new_static() -> Self {
        Self {
            kind: CallKind::Static,
            ..Default::default()
        }
    }

    pub fn value(mut self, value: Bytes32) -> Self {
        self.value = value;
        self
    }

    pub fn ink(mut self, ink: u64) -> Self {
        self.ink = Some(ink);
        self
    }

    pub fn call(self, contract: Bytes20, calldata: &[u8]) -> Result<Vec<u8>, Vec<u8>> {
        let mut outs_len = 0;
        if !self.value.is_zero() && self.kind != CallKind::Basic {
            return Err("unexpected value".into());
        }

        let ink = self.ink.unwrap_or(u64::MAX); // will be clamped by 63/64 rule
        let status = unsafe {
            match self.kind {
                CallKind::Basic => call_contract(
                    contract.ptr(),
                    calldata.as_ptr(),
                    calldata.len(),
                    self.value.ptr(),
                    ink,
                    &mut outs_len,
                ),
                CallKind::Delegate => delegate_call_contract(
                    contract.ptr(),
                    calldata.as_ptr(),
                    calldata.len(),
                    ink,
                    &mut outs_len,
                ),
                CallKind::Static => static_call_contract(
                    contract.ptr(),
                    calldata.as_ptr(),
                    calldata.len(),
                    ink,
                    &mut outs_len,
                ),
            }
        };

        let len = outs_len;
        let outs = if len == 0 {
            vec![]
        } else {
            unsafe {
                let mut outs = Vec::with_capacity(len);
                read_return_data(outs.as_mut_ptr());
                outs.set_len(len);
                outs
            }
        };
        match status {
            0 => Ok(outs),
            _ => Err(outs),
        }
    }
}
#[link(wasm_import_module = "forward")]
extern "C" {
    fn create1(
        code: *const u8,
        code_len: usize,
        endowment: *const u8,
        contract: *mut u8,
        revert_data_len: *mut usize,
    );

    fn create2(
        code: *const u8,
        code_len: usize,
        endowment: *const u8,
        salt: *const u8,
        contract: *mut u8,
        revert_data_len: *mut usize,
    );

    /// Returns 0 when there's never been a call
    fn return_data_size() -> u32;
}

pub fn create(code: &[u8], endowment: Bytes32, salt: Option<Bytes32>) -> Result<Bytes20, Vec<u8>> {
    let mut contract = [0; 20];
    let mut revert_data_len = 0;
    let contract = unsafe {
        if let Some(salt) = salt {
            create2(
                code.as_ptr(),
                code.len(),
                endowment.ptr(),
                salt.ptr(),
                contract.as_mut_ptr(),
                &mut revert_data_len as *mut _,
            );
        } else {
            create1(
                code.as_ptr(),
                code.len(),
                endowment.ptr(),
                contract.as_mut_ptr(),
                &mut revert_data_len as *mut _,
            );
        }
        Bytes20(contract)
    };
    if contract.is_zero() {
        unsafe {
            let mut revert_data = Vec::with_capacity(revert_data_len);
            read_return_data(revert_data.as_mut_ptr());
            revert_data.set_len(revert_data_len);
            return Err(revert_data);
        }
    }
    Ok(contract)
}

pub fn return_data_len() -> usize {
    unsafe { return_data_size() as usize }
}

#[link(wasm_import_module = "forward")]
extern "C" {
    pub(crate) fn contract_address(address: *mut u8);
}

pub fn address() -> Bytes20 {
    let mut data = [0; 20];
    unsafe { contract_address(data.as_mut_ptr()) };
    Bytes20(data)
}

pub fn balance() -> Bytes32 {
    addr::balance(address())
}
