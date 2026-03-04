// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::machine::{WasmEnv, WasmEnvMut};
use crate::memory::JitMemAccess;
use arbutil::{Bytes32, PreimageType};
use caller_env::wavmio::WavmState;
use caller_env::ExecEnv;
use rand::RngCore;

/// Newtype wrapping &mut WasmEnv to implement WavmState (orphan rule).
pub(crate) struct JitState<'a>(pub &'a mut WasmEnv);

/// Extracts (JitMemAccess, JitState) from a WasmEnvMut in place.
pub(crate) fn jit_env<'a>(env: &'a mut WasmEnvMut) -> (JitMemAccess<'a>, JitState<'a>) {
    let memory = env.data().memory.clone().unwrap();
    let (wenv, store) = env.data_and_store_mut();
    (JitMemAccess { memory, store }, JitState(wenv))
}

impl ExecEnv for JitState<'_> {
    fn advance_time(&mut self, ns: u64) {
        self.0.go_state.time += ns;
    }

    fn get_time(&self) -> u64 {
        self.0.go_state.time
    }

    fn next_rand_u32(&mut self) -> u32 {
        self.0.go_state.rng.next_u32()
    }

    fn print_string(&mut self, bytes: &[u8]) {
        match String::from_utf8(bytes.to_vec()) {
            Ok(s) => eprintln!("JIT: WASM says: {s}"), // TODO: this adds too many newlines since go calls this in chunks
            Err(e) => {
                let bytes = e.as_bytes();
                eprintln!("Go string {} is not valid utf8: {e:?}", hex::encode(bytes));
            }
        }
    }
}

impl WavmState for JitState<'_> {
    fn get_u64_global(&self, idx: usize) -> Option<u64> {
        self.0.small_globals.get(idx).copied()
    }

    fn set_u64_global(&mut self, idx: usize, val: u64) -> bool {
        match self.0.small_globals.get_mut(idx) {
            Some(g) => {
                *g = val;
                true
            }
            None => false,
        }
    }

    fn get_bytes32_global(&self, idx: usize) -> Option<&[u8; 32]> {
        self.0.large_globals.get(idx).map(|b| &b.0)
    }

    fn set_bytes32_global(&mut self, idx: usize, val: [u8; 32]) -> bool {
        match self.0.large_globals.get_mut(idx) {
            Some(g) => {
                *g = val.into();
                true
            }
            None => false,
        }
    }

    fn get_sequencer_message(&self, num: u64) -> Option<&[u8]> {
        self.0.sequencer_messages.get(&num).map(|v| v.as_slice())
    }

    fn get_delayed_message(&self, num: u64) -> Option<&[u8]> {
        self.0.delayed_messages.get(&num).map(|v| v.as_slice())
    }

    fn get_preimage(&self, preimage_type: u8, hash: &[u8; 32]) -> Option<&[u8]> {
        let pt: PreimageType = preimage_type.try_into().ok()?;
        self.0
            .preimages
            .get(&pt)
            .and_then(|m| m.get(&Bytes32::from(*hash)))
            .map(|v| v.as_slice())
    }
}
