use arbutil::{
    Bytes32,
    evm::{
        ARBOS_VERSION_STYLUS_CHARGING_FIXES, COLD_ACCOUNT_GAS, COLD_SLOAD_GAS, SSTORE_SENTRY_GAS,
        TLOAD_GAS, TSTORE_GAS, api::Gas, storage::StorageCache, user::UserOutcomeKind,
    },
    pricing::{EVM_API_INK, hostio},
};
use caller_env::{GuestPtr, MemAccess, wasmer_traits::WasmerMem};
use eyre::eyre;
use num_traits::Unsigned;
use prover::programs::meter::{GasMeteredMachine, MeteredMachine};
use wasmer::FunctionEnvMut;

use crate::{
    CallInputs, Escape, MaybeEscape, keccak,
    stylus::{StylusCustomEnvData, stylus_env},
};

pub fn msg_reentrant(mut ctx: FunctionEnvMut<StylusCustomEnvData>) -> Result<u32, Escape> {
    let data = ctx.data_mut();
    data.buy_ink(hostio::MSG_REENTRANT_BASE_INK)?;

    Ok(data.evm_data.reentrant)
}

pub fn read_args(mut ctx: FunctionEnvMut<StylusCustomEnvData>, ptr: GuestPtr) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::READ_ARGS_BASE_INK)?;
    data.pay_for_write(data.calldata.len() as u32)?;

    mem.write_slice(ptr, &data.calldata);

    Ok(())
}

pub fn storage_load_bytes32(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    key: GuestPtr,
    dest: GuestPtr,
) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::STORAGE_LOAD_BASE_INK)?;

    let arbos_version = data.evm_data.arbos_version;
    let evm_api_gas_to_use = if arbos_version < ARBOS_VERSION_STYLUS_CHARGING_FIXES {
        Gas(EVM_API_INK.0)
    } else {
        data.pricing().ink_to_gas(EVM_API_INK)
    };
    data.require_gas(COLD_SLOAD_GAS + StorageCache::REQUIRED_ACCESS_GAS + evm_api_gas_to_use)?;

    let key = mem.read_bytes32(key);

    let (value, gas_cost) = data.get_bytes32(key, evm_api_gas_to_use);
    data.buy_gas(gas_cost)?;
    mem.write_slice(dest, value.as_slice());

    Ok(())
}

pub fn transient_load_bytes32(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    key: GuestPtr,
    dest: GuestPtr,
) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::TRANSIENT_LOAD_BASE_INK)?;
    data.buy_gas(TLOAD_GAS)?;

    let key = mem.read_bytes32(key);
    let value = data.get_transient_bytes32(key);
    mem.write_slice(dest, value.as_slice());

    Ok(())
}

pub fn storage_cache_bytes32(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    key: GuestPtr,
    value: GuestPtr,
) -> MaybeEscape {
    let (mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::STORAGE_CACHE_BASE_INK)?;
    data.require_gas(SSTORE_SENTRY_GAS + StorageCache::REQUIRED_ACCESS_GAS)?;

    let key = mem.read_bytes32(key);
    let value = mem.read_bytes32(value);

    let gas_cost = data.cache_bytes32(key, value);
    data.buy_gas(gas_cost)?;

    Ok(())
}

pub fn transient_store_bytes32(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    key: GuestPtr,
    value: GuestPtr,
) -> MaybeEscape {
    let (mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::TRANSIENT_STORE_BASE_INK)?;
    data.buy_gas(TSTORE_GAS)?;

    let key = mem.read_bytes32(key);
    let value = mem.read_bytes32(value);

    data.set_transient_bytes32(key, value)?;

    Ok(())
}

pub fn storage_flush_cache(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    clear: u32,
) -> MaybeEscape {
    let data = ctx.data_mut();

    data.buy_ink(hostio::STORAGE_FLUSH_BASE_INK)?;
    data.require_gas(SSTORE_SENTRY_GAS)?;

    let gas_left = data.gas_left()?;
    let (gas_cost, outcome) = data.flush_storage_cache(clear != 0, gas_left)?;
    if data.evm_data.arbos_version >= ARBOS_VERSION_STYLUS_CHARGING_FIXES {
        data.buy_gas(gas_cost)?;
    }
    if outcome != UserOutcomeKind::Success {
        return Err(eyre!("outcome {outcome:?}").into());
    }

    Ok(())
}

pub fn write_result(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    ptr: GuestPtr,
    len: u32,
) -> MaybeEscape {
    let (mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::WRITE_RESULT_BASE_INK)?;
    data.pay_for_read(len)?;
    data.pay_for_read(len)?;

    data.outs = mem.read_slice(ptr, len as usize);

    Ok(())
}

pub fn pay_for_memory_grow(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    pages: u16,
) -> MaybeEscape {
    let data = ctx.data_mut();

    if pages == 0 {
        data.buy_ink(hostio::PAY_FOR_MEMORY_GROW_BASE_INK)?;
        return Ok(());
    }
    let gas_cost = data.add_pages(pages);
    data.buy_gas(gas_cost)?;

    Ok(())
}

pub fn exit_early(_ctx: FunctionEnvMut<StylusCustomEnvData>, status: u32) -> MaybeEscape {
    Err(Escape::Exit(status))
}

pub fn call_contract(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    contract: GuestPtr,
    data: GuestPtr,
    data_len: u32,
    value: GuestPtr,
    gas: u64,
    ret_len: GuestPtr,
) -> Result<u8, Escape> {
    let (mut mem, ctx_data) = stylus_env(&mut ctx);

    ctx_data.buy_ink(hostio::CALL_CONTRACT_BASE_INK)?;
    ctx_data.pay_for_read(data_len)?;
    ctx_data.pay_for_read(data_len)?;

    let CallInputs {
        contract,
        input,
        gas_left,
        gas_req,
        value,
    } = ctx_data.parse_call_inputs(&mem, contract, data, Gas(gas), data_len, Some(value))?;

    let (outs_len, gas_cost, status) =
        ctx_data.contract_call(contract, &input, gas_left, gas_req, value.unwrap());

    ctx_data.buy_gas(gas_cost)?;
    ctx_data.evm_data.return_data_len = outs_len;
    mem.write_u32(ret_len, outs_len);

    Ok(status as u8)
}

pub fn delegate_call_contract(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    contract: GuestPtr,
    data: GuestPtr,
    data_len: u32,
    gas: u64,
    ret_len: GuestPtr,
) -> Result<u8, Escape> {
    let (mut mem, ctx_data) = stylus_env(&mut ctx);

    ctx_data.buy_ink(hostio::CALL_CONTRACT_BASE_INK)?;
    ctx_data.pay_for_read(data_len)?;
    ctx_data.pay_for_read(data_len)?;

    let CallInputs {
        contract,
        input,
        gas_left,
        gas_req,
        ..
    } = ctx_data.parse_call_inputs(&mem, contract, data, Gas(gas), data_len, None)?;

    let (outs_len, gas_cost, status) = ctx_data.delegate_call(contract, &input, gas_left, gas_req);

    ctx_data.buy_gas(gas_cost)?;
    ctx_data.evm_data.return_data_len = outs_len;
    mem.write_u32(ret_len, outs_len);

    Ok(status as u8)
}

pub fn static_call_contract(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    contract: GuestPtr,
    data: GuestPtr,
    data_len: u32,
    gas: u64,
    ret_len: GuestPtr,
) -> Result<u8, Escape> {
    let (mut mem, ctx_data) = stylus_env(&mut ctx);

    ctx_data.buy_ink(hostio::CALL_CONTRACT_BASE_INK)?;
    ctx_data.pay_for_read(data_len)?;
    ctx_data.pay_for_read(data_len)?;

    let CallInputs {
        contract,
        input,
        gas_left,
        gas_req,
        ..
    } = ctx_data.parse_call_inputs(&mem, contract, data, Gas(gas), data_len, None)?;

    let (outs_len, gas_cost, status) = ctx_data.static_call(contract, &input, gas_left, gas_req);

    ctx_data.buy_gas(gas_cost)?;
    ctx_data.evm_data.return_data_len = outs_len;
    mem.write_u32(ret_len, outs_len);

    Ok(status as u8)
}

pub fn create1(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    code: GuestPtr,
    code_len: u32,
    endowment: GuestPtr,
    contract: GuestPtr,
    revert_data_len: GuestPtr,
) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::CREATE1_BASE_INK)?;
    data.pay_for_read(code_len)?;
    data.pay_for_read(code_len)?;

    let code = mem.read_slice(code, code_len as usize);
    let endowment = mem.read_bytes32(endowment);
    let gas = data.gas_left()?;

    let (result, ret_len, gas_cost) = data.create1(code, endowment, gas);
    let result = result?;

    data.buy_gas(gas_cost)?;
    data.evm_data.return_data_len = ret_len;
    mem.write_u32(revert_data_len, ret_len);
    mem.write_slice(contract, result.as_slice());

    Ok(())
}

pub fn create2(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    code: GuestPtr,
    code_len: u32,
    endowment: GuestPtr,
    salt: GuestPtr,
    contract: GuestPtr,
    revert_data_len: GuestPtr,
) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::CREATE2_BASE_INK)?;
    data.pay_for_read(code_len)?;
    data.pay_for_read(code_len)?;

    let code = mem.read_slice(code, code_len as usize);
    let endowment = mem.read_bytes32(endowment);
    let salt = mem.read_bytes32(salt);
    let gas = data.gas_left()?;

    let (result, ret_len, gas_cost) = data.create2(code, endowment, salt, gas);
    let result = result?;

    data.buy_gas(gas_cost)?;
    data.evm_data.return_data_len = ret_len;
    mem.write_u32(revert_data_len, ret_len);
    mem.write_slice(contract, result.as_slice());

    Ok(())
}

pub fn read_return_data(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    dest: GuestPtr,
    offset: u32,
    size: u32,
) -> Result<u32, Escape> {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::READ_RETURN_DATA_BASE_INK)?;

    let max = data.evm_data.return_data_len.saturating_sub(offset);
    data.pay_for_write(size.min(max))?;
    if max == 0 {
        return Ok(0);
    }

    let ret_data = data.get_return_data();
    let out_slice = slice_with_runoff(&ret_data, offset, offset.saturating_add(size));

    let out_len = out_slice.len() as u32;
    if out_len > 0 {
        mem.write_slice(dest, out_slice);
    }
    Ok(out_len)
}

pub fn return_data_size(mut ctx: FunctionEnvMut<StylusCustomEnvData>) -> Result<u32, Escape> {
    let data = ctx.data_mut();

    data.buy_ink(hostio::RETURN_DATA_SIZE_BASE_INK)?;
    Ok(data.evm_data.return_data_len)
}

pub fn emit_log(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    log_data: GuestPtr,
    len: u32,
    topics: u32,
) -> MaybeEscape {
    let (mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::EMIT_LOG_BASE_INK)?;
    if topics > 4 || len < topics * 32 {
        return Err("bad topic data".to_string().into());
    }
    data.pay_for_read(len)?;
    data.pay_for_evm_log(topics, len - topics * 32)?;

    let log_data = mem.read_slice(log_data, len as usize);
    data.emit_log(log_data, topics)
}

pub fn account_balance(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    address: GuestPtr,
    ptr: GuestPtr,
) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::ACCOUNT_BALANCE_BASE_INK)?;
    data.require_gas(COLD_ACCOUNT_GAS)?;
    let address = mem.read_bytes20(address);

    let (balance, gas_cost) = data.account_balance(address);
    data.buy_gas(gas_cost)?;
    mem.write_slice(ptr, balance.as_slice());

    Ok(())
}

pub fn account_code(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    address: GuestPtr,
    offset: u32,
    size: u32,
    dest: GuestPtr,
) -> Result<u32, Escape> {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::ACCOUNT_CODE_BASE_INK)?;
    data.require_gas(COLD_ACCOUNT_GAS)?;
    let address = mem.read_bytes20(address);
    let gas = data.gas_left()?;

    let arbos_version = data.evm_data.arbos_version;

    let (code, gas_cost) = data.account_code(arbos_version, address, gas);
    data.buy_gas(gas_cost)?;

    data.pay_for_write(code.len() as u32)?;

    let out_slice = slice_with_runoff(&code, offset, offset.saturating_add(size));
    let out_len = out_slice.len() as u32;
    mem.write_slice(dest, out_slice);

    Ok(out_len)
}

pub fn account_codehash(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    address: GuestPtr,
    ptr: GuestPtr,
) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::ACCOUNT_CODE_HASH_BASE_INK)?;
    data.require_gas(COLD_ACCOUNT_GAS)?;
    let address = mem.read_bytes20(address);

    let (hash, gas_cost) = data.account_codehash(address);
    data.buy_gas(gas_cost)?;
    mem.write_slice(ptr, hash.as_slice());

    Ok(())
}

pub fn account_code_size(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    address: GuestPtr,
) -> Result<u32, Escape> {
    let (mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::ACCOUNT_CODE_SIZE_BASE_INK)?;
    data.require_gas(COLD_ACCOUNT_GAS)?;
    let address = mem.read_bytes20(address);
    let gas = data.gas_left()?;

    let arbos_version = data.evm_data.arbos_version;

    let (code, gas_cost) = data.account_code(arbos_version, address, gas);
    data.buy_gas(gas_cost)?;

    Ok(code.len() as u32)
}

pub fn evm_gas_left(mut ctx: FunctionEnvMut<StylusCustomEnvData>) -> Result<u64, Escape> {
    let data = ctx.data_mut();

    data.buy_ink(hostio::EVM_GAS_LEFT_BASE_INK)?;
    Ok(data.gas_left()?.0)
}

pub fn evm_ink_left(mut ctx: FunctionEnvMut<StylusCustomEnvData>) -> Result<u64, Escape> {
    let data = ctx.data_mut();

    data.buy_ink(hostio::EVM_INK_LEFT_BASE_INK)?;
    Ok(data.ink_ready()?.0)
}

pub fn block_basefee(mut ctx: FunctionEnvMut<StylusCustomEnvData>, ptr: GuestPtr) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::BLOCK_BASEFEE_BASE_INK)?;
    mem.write_slice(ptr, data.evm_data.block_basefee.as_slice());

    Ok(())
}

pub fn chainid(mut ctx: FunctionEnvMut<StylusCustomEnvData>) -> Result<u64, Escape> {
    let data = ctx.data_mut();

    data.buy_ink(hostio::CHAIN_ID_BASE_INK)?;
    Ok(data.evm_data.chainid)
}

pub fn block_coinbase(mut ctx: FunctionEnvMut<StylusCustomEnvData>, ptr: GuestPtr) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::BLOCK_COINBASE_BASE_INK)?;
    mem.write_slice(ptr, data.evm_data.block_coinbase.as_slice());

    Ok(())
}

pub fn block_gas_limit(mut ctx: FunctionEnvMut<StylusCustomEnvData>) -> Result<u64, Escape> {
    let data = ctx.data_mut();

    data.buy_ink(hostio::BLOCK_GAS_LIMIT_BASE_INK)?;
    Ok(data.evm_data.block_gas_limit)
}

pub fn block_number(mut ctx: FunctionEnvMut<StylusCustomEnvData>) -> Result<u64, Escape> {
    let data = ctx.data_mut();

    data.buy_ink(hostio::BLOCK_NUMBER_BASE_INK)?;
    Ok(data.evm_data.block_number)
}

pub fn block_timestamp(mut ctx: FunctionEnvMut<StylusCustomEnvData>) -> Result<u64, Escape> {
    let data = ctx.data_mut();

    data.buy_ink(hostio::BLOCK_TIMESTAMP_BASE_INK)?;
    Ok(data.evm_data.block_timestamp)
}

pub fn contract_address(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    ptr: GuestPtr,
) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::ADDRESS_BASE_INK)?;
    mem.write_slice(ptr, data.evm_data.contract_address.as_slice());

    Ok(())
}

type U256 = ruint2::Uint<256, 4>;

fn read_u256(mem: &WasmerMem, ptr: GuestPtr) -> (U256, Bytes32) {
    let bytes = mem.read_bytes32(ptr);
    (bytes.into(), bytes)
}

pub fn math_div(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    value: GuestPtr,
    divisor: GuestPtr,
) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::MATH_DIV_BASE_INK)?;
    let (a, _) = read_u256(&mem, value);
    let (b, _) = read_u256(&mem, divisor);

    let result: Bytes32 = a.checked_div(b).unwrap_or_default().into();
    mem.write_slice(value, result.as_slice());

    Ok(())
}

pub fn math_mod(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    value: GuestPtr,
    modulus: GuestPtr,
) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::MATH_MOD_BASE_INK)?;
    let (a, _) = read_u256(&mem, value);
    let (b, _) = read_u256(&mem, modulus);

    let result: Bytes32 = a.checked_rem(b).unwrap_or_default().into();
    mem.write_slice(value, result.as_slice());

    Ok(())
}

pub fn math_pow(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    value: GuestPtr,
    exponent: GuestPtr,
) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::MATH_POW_BASE_INK)?;
    let (a, _) = read_u256(&mem, value);
    let (b, b32) = read_u256(&mem, exponent);

    data.pay_for_pow(&b32)?;
    let result: Bytes32 = a.wrapping_pow(b).into();
    mem.write_slice(value, result.as_slice());

    Ok(())
}

pub fn math_add_mod(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    value: GuestPtr,
    addend: GuestPtr,
    modulus: GuestPtr,
) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::MATH_ADD_MOD_BASE_INK)?;
    let (a, _) = read_u256(&mem, value);
    let (b, _) = read_u256(&mem, addend);
    let (c, _) = read_u256(&mem, modulus);

    let result: Bytes32 = a.add_mod(b, c).into();
    mem.write_slice(value, result.as_slice());

    Ok(())
}

pub fn math_mul_mod(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    value: GuestPtr,
    multiplier: GuestPtr,
    modulus: GuestPtr,
) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::MATH_MUL_MOD_BASE_INK)?;
    let (a, _) = read_u256(&mem, value);
    let (b, _) = read_u256(&mem, multiplier);
    let (c, _) = read_u256(&mem, modulus);

    let result: Bytes32 = a.mul_mod(b, c).into();
    mem.write_slice(value, result.as_slice());

    Ok(())
}

pub fn msg_sender(mut ctx: FunctionEnvMut<StylusCustomEnvData>, ptr: GuestPtr) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::MSG_SENDER_BASE_INK)?;
    mem.write_slice(ptr, data.evm_data.msg_sender.as_slice());

    Ok(())
}

pub fn msg_value(mut ctx: FunctionEnvMut<StylusCustomEnvData>, ptr: GuestPtr) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::MSG_VALUE_BASE_INK)?;
    mem.write_slice(ptr, data.evm_data.msg_value.as_slice());

    Ok(())
}

pub fn tx_gas_price(mut ctx: FunctionEnvMut<StylusCustomEnvData>, ptr: GuestPtr) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::TX_GAS_PRICE_BASE_INK)?;
    mem.write_slice(ptr, data.evm_data.tx_gas_price.as_slice());

    Ok(())
}

pub fn tx_ink_price(mut ctx: FunctionEnvMut<StylusCustomEnvData>) -> Result<u32, Escape> {
    let data = ctx.data_mut();

    data.buy_ink(hostio::TX_INK_PRICE_BASE_INK)?;
    Ok(data.pricing().ink_price)
}

pub fn tx_origin(mut ctx: FunctionEnvMut<StylusCustomEnvData>, ptr: GuestPtr) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.buy_ink(hostio::TX_ORIGIN_BASE_INK)?;
    mem.write_slice(ptr, data.evm_data.tx_origin.as_slice());

    Ok(())
}

pub fn native_keccak256(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    input: GuestPtr,
    len: u32,
    output: GuestPtr,
) -> MaybeEscape {
    let (mut mem, data) = stylus_env(&mut ctx);

    data.pay_for_keccak(len)?;
    let preimage = mem.read_slice(input, len as usize);
    let digest = keccak(&preimage);
    mem.write_slice(output, &digest);

    Ok(())
}

fn slice_with_runoff<T, I>(data: &impl AsRef<[T]>, start: I, end: I) -> &[T]
where
    I: TryInto<usize> + Unsigned,
{
    let start = start.try_into().unwrap_or(usize::MAX);
    let end = end.try_into().unwrap_or(usize::MAX);

    let data = data.as_ref();
    if start >= data.len() || end < start {
        return &[];
    }
    &data[start..end.min(data.len())]
}
