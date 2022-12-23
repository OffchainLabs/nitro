// Copyright 2022, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::env::{MaybeEscape, SystemState, WasmEnv, WasmEnvMut};
use eyre::Result;
use prover::programs::meter::{POLYGLOT_GAS_LEFT, POLYGLOT_GAS_STATUS};
use wasmer::{imports, Function, FunctionEnv, Global, Instance, Module, Store};

pub fn instance(path: &str, env: WasmEnv) -> Result<(Instance, FunctionEnv<WasmEnv>, Store)> {
    let mut store = env.config.store();
    let wat_or_wasm = std::fs::read(path)?;
    let module = Module::new(&store, &wat_or_wasm)?;

    let func_env = FunctionEnv::new(&mut store, env);
    let imports = imports! {
        "poly_host" => {
            "read_args" => Function::new_typed_with_env(&mut store, &func_env, read_args),
            "return_data" => Function::new_typed_with_env(&mut store, &func_env, return_data),
        },
    };
    let instance = Instance::new(&mut store, &module, &imports)?;
    let exports = &instance.exports;

    let expect_global = |name| -> Global { instance.exports.get_global(name).unwrap().clone() };

    let memory = exports.get_memory("memory")?.clone();
    let gas_left = expect_global(POLYGLOT_GAS_LEFT);
    let gas_status = expect_global(POLYGLOT_GAS_STATUS);

    let env = func_env.as_mut(&mut store);
    env.memory = Some(memory);
    env.state = Some(SystemState {
        gas_left,
        gas_status,
        wasm_gas_price: env.config.wasm_gas_price,
        hostio_cost: env.config.hostio_cost,
    });

    Ok((instance, func_env, store))
}

fn read_args(mut env: WasmEnvMut, ptr: u32) -> MaybeEscape {
    WasmEnv::begin(&mut env)?;

    let (env, memory) = WasmEnv::data(&mut env);
    memory.write_slice(ptr, &env.args)?;
    Ok(())
}

fn return_data(mut env: WasmEnvMut, ptr: u32, len: u32) -> MaybeEscape {
    let (mut state, mut store) = WasmEnv::begin(&mut env)?;

    let evm_words = |count: u64| count.saturating_mul(31) / 32;
    let evm_gas = evm_words(len.into()).saturating_mul(3); // 3 evm gas per word
    state.buy_evm_gas(&mut store, evm_gas)?;

    let (env, memory) = WasmEnv::data(&mut env);
    env.outs = memory.read_slice(ptr, len)?;
    Ok(())
}
