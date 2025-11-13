// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![allow(clippy::missing_safety_doc)]

use caller_env::{self, GuestPtr};

#[no_mangle]
pub unsafe extern "C" fn arbkeccak__keccak256(in_buf_ptr: GuestPtr, in_buf_len: u32, out_buf_ptr: GuestPtr)
{
    caller_env::arbkeccak::keccak256(
        &mut caller_env::static_caller::STATIC_MEM,
        &mut caller_env::static_caller::STATIC_ENV,
        in_buf_ptr, in_buf_len, out_buf_ptr
    )
}
