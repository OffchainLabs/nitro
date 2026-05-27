// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use brotli::{BrotliStatus, Dictionary};
use caller_env::{self, GuestPtr};

use crate::{
    caller_env::JitEnv,
    machine::{Escape, WasmEnvMut},
};

#[allow(clippy::too_many_arguments)]
pub fn brotli_compress(
    mut src: WasmEnvMut,
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
    out_len_ptr: GuestPtr,
    level: u32,
    window_size: u32,
    dictionary: Dictionary,
) -> Result<BrotliStatus, Escape> {
    let (mut mem, _) = src.jit_env();
    Ok(caller_env::brotli::brotli_compress(
        &mut mem,
        in_buf_ptr,
        in_buf_len,
        out_buf_ptr,
        out_len_ptr,
        level,
        window_size,
        dictionary,
    ))
}

#[allow(clippy::too_many_arguments)]
pub fn brotli_decompress(
    mut src: WasmEnvMut,
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
    out_len_ptr: GuestPtr,
    dictionary: Dictionary,
) -> Result<BrotliStatus, Escape> {
    let (mut mem, _) = src.jit_env();
    Ok(caller_env::brotli::brotli_decompress(
        &mut mem,
        in_buf_ptr,
        in_buf_len,
        out_buf_ptr,
        out_len_ptr,
        dictionary,
    ))
}
