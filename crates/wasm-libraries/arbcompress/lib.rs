// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![allow(clippy::missing_safety_doc)] // TODO: add safety docs

use brotli::{BrotliStatus, Dictionary};
use caller_env::{
    self, GuestPtr,
    static_caller::StaticMem
};

#[unsafe(no_mangle)]
pub unsafe extern "C" fn arbcompress__brotli_compress(
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
    out_len_ptr: GuestPtr,
    level: u32,
    window_size: u32,
    dictionary: Dictionary,
) -> BrotliStatus {
    caller_env::brotli::brotli_compress(
        &mut StaticMem,
        in_buf_ptr,
        in_buf_len,
        out_buf_ptr,
        out_len_ptr,
        level,
        window_size,
        dictionary,
    )
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn arbcompress__brotli_decompress(
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
    out_len_ptr: GuestPtr,
    dictionary: Dictionary,
) -> BrotliStatus {
    caller_env::brotli::brotli_decompress(
        &mut StaticMem,
        in_buf_ptr,
        in_buf_len,
        out_buf_ptr,
        out_len_ptr,
        dictionary,
    )
}
