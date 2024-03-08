// Copyright 2021-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::wavm;

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

type Uptr = usize;

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
/// The output buffer must be sufficiently large enough.
#[no_mangle]
pub unsafe extern "C" fn arbcompress__brotliDecompress(in_buf_ptr: Uptr, in_buf_len: usize, out_buf_ptr: Uptr, out_len_ptr: Uptr) -> BrotliStatus {
    let in_slice = wavm::read_slice_usize(in_buf_ptr, in_buf_len);
    let orig_output_len = wavm::caller_load32(out_len_ptr) as usize;
    let mut output = vec![0u8; orig_output_len as usize];
    let mut output_len = orig_output_len;
    let res = BrotliDecoderDecompress(
        in_buf_len as usize,
        in_slice.as_ptr(),
        &mut output_len,
        output.as_mut_ptr(),
    );
    if (res != BrotliStatus::Success) || (output_len > orig_output_len) {
        return BrotliStatus::Failure;
    }
    wavm::write_slice_usize(&output[..output_len], out_buf_ptr);
    wavm::caller_store32(out_len_ptr, output_len as u32);
    BrotliStatus::Success
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
pub unsafe extern "C" fn arbcompress__brotliCompress(in_buf_ptr: Uptr, in_buf_len: usize, out_buf_ptr: Uptr, out_len_ptr: Uptr, level: u32, window_size: u32) -> BrotliStatus {
    let in_slice = wavm::read_slice_usize(in_buf_ptr, in_buf_len);
    let orig_output_len = wavm::caller_load32(out_len_ptr) as usize;
    let mut output = vec![0u8; orig_output_len];
    let mut output_len = orig_output_len;

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
        return BrotliStatus::Failure;
    }
    wavm::write_slice_usize(&output[..output_len], out_buf_ptr);
    wavm::caller_store32(out_len_ptr, output_len as u32);
    BrotliStatus::Success
}
