// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![allow(clippy::missing_safety_doc)]

use caller_env::{self, GuestPtr};

#[no_mangle]
pub unsafe extern "C" fn arbcrypto__ecrecovery(
    hash_ptr: GuestPtr,
    hash_len: u32,
    sig_ptr: GuestPtr,
    sig_len: u32,
    pub_ptr: GuestPtr,
) -> u32 {
    caller_env::arbcrypto::ecrecovery(
        &mut caller_env::static_caller::STATIC_MEM,
        &mut caller_env::static_caller::STATIC_ENV,
        hash_ptr,
        hash_len,
        sig_ptr,
        sig_len,
        pub_ptr,
    )
}

#[no_mangle]
pub unsafe extern "C" fn arbcrypto__keccak256(
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
) {
    caller_env::arbcrypto::keccak256(
        &mut caller_env::static_caller::STATIC_MEM,
        &mut caller_env::static_caller::STATIC_ENV,
        in_buf_ptr,
        in_buf_len,
        out_buf_ptr,
    )
}
