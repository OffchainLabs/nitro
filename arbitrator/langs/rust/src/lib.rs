// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

pub use util::{Bytes20, Bytes32};

pub mod address;
pub mod block;
pub mod contract;
pub mod debug;
pub mod evm;
mod hostio;
pub mod msg;
pub mod tx;
mod util;

pub fn args(len: usize) -> Vec<u8> {
    let mut input = Vec::with_capacity(len);
    unsafe {
        hostio::read_args(input.as_mut_ptr());
        input.set_len(len);
    }
    input
}

pub fn output(data: Vec<u8>) {
    unsafe {
        hostio::return_data(data.as_ptr(), data.len());
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

pub fn load_bytes32(key: Bytes32) -> Bytes32 {
    let mut data = [0; 32];
    unsafe { hostio::account_load_bytes32(key.ptr(), data.as_mut_ptr()) };
    Bytes32(data)
}

pub fn store_bytes32(key: Bytes32, data: Bytes32) {
    unsafe { hostio::account_store_bytes32(key.ptr(), data.ptr()) };
}
