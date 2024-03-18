// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{BrotliStatus, Dictionary, DEFAULT_WINDOW_SIZE};

#[derive(Clone, Copy)]
#[repr(C)]
pub struct BrotliBuffer {
    /// Points to data owned by Go.
    ptr: *mut u8,
    /// The length in bytes. 
    len: *mut usize,
}

impl BrotliBuffer {
    fn as_slice(&self) -> &[u8] {
        let len = unsafe { *self.len };
        if len == 0 {
            return &[];
        }
        unsafe { std::slice::from_raw_parts(self.ptr, len) }
    }

    fn as_mut_slice(&mut self) -> &mut [u8] {
        let len = unsafe { *self.len };
        if len == 0 {
            return &mut [];
        }
        unsafe { std::slice::from_raw_parts_mut(self.ptr, len) }
    }
}

#[no_mangle]
pub extern "C" fn brotli_compress(
    input: BrotliBuffer,
    mut output: BrotliBuffer,
    dictionary: Dictionary,
    level: u32,
) -> BrotliStatus {
    let window = DEFAULT_WINDOW_SIZE;
    let buffer = output.as_mut_slice();
    match crate::compress_fixed(input.as_slice(), buffer, level, window, dictionary) {
        Ok(written) => unsafe { *output.len = written },
        Err(status) => return status,
    }
    BrotliStatus::Success
}

#[no_mangle]
pub extern "C" fn brotli_decompress(
    input: BrotliBuffer,
    mut output: BrotliBuffer,
    dictionary: Dictionary,
) -> BrotliStatus {
    match crate::decompress_fixed(input.as_slice(), output.as_mut_slice(), dictionary) {
        Ok(written) => unsafe { *output.len = written },
        Err(status) => return status,
    }
    BrotliStatus::Success
}
