// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! Brotli via the `arbcompress` host-import module, for replay-style wasm builds.
//!
//! Nothing is linked here: the module only declares imports, and the runner resolves them —
//! the arbitrator with the `arbcompress.wasm` library, the JIT with native bindings, etc. The ABI
//! mirrors `arbcompress/wasm.go` and the exports of `crates/wasm-libraries/arbcompress`.

use alloc::{vec, vec::Vec};
use core::mem::MaybeUninit;

use crate::{BrotliStatus, Dictionary};

#[link(wasm_import_module = "arbcompress")]
unsafe extern "C" {
    fn brotli_compress(
        in_buf_ptr: *const u8,
        in_buf_len: u32,
        out_buf_ptr: *mut u8,
        out_len_ptr: *mut u32,
        level: u32,
        window_size: u32,
        dictionary: Dictionary,
    ) -> u32;

    fn brotli_decompress(
        in_buf_ptr: *const u8,
        in_buf_len: u32,
        out_buf_ptr: *mut u8,
        out_len_ptr: *mut u32,
        dictionary: Dictionary,
    ) -> u32;
}

const SUCCESS: u32 = BrotliStatus::Success as u32;

/// Upper bound for `compress` output; Go's `compressedBufferSizeFor` formula.
pub fn compression_bound(len: usize, _level: u32) -> usize {
    len + (len >> 10) * 8 + 64
}

/// Brotli compresses a slice into a vec, through the host.
pub fn compress(
    input: &[u8],
    level: u32,
    window_size: u32,
    dictionary: Dictionary,
) -> Result<Vec<u8>, BrotliStatus> {
    let mut output = vec![0u8; compression_bound(input.len(), level)];
    let mut out_len = output.len() as u32;
    let status = unsafe {
        brotli_compress(
            input.as_ptr(),
            input.len() as u32,
            output.as_mut_ptr(),
            &mut out_len,
            level,
            window_size,
            dictionary,
        )
    };
    if status != SUCCESS {
        return Err(BrotliStatus::Failure);
    }
    output.truncate(out_len as usize);
    Ok(output)
}

/// Brotli decompresses a slice into a vec, through the host.
///
/// The host call needs the output buffer upfront, so the buffer is grown by
/// doubling until the payload fits. Deterministic: the same input always takes
/// the same retry sequence and yields the same bytes. Corrupt input costs the
/// full retry ladder before failing; size-capped callers should prefer
/// [`decompress_fixed`].
pub fn decompress(input: &[u8], dictionary: Dictionary) -> Result<Vec<u8>, BrotliStatus> {
    // Brotli caps the compression ratio near 4096:1, so 12 doublings of the initial guess always
    // suffice for valid input.
    let mut len = input.len().max(1024) * 4;
    for _ in 0..12 {
        let mut output = vec![MaybeUninit::uninit(); len];
        if let Ok(data) = decompress_fixed(input, &mut output, dictionary) {
            return Ok(data.to_vec());
        }
        len *= 2;
    }
    Err(BrotliStatus::Failure)
}

/// Brotli decompresses a slice into a buffer of limited capacity, through the host.
pub fn decompress_fixed<'a>(
    input: &[u8],
    output: &'a mut [MaybeUninit<u8>],
    dictionary: Dictionary,
) -> Result<&'a [u8], BrotliStatus> {
    let mut out_len = output.len() as u32;
    let status = unsafe {
        brotli_decompress(
            input.as_ptr(),
            input.len() as u32,
            output.as_mut_ptr() as *mut u8,
            &mut out_len,
            dictionary,
        )
    };
    if status != SUCCESS {
        return Err(BrotliStatus::Failure);
    }
    // SAFETY: the host initialized `out_len` bytes.
    unsafe {
        Ok(core::mem::transmute::<&[MaybeUninit<u8>], &[u8]>(
            &output[..out_len as usize],
        ))
    }
}
