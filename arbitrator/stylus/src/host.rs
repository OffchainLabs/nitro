// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![allow(clippy::too_many_arguments)]

use crate::env::{Escape, MaybeEscape, WasmEnv, WasmEnvMut};
use arbutil::{
    evm::{self, api::EvmApi, user::UserOutcomeKind},
    Bytes20, Bytes32,
};
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
    let (value, gas_cost) = env.evm_api.get_bytes32(key);
    env.write_slice(dest, &value.0)?;
    env.buy_gas(gas_cost)?;
    Ok(())
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
    let gas_cost = env.evm_api.set_bytes32(key, value)?;
    env.buy_gas(gas_cost)?;
    Ok(())
}

pub(crate) fn call_contract<E: EvmApi>(
    env: WasmEnvMut<E>,
    contract: u32,
    data: u32,
    data_len: u32,
    value: u32,
    gas: u64,
    ret_len: u32,
) -> Result<u8, Escape> {
    let value = Some(value);
    let call = |api: &mut E, contract, data, gas, value: Option<_>| {
        api.contract_call(contract, data, gas, value.unwrap())
    };
    do_call(env, contract, data, data_len, value, gas, ret_len, call)
}

pub(crate) fn delegate_call_contract<E: EvmApi>(
    env: WasmEnvMut<E>,
    contract: u32,
    data: u32,
    data_len: u32,
    gas: u64,
    ret_len: u32,
) -> Result<u8, Escape> {
    let call = |api: &mut E, contract, data, gas, _| api.delegate_call(contract, data, gas);
    do_call(env, contract, data, data_len, None, gas, ret_len, call)
}

pub(crate) fn static_call_contract<E: EvmApi>(
    env: WasmEnvMut<E>,
    contract: u32,
    data: u32,
    data_len: u32,
    gas: u64,
    ret_len: u32,
) -> Result<u8, Escape> {
    let call = |api: &mut E, contract, data, gas, _| api.static_call(contract, data, gas);
    do_call(env, contract, data, data_len, None, gas, ret_len, call)
}

pub(crate) fn do_call<F, E>(
    mut env: WasmEnvMut<E>,
    contract: u32,
    calldata: u32,
    calldata_len: u32,
    value: Option<u32>,
    mut gas: u64,
    return_data_len: u32,
    call: F,
) -> Result<u8, Escape>
where
    E: EvmApi,
    F: FnOnce(&mut E, Bytes20, Vec<u8>, u64, Option<Bytes32>) -> (u32, u64, UserOutcomeKind),
{
    let mut env = WasmEnv::start(&mut env)?;
    env.pay_for_evm_copy(calldata_len.into())?;
    gas = gas.min(env.gas_left()?); // provide no more than what the user has

    let contract = env.read_bytes20(contract)?;
    let input = env.read_slice(calldata, calldata_len)?;
    let value = value.map(|x| env.read_bytes32(x)).transpose()?;
    let api = &mut env.evm_api;

    let (outs_len, gas_cost, status) = call(api, contract, input, gas, value);
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
    let gas = env.gas_left()?;

    let (result, ret_len, gas_cost) = env.evm_api.create1(code, endowment, gas);
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
    let gas = env.gas_left()?;

    let (result, ret_len, gas_cost) = env.evm_api.create2(code, endowment, salt, gas);
    env.evm_data.return_data_len = ret_len;
    env.write_u32(revert_data_len, ret_len)?;
    env.buy_gas(gas_cost)?;
    env.write_bytes20(contract, result?)?;
    Ok(())
}

pub(crate) fn read_return_data<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    dest: u32,
    offset: u32,
    size: u32,
) -> Result<u32, Escape> {
    let mut env = WasmEnv::start(&mut env)?;

    let data = env.evm_api.get_return_data(offset, size);
    env.pay_for_evm_copy(data.len() as u64)?;
    assert!(data.len() <= size as usize);
    env.write_slice(dest, &data)?;
    Ok(data.len() as u32)
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
    if topics > 4 || len < topics * 32 {
        return Escape::logical("bad topic data");
    }
    env.pay_for_evm_log(topics, len - topics * 32)?;

    let data = env.read_slice(data, len)?;
    env.evm_api.emit_log(data, topics)?;
    Ok(())
}

pub(crate) fn account_balance<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    address: u32,
    ptr: u32,
) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    let address = env.read_bytes20(address)?;
    let (balance, gas_cost) = env.evm_api.account_balance(address);
    env.write_slice(ptr, &balance.0)?;
    env.buy_gas(gas_cost)?;
    Ok(())
}

pub(crate) fn account_codehash<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    address: u32,
    ptr: u32,
) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    let address = env.read_bytes20(address)?;
    let (hash, gas_cost) = env.evm_api.account_codehash(address);
    env.write_slice(ptr, &hash.0)?;
    env.buy_gas(gas_cost)?;
    Ok(())
}

pub(crate) fn evm_gas_left<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    let mut env = WasmEnv::start(&mut env)?;
    env.buy_gas(evm::GASLEFT_GAS)?;
    Ok(env.gas_left()?)
}

pub(crate) fn evm_ink_left<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    let mut env = WasmEnv::start(&mut env)?;
    env.buy_gas(evm::GASLEFT_GAS)?;
    Ok(env.ink_ready()?)
}

pub(crate) fn block_basefee<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    env.buy_gas(evm::BASEFEE_GAS)?;
    env.write_bytes32(ptr, env.evm_data.block_basefee)?;
    Ok(())
}

pub(crate) fn block_chainid<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    env.buy_gas(evm::CHAINID_GAS)?;
    env.write_bytes32(ptr, env.evm_data.block_chainid)?;
    Ok(())
}

pub(crate) fn block_coinbase<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    env.buy_gas(evm::COINBASE_GAS)?;
    env.write_bytes20(ptr, env.evm_data.block_coinbase)?;
    Ok(())
}

pub(crate) fn block_gas_limit<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    let mut env = WasmEnv::start(&mut env)?;
    env.buy_gas(evm::GASLIMIT_GAS)?;
    Ok(env.evm_data.block_gas_limit)
}

pub(crate) fn block_number<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    env.buy_gas(evm::NUMBER_GAS)?;
    env.write_bytes32(ptr, env.evm_data.block_number)?;
    Ok(())
}

pub(crate) fn block_timestamp<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    let mut env = WasmEnv::start(&mut env)?;
    env.buy_gas(evm::TIMESTAMP_GAS)?;
    Ok(env.evm_data.block_timestamp)
}

pub(crate) fn contract_address<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    env.buy_gas(evm::ADDRESS_GAS)?;
    env.write_bytes20(ptr, env.evm_data.contract_address)?;
    Ok(())
}

pub(crate) fn msg_sender<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    env.buy_gas(evm::CALLER_GAS)?;
    env.write_bytes20(ptr, env.evm_data.msg_sender)?;
    Ok(())
}

pub(crate) fn msg_value<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    env.buy_gas(evm::CALLVALUE_GAS)?;
    env.write_bytes32(ptr, env.evm_data.msg_value)?;
    Ok(())
}

pub(crate) fn tx_gas_price<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    env.buy_gas(evm::GASPRICE_GAS)?;
    env.write_bytes32(ptr, env.evm_data.tx_gas_price)?;
    Ok(())
}

pub(crate) fn tx_ink_price<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    let mut env = WasmEnv::start(&mut env)?;
    env.buy_gas(evm::GASPRICE_GAS)?;
    Ok(env.pricing().ink_price)
}

pub(crate) fn tx_origin<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    env.buy_gas(evm::ORIGIN_GAS)?;
    env.write_bytes20(ptr, env.evm_data.tx_origin)?;
    Ok(())
}

pub(crate) fn memory_grow<E: EvmApi>(mut env: WasmEnvMut<E>, pages: u16) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env)?;
    let gas_cost = env.evm_api.add_pages(pages);
    env.buy_gas(gas_cost)?;
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
