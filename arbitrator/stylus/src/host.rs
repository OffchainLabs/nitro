// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

use crate::{
    api::EvmApi,
    env::{Escape, MaybeEscape, WasmEnv, WasmEnvMut},
};
use arbutil::evm;
use prover::{programs::prelude::*, value::Value};

pub(crate) fn read_args<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    env.pay_for_evm_copy(env.args.len() as u64)?;
    env.write_slice(ptr, &env.args)?;
    Ok(())
}

pub(crate) fn return_data<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32, len: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    env.pay_for_evm_copy(len.into())?;
    env.outs = env.read_slice(ptr, len)?;
    Ok(())
}

pub(crate) fn account_load_bytes32<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    key: u32,
    dest: u32,
) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    let key = env.read_bytes32(key)?;
    let (value, gas_cost) = env.evm.get_bytes32(key);
    env.write_slice(dest, &value.0)?;
    env.buy_gas(gas_cost)
}

pub(crate) fn account_store_bytes32<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    key: u32,
    value: u32,
) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    env.require_gas(evm::SSTORE_SENTRY_GAS)?; // see operations_acl_arbitrum.go

    let key = env.read_bytes32(key)?;
    let value = env.read_bytes32(value)?;
    let gas_cost = env.evm.set_bytes32(key, value)?;
    env.buy_gas(gas_cost)
}

pub(crate) fn call_contract<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    contract: u32,
    calldata: u32,
    calldata_len: u32,
    value: u32,
    mut ink: u64,
    return_data_len: u32,
) -> Result<u8, Escape> {
    let mut env = WasmEnv::start(&mut env)?;
    env.pay_for_evm_copy(calldata_len.into())?;
    ink = ink.min(env.ink_left().into()); // provide no more than what the user has

    let gas = env.pricing().ink_to_gas(ink);
    let contract = env.read_bytes20(contract)?;
    let input = env.read_slice(calldata, calldata_len)?;
    let value = env.read_bytes32(value)?;

    let (outs_len, gas_cost, status) = env.evm.contract_call(contract, input, gas, value);
    env.evm_data.return_data_len = outs_len;
    env.write_u32(return_data_len, outs_len)?;
    env.buy_gas(gas_cost)?;
    Ok(status as u8)
}

pub(crate) fn delegate_call_contract<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    contract: u32,
    calldata: u32,
    calldata_len: u32,
    mut ink: u64,
    return_data_len: u32,
) -> Result<u8, Escape> {
    let mut env = WasmEnv::start(&mut env)?;
    env.pay_for_evm_copy(calldata_len.into())?;
    ink = ink.min(env.ink_left().into()); // provide no more than what the user has

    let gas = env.pricing().ink_to_gas(ink);
    let contract = env.read_bytes20(contract)?;
    let input = env.read_slice(calldata, calldata_len)?;

    let (outs_len, gas_cost, status) = env.evm.delegate_call(contract, input, gas);
    env.evm_data.return_data_len = outs_len;
    env.write_u32(return_data_len, outs_len)?;
    env.buy_gas(gas_cost)?;
    Ok(status as u8)
}

pub(crate) fn static_call_contract<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    contract: u32,
    calldata: u32,
    calldata_len: u32,
    mut ink: u64,
    return_data_len: u32,
) -> Result<u8, Escape> {
    let mut env = WasmEnv::start(&mut env)?;
    env.pay_for_evm_copy(calldata_len.into())?;
    ink = ink.min(env.ink_left().into()); // provide no more than what the user has

    let gas = env.pricing().ink_to_gas(ink);
    let contract = env.read_bytes20(contract)?;
    let input = env.read_slice(calldata, calldata_len)?;

    let (outs_len, gas_cost, status) = env.evm.static_call(contract, input, gas);
    env.evm_data.return_data_len = outs_len;
    env.write_u32(return_data_len, outs_len)?;
    env.buy_gas(gas_cost)?;
    Ok(status as u8)
}

pub(crate) fn create1<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    code: u32,
    code_len: u32,
    endowment: u32,
    contract: u32,
    revert_data_len: u32,
) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    env.pay_for_evm_copy(code_len.into())?;

    let code = env.read_slice(code, code_len)?;
    let endowment = env.read_bytes32(endowment)?;
    let gas = env.gas_left();

    let (result, ret_len, gas_cost) = env.evm.create1(code, endowment, gas);
    env.evm_data.return_data_len = ret_len;
    env.write_u32(revert_data_len, ret_len)?;
    env.buy_gas(gas_cost)?;
    env.write_bytes20(contract, result?)?;
    Ok(())
}

pub(crate) fn create2<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    code: u32,
    code_len: u32,
    endowment: u32,
    salt: u32,
    contract: u32,
    revert_data_len: u32,
) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    env.pay_for_evm_copy(code_len.into())?;

    let code = env.read_slice(code, code_len)?;
    let endowment = env.read_bytes32(endowment)?;
    let salt = env.read_bytes32(salt)?;
    let gas = env.gas_left();

    let (result, ret_len, gas_cost) = env.evm.create2(code, endowment, salt, gas);
    env.evm_data.return_data_len = ret_len;
    env.write_u32(revert_data_len, ret_len)?;
    env.buy_gas(gas_cost)?;
    env.write_bytes20(contract, result?)?;
    Ok(())
}

pub(crate) fn read_return_data<E: EvmApi>(mut env: WasmEnvMut<E>, dest: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    let len = env.evm_data.return_data_len;
    env.pay_for_evm_copy(len.into())?;

    let data = env.evm.load_return_data();
    env.write_slice(dest, &data)?;
    assert_eq!(data.len(), len as usize);
    Ok(())
}

pub(crate) fn return_data_size<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u32, Escape> {
    let env = WasmEnv::start(&mut env)?;
    let len = env.evm_data.return_data_len;
    Ok(len)
}

pub(crate) fn emit_log<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    data: u32,
    len: u32,
    topics: u32,
) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    let topics: u64 = topics.into();
    let length: u64 = len.into();
    if length < topics * 32 || topics > 4 {
        return Escape::logical("bad topic data");
    }
    env.buy_gas((1 + topics) * evm::LOG_TOPIC_GAS)?;
    env.buy_gas((length - topics * 32) * evm::LOG_DATA_GAS)?;

    let data = env.read_slice(data, len)?;
    env.evm.emit_log(data, topics as usize)?;
    Ok(())
}

pub(crate) fn tx_origin<E: EvmApi>(mut env: WasmEnvMut<E>, data: u32) -> MaybeEscape {
    let env = WasmEnv::start(&mut env)?;
    let origin = env.evm_data.origin;
    env.write_bytes20(data, origin)?;
    Ok(())
}

pub(crate) fn console_log_text<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    ptr: u32,
    len: u32,
) -> MaybeEscape {
    let env = WasmEnv::start_free(&mut env);
    let text = env.read_slice(ptr, len)?;
    env.say(String::from_utf8_lossy(&text));
    Ok(())
}

pub(crate) fn console_log<E: EvmApi, T: Into<Value>>(
    mut env: WasmEnvMut<E>,
    value: T,
) -> MaybeEscape {
    let env = WasmEnv::start_free(&mut env);
    env.say(value.into());
    Ok(())
}

pub(crate) fn console_tee<E: EvmApi, T: Into<Value> + Copy>(
    mut env: WasmEnvMut<E>,
    value: T,
) -> Result<T, Escape> {
    let env = WasmEnv::start_free(&mut env);
    env.say(value.into());
    Ok(value)
}
