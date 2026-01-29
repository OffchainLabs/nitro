// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// The default for JIT binary, no need for LLVM right now
pub(crate) const DEFAULT_JIT_CRANELIFT: bool = true;

pub(crate) const TARGET_ARM_64: &str = "arm64";
pub(crate) const TARGET_AMD_64: &str = "amd64";
pub(crate) const TARGET_HOST: &str = "host";

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
            jit_cranelift: DEFAULT_JIT_CRANELIFT,
            wasm_memory_usage_limit: 1 << 32,
        }
    }
}
