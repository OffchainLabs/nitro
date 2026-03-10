//! This module implements arbcompression functions required by Arbitrum.

use crate::state::{gp, sp1_env};
use crate::{Escape, Ptr, replay::CustomEnvData};
use brotli::Dictionary;
use wasmer::FunctionEnvMut;

pub fn brotli_compress(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    in_buf_ptr: Ptr,
    in_buf_len: u32,
    out_buf_ptr: Ptr,
    out_len_ptr: Ptr,
    level: u32,
    window_size: u32,
    dictionary: u8,
) -> Result<u32, Escape> {
    let (mut mem, state) = sp1_env(&mut ctx);
    let dictionary = Dictionary::try_from(dictionary).expect("unknown dictionary");
    Ok(caller_env::brotli::brotli_compress(
        &mut mem,
        state,
        gp(in_buf_ptr),
        in_buf_len,
        gp(out_buf_ptr),
        gp(out_len_ptr),
        level,
        window_size,
        dictionary,
    )
    .into())
}

pub fn brotli_decompress(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    in_buf_ptr: Ptr,
    in_buf_len: u32,
    out_buf_ptr: Ptr,
    out_len_ptr: Ptr,
    dictionary: u8,
) -> Result<u32, Escape> {
    let (mut mem, state) = sp1_env(&mut ctx);
    let dictionary = Dictionary::try_from(dictionary).expect("unknown dictionary");
    Ok(caller_env::brotli::brotli_decompress(
        &mut mem,
        state,
        gp(in_buf_ptr),
        in_buf_len,
        gp(out_buf_ptr),
        gp(out_len_ptr),
        dictionary,
    )
    .into())
}
