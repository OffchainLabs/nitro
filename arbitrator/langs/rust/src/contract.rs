// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{address as addr, hostio, tx, Bytes20, Bytes32};

#[derive(Clone, Default)]
#[must_use]
pub struct Call {
    kind: CallKind,
    value: Bytes32,
    gas: Option<u64>,
    offset: usize,
    size: Option<usize>,
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

    pub fn value(mut self, callvalue: Bytes32) -> Self {
        if self.kind != CallKind::Basic {
            panic!("cannot set value for delegate or static calls");
        }
        self.value = callvalue;
        self
    }

    pub fn gas(mut self, gas: u64) -> Self {
        self.gas = Some(gas);
        self
    }

    pub fn ink(mut self, ink: u64) -> Self {
        self.gas = Some(tx::ink_to_gas(ink));
        self
    }

    pub fn limit_return_data(mut self, offset: usize, size: usize) -> Self {
        self.offset = offset;
        self.size = Some(size);
        self
    }

    pub fn skip_return_data(self) -> Self {
        self.limit_return_data(0, 0)
    }

    pub fn call(self, contract: Bytes20, calldata: &[u8]) -> Result<Vec<u8>, Vec<u8>> {
        let mut outs_len = 0;
        let gas = self.gas.unwrap_or(u64::MAX); // will be clamped by 63/64 rule
        let status = unsafe {
            match self.kind {
                CallKind::Basic => hostio::call_contract(
                    contract.ptr(),
                    calldata.as_ptr(),
                    calldata.len(),
                    self.value.ptr(),
                    gas,
                    &mut outs_len,
                ),
                CallKind::Delegate => hostio::delegate_call_contract(
                    contract.ptr(),
                    calldata.as_ptr(),
                    calldata.len(),
                    gas,
                    &mut outs_len,
                ),
                CallKind::Static => hostio::static_call_contract(
                    contract.ptr(),
                    calldata.as_ptr(),
                    calldata.len(),
                    gas,
                    &mut outs_len,
                ),
            }
        };

        unsafe {
            hostio::CACHED_RETURN_DATA_SIZE.set(outs_len as u32);
        }

        let outs = partial_return_data_impl(self.offset, self.size, outs_len);
        match status {
            0 => Ok(outs),
            _ => Err(outs),
        }
    }
}

fn partial_return_data_impl(offset: usize, size: Option<usize>, full_size: usize) -> Vec<u8> {
    let mut offset = offset;
    if offset > full_size {
        offset = full_size;
    }
    let remaining_size = full_size - offset;
    let mut allocated_len = size.unwrap_or(remaining_size);
    if allocated_len > remaining_size {
        allocated_len = remaining_size;
    }
    let mut data = Vec::with_capacity(allocated_len);
    if allocated_len > 0 {
        unsafe {
            let written_size = hostio::read_return_data(data.as_mut_ptr(), offset, allocated_len);
            assert!(written_size <= allocated_len);
            data.set_len(written_size);
        }
    };

    data
}

#[derive(Clone, Default)]
#[must_use]
pub struct Deploy {
    salt: Option<Bytes32>,
    offset: usize,
    size: Option<usize>,
}

impl Deploy {
    pub fn new() -> Self {
        Default::default()
    }

    pub fn salt(mut self, salt: Bytes32) -> Self {
        self.salt = Some(salt);
        self
    }

    pub fn salt_option(mut self, salt: Option<Bytes32>) -> Self {
        self.salt = salt;
        self
    }

    pub fn limit_return_data(mut self, offset: usize, size: usize) -> Self {
        self.offset = offset;
        self.size = Some(size);
        self
    }

    pub fn skip_return_data(self) -> Self {
        self.limit_return_data(0, 0)
    }

    pub fn deploy(self, code: &[u8], endowment: Bytes32) -> Result<Bytes20, Vec<u8>> {
        let mut contract = [0; 20];
        let mut revert_data_len = 0;
        let contract = unsafe {
            if let Some(salt) = self.salt {
                hostio::create2(
                    code.as_ptr(),
                    code.len(),
                    endowment.ptr(),
                    salt.ptr(),
                    contract.as_mut_ptr(),
                    &mut revert_data_len as *mut _,
                );
            } else {
                hostio::create1(
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
            let revert_data = if revert_data_len == 0 {
                vec![]
            } else {
                partial_return_data_impl(self.offset, self.size, revert_data_len)
            };

            return Err(revert_data);
        }
        Ok(contract)
    }
}

pub fn address() -> Bytes20 {
    let mut data = [0; 20];
    unsafe { hostio::contract_address(data.as_mut_ptr()) };
    Bytes20(data)
}

pub fn balance() -> Bytes32 {
    addr::balance(address())
}
pub fn partial_return_data(offset: usize, size: usize) -> Vec<u8> {
    partial_return_data_impl(offset, Some(size), return_data_len())
}

fn return_data_len() -> usize {
    unsafe { hostio::CACHED_RETURN_DATA_SIZE.get() as usize }
}
