#![cfg_attr(target_os = "zkvm", no_main)]

#[cfg(target_os = "zkvm")]
sp1_zkvm::entrypoint!(main);

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

fn main() {
    let version = sp1_zkvm::io::read::<u16>();
    let debug = sp1_zkvm::io::read::<bool>();
    let wasm = sp1_zkvm::io::read::<Vec<u8>>();

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
    let rv64_binary = module.serialize().expect("serialize module");

    sp1_zkvm::io::commit(&rv64_binary.to_vec());
}

// Those are referenced by wasmer runtimes, but are never invoked
#[unsafe(no_mangle)]
pub extern "C" fn __negdf2(_x: f64) -> f64 {
    todo!()
}

#[unsafe(no_mangle)]
pub extern "C" fn __negsf2(_x: f32) -> f32 {
    todo!()
}
