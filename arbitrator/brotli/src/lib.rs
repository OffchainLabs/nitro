// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![cfg_attr(target_arch = "wasm32", no_std)]

extern crate alloc;

#[cfg(target_arch = "wasm32")]
use alloc::vec::Vec;

use core::{
    ffi::c_void,
    mem::{self, MaybeUninit},
    ptr,
};

pub mod cgo;
mod dicts;
mod types;

#[cfg(feature = "wasmer_traits")]
mod wasmer_traits;

pub use dicts::Dictionary;
use types::*;
pub use types::{BrotliStatus, DEFAULT_WINDOW_SIZE};

type DecoderState = c_void;
type EncoderState = c_void;
type EncoderPreparedDictionary = c_void;
type CustomAllocator = c_void;
type HeapItem = c_void;

// compression API
extern "C" {
    fn BrotliEncoderCreateInstance(
        alloc: Option<extern "C" fn(opaque: *const CustomAllocator, size: usize) -> *mut HeapItem>,
        free: Option<extern "C" fn(opaque: *const CustomAllocator, address: *mut HeapItem)>,
        opaque: *mut CustomAllocator,
    ) -> *mut EncoderState;

    /// Quality must be at least 2 for this bound to be correct.
    fn BrotliEncoderMaxCompressedSize(input_size: usize) -> usize;

    fn BrotliEncoderSetParameter(
        state: *mut EncoderState,
        param: BrotliEncoderParameter,
        value: u32,
    ) -> BrotliBool;

    fn BrotliEncoderAttachPreparedDictionary(
        state: *mut EncoderState,
        dictionary: *const EncoderPreparedDictionary,
    ) -> BrotliBool;

    fn BrotliEncoderCompressStream(
        state: *mut EncoderState,
        op: BrotliEncoderOperation,
        input_len: *mut usize,
        input_ptr: *mut *const u8,
        out_left: *mut usize,
        out_ptr: *mut *mut u8,
        out_len: *mut usize,
    ) -> BrotliBool;

    fn BrotliEncoderIsFinished(state: *mut EncoderState) -> BrotliBool;

    fn BrotliEncoderDestroyInstance(state: *mut EncoderState);
}

// decompression API
extern "C" {
    fn BrotliDecoderCreateInstance(
        alloc: Option<extern "C" fn(opaque: *const CustomAllocator, size: usize) -> *mut HeapItem>,
        free: Option<extern "C" fn(opaque: *const CustomAllocator, address: *mut HeapItem)>,
        opaque: *mut CustomAllocator,
    ) -> *mut DecoderState;

    fn BrotliDecoderAttachDictionary(
        state: *mut DecoderState,
        kind: BrotliSharedDictionaryType,
        len: usize,
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

/// Determines the maximum size a brotli compression could be.
/// Note: assumes the user never calls "flush" except during "finish" at the end.
pub fn compression_bound(len: usize, level: u32) -> usize {
    let mut bound = unsafe { BrotliEncoderMaxCompressedSize(len) };
    if level <= 2 {
        bound = bound.max(len + (len >> 10) * 8 + 64);
    }
    bound
}

/// Brotli compresses a slice into a vec.
pub fn compress(
    input: &[u8],
    level: u32,
    window_size: u32,
    dictionary: Dictionary,
) -> Result<Vec<u8>, BrotliStatus> {
    compress_into(input, Vec::new(), level, window_size, dictionary)
}

/// Brotli compresses a slice, extending the `output` specified.
pub fn compress_into(
    input: &[u8],
    mut output: Vec<u8>,
    level: u32,
    window_size: u32,
    dictionary: Dictionary,
) -> Result<Vec<u8>, BrotliStatus> {
    let max_size = compression_bound(input.len(), level);
    let needed = max_size.saturating_sub(output.spare_capacity_mut().len());
    output.reserve_exact(needed);

    let space = output.spare_capacity_mut();
    let count = compress_fixed(input, space, level, window_size, dictionary)?.len();
    unsafe { output.set_len(output.len() + count) }
    Ok(output)
}

/// Brotli compresses a slice into a buffer of limited capacity.
pub fn compress_fixed<'a>(
    input: &'a [u8],
    output: &'a mut [MaybeUninit<u8>],
    level: u32,
    window_size: u32,
    dictionary: Dictionary,
) -> Result<&'a [u8], BrotliStatus> {
    unsafe {
        let state = BrotliEncoderCreateInstance(None, None, ptr::null_mut());

        macro_rules! check {
            ($ret:expr) => {
                if $ret.is_err() {
                    BrotliEncoderDestroyInstance(state);
                    return Err(BrotliStatus::Failure);
                }
            };
        }

        check!(BrotliEncoderSetParameter(
            state,
            BrotliEncoderParameter::Quality,
            level
        ));
        check!(BrotliEncoderSetParameter(
            state,
            BrotliEncoderParameter::WindowSize,
            window_size
        ));

        if let Some(dict) = dictionary.ptr(level) {
            check!(BrotliEncoderAttachPreparedDictionary(state, dict));
        }

        let mut in_len = input.len();
        let mut in_ptr = input.as_ptr();
        let mut out_left = output.len();
        let mut out_ptr = output.as_mut_ptr() as *mut u8;
        let mut out_len = out_left;

        let status = BrotliEncoderCompressStream(
            state,
            BrotliEncoderOperation::Finish,
            &mut in_len as _,
            &mut in_ptr as _,
            &mut out_left as _,
            &mut out_ptr as _,
            &mut out_len as _,
        );
        check!(status);
        check!(BrotliEncoderIsFinished(state));
        BrotliEncoderDestroyInstance(state);

        // SAFETY: brotli initialized this span of bytes
        let output = mem::transmute(&output[..out_len]);
        Ok(output)
    }
}

/// Brotli compresses a slice into a buffer of limited capacity.
pub fn decompress(input: &[u8], dictionary: Dictionary) -> Result<Vec<u8>, BrotliStatus> {
    unsafe {
        let state = BrotliDecoderCreateInstance(None, None, ptr::null_mut());
        let mut output: Vec<u8> = Vec::with_capacity(4 * input.len());

        macro_rules! check {
            ($ret:expr) => {
                if $ret.is_err() {
                    BrotliDecoderDestroyInstance(state);
                    return Err(BrotliStatus::Failure);
                }
            };
        }

        // TODO: consider window and quality check?
        // TODO: fuzz
        if let Some(dict) = dictionary.slice() {
            let attatched = BrotliDecoderAttachDictionary(
                state,
                BrotliSharedDictionaryType::Raw,
                dict.len(),
                dict.as_ptr(),
            );
            check!(attatched);
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
            check!(status);
            check!(BrotliDecoderIsFinished(state));
            break;
        }

        BrotliDecoderDestroyInstance(state);
        Ok(output)
    }
}

/// Brotli decompresses a slice into
pub fn decompress_fixed<'a>(
    input: &'a [u8],
    output: &'a mut [MaybeUninit<u8>],
    dictionary: Dictionary,
) -> Result<&'a [u8], BrotliStatus> {
    unsafe {
        let state = BrotliDecoderCreateInstance(None, None, ptr::null_mut());

        macro_rules! check {
            ($cond:expr) => {
                if !$cond {
                    BrotliDecoderDestroyInstance(state);
                    return Err(BrotliStatus::Failure);
                }
            };
        }

        if let Some(dict) = dictionary.slice() {
            let attatched = BrotliDecoderAttachDictionary(
                state,
                BrotliSharedDictionaryType::Raw,
                dict.len(),
                dict.as_ptr(),
            );
            check!(attatched == BrotliBool::True);
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
        check!(status == BrotliStatus::Success);
        check!(BrotliDecoderIsFinished(state) == BrotliBool::True);
        BrotliDecoderDestroyInstance(state);

        // SAFETY: brotli initialized this span of bytes
        let output = mem::transmute(&output[..out_len]);
        Ok(output)
    }
}
