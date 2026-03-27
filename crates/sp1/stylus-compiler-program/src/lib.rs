use prover::programs::{
    MiddlewareWrapper, config::CompileConfig, depth::DepthChecker, dynamic::DynamicMeter,
    heap::HeapBound, meter::Meter, start::StartMover,
};
use std::str::FromStr;
use std::sync::Arc;
use wasmer::{
    Module, Store,
    sys::{CompilerConfig, CpuFeature, EngineBuilder, Singlepass, Target, Triple},
};

/// Compiles a Stylus WASM program to a rv64 binary using the wasmer singlepass compiler.
///
/// Applies the same middleware stack used in the standard Stylus compilation pipeline:
/// `StartMover`, `Meter`, `DynamicMeter`, `DepthChecker`, and `HeapBound`.
pub fn compile(version: u16, debug: bool, wasm: &[u8]) -> Vec<u8> {
    let compile_config = CompileConfig::version(version, debug);
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

    let triple = Triple::from_str("riscv64").expect("target triple");
    let engine = EngineBuilder::new(config)
        .set_target(Some(Target::new(triple, CpuFeature::set())))
        .engine();

    let store = Store::new(engine);
    let module = Module::new(&store, wasm).expect("compilation failed");
    module.serialize().expect("serialize module").to_vec()
}
