// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{gostack::GoStack, machine::WasmEnvMut};

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

#[derive(PartialEq)]
#[repr(u32)]
pub enum BrotliStatus {
    Failure,
    Success,
}

/// go side: λ(inBuf []byte, outBuf []byte, level, windowSize uint64) (outLen uint64, status BrotliStatus)
pub fn brotli_compress(mut env: WasmEnvMut, sp: u32) {
    let mut sp = GoStack::simple(sp, &mut env);
    let (in_buf_ptr, in_buf_len) = sp.read_go_slice();
    let (out_buf_ptr, out_buf_len) = sp.read_go_slice();
    let level = sp.read_u32();
    let windowsize = sp.read_u32();

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

    if (res != BrotliStatus::Success) || (output_len as u64 > out_buf_len) {
        sp.skip_u64();
        sp.write_u32(BrotliStatus::Failure as _);
        return;
    }
    sp.write_slice(out_buf_ptr, &output[..output_len]);
    sp.write_u64(output_len as u64);
    sp.write_u32(BrotliStatus::Success as _);
}

/// go side: λ(inBuf []byte, outBuf []byte) (outLen uint64, status BrotliStatus)
pub fn brotli_decompress(mut env: WasmEnvMut, sp: u32) {
    let mut sp = GoStack::simple(sp, &mut env);
    let (in_buf_ptr, in_buf_len) = sp.read_go_slice();
    let (out_buf_ptr, out_buf_len) = sp.read_go_slice();

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

    if (res != BrotliStatus::Success) || (output_len as u64 > out_buf_len) {
        sp.skip_u64();
        sp.write_u32(BrotliStatus::Failure as _);
        return;
    }
    sp.write_slice(out_buf_ptr, &output[..output_len]);
    sp.write_u64(output_len as u64);
    sp.write_u32(BrotliStatus::Success as _);
}
