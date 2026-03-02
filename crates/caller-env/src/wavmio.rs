// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use alloc::string::String;
use crate::{GuestPtr, MemAccess};

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
