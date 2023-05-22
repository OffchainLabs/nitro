// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

pub use util::{Bytes20, Bytes32};

pub mod address;
pub mod block;
pub mod contract;
pub mod debug;
pub mod evm;
pub mod msg;
pub mod tx;
mod util;

#[link(wasm_import_module = "forward")]
extern "C" {
    pub(crate) fn read_args(dest: *mut u8);
    pub(crate) fn return_data(data: *const u8, len: usize);
    pub fn memory_grow(pages: u32);
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
        /// Force the compiler to import these symbols
        /// Note: calling these functions will unproductively consume gas
        #[no_mangle]
        pub unsafe fn mark_used() {
            arbitrum::memory_grow(0);
            panic!();
        }

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
    pub(crate) fn account_load_bytes32(key: *const u8, dest: *mut u8);
    pub(crate) fn account_store_bytes32(key: *const u8, value: *const u8);
}

pub fn load_bytes32(key: Bytes32) -> Bytes32 {
    let mut data = [0; 32];
    unsafe { account_load_bytes32(key.ptr(), data.as_mut_ptr()) };
    Bytes32(data)
}

pub fn store_bytes32(key: Bytes32, data: Bytes32) {
    unsafe { account_store_bytes32(key.ptr(), data.ptr()) };
}
