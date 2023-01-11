// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::wavm;
use go_abi::*;

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

/// Brotli decompresses a go slice
///
/// # Safety
///
/// The go side has the following signature, which must be respected.
///     λ(inBuf []byte, outBuf []byte) (outLen uint64, status BrotliStatus)
///
/// The output buffer must be sufficiently large enough.
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbcompress_brotliDecompress(sp: usize) {
    let mut sp = GoStack::new(sp);
    let (in_buf_ptr, in_buf_len) = sp.read_go_slice();
    let (out_buf_ptr, out_buf_len) = sp.read_go_slice();

    let in_slice = wavm::read_slice(in_buf_ptr, in_buf_len);
    let mut output = vec![0u8; out_buf_len as usize];
    let mut output_len = out_buf_len as usize;
    let res = BrotliDecoderDecompress(
        in_buf_len as usize,
        in_slice.as_ptr(),
        &mut output_len,
        output.as_mut_ptr(),
    );
    if (res != BrotliStatus::Success) || (output_len as u64 > out_buf_len) {
        sp.skip_u64();
        sp.write_u32(BrotliStatus::Failure as _);
        return;
    }
    wavm::write_slice(&output[..output_len], out_buf_ptr);
    sp.write_u64(output_len as u64);
    sp.write_u32(BrotliStatus::Success as _);
}

/// Brotli compresses a go slice
///
/// # Safety
///
/// The go side has the following signature, which must be respected.
///     λ(inBuf []byte, outBuf []byte, level, windowSize uint64) (outLen uint64, status BrotliStatus)
///
/// The output buffer must be sufficiently large enough.
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbcompress_brotliCompress(sp: usize) {
    let mut sp = GoStack::new(sp);
    let (in_buf_ptr, in_buf_len) = sp.read_go_slice();
    let (out_buf_ptr, out_buf_len) = sp.read_go_slice();
    let level = sp.read_u32();
    let windowsize = sp.read_u32();

    let in_slice = wavm::read_slice(in_buf_ptr, in_buf_len);
    let mut output = vec![0u8; out_buf_len as usize];
    let mut output_len = out_buf_len as usize;
    let res = BrotliEncoderCompress(
        level,
        windowsize,
        BROTLI_MODE_GENERIC,
        in_buf_len as usize,
        in_slice.as_ptr(),
        &mut output_len,
        output.as_mut_ptr(),
    );
    if (res != BrotliStatus::Success) || (output_len as u64 > out_buf_len) {
        sp.skip_u64();
        sp.write_u32(BrotliStatus::Failure as _);
        return;
    }
    wavm::write_slice(&output[..output_len], out_buf_ptr);
    sp.write_u64(output_len as u64);
    sp.write_u32(BrotliStatus::Success as _);
}
