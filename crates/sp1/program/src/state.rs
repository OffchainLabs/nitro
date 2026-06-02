// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use caller_env::{
    ExecEnv,
    wasmer_traits::{HasMemory, WasmerMem},
};
use rand::Rng;
use wasmer::FunctionEnvMut;

use crate::replay::CustomEnvData;

impl ExecEnv for CustomEnvData {
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
        crate::platform::print_string(1, bytes);
    }
}

/// Extracts (WasmerMem, &mut CustomEnvData) from a FunctionEnvMut in place.
pub(crate) fn sp1_env<'a>(
    ctx: &'a mut FunctionEnvMut<'_, CustomEnvData>,
) -> (WasmerMem<'a>, &'a mut CustomEnvData) {
    let memory = ctx.data().memory();
    let (data, store) = ctx.data_and_store_mut();
    (WasmerMem::new(memory, store), data)
}
