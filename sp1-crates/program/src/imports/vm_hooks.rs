use crate::{
    CallInputs, Escape, MaybeEscape, Ptr, keccak, read_bytes20, read_bytes32, read_slice,
    stylus::StylusCustomEnvData,
};
use arbutil::{
    Bytes32,
    evm::{
        ARBOS_VERSION_STYLUS_CHARGING_FIXES, COLD_ACCOUNT_GAS, COLD_SLOAD_GAS, SSTORE_SENTRY_GAS,
        TLOAD_GAS, TSTORE_GAS, api::Gas, storage::StorageCache, user::UserOutcomeKind,
    },
    pricing::{EVM_API_INK, hostio},
};
use eyre::eyre;
use prover::programs::meter::{GasMeteredMachine, MeteredMachine};
use wasmer::{FunctionEnvMut, MemoryView};

pub fn msg_reentrant(mut ctx: FunctionEnvMut<StylusCustomEnvData>) -> u32 {
    let data = ctx.data_mut();
    data.buy_ink(hostio::MSG_REENTRANT_BASE_INK)
        .expect("buy ink");

    data.evm_data.reentrant
}

pub fn read_args(mut ctx: FunctionEnvMut<StylusCustomEnvData>, ptr: Ptr) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::READ_ARGS_BASE_INK)?;
    data.pay_for_write(data.calldata.len() as u32)?;

    memory.write(ptr.offset() as u64, &data.calldata)?;

    Ok(())
}

pub fn storage_load_bytes32(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    key: Ptr,
    dest: Ptr,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::STORAGE_LOAD_BASE_INK)?;

    let arbos_version = data.evm_data.arbos_version;
    let evm_api_gas_to_use = if arbos_version < ARBOS_VERSION_STYLUS_CHARGING_FIXES {
        Gas(EVM_API_INK.0)
    } else {
        data.pricing().ink_to_gas(EVM_API_INK)
    };
    data.require_gas(COLD_SLOAD_GAS + StorageCache::REQUIRED_ACCESS_GAS + evm_api_gas_to_use)?;

    let key = read_bytes32(key, &memory)?;

    let (value, gas_cost) = data.get_bytes32(key, evm_api_gas_to_use);
    data.buy_gas(gas_cost)?;
    memory.write(dest.offset() as u64, value.as_slice())?;

    Ok(())
}

pub fn transient_load_bytes32(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    key: Ptr,
    dest: Ptr,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::TRANSIENT_LOAD_BASE_INK)?;
    data.buy_gas(TLOAD_GAS)?;

    let key = read_bytes32(key, &memory)?;
    let value = data.get_transient_bytes32(key);
    memory.write(dest.offset() as u64, value.as_slice())?;

    Ok(())
}

pub fn storage_cache_bytes32(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    key: Ptr,
    value: Ptr,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::STORAGE_CACHE_BASE_INK)?;
    data.require_gas(SSTORE_SENTRY_GAS + StorageCache::REQUIRED_ACCESS_GAS)?;

    let key = read_bytes32(key, &memory)?;
    let value = read_bytes32(value, &memory)?;

    let gas_cost = data.cache_bytes32(key, value);
    data.buy_gas(gas_cost)?;

    Ok(())
}

pub fn transient_store_bytes32(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    key: Ptr,
    value: Ptr,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::TRANSIENT_STORE_BASE_INK)?;
    data.buy_gas(TSTORE_GAS)?;

    let key = read_bytes32(key, &memory)?;
    let value = read_bytes32(value, &memory)?;

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
    ptr: Ptr,
    len: u32,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::WRITE_RESULT_BASE_INK)?;
    data.pay_for_read(len)?;
    data.pay_for_read(len)?;

    data.outs = read_slice(ptr, len as usize, &memory)?;

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
    contract: Ptr,
    data: Ptr,
    data_len: u32,
    value: Ptr,
    gas: u64,
    ret_len: Ptr,
) -> Result<u8, Escape> {
    let (ctx_data, store) = ctx.data_and_store_mut();
    let memory = ctx_data.memory.clone().unwrap().view(&store);

    ctx_data.buy_ink(hostio::CALL_CONTRACT_BASE_INK)?;
    ctx_data.pay_for_read(data_len)?;
    ctx_data.pay_for_read(data_len)?;

    let CallInputs {
        contract,
        input,
        gas_left,
        gas_req,
        value,
    } = ctx_data.parse_call_inputs(&memory, contract, data, Gas(gas), data_len, Some(value))?;

    let (outs_len, gas_cost, status) =
        ctx_data.contract_call(contract, &input, gas_left, gas_req, value.unwrap());

    ctx_data.buy_gas(gas_cost)?;
    ctx_data.evm_data.return_data_len = outs_len;
    ret_len.write(&memory, outs_len)?;

    Ok(status as u8)
}

pub fn delegate_call_contract(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    contract: Ptr,
    data: Ptr,
    data_len: u32,
    gas: u64,
    ret_len: Ptr,
) -> Result<u8, Escape> {
    let (ctx_data, store) = ctx.data_and_store_mut();
    let memory = ctx_data.memory.clone().unwrap().view(&store);

    ctx_data.buy_ink(hostio::CALL_CONTRACT_BASE_INK)?;
    ctx_data.pay_for_read(data_len)?;
    ctx_data.pay_for_read(data_len)?;

    let CallInputs {
        contract,
        input,
        gas_left,
        gas_req,
        ..
    } = ctx_data.parse_call_inputs(&memory, contract, data, Gas(gas), data_len, None)?;

    let (outs_len, gas_cost, status) = ctx_data.delegate_call(contract, &input, gas_left, gas_req);

    ctx_data.buy_gas(gas_cost)?;
    ctx_data.evm_data.return_data_len = outs_len;
    ret_len.write(&memory, outs_len)?;

    Ok(status as u8)
}

pub fn static_call_contract(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    contract: Ptr,
    data: Ptr,
    data_len: u32,
    gas: u64,
    ret_len: Ptr,
) -> Result<u8, Escape> {
    let (ctx_data, store) = ctx.data_and_store_mut();
    let memory = ctx_data.memory.clone().unwrap().view(&store);

    ctx_data.buy_ink(hostio::CALL_CONTRACT_BASE_INK)?;
    ctx_data.pay_for_read(data_len)?;
    ctx_data.pay_for_read(data_len)?;

    let CallInputs {
        contract,
        input,
        gas_left,
        gas_req,
        ..
    } = ctx_data.parse_call_inputs(&memory, contract, data, Gas(gas), data_len, None)?;

    let (outs_len, gas_cost, status) = ctx_data.static_call(contract, &input, gas_left, gas_req);

    ctx_data.buy_gas(gas_cost)?;
    ctx_data.evm_data.return_data_len = outs_len;
    ret_len.write(&memory, outs_len)?;

    Ok(status as u8)
}

pub fn create1(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    code: Ptr,
    code_len: u32,
    endowment: Ptr,
    contract: Ptr,
    revert_data_len: Ptr,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::CREATE1_BASE_INK)?;
    data.pay_for_read(code_len)?;
    data.pay_for_read(code_len)?;

    let code = read_slice(code, code_len as usize, &memory)?;
    let endowment = read_bytes32(endowment, &memory)?;
    let gas = data.gas_left()?;

    let (result, ret_len, gas_cost) = data.create1(code, endowment, gas);
    let result = result?;

    data.buy_gas(gas_cost)?;
    data.evm_data.return_data_len = ret_len;
    revert_data_len.write(&memory, ret_len)?;
    memory.write(contract.offset() as u64, result.as_slice())?;

    Ok(())
}

pub fn create2(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    code: Ptr,
    code_len: u32,
    endowment: Ptr,
    salt: Ptr,
    contract: Ptr,
    revert_data_len: Ptr,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::CREATE2_BASE_INK)?;
    data.pay_for_read(code_len)?;
    data.pay_for_read(code_len)?;

    let code = read_slice(code, code_len as usize, &memory)?;
    let endowment = read_bytes32(endowment, &memory)?;
    let salt = read_bytes32(salt, &memory)?;
    let gas = data.gas_left()?;

    let (result, ret_len, gas_cost) = data.create2(code, endowment, salt, gas);
    let result = result?;

    data.buy_gas(gas_cost)?;
    data.evm_data.return_data_len = ret_len;
    revert_data_len.write(&memory, ret_len)?;
    memory.write(contract.offset() as u64, result.as_slice())?;

    Ok(())
}

pub fn read_return_data(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    dest: Ptr,
    offset: u32,
    size: u32,
) -> Result<u32, Escape> {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

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
        memory.write(dest.offset() as u64, out_slice)?;
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
    log_data: Ptr,
    len: u32,
    topics: u32,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::EMIT_LOG_BASE_INK)?;
    if topics > 4 || len < topics * 32 {
        return Err("bad topic data".to_string().into());
    }
    data.pay_for_read(len)?;
    data.pay_for_evm_log(topics, len - topics * 32)?;

    let log_data = read_slice(log_data, len as usize, &memory)?;
    data.emit_log(log_data, topics)
}

pub fn account_balance(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    address: Ptr,
    ptr: Ptr,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::ACCOUNT_BALANCE_BASE_INK)?;
    data.require_gas(COLD_ACCOUNT_GAS)?;
    let address = read_bytes20(address, &memory)?;

    let (balance, gas_cost) = data.account_balance(address);
    data.buy_gas(gas_cost)?;
    memory.write(ptr.offset() as u64, balance.as_slice())?;

    Ok(())
}

pub fn account_code(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    address: Ptr,
    offset: u32,
    size: u32,
    dest: Ptr,
) -> Result<u32, Escape> {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::ACCOUNT_CODE_BASE_INK)?;
    data.require_gas(COLD_ACCOUNT_GAS)?;
    let address = read_bytes20(address, &memory)?;
    let gas = data.gas_left()?;

    let arbos_version = data.evm_data.arbos_version;

    let (code, gas_cost) = data.account_code(arbos_version, address, gas);
    data.buy_gas(gas_cost)?;

    data.pay_for_write(code.len() as u32)?;

    let out_slice = slice_with_runoff(&code, offset, offset.saturating_add(size));
    let out_len = out_slice.len() as u32;
    memory.write(dest.offset() as u64, out_slice)?;

    Ok(out_len)
}

pub fn account_codehash(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    address: Ptr,
    ptr: Ptr,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::ACCOUNT_CODE_HASH_BASE_INK)?;
    data.require_gas(COLD_ACCOUNT_GAS)?;
    let address = read_bytes20(address, &memory)?;

    let (hash, gas_cost) = data.account_codehash(address);
    data.buy_gas(gas_cost)?;
    memory.write(ptr.offset() as u64, hash.as_slice())?;

    Ok(())
}

pub fn account_code_size(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    address: Ptr,
) -> Result<u32, Escape> {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::ACCOUNT_CODE_SIZE_BASE_INK)?;
    data.require_gas(COLD_ACCOUNT_GAS)?;
    let address = read_bytes20(address, &memory)?;
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

pub fn block_basefee(mut ctx: FunctionEnvMut<StylusCustomEnvData>, ptr: Ptr) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::BLOCK_BASEFEE_BASE_INK)?;
    memory.write(ptr.offset() as u64, data.evm_data.block_basefee.as_slice())?;

    Ok(())
}

pub fn chainid(mut ctx: FunctionEnvMut<StylusCustomEnvData>) -> Result<u64, Escape> {
    let data = ctx.data_mut();

    data.buy_ink(hostio::CHAIN_ID_BASE_INK)?;
    Ok(data.evm_data.chainid)
}

pub fn block_coinbase(mut ctx: FunctionEnvMut<StylusCustomEnvData>, ptr: Ptr) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::BLOCK_COINBASE_BASE_INK)?;
    memory.write(ptr.offset() as u64, data.evm_data.block_coinbase.as_slice())?;

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

pub fn contract_address(mut ctx: FunctionEnvMut<StylusCustomEnvData>, ptr: Ptr) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::ADDRESS_BASE_INK)?;
    memory.write(
        ptr.offset() as u64,
        data.evm_data.contract_address.as_slice(),
    )?;

    Ok(())
}

type U256 = ruint2::Uint<256, 4>;

fn read_u256(ptr: Ptr, memory: &MemoryView) -> Result<(U256, Bytes32), Escape> {
    let bytes = read_bytes32(ptr, memory)?;
    Ok((bytes.clone().into(), bytes))
}

pub fn math_div(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    value: Ptr,
    divisor: Ptr,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::MATH_DIV_BASE_INK)?;
    let (a, _) = read_u256(value, &memory)?;
    let (b, _) = read_u256(divisor, &memory)?;

    let result: Bytes32 = a.checked_div(b).unwrap_or_default().into();
    memory.write(value.offset() as u64, result.as_slice())?;

    Ok(())
}

pub fn math_mod(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    value: Ptr,
    modulus: Ptr,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::MATH_MOD_BASE_INK)?;
    let (a, _) = read_u256(value, &memory)?;
    let (b, _) = read_u256(modulus, &memory)?;

    let result: Bytes32 = a.checked_rem(b).unwrap_or_default().into();
    memory.write(value.offset() as u64, result.as_slice())?;

    Ok(())
}

pub fn math_pow(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    value: Ptr,
    exponent: Ptr,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::MATH_POW_BASE_INK)?;
    let (a, _) = read_u256(value, &memory)?;
    let (b, b32) = read_u256(exponent, &memory)?;

    data.pay_for_pow(&b32)?;
    let result: Bytes32 = a.wrapping_pow(b).into();
    memory.write(value.offset() as u64, result.as_slice())?;

    Ok(())
}

pub fn math_add_mod(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    value: Ptr,
    addend: Ptr,
    modulus: Ptr,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::MATH_ADD_MOD_BASE_INK)?;
    let (a, _) = read_u256(value, &memory)?;
    let (b, _) = read_u256(addend, &memory)?;
    let (c, _) = read_u256(modulus, &memory)?;

    let result: Bytes32 = a.add_mod(b, c).into();
    memory.write(value.offset() as u64, result.as_slice())?;

    Ok(())
}

pub fn math_mul_mod(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    value: Ptr,
    multiplier: Ptr,
    modulus: Ptr,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::MATH_MUL_MOD_BASE_INK)?;
    let (a, _) = read_u256(value, &memory)?;
    let (b, _) = read_u256(multiplier, &memory)?;
    let (c, _) = read_u256(modulus, &memory)?;

    let result: Bytes32 = a.mul_mod(b, c).into();
    memory.write(value.offset() as u64, result.as_slice())?;

    Ok(())
}

pub fn msg_sender(mut ctx: FunctionEnvMut<StylusCustomEnvData>, ptr: Ptr) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::MSG_SENDER_BASE_INK)?;
    memory.write(ptr.offset() as u64, data.evm_data.msg_sender.as_slice())?;

    Ok(())
}

pub fn msg_value(mut ctx: FunctionEnvMut<StylusCustomEnvData>, ptr: Ptr) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::MSG_VALUE_BASE_INK)?;
    memory.write(ptr.offset() as u64, data.evm_data.msg_value.as_slice())?;

    Ok(())
}

pub fn tx_gas_price(mut ctx: FunctionEnvMut<StylusCustomEnvData>, ptr: Ptr) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::TX_GAS_PRICE_BASE_INK)?;
    memory.write(ptr.offset() as u64, data.evm_data.tx_gas_price.as_slice())?;

    Ok(())
}

pub fn tx_ink_price(mut ctx: FunctionEnvMut<StylusCustomEnvData>) -> Result<u32, Escape> {
    let data = ctx.data_mut();

    data.buy_ink(hostio::TX_INK_PRICE_BASE_INK)?;
    Ok(data.pricing().ink_price)
}

pub fn tx_origin(mut ctx: FunctionEnvMut<StylusCustomEnvData>, ptr: Ptr) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.buy_ink(hostio::TX_ORIGIN_BASE_INK)?;
    memory.write(ptr.offset() as u64, data.evm_data.tx_origin.as_slice())?;

    Ok(())
}

pub fn native_keccak256(
    mut ctx: FunctionEnvMut<StylusCustomEnvData>,
    input: Ptr,
    len: u32,
    output: Ptr,
) -> MaybeEscape {
    let (data, store) = ctx.data_and_store_mut();
    let memory = data.memory.clone().unwrap().view(&store);

    data.pay_for_keccak(len)?;
    let preimage = read_slice(input, len as usize, &memory)?;
    let digest = keccak(&preimage);
    memory.write(output.offset() as u64, &digest)?;

    Ok(())
}

use num_traits::Unsigned;
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
