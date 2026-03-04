// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use caller_env::wavmio::WavmIo;
use caller_env::{ExecEnv, GuestPtr};
use rand::RngCore;
use wasmer::FunctionEnvMut;

use crate::memory::Sp1MemAccess;
use crate::replay::CustomEnvData;

/// Newtype wrapper to implement WavmIo and ExecEnv for CustomEnvData.
pub(crate) struct Sp1State<'a>(pub &'a mut CustomEnvData);

impl ExecEnv for Sp1State<'_> {
    fn advance_time(&mut self, ns: u64) {
        self.0.time += ns;
    }

    fn get_time(&self) -> u64 {
        self.0.time
    }

    fn next_rand_u32(&mut self) -> u32 {
        self.0.pcg.next_u32()
    }

    fn print_string(&mut self, bytes: &[u8]) {
        crate::platform::print_string(2, bytes);
    }
}

impl WavmIo for Sp1State<'_> {
    fn get_u64_global(&self, idx: usize) -> Option<u64> {
        self.0.input().small_globals.get(idx).copied()
    }

    fn set_u64_global(&mut self, idx: usize, val: u64) -> bool {
        match self.0.input_mut().small_globals.get_mut(idx) {
            Some(g) => {
                *g = val;
                true
            }
            None => false,
        }
    }

    fn get_bytes32_global(&self, idx: usize) -> Option<&[u8; 32]> {
        self.0.input().large_globals.get(idx)
    }

    fn set_bytes32_global(&mut self, idx: usize, val: [u8; 32]) -> bool {
        match self.0.input_mut().large_globals.get_mut(idx) {
            Some(g) => {
                *g = val;
                true
            }
            None => false,
        }
    }

    fn get_sequencer_message(&self, num: u64) -> Option<&[u8]> {
        self.0.input().sequencer_messages.get(&num).map(|v| v.as_slice())
    }

    fn get_delayed_message(&self, num: u64) -> Option<&[u8]> {
        self.0.input().delayed_messages.get(&num).map(|v| v.as_slice())
    }

    fn get_preimage(&self, preimage_type: u8, hash: &[u8; 32]) -> Option<&[u8]> {
        self.0
            .input()
            .preimages
            .get(&preimage_type)
            .and_then(|m| m.get(hash))
            .map(|v| v.as_slice())
    }
}

/// Extracts (Sp1MemAccess, Sp1State) from a FunctionEnvMut in place.
pub(crate) fn sp1_env<'a>(
    ctx: &'a mut FunctionEnvMut<'_, CustomEnvData>,
) -> (Sp1MemAccess<'a>, Sp1State<'a>) {
    let memory = ctx.data().memory.clone().unwrap();
    let (data, store) = ctx.data_and_store_mut();
    (Sp1MemAccess { memory, store }, Sp1State(data))
}

/// Converts a wasmer `Ptr` (WasmPtr<u32>) to a caller-env `GuestPtr`.
pub(crate) fn gp(p: crate::Ptr) -> GuestPtr {
    GuestPtr(p.offset())
}
