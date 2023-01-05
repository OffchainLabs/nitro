// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::env::{MaybeEscape, SystemStateData, WasmEnv, WasmEnvMut};
use eyre::Result;
use prover::programs::{
    meter::{STYLUS_GAS_LEFT, STYLUS_GAS_STATUS},
    native::NativeInstance, config::StylusConfig,
};
use wasmer::{imports, Function, FunctionEnv, Global, Instance, Module};

pub fn module(wasm: &[u8], config: StylusConfig) -> Result<Vec<u8>> {
    let mut store = config.store();
    let module = Module::new(&store, wasm)?;
    let imports = imports! {
        "forward" => {
            "read_args" => Function::new_typed(&mut store, |_: u32| {}),
            "return_data" => Function::new_typed(&mut store, |_: u32, _: u32| {}),
        },
    };
    Instance::new(&mut store, &module, &imports)?;

    let module = module.serialize()?;
    Ok(module.to_vec())
}

pub fn instance(path: &str, env: WasmEnv) -> Result<(NativeInstance, FunctionEnv<WasmEnv>)> {
    let mut store = env.config.store();
    let wat_or_wasm = std::fs::read(path)?;
    let module = Module::new(&store, wat_or_wasm)?;

    let func_env = FunctionEnv::new(&mut store, env);
    let imports = imports! {
        "forward" => {
            "read_args" => Function::new_typed_with_env(&mut store, &func_env, read_args),
            "return_data" => Function::new_typed_with_env(&mut store, &func_env, return_data),
        },
    };
    let instance = Instance::new(&mut store, &module, &imports)?;
    let exports = &instance.exports;

    let expect_global = |name| -> Global { instance.exports.get_global(name).unwrap().clone() };

    let memory = exports.get_memory("memory")?.clone();
    let gas_left = expect_global(STYLUS_GAS_LEFT);
    let gas_status = expect_global(STYLUS_GAS_STATUS);

    let env = func_env.as_mut(&mut store);
    env.memory = Some(memory);
    env.state = Some(SystemStateData {
        gas_left,
        gas_status,
        pricing: env.config.pricing,
    });

    let native = NativeInstance::new(instance, store);
    Ok((native, func_env))
}

fn read_args(mut env: WasmEnvMut, ptr: u32) -> MaybeEscape {
    WasmEnv::begin(&mut env)?;

    let (env, memory) = WasmEnv::data(&mut env);
    memory.write_slice(ptr, &env.args)?;
    Ok(())
}

fn return_data(mut env: WasmEnvMut, ptr: u32, len: u32) -> MaybeEscape {
    let mut state = WasmEnv::begin(&mut env)?;

    let evm_words = |count: u64| count.saturating_mul(31) / 32;
    let evm_gas = evm_words(len.into()).saturating_mul(3); // 3 evm gas per word
    state.buy_evm_gas(evm_gas)?;

    let (env, memory) = WasmEnv::data(&mut env);
    env.outs = memory.read_slice(ptr, len)?;
    Ok(())
}
