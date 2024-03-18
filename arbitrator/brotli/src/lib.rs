// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![cfg_attr(target_arch = "wasm32", no_std)]

extern crate alloc;

use alloc::vec::Vec;
use core::{
    ffi::c_void,
    mem::{self, MaybeUninit},
    ptr,
};

pub mod cgo;
mod types;

#[cfg(feature = "wasmer_traits")]
mod wasmer_traits;

use types::*;
pub use types::{BrotliStatus, Dictionary, DEFAULT_WINDOW_SIZE};

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

    fn BrotliDecoderIsFinished(state: *const DecoderState) -> BrotliBool;

    fn BrotliDecoderDestroyInstance(state: *mut DecoderState);
}

/// Brotli compresses a slice into a vec, growing as needed.
/// The output buffer must be sufficiently large.
pub fn compress(input: &[u8], output: &mut Vec<u8>, level: u32, window_size: u32) -> BrotliStatus {
    let mut output_len = output.capacity();
    unsafe {
        let res = BrotliEncoderCompress(
            level,
            window_size,
            BROTLI_MODE_GENERIC,
            input.len(),
            input.as_ptr(),
            &mut output_len,
            output.as_mut_ptr(),
        );
        if res != BrotliStatus::Success {
            return BrotliStatus::Failure;
        }
        output.set_len(output_len);
    }
    BrotliStatus::Success
}

/// Brotli compresses a slice.
/// The output buffer must be sufficiently large.
pub fn compress_fixed<'a>(
    input: &'a [u8],
    output: &'a mut [MaybeUninit<u8>],
    level: u32,
    window_size: u32,
    dictionary: Dictionary,
) -> Result<&'a [u8], BrotliStatus> {
    let mut out_len = output.len();
    unsafe {
        let res = BrotliEncoderCompress(
            level,
            window_size,
            BROTLI_MODE_GENERIC,
            input.len(),
            input.as_ptr(),
            &mut out_len,
            output.as_mut_ptr() as *mut u8,
        );
        if res != BrotliStatus::Success {
            return Err(BrotliStatus::Failure);
        }
    }

    // SAFETY: brotli initialized this span of bytes
    let output = unsafe { mem::transmute(&output[..out_len]) };
    Ok(output)
}

/// Brotli decompresses a slice into a vec, growing as needed.
pub fn decompress(input: &[u8], output: &mut Vec<u8>, dictionary: Dictionary) -> BrotliStatus {
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
        let mut out_left = output.capacity();
        let mut out_ptr = output.as_mut_ptr();
        let mut out_len = out_left;

        loop {
            let status = BrotliDecoderDecompressStream(
                state,
                &mut in_len as _,
                &mut in_ptr as _,
                &mut out_left as _,
                &mut out_ptr as _,
                &mut out_len as _,
            );
            output.set_len(out_len);

            if status == BrotliStatus::NeedsMoreOutput {
                output.reserve(24 * 1024);
                out_ptr = output.as_mut_ptr().add(out_len);
                out_left = output.capacity() - out_len;
                continue;
            }
            require!(status == BrotliStatus::Success);
            require!(BrotliDecoderIsFinished(state) == BrotliBool::True);
            break;
        }

        BrotliDecoderDestroyInstance(state);
    }
    BrotliStatus::Success
}

/// Brotli decompresses a slice, returning the number of bytes written.
pub fn decompress_fixed<'a>(
    input: &'a [u8],
    output: &'a mut [MaybeUninit<u8>],
    dictionary: Dictionary,
) -> Result<&'a [u8], BrotliStatus> {
    unsafe {
        let state = BrotliDecoderCreateInstance(None, None, ptr::null_mut());

        macro_rules! require {
            ($cond:expr) => {
                if !$cond {
                    BrotliDecoderDestroyInstance(state);
                    return Err(BrotliStatus::Failure);
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
        let mut out_left = output.len();
        let mut out_ptr = output.as_mut_ptr() as *mut u8;
        let mut out_len = out_left;

        let status = BrotliDecoderDecompressStream(
            state,
            &mut in_len as _,
            &mut in_ptr as _,
            &mut out_left as _,
            &mut out_ptr as _,
            &mut out_len as _,
        );
        require!(status == BrotliStatus::Success);
        require!(BrotliDecoderIsFinished(state) == BrotliBool::True);
        BrotliDecoderDestroyInstance(state);

        // SAFETY: brotli initialized this span of bytes
        let output = mem::transmute(&output[..out_len]);
        Ok(output)
    }
}
