// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use caller_env::{ExecEnv, wasmer_traits::WasmerMem};
use rand::Rng;

use crate::machine::{WasmEnv, WasmEnvMut};

pub(crate) trait JitEnv<'a> {
    fn jit_env(&mut self) -> (WasmerMem<'_>, &mut WasmEnv);
}

impl<'a> JitEnv<'a> for WasmEnvMut<'a> {
    fn jit_env(&mut self) -> (WasmerMem<'_>, &mut WasmEnv) {
        let memory = self.data().memory.clone().unwrap();
        let (wenv, store) = self.data_and_store_mut();
        (WasmerMem::new(memory, store), wenv)
    }
}

impl ExecEnv for WasmEnv {
    fn advance_time(&mut self, ns: u64) {
        self.go_state.time += ns;
    }

    fn get_time(&self) -> u64 {
        self.go_state.time
    }

    fn next_rand_u32(&mut self) -> u32 {
        self.go_state.rng.next_u32()
    }

    fn print_string(&mut self, bytes: &[u8]) {
        match String::from_utf8(bytes.to_vec()) {
            Ok(s) => eprintln!("JIT: WASM says: {s}"), // TODO: this adds too many newlines
            // since go calls this in chunks
            Err(e) => {
                let bytes = e.as_bytes();
                eprintln!("Go string {} is not valid utf8: {e:?}", hex::encode(bytes));
            }
        }
    }
}
