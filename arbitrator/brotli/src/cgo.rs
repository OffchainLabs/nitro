// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{BrotliStatus, Dictionary, DEFAULT_WINDOW_SIZE};
use core::{mem::MaybeUninit, slice};

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
