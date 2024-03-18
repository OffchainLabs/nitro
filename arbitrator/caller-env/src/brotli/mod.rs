// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(clippy::too_many_arguments)]

use crate::{ExecEnv, GuestPtr, MemAccess};
use alloc::vec::Vec;
use brotli::{BrotliStatus, Dictionary};

/// Brotli compresses a go slice
///
/// The output buffer must be sufficiently large.
/// The pointers must not be null.
pub fn brotli_compress<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    _env: &mut E,
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
    out_len_ptr: GuestPtr,
    level: u32,
    window_size: u32,
    dictionary: Dictionary,
) -> BrotliStatus {
    let input = mem.read_slice(in_buf_ptr, in_buf_len as usize);
    let mut output = Vec::with_capacity(mem.read_u32(out_len_ptr) as usize);

    let status = brotli::compress_fixed(&input, &mut output, level, window_size, dictionary);
    match status {
        Ok(written) => unsafe {
            output.set_len(written);
            mem.write_slice(out_buf_ptr, &output[..written]);
            mem.write_u32(out_len_ptr, written as u32);
            BrotliStatus::Success
        },
        Err(status) => status,
    }
}

/// Brotli decompresses a go slice using a custom dictionary.
///
/// # Safety
///
/// The output buffer must be sufficiently large.
/// The pointers must not be null.
pub fn brotli_decompress<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    _env: &mut E,
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
    out_len_ptr: GuestPtr,
    dictionary: Dictionary,
) -> BrotliStatus {
    let input = mem.read_slice(in_buf_ptr, in_buf_len as usize);
    let mut output = Vec::with_capacity(mem.read_u32(out_len_ptr) as usize);

    let status = brotli::decompress_fixed(&input, &mut output, dictionary);
    match status {
        Ok(written) => unsafe {
            output.set_len(written);
            mem.write_slice(out_buf_ptr, &output[..written]);
            mem.write_u32(out_len_ptr, written as u32);
            BrotliStatus::Success
        },
        Err(status) => status,
    }
}
