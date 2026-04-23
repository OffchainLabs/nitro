// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::path::{Path, PathBuf};

use arbutil::Bytes32;

pub mod execution;
pub mod machine;
pub mod machine_locator;

pub(crate) const DEFAULT_JIT_CRANELIFT: bool = true;
pub(crate) const DEFAULT_WASM_MEMORY_USAGE_LIMIT: u64 = 1 << 32;
const REPLAY_WASM: &str = "replay.wasm";

pub type ModuleRoot = Bytes32;

pub fn replay_binary(binary_path: &Path) -> PathBuf {
    binary_path.join(REPLAY_WASM)
}
