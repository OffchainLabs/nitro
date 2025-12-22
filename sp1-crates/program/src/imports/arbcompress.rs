//! This module implements arbcompression functions required by Arbitrum.
//! It is based on:
//! https://github.com/OffchainLabs/nitro/blob/d2dba175c037c47e68cf3038f0d4b06b54983644/arbitrator/caller-env/src/brotli/mod.rs
//! But a pure Rust brotli implementation is used instead of the C++ FFI one.
//! We have verified (in a small scale) that the 2 brotli implementations generate
//! the same bytes in both compression and decompression.

use crate::{Escape, Ptr, read_slice, replay::CustomEnvData};
use brotli::{Allocator, CompressorWriter, Decompressor, HeapAlloc, SliceWrapperMut};
use std::io::{Cursor, Read, Write};
use wasmer::FunctionEnvMut;

const STYLUS_DICTIONARY: &[u8] =
    include_bytes!("../../../../arbitrator/brotli/src/dicts/stylus-program-11.lz");

// Following Arbitrum's convention
pub const BROTLI_SUCCESS: u32 = 1;

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
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    let input = read_slice(in_buf_ptr, in_buf_len as usize, &memory)?;
    let mut output = vec![];
    {
        let mut writer = CompressorWriter::new(&mut output, 4096, level, window_size);
        match dictionary {
            0 => (), // Empty dictionary
            1 => writer.set_custom_dictionary(STYLUS_DICTIONARY),
            _ => panic!("Unknown dictionary value: {dictionary}"),
        }
        writer.write_all(&input)?;
    }
    let out_len = out_len_ptr.read(&memory)?;
    assert!(output.len() <= out_len as usize);

    memory.write(out_buf_ptr.offset() as u64, &output)?;
    out_len_ptr.write(&memory, output.len() as u32)?;

    Ok(BROTLI_SUCCESS)
}

pub fn brotli_decompress(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    in_buf_ptr: Ptr,
    in_buf_len: u32,
    out_buf_ptr: Ptr,
    out_len_ptr: Ptr,
    dictionary: u8,
) -> Result<u32, Escape> {
    // Keep the allocator alive for the duration of this method
    let mut allocator = HeapAlloc::default();

    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    let input = read_slice(in_buf_ptr, in_buf_len as usize, &memory)?;
    let mut decompressor = match dictionary {
        0 => Decompressor::new(Cursor::new(input), 4096),
        1 => {
            // This is slow(requires copying for every operation), but it might work now.
            let mut buffer = allocator.alloc_cell(STYLUS_DICTIONARY.len());
            buffer.slice_mut().copy_from_slice(STYLUS_DICTIONARY);
            Decompressor::new_with_custom_dict(Cursor::new(input), 4096, buffer)
        }
        _ => panic!("Unknown dictionary value: {dictionary}"),
    };
    let mut output = vec![];
    decompressor.read_to_end(&mut output)?;

    let out_len = out_len_ptr.read(&memory)?;
    assert!(output.len() <= out_len as usize);

    memory.write(out_buf_ptr.offset() as u64, &output)?;
    out_len_ptr.write(&memory, output.len() as u32)?;

    Ok(BROTLI_SUCCESS)
}
