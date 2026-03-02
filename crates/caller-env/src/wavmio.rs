// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::{GuestPtr, MemAccess};
use alloc::string::String;
use alloc::format;
use core::cmp::min;

/// Trait for accessing wavmio host state (globals, inbox, preimages).
pub trait WavmState {
    fn get_u64_global(&self, idx: usize) -> Option<u64>;
    fn set_u64_global(&mut self, idx: usize, val: u64) -> bool;
    fn get_bytes32_global(&self, idx: usize) -> Option<&[u8; 32]>;
    fn set_bytes32_global(&mut self, idx: usize, val: [u8; 32]) -> bool;
    fn get_sequencer_message(&self, num: u64) -> Option<&[u8]>;
    fn get_delayed_message(&self, num: u64) -> Option<&[u8]>;
    fn get_preimage(&self, preimage_type: u8, hash: &[u8; 32]) -> Option<&[u8]>;
}

/// Reads 32-bytes of global state and writes to guest memory.
pub fn get_global_state_bytes32<M: MemAccess, S: WavmState>(
    mem: &mut M,
    state: &S,
    idx: u32,
    out_ptr: GuestPtr,
) -> Result<(), String> {
    let Some(global) = state.get_bytes32_global(idx as usize) else {
        return Err("global read out of bounds in wavmio.getGlobalStateBytes32".into());
    };
    mem.write_slice(out_ptr, &global[..]);
    Ok(())
}

/// Reads 32-bytes from guest memory and writes to global state.
pub fn set_global_state_bytes32<M: MemAccess, S: WavmState>(
    mem: &M,
    state: &mut S,
    idx: u32,
    src_ptr: GuestPtr,
) -> Result<(), String> {
    let val = mem.read_fixed(src_ptr);
    if !state.set_bytes32_global(idx as usize, val) {
        return Err("global write oob in wavmio.setGlobalStateBytes32".into());
    }
    Ok(())
}

/// Reads 8-bytes of global state.
pub fn get_global_state_u64<S: WavmState>(state: &S, idx: u32) -> Result<u64, String> {
    match state.get_u64_global(idx as usize) {
        Some(val) => Ok(val),
        None => Err("global read out of bounds in wavmio.getGlobalStateU64".into()),
    }
}

/// Writes 8-bytes of global state.
pub fn set_global_state_u64<S: WavmState>(state: &mut S, idx: u32, val: u64) -> Result<(), String> {
    if !state.set_u64_global(idx as usize, val) {
        return Err("global write out of bounds in wavmio.setGlobalStateU64".into());
    }
    Ok(())
}

/// Reads up to 32 bytes of a sequencer inbox message at the given offset.
pub fn read_inbox_message<M: MemAccess, S: WavmState>(
    mem: &mut M,
    state: &S,
    msg_num: u64,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, String> {
    let message = match state.get_sequencer_message(msg_num) {
        Some(message) => message,
        None => return Err(format!("missing sequencer inbox message {msg_num}")),
    };
    let offset = offset as usize;
    let len = min(32, message.len().saturating_sub(offset));
    let read = message.get(offset..(offset + len)).unwrap_or_default();
    mem.write_slice(out_ptr, read);
    Ok(read.len() as u32)
}

/// Reads up to 32 bytes of a delayed inbox message at the given offset.
pub fn read_delayed_inbox_message<M: MemAccess, S: WavmState>(
    mem: &mut M,
    state: &S,
    msg_num: u64,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, String> {
    let message = match state.get_delayed_message(msg_num) {
        Some(message) => message,
        None => return Err(format!("missing delayed inbox message {msg_num}")),
    };
    let offset = offset as usize;
    let len = min(32, message.len().saturating_sub(offset));
    let read = message.get(offset..(offset + len)).unwrap_or_default();
    mem.write_slice(out_ptr, read);
    Ok(read.len() as u32)
}

/// Looks up a preimage by type and hash, reads up to 32 bytes at an aligned offset.
pub fn resolve_preimage<M: MemAccess, S: WavmState>(
    mem: &mut M,
    state: &S,
    preimage_type: u8,
    hash_ptr: GuestPtr,
    offset: u32,
    out_ptr: GuestPtr,
    name: &str,
) -> Result<u32, String> {
    let hash = mem.read_fixed(hash_ptr);
    let offset = offset as usize;

    let Some(preimage) = state.get_preimage(preimage_type, &hash) else {
        let hash_hex = hex::encode(hash);
        return Err(format!(
            "Missing requested preimage for hash {hash_hex} in {name}"
        ));
    };

    if offset % 32 != 0 {
        return Err(format!("bad offset {offset} in {name}"));
    }

    let len = min(32, preimage.len().saturating_sub(offset));
    let read = preimage.get(offset..(offset + len)).unwrap_or_default();
    mem.write_slice(out_ptr, read);
    Ok(read.len() as u32)
}

/// Returns 1 if a preimage exists for the given type and hash, 0 otherwise.
pub fn validate_certificate<M: MemAccess, S: WavmState>(
    mem: &M,
    state: &S,
    preimage_type: u8,
    hash_ptr: GuestPtr,
) -> u8 {
    let hash = mem.read_fixed(hash_ptr);
    match state.get_preimage(preimage_type, &hash) {
        Some(_) => 1,
        None => 0,
    }
}
