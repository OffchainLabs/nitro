// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::str::FromStr;

use anyhow::{Context, Result};
use prover::programs::config::CompileConfig;
use serde::{Deserialize, Serialize};
use wasmer::{
    Module, Store,
    sys::{CompilerConfig, CpuFeature, EngineBuilder, Singlepass, Target, Triple},
};

/// Input parameters for Stylus WASM compilation.
#[derive(Serialize, Deserialize)]
pub struct CompileInput {
    pub version: u16,
    pub debug: bool,
    pub wasm: Vec<u8>,
}

/// Compiles a Stylus WASM program to a rv64 binary using the wasmer singlepass compiler.
pub fn compile(input: &CompileInput) -> Result<Vec<u8>> {
    anyhow::ensure!(
        input.version <= 3,
        "unsupported Stylus version {}, expected 0..=3",
        input.version
    );
    let compile_config = CompileConfig::version(input.version, input.debug);
    let mut config = Singlepass::new();
    config.canonicalize_nans(true);
    config.enable_verifier();

    compile_config.push_stylus_middlewares(&mut config);

    let triple =
        Triple::from_str("riscv64").map_err(|e| anyhow::anyhow!("invalid target triple: {e}"))?;
    let engine = EngineBuilder::new(config)
        .set_target(Some(Target::new(triple, CpuFeature::set())))
        .engine();

    let store = Store::new(engine);
    let module = Module::new(&store, &input.wasm).context("wasm compilation failed")?;
    let rv64_binary = module.serialize().context("module serialization failed")?;
    Ok(rv64_binary.to_vec())
}
