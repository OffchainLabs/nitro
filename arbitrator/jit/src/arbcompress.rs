// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::callerenv::JitCallerEnv;
use crate::machine::Escape;
use crate::machine::WasmEnvMut;
use callerenv::CallerEnv;

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

type Uptr = u32;

/// Brotli decompresses a go slice
///
/// # Safety
///
/// The output buffer must be sufficiently large enough.
pub fn brotli_decompress(
    mut env: WasmEnvMut,
    in_buf_ptr: Uptr,
    in_buf_len: u32,
    out_buf_ptr: Uptr,
    out_len_ptr: Uptr,
) -> Result<u32, Escape> {
    let mut caller_env = JitCallerEnv::new(&mut env);
    let in_slice = caller_env.caller_read_slice(in_buf_ptr, in_buf_len);
    let orig_output_len = caller_env.caller_read_u32(out_len_ptr) as usize;
    let mut output = vec![0u8; orig_output_len as usize];
    let mut output_len = orig_output_len;
    unsafe {
        let res = BrotliDecoderDecompress(
            in_buf_len as usize,
            in_slice.as_ptr(),
            &mut output_len,
            output.as_mut_ptr(),
        );
        if (res != BrotliStatus::Success) || (output_len > orig_output_len) {
            return Ok(0);
        }
    }
    caller_env.caller_write_slice(out_buf_ptr, &output[..output_len]);
    caller_env.caller_write_u32(out_len_ptr, output_len as u32);
    Ok(1)
}

/// Brotli compresses a go slice
///
/// The output buffer must be sufficiently large enough.
pub fn brotli_compress(
    mut env: WasmEnvMut,
    in_buf_ptr: Uptr,
    in_buf_len: u32,
    out_buf_ptr: Uptr,
    out_len_ptr: Uptr,
    level: u32,
    window_size: u32,
) -> Result<u32, Escape> {
    let mut caller_env = JitCallerEnv::new(&mut env);
    let in_slice = caller_env.caller_read_slice(in_buf_ptr, in_buf_len);
    let orig_output_len = caller_env.caller_read_u32(out_len_ptr) as usize;
    let mut output = vec![0u8; orig_output_len];
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
            return Ok(0);
        }
    }
    caller_env.caller_write_slice(out_buf_ptr, &output[..output_len]);
    caller_env.caller_write_u32(out_len_ptr, output_len as u32);
    Ok(1)
}
