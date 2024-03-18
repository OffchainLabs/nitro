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
) -> BrotliStatus {
    let input = mem.read_slice(in_buf_ptr, in_buf_len as usize);
    let mut output = Vec::with_capacity(mem.read_u32(out_len_ptr) as usize);

    let status = brotli::compress(&input, &mut output, level, window_size);
    if status.is_ok() {
        let out_len = output.len();
        mem.write_slice(out_buf_ptr, &output[..out_len]);
        mem.write_u32(out_len_ptr, out_len as u32);
    }
    status
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

    let status = brotli::decompress(&input, &mut output, dictionary, false);
    if status.is_ok() {
        let out_len = output.len();
        mem.write_slice(out_buf_ptr, &output[..out_len]);
        mem.write_u32(out_len_ptr, out_len as u32);
    }
    status
}
