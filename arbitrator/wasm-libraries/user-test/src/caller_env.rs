// Copyright 2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use arbutil::Bytes32;
use caller_env::{static_caller::STATIC_MEM, GuestPtr, MemAccess};

pub struct UserMem;

impl UserMem {
    pub fn read_bytes32(ptr: GuestPtr) -> Bytes32 {
        unsafe { STATIC_MEM.read_fixed(ptr).into() }
    }

    pub fn read_slice(ptr: GuestPtr, len: u32) -> Vec<u8> {
        unsafe { STATIC_MEM.read_slice(ptr, len as usize) }
    }

    pub fn write_slice(ptr: GuestPtr, src: &[u8]) {
        unsafe { STATIC_MEM.write_slice(ptr, src) }
    }
}
