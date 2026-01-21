// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#[derive(Clone, Debug)]
pub struct JitMachineConfig {
    pub prover_bin_path: String,
    pub jit_cranelift: bool,
    pub wasm_memory_usage_limit: u64,
}

impl Default for JitMachineConfig {
    fn default() -> Self {
        Self {
            prover_bin_path: "replay.wasm".to_owned(),
            jit_cranelift: true,
            wasm_memory_usage_limit: 4_294_967_296,
        }
    }
}
