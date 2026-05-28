// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![allow(clippy::too_many_arguments)]

use alloc::vec::Vec;

use brotli::{BrotliStatus, Dictionary};

use crate::{GuestPtr, MemAccess};

pub fn brotli_compress<M: MemAccess>(
    mem: &mut M,
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
    out_len_ptr: GuestPtr,
    level: u32,
    window_size: u32,
    dictionary: Dictionary,
) -> BrotliStatus {
    let input = mem.read_slice(in_buf_ptr, in_buf_len as usize);
    let mut output = Vec::with_capacity(mem.read_u32(out_len_ptr) as usize);

    let result = brotli::compress_fixed(
        &input,
        output.spare_capacity_mut(),
        level,
        window_size,
        dictionary,
    );
    write_output(mem, out_buf_ptr, out_len_ptr, result)
}

pub fn brotli_decompress<M: MemAccess>(
    mem: &mut M,
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
    out_len_ptr: GuestPtr,
    dictionary: Dictionary,
) -> BrotliStatus {
    let input = mem.read_slice(in_buf_ptr, in_buf_len as usize);
    let mut output = Vec::with_capacity(mem.read_u32(out_len_ptr) as usize);

    let result = brotli::decompress_fixed(&input, output.spare_capacity_mut(), dictionary);
    write_output(mem, out_buf_ptr, out_len_ptr, result)
}

fn write_output<M: MemAccess>(
    mem: &mut M,
    out_buf_ptr: GuestPtr,
    out_len_ptr: GuestPtr,
    result: Result<&[u8], BrotliStatus>,
) -> BrotliStatus {
    match result {
        Ok(slice) => {
            mem.write_slice(out_buf_ptr, slice);
            mem.write_u32(out_len_ptr, slice.len() as u32);
            BrotliStatus::Success
        }
        Err(status) => status,
    }
}

#[cfg(feature = "wasmer_traits")]
pub mod host {
    use brotli::{BrotliStatus, Dictionary};

    use crate::GuestPtr;

    host_fn!(fn brotli_compress(a: GuestPtr, b: u32, c: GuestPtr, d: GuestPtr, e: u32, f: u32, g: Dictionary) -> BrotliStatus);
    host_fn!(fn brotli_decompress(a: GuestPtr, b: u32, c: GuestPtr, d: GuestPtr, e: Dictionary) -> BrotliStatus);
}
