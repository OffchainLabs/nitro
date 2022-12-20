// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use arbutil::{crypto, format};
use eyre::Result;
use std::time::{Duration, Instant};
use wasmer::{CompilerConfig, Imports, Instance, Module, Store};
use wasmer_compiler_cranelift::{Cranelift, CraneliftOptLevel};
use wasmer_compiler_llvm::{LLVMOptLevel, LLVM};
use wasmer_compiler_singlepass::Singlepass;

#[test]
fn benchmark_wasmer() -> Result<()> {
    // benchmarks wasmer across all compiler backends

    fn single() -> Store {
        let mut compiler = Singlepass::new();
        compiler.canonicalize_nans(true);
        compiler.enable_verifier();
        Store::new(compiler)
    }

    fn cranelift() -> Store {
        let mut compiler = Cranelift::new();
        compiler.canonicalize_nans(true);
        compiler.enable_verifier();
        compiler.opt_level(CraneliftOptLevel::Speed);
        Store::new(compiler)
    }

    fn llvm() -> Store {
        let mut compiler = LLVM::new();
        compiler.canonicalize_nans(true);
        compiler.enable_verifier();
        compiler.opt_level(LLVMOptLevel::Aggressive);
        Store::new(compiler)
    }

    fn emulated(mut store: Store) -> Result<Duration> {
        let wat = std::fs::read("tests/keccak/target/wasm32-unknown-unknown/release/keccak.wasm")?;
        let module = Module::new(&mut store, &wat)?;
        let instance = Instance::new(&mut store, &module, &Imports::new())?;

        let exports = instance.exports;
        let main = exports.get_typed_function::<(i32, i32), i32>(&store, "main")?;

        let time = Instant::now();
        main.call(&mut store, 0, 0)?;
        Ok(time.elapsed())
    }

    fn native() -> Duration {
        let time = Instant::now();
        let mut data = [0; 32];
        for _ in 0..100 {
            data = crypto::keccak(&data);
        }
        assert_ne!(data, [0; 32]); // keeps the optimizer from pruning `data`
        time.elapsed()
    }

    println!("Native:  {}", format::time(native()));
    println!("LLVM:    {}", format::time(emulated(llvm())?));
    println!("Crane:   {}", format::time(emulated(cranelift())?));
    println!("Single:  {}", format::time(emulated(single())?));
    Ok(())
}
