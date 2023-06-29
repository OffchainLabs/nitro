// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{Bytes20, Bytes32};

#[derive(Default)]
#[must_use]
pub struct Call {
    kind: CallKind,
    value: Bytes32,
    ink: Option<u64>,
}

#[derive(PartialEq)]
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
        if self.value != Bytes32::default() && self.kind != CallKind::Basic {
            return Err("unexpected value".into());
        }

        let ink = self.ink.unwrap_or(u64::MAX); // will be clamped by 63/64 rule
        let status = match self.kind {
            CallKind::Basic => unsafe {
                call_contract(
                    contract.ptr(),
                    calldata.as_ptr(),
                    calldata.len(),
                    self.value.ptr(),
                    ink,
                    &mut outs_len,
                )
            },
            CallKind::Delegate => unsafe {
                delegate_call_contract(
                    contract.ptr(),
                    calldata.as_ptr(),
                    calldata.len(),
                    ink,
                    &mut outs_len,
                )
            },
            CallKind::Static => unsafe {
                static_call_contract(
                    contract.ptr(),
                    calldata.as_ptr(),
                    calldata.len(),
                    ink,
                    &mut outs_len,
                )
            },
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
