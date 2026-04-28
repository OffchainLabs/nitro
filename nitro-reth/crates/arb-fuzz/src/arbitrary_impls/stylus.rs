use arbitrary::{Arbitrary, Unstructured};
use serde::Serialize;
use wasm_smith::{Config, Module};

use super::tx::BoundedBytes;

#[derive(Debug, Clone, Arbitrary, Serialize)]
pub struct StylusFuzzInput {
    pub wasm_seed: u64,
    pub calldata: BoundedBytes<512>,
    pub gas_budget: u64,
}

#[derive(Debug, thiserror::Error)]
pub enum SmithError {
    #[error("wasm-smith generation error: {0}")]
    Smith(#[from] arbitrary::Error),
    #[error("wasm encoding error: {0}")]
    Encode(String),
}

/// Mint a deterministic Stylus-shaped WASM module from `seed`.
pub fn smith_wasm(seed: u64) -> Result<Vec<u8>, SmithError> {
    let mut buf = [0u8; 1024];
    let mut state = seed.wrapping_mul(0x9E37_79B9_7F4A_7C15).wrapping_add(1);
    for byte in buf.iter_mut() {
        state = state
            .wrapping_mul(6_364_136_223_846_793_005)
            .wrapping_add(1_442_695_040_888_963_407);
        *byte = (state >> 33) as u8;
    }
    let mut u = Unstructured::new(&buf);

    let mut cfg = Config::default();
    cfg.max_funcs = 4;
    cfg.max_globals = 2;
    cfg.max_memories = 1;
    cfg.max_memory32_bytes = 64 * 1024;
    cfg.max_imports = 0;
    cfg.max_exports = 4;
    cfg.max_instructions = 256;
    cfg.bulk_memory_enabled = false;
    cfg.reference_types_enabled = false;
    cfg.simd_enabled = false;
    cfg.threads_enabled = false;
    cfg.exceptions_enabled = false;
    cfg.gc_enabled = false;
    cfg.allow_start_export = false;

    let module = Module::new(cfg, &mut u).map_err(SmithError::Smith)?;
    Ok(module.to_bytes())
}
