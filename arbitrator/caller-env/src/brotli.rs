// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![allow(clippy::too_many_arguments)]

use crate::{ExecEnv, GuestPtr, MemAccess};
use alloc::vec;
use num_enum::{IntoPrimitive, TryFromPrimitive};

#[derive(PartialEq, IntoPrimitive, TryFromPrimitive)]
#[repr(u32)]
pub enum BrotliStatus {
    Failure,
    Success,
}

extern "C" {
    pub fn BrotliDecoderDecompress(
        encoded_size: usize,
        encoded_buffer: *const u8,
        decoded_size: *mut usize,
        decoded_buffer: *mut u8,
    ) -> BrotliStatus;

    pub fn BrotliEncoderCompress(
        quality: u32,
        lgwin: u32,
        mode: u32,
        input_size: usize,
        input_buffer: *const u8,
        encoded_size: *mut usize,
        encoded_buffer: *mut u8,
    ) -> BrotliStatus;
}

const BROTLI_MODE_GENERIC: u32 = 0;

/// Brotli decompresses a go slice.
///
/// # Safety
///
/// The output buffer must be sufficiently large enough.
pub fn brotli_decompress<M: MemAccess, E: ExecEnv>(
    mem: &mut M,
    _env: &mut E,
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
    out_len_ptr: GuestPtr,
) -> BrotliStatus {
    let in_slice = mem.read_slice(in_buf_ptr, in_buf_len as usize);
    let orig_output_len = mem.read_u32(out_len_ptr) as usize;
    let mut output = vec![0; orig_output_len];
    let mut output_len = orig_output_len;
    unsafe {
        let res = BrotliDecoderDecompress(
            in_buf_len as usize,
            in_slice.as_ptr(),
            &mut output_len,
            output.as_mut_ptr(),
        );
        if (res != BrotliStatus::Success) || (output_len > orig_output_len) {
            return BrotliStatus::Failure;
        }
    }
    mem.write_slice(out_buf_ptr, &output[..output_len]);
    mem.write_u32(out_len_ptr, output_len as u32);
    BrotliStatus::Success
}

/// Brotli compresses a go slice
///
/// The output buffer must be large enough.
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
    let in_slice = mem.read_slice(in_buf_ptr, in_buf_len as usize);
    let orig_output_len = mem.read_u32(out_len_ptr) as usize;
    let mut output = vec![0; orig_output_len];
    let mut output_len = orig_output_len;

    unsafe {
        let res = BrotliEncoderCompress(
            level,
            window_size,
            BROTLI_MODE_GENERIC,
            in_buf_len as usize,
            in_slice.as_ptr(),
            &mut output_len,
            output.as_mut_ptr(),
        );
        if (res != BrotliStatus::Success) || (output_len > orig_output_len) {
            return BrotliStatus::Failure;
        }
    }
    mem.write_slice(out_buf_ptr, &output[..output_len]);
    mem.write_u32(out_len_ptr, output_len as u32);
    BrotliStatus::Success
}
