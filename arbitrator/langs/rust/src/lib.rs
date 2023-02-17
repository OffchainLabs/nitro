// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use std::array::TryFromSliceError;

#[link(wasm_import_module = "forward")]
extern "C" {
    pub fn read_args(dest: *mut u8);
    pub fn return_data(data: *const u8, len: usize);
}

pub fn args(len: usize) -> Vec<u8> {
    let mut input = Vec::with_capacity(len);
    unsafe {
        read_args(input.as_mut_ptr());
        input.set_len(len);
    }
    input
}

pub fn output(data: Vec<u8>) {
    unsafe {
        return_data(data.as_ptr(), data.len());
    }
}

#[macro_export]
macro_rules! arbitrum_main {
    ($name:expr) => {
        #[no_mangle]
        pub extern "C" fn arbitrum_main(len: usize) -> usize {
            let input = arbitrum::args(len);
            let (data, status) = match $name(input) {
                Ok(data) => (data, 0),
                Err(data) => (data, 1),
            };
            arbitrum::output(data);
            status
        }
    };
}

#[link(wasm_import_module = "forward")]
extern "C" {
    pub fn account_load_bytes32(key: *const u8, dest: *mut u8);
    pub fn account_store_bytes32(key: *const u8, value: *const u8);
}

#[derive(Default)]
#[repr(C)]
pub struct Bytes32(pub [u8; 32]);

impl Bytes32 {
    pub fn ptr(&self) -> *const u8 {
        self.0.as_ptr()
    }
}

pub fn load_bytes32(key: Bytes32) -> Bytes32 {
    let mut data = [0; 32];
    unsafe { account_load_bytes32(key.ptr(), data.as_mut_ptr()) };
    Bytes32(data)
}

pub fn store_bytes32(key: Bytes32, data: Bytes32) {
    unsafe { account_store_bytes32(key.ptr(), data.ptr()) };
}

impl Bytes32 {
    pub fn from_slice(data: &[u8]) -> Result<Self, TryFromSliceError> {
        Ok(Self(data.try_into()?))
    }
}
