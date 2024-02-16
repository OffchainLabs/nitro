// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::{gostack::GoStack, machine::WasmEnvMut};

extern "C" {
    pub fn BrotliDecoderDecompress(
        encoded_size: usize,
        encoded_buffer: *const u8,
        decoded_size: *mut usize,
        decoded_buffer: *mut u8,
    ) -> u32;

    pub fn BrotliEncoderCompress(
        quality: u32,
        lgwin: u32,
        mode: u32,
        input_size: usize,
        input_buffer: *const u8,
        encoded_size: *mut usize,
        encoded_buffer: *mut u8,
    ) -> u32;
}

const BROTLI_MODE_GENERIC: u32 = 0;
const BROTLI_RES_SUCCESS: u32 = 1;

pub fn brotli_compress(mut env: WasmEnvMut, sp: u32) {
    let (sp, _) = GoStack::new(sp, &mut env);

    //(inBuf []byte, outBuf []byte, level int, windowSize int) int
    let in_buf_ptr = sp.read_u64(0);
    let in_buf_len = sp.read_u64(1);
    let out_buf_ptr = sp.read_u64(3);
    let out_buf_len = sp.read_u64(4);
    let level = sp.read_u64(6) as u32;
    let windowsize = sp.read_u64(7) as u32;
    let output_arg = 8;

    let in_slice = sp.read_slice(in_buf_ptr, in_buf_len);
    let mut output = vec![0u8; out_buf_len as usize];
    let mut output_len = out_buf_len as usize;

    let res = unsafe {
        BrotliEncoderCompress(
            level,
            windowsize,
            BROTLI_MODE_GENERIC,
            in_buf_len as usize,
            in_slice.as_ptr(),
            &mut output_len,
            output.as_mut_ptr(),
        )
    };

    if (res != BROTLI_RES_SUCCESS) || (output_len as u64 > out_buf_len) {
        sp.write_u64(output_arg, u64::MAX);
        return;
    }
    sp.write_slice(out_buf_ptr, &output[..output_len]);
    sp.write_u64(output_arg, output_len as u64);
}

pub fn brotli_decompress(mut env: WasmEnvMut, sp: u32) {
    let (sp, _) = GoStack::new(sp, &mut env);

    //(inBuf []byte, outBuf []byte) int
    let in_buf_ptr = sp.read_u64(0);
    let in_buf_len = sp.read_u64(1);
    let out_buf_ptr = sp.read_u64(3);
    let out_buf_len = sp.read_u64(4);
    let output_arg = 6;

    let in_slice = sp.read_slice(in_buf_ptr, in_buf_len);
    let mut output = vec![0u8; out_buf_len as usize];
    let mut output_len = out_buf_len as usize;

    let res = unsafe {
        BrotliDecoderDecompress(
            in_buf_len as usize,
            in_slice.as_ptr(),
            &mut output_len,
            output.as_mut_ptr(),
        )
    };

    if (res != BROTLI_RES_SUCCESS) || (output_len as u64 > out_buf_len) {
        sp.write_u64(output_arg, u64::MAX);
        return;
    }
    sp.write_slice(out_buf_ptr, &output[..output_len]);
    sp.write_u64(output_arg, output_len as u64);
}
