//! This module implements arbcompression functions required by Arbitrum.

use caller_env::GuestPtr;
use wasmer::FunctionEnvMut;

use crate::{
    Escape,
    replay::CustomEnvData,
    state::sp1_env,
};

#[allow(clippy::too_many_arguments)]
pub fn brotli_compress(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
    out_len_ptr: GuestPtr,
    level: u32,
    window_size: u32,
    dictionary: u8,
) -> Result<u32, Escape> {
    let (mut mem, state) = sp1_env(&mut ctx);
    Ok(caller_env::brotli::brotli_compress(
        &mut mem,
        state,
        in_buf_ptr,
        in_buf_len,
        out_buf_ptr,
        out_len_ptr,
        level,
        window_size,
        dictionary.try_into().expect("unknown dictionary"),
    )
    .into())
}

pub fn brotli_decompress(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    in_buf_ptr: GuestPtr,
    in_buf_len: u32,
    out_buf_ptr: GuestPtr,
    out_len_ptr: GuestPtr,
    dictionary: u8,
) -> Result<u32, Escape> {
    let (mut mem, state) = sp1_env(&mut ctx);
    Ok(caller_env::brotli::brotli_decompress(
        &mut mem,
        state,
        in_buf_ptr,
        in_buf_len,
        out_buf_ptr,
        out_len_ptr,
        dictionary.try_into().expect("unknown dictionary"),
    )
    .into())
}
