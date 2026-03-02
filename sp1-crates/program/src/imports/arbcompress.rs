//! This module implements arbcompression functions required by Arbitrum.
//! It is based on:
//! https://github.com/OffchainLabs/nitro/blob/d2dba175c037c47e68cf3038f0d4b06b54983644/arbitrator/caller-env/src/brotli/mod.rs
//! But a pure Rust brotli implementation is used instead of the C++ FFI one.
//! We have verified (in a small scale) that the 2 brotli implementations generate
//! the same bytes in both compression and decompression.

use crate::{Escape, Ptr, read_slice, replay::CustomEnvData};
use brotli::{BrotliStatus, Dictionary};
use std::io::{Cursor, Read, Write};
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
    let dictionary: Dictionary = dictionary.try_into().unwrap();

    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    let input = read_slice(in_buf_ptr, in_buf_len as usize, &memory)?;
    let out_len = out_len_ptr.read(&memory)?;    
    let mut output = Vec::with_capacity(out_len as usize);

    let result = brotli::compress_fixed(
        &input,
        output.spare_capacity_mut(),
        level,
        window_size,
        dictionary,
    );
    Ok(match result {
        Ok(slice) => {
            memory.write(out_buf_ptr.offset() as u64, slice)?;
            out_len_ptr.write(&memory, slice.len() as u32)?;
            BrotliStatus::Success
        }
        Err(status) => status,
    } as u32)
}

pub fn brotli_decompress(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    in_buf_ptr: Ptr,
    in_buf_len: u32,
    out_buf_ptr: Ptr,
    out_len_ptr: Ptr,
    dictionary: u8,
) -> Result<u32, Escape> {
    let dictionary: Dictionary = dictionary.try_into().unwrap();

    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    let input = read_slice(in_buf_ptr, in_buf_len as usize, &memory)?;
    let out_len = out_len_ptr.read(&memory)?;    
    let mut output = Vec::with_capacity(out_len as usize);

    let result = brotli::decompress_fixed(&input, output.spare_capacity_mut(), dictionary);
    Ok(match result {
        Ok(slice) => {
            memory.write(out_buf_ptr.offset() as u64, slice)?;
            out_len_ptr.write(&memory, slice.len() as u32)?;
            BrotliStatus::Success            
        }
        Err(status) => status,
    } as u32)
}
