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

fn brotli_stream_op(
    state: *mut EncoderState,
    input: Option<BrotliBuffer>,
    mut output: BrotliBuffer,
    op: BrotliEncoderOperation,
) -> usize {
    unsafe {
        let buffer = output.as_uninit();
        let input_slice = match &input {
            Some(buf) => buf.as_slice(),
            None => &[],
        };

        let mut in_len = input_slice.len();
        let mut in_ptr = input_slice.as_ptr();
        let mut out_left = buffer.len();
        let mut out_ptr = buffer.as_mut_ptr() as *mut u8;
        let mut out_len = out_left;

        let status = BrotliEncoderCompressStream(
            state,
            op,
            &mut in_len,
            &mut in_ptr,
            &mut out_left,
            &mut out_ptr,
            &mut out_len,
        );

        if status.is_err() {
            panic!("Error compressing stream");
        }
        out_len
    }
}

#[no_mangle]
pub extern "C" fn brotli_write_to_stream(
    state: *mut EncoderState,
    input: BrotliBuffer,
    output: BrotliBuffer,
) -> usize {
    brotli_stream_op(state, Some(input), output, BrotliEncoderOperation::Process)
}

#[no_mangle]
pub extern "C" fn brotli_flush_stream(state: *mut EncoderState, output: BrotliBuffer) {
    brotli_stream_op(state, None, output, BrotliEncoderOperation::Flush);
}

#[no_mangle]
pub extern "C" fn brotli_close_stream(state: *mut EncoderState, output: BrotliBuffer) {
    brotli_stream_op(state, None, output, BrotliEncoderOperation::Finish);
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
