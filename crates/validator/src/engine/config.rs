// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::path::PathBuf;

use arbutil::Bytes32;

// The default for JIT binary, no need for LLVM right now
pub(crate) const DEFAULT_JIT_CRANELIFT: bool = true;

pub(crate) const TARGET_ARM_64: &str = "arm64";
pub(crate) const TARGET_AMD_64: &str = "amd64";
pub(crate) const TARGET_HOST: &str = "host";

pub type ModuleRoot = Bytes32;

#[derive(Clone, Debug)]
pub struct JitManagerConfig {
    pub prover_bin_path: PathBuf,
    pub jit_cranelift: bool,
    pub wasm_memory_usage_limit: u64,
}

impl Default for JitManagerConfig {
    fn default() -> Self {
        Self {
            prover_bin_path: "replay.wasm".into(),
            jit_cranelift: DEFAULT_JIT_CRANELIFT,
            wasm_memory_usage_limit: 1 << 32,
        }
    }
}
