// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

#![allow(clippy::too_many_arguments)]

use crate::env::{Escape, MaybeEscape, WasmEnv, WasmEnvMut};
use arbutil::{
    crypto,
    evm::{self, api::EvmApi, user::UserOutcomeKind},
    pricing::{EVM_API_INK, HOSTIO_INK, PTR_INK},
    Bytes20, Bytes32,
};
use eyre::Result;
use prover::{programs::prelude::*, value::Value};

macro_rules! be {
    ($int:expr) => {
        $int.to_be_bytes()
    };
}

macro_rules! trace {
    ($name:expr, $env:expr, [$($args:expr),+], $outs:expr) => {{
        trace!($name, $env, [$($args),+], $outs, ())
    }};
    ($name:expr, $env:expr, $args:expr, $outs:expr) => {{
        trace!($name, $env, $args, $outs, ())
    }};
    ($name:expr, $env:expr, [$($args:expr),+], [$($outs:expr),+], $ret:expr) => {{
        if $env.evm_data.tracing {
            let ink = $env.ink_ready()?;
            let mut args = vec![];
            $(args.extend($args);)*
            let mut outs = vec![];
            $(outs.extend($outs);)*
            $env.trace($name, &args, &outs, ink);
        }
        Ok($ret)
    }};
    ($name:expr, $env:expr, [$($args:expr),+], $outs:expr, $ret:expr) => {{
        if $env.evm_data.tracing {
            let ink = $env.ink_ready()?;
            let mut args = vec![];
            $(args.extend($args);)*
            $env.trace($name, &args, $outs.as_slice(), ink);
        }
        Ok($ret)
    }};
    ($name:expr, $env:expr, $args:expr, $outs:expr, $ret:expr) => {{
        if $env.evm_data.tracing {
            let ink = $env.ink_ready()?;
            $env.trace($name, $args.as_slice(), $outs.as_slice(), ink);
        }
        Ok($ret)
    }};
}

pub(crate) fn read_args<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env, 0)?;
    env.pay_for_write(env.args.len() as u64)?;
    env.write_slice(ptr, &env.args)?;
    trace!("read_args", env, env.args, &[])
}

pub(crate) fn write_result<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32, len: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env, 0)?;
    env.pay_for_read(len.into())?;
    env.outs = env.read_slice(ptr, len)?;
    trace!("write_result", env, &[], &env.outs)
}

pub(crate) fn storage_load_bytes32<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    key: u32,
    dest: u32,
) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env, 2 * PTR_INK + EVM_API_INK)?;
    let key = env.read_bytes32(key)?;

    let (value, gas_cost) = env.evm_api.get_bytes32(key);
    env.buy_gas(gas_cost)?;
    env.write_bytes32(dest, value)?;
    trace!("storage_load_bytes32", env, key, value)
}

pub(crate) fn storage_store_bytes32<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    key: u32,
    value: u32,
) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env, 2 * PTR_INK + EVM_API_INK)?;
    env.require_gas(evm::SSTORE_SENTRY_GAS)?; // see operations_acl_arbitrum.go

    let key = env.read_bytes32(key)?;
    let value = env.read_bytes32(value)?;

    let gas_cost = env.evm_api.set_bytes32(key, value)?;
    env.buy_gas(gas_cost)?;
    trace!("storage_store_bytes32", env, [key, value], [])
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
    let call = |api: &mut E, contract, data: &_, gas, value: Option<_>| {
        api.contract_call(contract, data, gas, value.unwrap())
    };
    do_call(env, contract, data, data_len, value, gas, ret_len, call, "")
}

pub(crate) fn delegate_call_contract<E: EvmApi>(
    env: WasmEnvMut<E>,
    contract: u32,
    data: u32,
    data_len: u32,
    gas: u64,
    ret_len: u32,
) -> Result<u8, Escape> {
    let call = |api: &mut E, contract, data: &_, gas, _| api.delegate_call(contract, data, gas);
    do_call(
        env, contract, data, data_len, None, gas, ret_len, call, "delegate",
    )
}

pub(crate) fn static_call_contract<E: EvmApi>(
    env: WasmEnvMut<E>,
    contract: u32,
    data: u32,
    data_len: u32,
    gas: u64,
    ret_len: u32,
) -> Result<u8, Escape> {
    let call = |api: &mut E, contract, data: &_, gas, _| api.static_call(contract, data, gas);
    do_call(
        env, contract, data, data_len, None, gas, ret_len, call, "static",
    )
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
    name: &str,
) -> Result<u8, Escape>
where
    E: EvmApi,
    F: FnOnce(&mut E, Bytes20, &[u8], u64, Option<Bytes32>) -> (u32, u64, UserOutcomeKind),
{
    let mut env = WasmEnv::start(&mut env, 3 * PTR_INK + EVM_API_INK)?;
    env.pay_for_read(calldata_len.into())?;
    gas = gas.min(env.gas_left()?); // provide no more than what the user has

    let contract = env.read_bytes20(contract)?;
    let input = env.read_slice(calldata, calldata_len)?;
    let value = value.map(|x| env.read_bytes32(x)).transpose()?;
    let api = &mut env.evm_api;

    let (outs_len, gas_cost, status) = call(api, contract, &input, gas, value);
    env.buy_gas(gas_cost)?;
    env.evm_data.return_data_len = outs_len;
    env.write_u32(return_data_len, outs_len)?;
    let status = status as u8;

    if env.evm_data.tracing {
        let underscore = (!name.is_empty()).then_some("_").unwrap_or_default();
        let name = format!("{name}{underscore}call_contract");
        return trace!(
            &name,
            env,
            [contract, &input, be!(gas), value.unwrap_or_default()],
            [be!(outs_len), be!(status)],
            status
        );
    }
    Ok(status)
}

pub(crate) fn create1<E: EvmApi>(
    env: WasmEnvMut<E>,
    code: u32,
    code_len: u32,
    endowment: u32,
    contract: u32,
    revert_data_len: u32,
) -> MaybeEscape {
    let call = |api: &mut E, code, value, _, gas| api.create1(code, value, gas);
    do_create(
        env,
        code,
        code_len,
        endowment,
        None,
        contract,
        revert_data_len,
        3 * PTR_INK + EVM_API_INK,
        call,
        "create1",
    )
}

pub(crate) fn create2<E: EvmApi>(
    env: WasmEnvMut<E>,
    code: u32,
    code_len: u32,
    endowment: u32,
    salt: u32,
    contract: u32,
    revert_data_len: u32,
) -> MaybeEscape {
    let call = |api: &mut E, code, value, salt: Option<_>, gas| {
        api.create2(code, value, salt.unwrap(), gas)
    };
    do_create(
        env,
        code,
        code_len,
        endowment,
        Some(salt),
        contract,
        revert_data_len,
        4 * PTR_INK + EVM_API_INK,
        call,
        "create2",
    )
}

pub(crate) fn do_create<F, E>(
    mut env: WasmEnvMut<E>,
    code: u32,
    code_len: u32,
    endowment: u32,
    salt: Option<u32>,
    contract: u32,
    revert_data_len: u32,
    cost: u64,
    call: F,
    name: &str,
) -> MaybeEscape
where
    E: EvmApi,
    F: FnOnce(&mut E, Vec<u8>, Bytes32, Option<Bytes32>, u64) -> (Result<Bytes20>, u32, u64),
{
    let mut env = WasmEnv::start(&mut env, cost)?;
    env.pay_for_read(code_len.into())?;

    let code = env.read_slice(code, code_len)?;
    let code_copy = env.evm_data.tracing.then(|| code.clone());

    let endowment = env.read_bytes32(endowment)?;
    let salt = salt.map(|x| env.read_bytes32(x)).transpose()?;
    let gas = env.gas_left()?;
    let api = &mut env.evm_api;

    let (result, ret_len, gas_cost) = call(api, code, endowment, salt, gas);
    let result = result?;

    env.buy_gas(gas_cost)?;
    env.evm_data.return_data_len = ret_len;
    env.write_u32(revert_data_len, ret_len)?;
    env.write_bytes20(contract, result)?;

    let salt = salt.unwrap_or_default();
    trace!(
        name,
        env,
        [code_copy.unwrap(), endowment, salt, be!(gas)],
        [result, be!(ret_len), be!(gas_cost)],
        ()
    )
}

pub(crate) fn read_return_data<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    dest: u32,
    offset: u32,
    size: u32,
) -> Result<u32, Escape> {
    let mut env = WasmEnv::start(&mut env, EVM_API_INK)?;
    env.pay_for_write(size.into())?;

    let data = env.evm_api.get_return_data(offset, size);
    assert!(data.len() <= size as usize);
    env.write_slice(dest, &data)?;

    let len = data.len() as u32;
    trace!("read_return_data", env, [be!(dest), be!(offset)], data, len)
}

pub(crate) fn return_data_size<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u32, Escape> {
    let mut env = WasmEnv::start(&mut env, 0)?;
    let len = env.evm_data.return_data_len;
    trace!("return_data_size", env, be!(len), [], len)
}

pub(crate) fn emit_log<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    data: u32,
    len: u32,
    topics: u32,
) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env, EVM_API_INK)?;
    if topics > 4 || len < topics * 32 {
        return Escape::logical("bad topic data");
    }
    env.pay_for_read(len.into())?;
    env.pay_for_evm_log(topics, len - topics * 32)?;

    let data = env.read_slice(data, len)?;
    env.evm_api.emit_log(data.clone(), topics)?;
    trace!("emit_log", env, [data, be!(topics)], [])
}

pub(crate) fn account_balance<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    address: u32,
    ptr: u32,
) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env, 2 * PTR_INK + EVM_API_INK)?;
    let address = env.read_bytes20(address)?;

    let (balance, gas_cost) = env.evm_api.account_balance(address);
    env.buy_gas(gas_cost)?;
    env.write_bytes32(ptr, balance)?;
    trace!("account_balance", env, [], balance)
}

pub(crate) fn account_codehash<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    address: u32,
    ptr: u32,
) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env, 2 * PTR_INK + EVM_API_INK)?;
    let address = env.read_bytes20(address)?;

    let (hash, gas_cost) = env.evm_api.account_codehash(address);
    env.buy_gas(gas_cost)?;
    env.write_bytes32(ptr, hash)?;
    trace!("account_codehash", env, [], hash)
}

pub(crate) fn evm_gas_left<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    let mut env = WasmEnv::start(&mut env, 0)?;
    let gas = env.gas_left()?;
    trace!("evm_gas_left", env, be!(gas), [], gas)
}

pub(crate) fn evm_ink_left<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    let mut env = WasmEnv::start(&mut env, 0)?;
    let ink = env.ink_ready()?;
    trace!("evm_ink_left", env, be!(ink), [], ink)
}

pub(crate) fn block_basefee<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env, PTR_INK)?;
    env.write_bytes32(ptr, env.evm_data.block_basefee)?;
    trace!("block_basefee", env, [], env.evm_data.block_basefee)
}

pub(crate) fn block_coinbase<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env, PTR_INK)?;
    env.write_bytes20(ptr, env.evm_data.block_coinbase)?;
    trace!("block_coinbase", env, [], env.evm_data.block_coinbase)
}

pub(crate) fn block_gas_limit<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    let mut env = WasmEnv::start(&mut env, 0)?;
    let limit = env.evm_data.block_gas_limit;
    trace!("block_gas_limit", env, [], be!(limit), limit)
}

pub(crate) fn block_number<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    let mut env = WasmEnv::start(&mut env, 0)?;
    let number = env.evm_data.block_number;
    trace!("block_number", env, [], be!(number), number)
}

pub(crate) fn block_timestamp<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    let mut env = WasmEnv::start(&mut env, 0)?;
    let timestamp = env.evm_data.block_timestamp;
    trace!("block_timestamp", env, [], be!(timestamp), timestamp)
}

pub(crate) fn chainid<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u64, Escape> {
    let mut env = WasmEnv::start(&mut env, 0)?;
    let chainid = env.evm_data.chainid;
    trace!("chainid", env, [], be!(chainid), chainid)
}

pub(crate) fn contract_address<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env, PTR_INK)?;
    env.write_bytes20(ptr, env.evm_data.contract_address)?;
    trace!("contract_address", env, [], env.evm_data.contract_address)
}

pub(crate) fn msg_reentrant<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u32, Escape> {
    let mut env = WasmEnv::start(&mut env, 0)?;
    let reentrant = env.evm_data.reentrant;
    trace!("msg_reentrant", env, [], be!(reentrant), reentrant)
}

pub(crate) fn msg_sender<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env, PTR_INK)?;
    env.write_bytes20(ptr, env.evm_data.msg_sender)?;
    trace!("msg_sender", env, [], env.evm_data.msg_sender)
}

pub(crate) fn msg_value<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env, PTR_INK)?;
    env.write_bytes32(ptr, env.evm_data.msg_value)?;
    trace!("msg_value", env, [], env.evm_data.msg_value)
}

pub(crate) fn native_keccak256<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    input: u32,
    len: u32,
    output: u32,
) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env, 0)?;
    env.pay_for_keccak(len.into())?;

    let preimage = env.read_slice(input, len)?;
    let digest = crypto::keccak(&preimage);
    env.write_bytes32(output, digest.into())?;
    trace!("native_keccak256", env, preimage, digest)
}

pub(crate) fn tx_gas_price<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env, PTR_INK)?;
    env.write_bytes32(ptr, env.evm_data.tx_gas_price)?;
    trace!("tx_gas_price", env, [], env.evm_data.tx_gas_price)
}

pub(crate) fn tx_ink_price<E: EvmApi>(mut env: WasmEnvMut<E>) -> Result<u32, Escape> {
    let mut env = WasmEnv::start(&mut env, 0)?;
    let ink_price = env.pricing().ink_price;
    trace!("tx_ink_price", env, [], be!(ink_price), ink_price)
}

pub(crate) fn tx_origin<E: EvmApi>(mut env: WasmEnvMut<E>, ptr: u32) -> MaybeEscape {
    let mut env = WasmEnv::start(&mut env, PTR_INK)?;
    env.write_bytes20(ptr, env.evm_data.tx_origin)?;
    trace!("tx_origin", env, [], env.evm_data.tx_origin)
}

pub(crate) fn memory_grow<E: EvmApi>(mut env: WasmEnvMut<E>, pages: u16) -> MaybeEscape {
    let mut env = WasmEnv::start_free(&mut env);
    if pages == 0 {
        env.buy_ink(HOSTIO_INK)?;
        return Ok(());
    }
    let gas_cost = env.evm_api.add_pages(pages);
    env.buy_gas(gas_cost)?;
    trace!("memory_grow", env, be!(pages), [])
}

pub(crate) fn console_log_text<E: EvmApi>(
    mut env: WasmEnvMut<E>,
    ptr: u32,
    len: u32,
) -> MaybeEscape {
    let mut env = WasmEnv::start_free(&mut env);
    let text = env.read_slice(ptr, len)?;
    env.say(String::from_utf8_lossy(&text));
    trace!("console_log_text", env, text, [])
}

pub(crate) fn console_log<E: EvmApi, T: Into<Value>>(
    mut env: WasmEnvMut<E>,
    value: T,
) -> MaybeEscape {
    let mut env = WasmEnv::start_free(&mut env);
    let value = value.into();
    env.say(value);
    trace!("console_log", env, [format!("{value}").as_bytes()], [])
}

pub(crate) fn console_tee<E: EvmApi, T: Into<Value> + Copy>(
    mut env: WasmEnvMut<E>,
    value: T,
) -> Result<T, Escape> {
    let env = WasmEnv::start_free(&mut env);
    env.say(value.into());
    Ok(value)
}

pub(crate) fn null_host<E: EvmApi>(_: WasmEnvMut<E>) {}
