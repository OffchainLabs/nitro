// Copyright 2021-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use jit::program::create_stylus_config;
use jit::machine::WasmEnv;
use wasmer::{ Function, FunctionEnv, Store, CompilerConfig, Value };
use wasmer_compiler_cranelift::Cranelift;

fn main() -> eyre::Result<()> {
    let env = WasmEnv::default();

    let mut compiler = Cranelift::new();
    compiler.canonicalize_nans(true);
    compiler.enable_verifier();
    let mut store = Store::new(compiler);

    let func_env = FunctionEnv::new(&mut store, env);
    let f_create_stylus_config = Function::new_typed_with_env(&mut store, &func_env, create_stylus_config);

    let ret = f_create_stylus_config.call(&mut store, &[Value::I32(0), Value::I32(10000), Value::I32(1), Value::I32(0)]).unwrap();

    println!("Hello, world2!");
    println!("{:?}", ret);
    Ok(())
}
