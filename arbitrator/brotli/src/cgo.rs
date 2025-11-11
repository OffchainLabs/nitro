// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::types::{BrotliEncoderOperation, BrotliEncoderParameter};
use crate::{
    BrotliEncoderCompressStream, BrotliEncoderCreateInstance, BrotliEncoderDestroyInstance,
    BrotliEncoderSetParameter, BrotliStatus, Dictionary, EncoderState, DEFAULT_WINDOW_SIZE,
};
use core::{mem::MaybeUninit, slice};
use std::ptr;
use BrotliEncoderParameter::*;

/// Mechanism for passing data between Go and Rust where Rust can specify the initialized length.
#[derive(Clone, Copy)]
#[repr(C)]
pub struct BrotliBuffer {
    /// Points to data owned by Go.
    ptr: *mut u8,
    /// The length in bytes. Rust may mutate this value to indicate the number of bytes initialized.
    len: *mut usize,
}

impl BrotliBuffer {
    /// Interprets the underlying Go data as a Rust slice.
    fn as_slice(&self) -> &[u8] {
        let len = unsafe { *self.len };
        if len == 0 {
            return &[];
        }
        unsafe { slice::from_raw_parts(self.ptr, len) }
    }

    /// Interprets the underlying Go data as a Rust slice of uninitialized data.
    fn as_uninit(&mut self) -> &mut [MaybeUninit<u8>] {
        let len = unsafe { *self.len };
        if len == 0 {
            return &mut [];
        }
        unsafe { slice::from_raw_parts_mut(self.ptr as _, len) }
    }
}

/// Brotli compresses the given Go data into a buffer of limited capacity.
#[no_mangle]
pub extern "C" fn brotli_compress(
    input: BrotliBuffer,
    mut output: BrotliBuffer,
    dictionary: Dictionary,
    level: u32,
) -> BrotliStatus {
    let window = DEFAULT_WINDOW_SIZE;
    let buffer = output.as_uninit();
    match crate::compress_fixed(input.as_slice(), buffer, level, window, dictionary) {
        Ok(slice) => unsafe { *output.len = slice.len() },
        Err(status) => return status,
    }
    BrotliStatus::Success
}

#[no_mangle]
pub extern "C" fn brotli_create_compressing_writer(level: u32) -> *mut EncoderState {
    unsafe {
        let state = BrotliEncoderCreateInstance(None, None, ptr::null_mut());

        if BrotliEncoderSetParameter(state, Quality, level).is_err() {
            panic!("Aaaaaa qualityyyy")
        }
        if BrotliEncoderSetParameter(state, WindowSize, DEFAULT_WINDOW_SIZE).is_err() {
            panic!("Aaaaaa windowwwww")
        }

        state
    }
}

#[no_mangle]
pub extern "C" fn brotli_write_to_stream(
    state: *mut EncoderState,
    input: BrotliBuffer,
    mut output: BrotliBuffer,
) -> usize {
    unsafe {
        let buffer = output.as_uninit();
        let input = input.as_slice();

        let mut in_len = input.len();
        let mut in_ptr = input.as_ptr();
        let mut out_left = buffer.len();
        let mut out_ptr = buffer.as_mut_ptr() as *mut u8;
        let mut out_len = out_left;

        let status = BrotliEncoderCompressStream(
            state,
            BrotliEncoderOperation::Process,
            &mut in_len as _,
            &mut in_ptr as _,
            &mut out_left as _,
            &mut out_ptr as _,
            &mut out_len as _,
        );

        if status.is_err() {
            panic!("Error compressing stream");
        }
        out_len
    }
}

#[no_mangle]
pub extern "C" fn brotli_flush_stream(state: *mut EncoderState, mut output: BrotliBuffer) {
    unsafe {
        let buffer = output.as_uninit();
        let input = &[];

        let mut in_len = input.len();
        let mut in_ptr = input.as_ptr();
        let mut out_left = buffer.len();
        let mut out_ptr = buffer.as_mut_ptr() as *mut u8;
        let mut out_len = out_left;

        let status = BrotliEncoderCompressStream(
            state,
            BrotliEncoderOperation::Flush,
            &mut in_len as _,
            &mut in_ptr as _,
            &mut out_left as _,
            &mut out_ptr as _,
            &mut out_len as _,
        );

        if status.is_err() {
            panic!("Error compressing stream");
        }
    }
}

#[no_mangle]
pub extern "C" fn brotli_close_stream(state: *mut EncoderState, mut output: BrotliBuffer) {
    unsafe {
        let buffer = output.as_uninit();
        let input = &[];

        let mut in_len = input.len();
        let mut in_ptr = input.as_ptr();
        let mut out_left = buffer.len();
        let mut out_ptr = buffer.as_mut_ptr() as *mut u8;
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

        if status.is_err() {
            panic!("Error compressing stream");
        }
    }
}

/// Brotli decompresses the given Go data into a buffer of limited capacity.
#[no_mangle]
pub extern "C" fn brotli_decompress(
    input: BrotliBuffer,
    mut output: BrotliBuffer,
    dictionary: Dictionary,
) -> BrotliStatus {
    match crate::decompress_fixed(input.as_slice(), output.as_uninit(), dictionary) {
        Ok(slice) => unsafe { *output.len = slice.len() },
        Err(status) => return status,
    }
    BrotliStatus::Success
}
