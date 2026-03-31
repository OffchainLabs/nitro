use std::{str::FromStr, sync::Arc};

use anyhow::{Context, Result};
use prover::programs::{
    MiddlewareWrapper, config::CompileConfig, depth::DepthChecker, dynamic::DynamicMeter,
    heap::HeapBound, meter::Meter, start::StartMover,
};
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
///
/// Applies the same middleware stack used in the standard Stylus compilation pipeline:
/// `StartMover`, `Meter`, `DynamicMeter`, `DepthChecker`, and `HeapBound`.
pub fn compile(input: &CompileInput) -> Result<Vec<u8>> {
    let compile_config = CompileConfig::version(input.version, input.debug);
    let mut config = Singlepass::new();
    config.canonicalize_nans(true);
    config.enable_verifier();

    let start = MiddlewareWrapper::new(StartMover::new(compile_config.debug.debug_info));
    let meter = MiddlewareWrapper::new(Meter::new(&compile_config.pricing));
    let dygas = MiddlewareWrapper::new(DynamicMeter::new(&compile_config.pricing));
    let depth = MiddlewareWrapper::new(DepthChecker::new(compile_config.bounds));
    let bound = MiddlewareWrapper::new(HeapBound::new(compile_config.bounds));

    config.push_middleware(Arc::new(start));
    config.push_middleware(Arc::new(meter));
    config.push_middleware(Arc::new(dygas));
    config.push_middleware(Arc::new(depth));
    config.push_middleware(Arc::new(bound));

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
