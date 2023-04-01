// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::env::{Escape, MaybeEscape, WasmEnv, WasmEnvMut};
use arbutil::Color;
use prover::programs::prelude::*;

// params.SstoreSentryGasEIP2200 (see operations_acl_arbitrum.go)
const SSTORE_SENTRY_EVM_GAS: u64 = 2300;

// params.LogGas and params.LogDataGas
const LOG_TOPIC_GAS: u64 = 375;
const LOG_DATA_GAS: u64 = 8;

pub(crate) fn read_args(mut env: WasmEnvMut, ptr: u32) -> MaybeEscape {
    WasmEnv::begin(&mut env)?;

    let (env, memory) = WasmEnv::data(&mut env);
    memory.write_slice(ptr, &env.args)?;
    Ok(())
}

pub(crate) fn return_data(mut env: WasmEnvMut, ptr: u32, len: u32) -> MaybeEscape {
    let mut meter = WasmEnv::begin(&mut env)?;
    meter.pay_for_evm_copy(len as usize)?;

    let (env, memory) = WasmEnv::data(&mut env);
    env.outs = memory.read_slice(ptr, len)?;
    Ok(())
}

pub(crate) fn account_load_bytes32(mut env: WasmEnvMut, key: u32, dest: u32) -> MaybeEscape {
    WasmEnv::begin(&mut env)?;

    let (data, memory) = WasmEnv::data(&mut env);
    let key = memory.read_bytes32(key)?;
    let (value, cost) = data.evm().load_bytes32(key);
    memory.write_slice(dest, &value.0)?;

    let mut meter = WasmEnv::meter(&mut env);
    meter.buy_evm_gas(cost)
}

pub(crate) fn account_store_bytes32(mut env: WasmEnvMut, key: u32, value: u32) -> MaybeEscape {
    let mut meter = WasmEnv::begin(&mut env)?;
    meter.require_evm_gas(SSTORE_SENTRY_EVM_GAS)?;

    let (data, memory) = WasmEnv::data(&mut env);
    let key = memory.read_bytes32(key)?;
    let value = memory.read_bytes32(value)?;
    let cost = data.evm().store_bytes32(key, value)?;

    let mut meter = WasmEnv::meter(&mut env);
    meter.buy_evm_gas(cost)
}

pub(crate) fn call_contract(
    mut env: WasmEnvMut,
    contract: u32,
    calldata: u32,
    calldata_len: u32,
    value: u32,
    mut wasm_gas: u64,
    return_data_len: u32,
) -> Result<u8, Escape> {
    let mut env = WasmEnv::start(&mut env)?;
    env.pay_for_evm_copy(calldata_len as usize)?;
    wasm_gas = wasm_gas.min(env.gas_left().into()); // provide no more than what the user has

    let evm_gas = env.meter().pricing.wasm_to_evm(wasm_gas);
    let contract = env.read_bytes20(contract)?;
    let input = env.read_slice(calldata, calldata_len)?;
    let value = env.read_bytes32(value)?;

    let (outs_len, evm_cost, status) = env.evm().contract_call(contract, input, evm_gas, value);
    env.set_return_data_len(outs_len);
    env.write_u32(return_data_len, outs_len);
    env.buy_evm_gas(evm_cost)?;
    Ok(status as u8)
}

pub(crate) fn delegate_call_contract(
    mut env: WasmEnvMut,
    contract: u32,
    calldata: u32,
    calldata_len: u32,
    mut wasm_gas: u64,
    return_data_len: u32,
) -> Result<u8, Escape> {
    let mut env = WasmEnv::start(&mut env)?;
    env.pay_for_evm_copy(calldata_len as usize)?;
    wasm_gas = wasm_gas.min(env.gas_left().into()); // provide no more than what the user has

    let evm_gas = env.meter().pricing.wasm_to_evm(wasm_gas);
    let contract = env.read_bytes20(contract)?;
    let input = env.read_slice(calldata, calldata_len)?;

    let (outs_len, evm_cost, status) = env.evm().delegate_call(contract, input, evm_gas);
    env.set_return_data_len(outs_len);
    env.write_u32(return_data_len, outs_len);
    env.buy_evm_gas(evm_cost)?;
    Ok(status as u8)
}

pub(crate) fn static_call_contract(
    mut env: WasmEnvMut,
    contract: u32,
    calldata: u32,
    calldata_len: u32,
    mut wasm_gas: u64,
    return_data_len: u32,
) -> Result<u8, Escape> {
    let mut env = WasmEnv::start(&mut env)?;
    env.pay_for_evm_copy(calldata_len as usize)?;
    wasm_gas = wasm_gas.min(env.gas_left().into()); // provide no more than what the user has

    let evm_gas = env.meter().pricing.wasm_to_evm(wasm_gas);
    let contract = env.read_bytes20(contract)?;
    let input = env.read_slice(calldata, calldata_len)?;

    let (outs_len, evm_cost, status) = env.evm().static_call(contract, input, evm_gas);
    env.set_return_data_len(outs_len);
    env.write_u32(return_data_len, outs_len);
    env.buy_evm_gas(evm_cost)?;
    Ok(status as u8)
}

pub(crate) fn read_return_data(mut env: WasmEnvMut, dest: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    let len = env.return_data_len();
    env.pay_for_evm_copy(len as usize)?;

    let data = env.evm().load_return_data();
    env.write_slice(dest, &data)?;
    assert_eq!(data.len(), len as usize);
    Ok(())
}

pub(crate) fn emit_log(mut env: WasmEnvMut, data: u32, len: u32, topics: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    let topics: u64 = topics.into();
    let length: u64 = len.into();
    if length < topics * 32 || topics > 4 {
        return Escape::logical("bad topic data");
    }
    env.buy_evm_gas((1 + topics) * LOG_TOPIC_GAS)?;
    env.buy_evm_gas((length - topics * 32) * LOG_DATA_GAS)?;

    let data = env.read_slice(data, len)?;
    env.evm().emit_log(data, topics as usize)?;
    Ok(())
}

pub(crate) fn debug_println(mut env: WasmEnvMut, ptr: u32, len: u32) -> MaybeEscape {
    let memory = WasmEnv::memory(&mut env);
    let text = memory.read_slice(ptr, len)?;
    println!(
        "{} {}",
        "Stylus says:".yellow(),
        String::from_utf8_lossy(&text)
    );
    Ok(())
}
