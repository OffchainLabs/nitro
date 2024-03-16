// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(clippy::too_many_arguments)]

use crate::{ExecEnv, GuestPtr, MemAccess};
use alloc::vec::Vec;
use core::{ffi::c_void, ptr};

mod types;

pub use types::*;

type DecoderState = c_void;
type CustomAllocator = c_void;
type HeapItem = c_void;

// one-shot brotli API
extern "C" {
    fn BrotliEncoderCompress(
        quality: u32,
        lgwin: u32,
        mode: u32,
        input_size: usize,
        input_buffer: *const u8,
        encoded_size: *mut usize,
        encoded_buffer: *mut u8,
    ) -> BrotliStatus;
}

// custom dictionary API
extern "C" {
    fn BrotliDecoderCreateInstance(
        alloc: Option<extern "C" fn(opaque: *const CustomAllocator, size: usize) -> *mut HeapItem>,
        free: Option<extern "C" fn(opaque: *const CustomAllocator, address: *mut HeapItem)>,
        opaque: *mut CustomAllocator,
    ) -> *mut DecoderState;

    fn BrotliDecoderAttachDictionary(
        state: *mut DecoderState,
        dict_type: BrotliSharedDictionaryType,
        dict_len: usize,
        dictionary: *const u8,
    ) -> BrotliBool;

    fn BrotliDecoderDecompressStream(
        state: *mut DecoderState,
        input_len: *mut usize,
        input_ptr: *mut *const u8,
        out_left: *mut usize,
        out_ptr: *mut *mut u8,
        out_len: *mut usize,
    ) -> BrotliStatus;

    fn BrotliDecoderDestroyInstance(state: *mut DecoderState);
}

const BROTLI_MODE_GENERIC: u32 = 0;

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
    let prior_out_len = mem.read_u32(out_len_ptr) as usize;

    let mut output = Vec::with_capacity(prior_out_len);
    let mut out_len = prior_out_len;
    unsafe {
        let state = BrotliDecoderCreateInstance(None, None, ptr::null_mut());

        macro_rules! require {
            ($cond:expr) => {
                if !$cond {
                    BrotliDecoderDestroyInstance(state);
                    return BrotliStatus::Failure;
                }
            };
        }

        if dictionary != Dictionary::Empty {
            let attatched = BrotliDecoderAttachDictionary(
                state,
                BrotliSharedDictionaryType::Raw,
                dictionary.len(),
                dictionary.data(),
            );
            require!(attatched == BrotliBool::True);
        }

        let mut in_len = input.len();
        let mut in_ptr = input.as_ptr();
        let mut out_left = prior_out_len;
        let mut out_ptr = output.as_mut_ptr();

        let status = BrotliDecoderDecompressStream(
            state,
            &mut in_len as _,
            &mut in_ptr as _,
            &mut out_left as _,
            &mut out_ptr as _,
            &mut out_len as _,
        );
        require!(status == BrotliStatus::Success && out_len <= prior_out_len);

        BrotliDecoderDestroyInstance(state);
        output.set_len(out_len);
    }
    mem.write_slice(out_buf_ptr, &output[..out_len]);
    mem.write_u32(out_len_ptr, out_len as u32);
    BrotliStatus::Success
}

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
    let in_slice = mem.read_slice(in_buf_ptr, in_buf_len as usize);
    let orig_output_len = mem.read_u32(out_len_ptr) as usize;
    let mut output = Vec::with_capacity(orig_output_len);
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
        output.set_len(output_len);
    }
    mem.write_slice(out_buf_ptr, &output[..output_len]);
    mem.write_u32(out_len_ptr, output_len as u32);
    BrotliStatus::Success
}
