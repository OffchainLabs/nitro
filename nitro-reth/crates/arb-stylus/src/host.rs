use alloy_primitives::{Address, B256, U256};
use wasmer::FunctionEnvMut;

use arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS_CHARGING_FIXES;

use crate::{
    env::WasmEnv,
    error::{Escape, MaybeEscape},
    evm_api::{EvmApi, UserOutcomeKind},
    ink::Gas,
    meter::{GasMeteredMachine, MeteredMachine},
    pricing::{evm_gas, hostio as hio},
};

macro_rules! hostio {
    ($env:expr) => {
        WasmEnv::program($env)?
    };
}

/// Read the program's arguments into WASM memory.
pub fn read_args<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>, ptr: u32) -> MaybeEscape {
    crate::trace::record_leaf("read_args", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::READ_ARGS_BASE_INK)?;
    let args = info.env.args.clone();
    info.pay_for_write(args.len() as u32)?;
    info.write_slice(ptr, &args)?;
    Ok(())
}

/// Write the program's result from WASM memory.
pub fn write_result<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    ptr: u32,
    len: u32,
) -> MaybeEscape {
    crate::trace::record_leaf("write_result", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::WRITE_RESULT_BASE_INK)?;
    info.pay_for_read(len)?;
    info.pay_for_read(len)?; // read from geth
    let data = info.read_slice(ptr, len)?;
    info.env.outs = data;
    Ok(())
}

/// Exit the program early with a status code.
pub fn exit_early<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>, status: u32) -> MaybeEscape {
    crate::trace::record_leaf("exit_early", Default::default(), Default::default());
    let _info = hostio!(&mut env);
    Err(Escape::Exit(status))
}

/// Load a 32-byte storage value.
pub fn storage_load_bytes32<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    key_ptr: u32,
    dest_ptr: u32,
) -> MaybeEscape {
    let mut info = hostio!(&mut env);
    let trace_on = crate::trace::is_active();
    let start_ink = if trace_on { info.ink_ready()?.0 } else { 0 };
    info.buy_ink(hio::STORAGE_LOAD_BASE_INK)?;
    let arbos_version = info.env.evm_data.arbos_version;
    // Preserve wrong behavior for old arbos versions
    let evm_api_gas = if arbos_version < ARBOS_VERSION_STYLUS_CHARGING_FIXES {
        Gas(crate::pricing::EVM_API_INK.0)
    } else {
        info.pricing().ink_to_gas(crate::pricing::EVM_API_INK)
    };
    info.require_gas(
        evm_gas::COLD_SLOAD_GAS + evm_gas::STORAGE_CACHE_REQUIRED_ACCESS_GAS + evm_api_gas.0,
    )?;
    let key = B256::from(info.read_fixed::<32>(key_ptr)?);
    let (value, gas_cost) = info
        .env
        .evm_api
        .get_bytes32(key, evm_api_gas)
        .map_err(|e| Escape::Internal(e.to_string()))?;
    info.buy_gas(gas_cost.0)?;
    info.write_slice(dest_ptr, value.as_slice())?;
    if trace_on {
        let end_ink = info.ink_ready().map(|i| i.0).unwrap_or(0);
        crate::trace::record(
            "storage_load_bytes32",
            alloy_primitives::Bytes::copy_from_slice(key.as_slice()),
            alloy_primitives::Bytes::copy_from_slice(value.as_slice()),
            start_ink,
            end_ink,
            None,
        );
    }
    Ok(())
}

/// Cache a 32-byte storage value for later flushing.
pub fn storage_cache_bytes32<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    key_ptr: u32,
    value_ptr: u32,
) -> MaybeEscape {
    crate::trace::record_leaf(
        "storage_cache_bytes32",
        Default::default(),
        Default::default(),
    );
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::STORAGE_CACHE_BASE_INK)?;
    info.require_gas(evm_gas::SSTORE_SENTRY_GAS + evm_gas::STORAGE_CACHE_REQUIRED_ACCESS_GAS)?;
    let key = B256::from(info.read_fixed::<32>(key_ptr)?);
    let value = B256::from(info.read_fixed::<32>(value_ptr)?);
    let gas_cost = info
        .env
        .evm_api
        .cache_bytes32(key, value)
        .map_err(|e| Escape::Internal(e.to_string()))?;
    info.buy_gas(gas_cost.0)?;
    Ok(())
}

/// Flush the storage cache.
pub fn storage_flush_cache<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    clear: u32,
) -> MaybeEscape {
    crate::trace::record_leaf(
        "storage_flush_cache",
        Default::default(),
        Default::default(),
    );
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::STORAGE_FLUSH_BASE_INK)?;
    info.require_gas(evm_gas::SSTORE_SENTRY_GAS)?;
    let gas_left = info.ink_ready().map(|ink| info.pricing().ink_to_gas(ink))?;
    let (gas_cost, status) = info
        .env
        .evm_api
        .flush_storage_cache(clear != 0, Gas(gas_left.0 + 1))
        .map_err(|e| Escape::Internal(e.to_string()))?;
    if info.env.evm_data.arbos_version >= ARBOS_VERSION_STYLUS_CHARGING_FIXES {
        info.buy_gas(gas_cost.0)?;
    }
    if status != UserOutcomeKind::Success {
        return Escape::logical("storage flush failed");
    }
    Ok(())
}

/// Load a transient storage value.
pub fn transient_load_bytes32<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    key_ptr: u32,
    dest_ptr: u32,
) -> MaybeEscape {
    crate::trace::record_leaf(
        "transient_load_bytes32",
        Default::default(),
        Default::default(),
    );
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::TRANSIENT_LOAD_BASE_INK)?;
    info.buy_gas(evm_gas::TLOAD_GAS)?;
    let key = B256::from(info.read_fixed::<32>(key_ptr)?);
    let value = info
        .env
        .evm_api
        .get_transient_bytes32(key)
        .map_err(|e| Escape::Internal(e.to_string()))?;
    info.write_slice(dest_ptr, value.as_slice())?;
    Ok(())
}

/// Store a transient storage value.
pub fn transient_store_bytes32<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    key_ptr: u32,
    value_ptr: u32,
) -> MaybeEscape {
    crate::trace::record_leaf(
        "transient_store_bytes32",
        Default::default(),
        Default::default(),
    );
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::TRANSIENT_STORE_BASE_INK)?;
    info.buy_gas(evm_gas::TSTORE_GAS)?;
    let key = B256::from(info.read_fixed::<32>(key_ptr)?);
    let value = B256::from(info.read_fixed::<32>(value_ptr)?);
    let status = info
        .env
        .evm_api
        .set_transient_bytes32(key, value)
        .map_err(|e| Escape::Internal(e.to_string()))?;
    if status == UserOutcomeKind::Failure {
        return Escape::logical("transient store failed");
    }
    Ok(())
}

/// Execute a CALL.
pub fn call_contract<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    contract_ptr: u32,
    calldata_ptr: u32,
    calldata_len: u32,
    value_ptr: u32,
    gas: u64,
    ret_len_ptr: u32,
) -> Result<u8, Escape> {
    crate::trace::record_leaf("call_contract", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::CALL_CONTRACT_BASE_INK)?;
    info.pay_for_read(calldata_len)?;
    info.pay_for_read(calldata_len)?; // read from geth
    let contract = Address::from_slice(&info.read_fixed::<20>(contract_ptr)?);
    let calldata = info.read_slice(calldata_ptr, calldata_len)?;
    let value = U256::from_be_bytes(info.read_fixed::<32>(value_ptr)?);
    let gas_left = info.ink_ready().map(|ink| info.pricing().ink_to_gas(ink))?;
    let gas_req = Gas(gas.min(gas_left.0));
    let (ret_len, gas_cost, status) = info
        .env
        .evm_api
        .contract_call(contract, &calldata, gas_left, gas_req, value)
        .map_err(|e| Escape::Internal(e.to_string()))?;
    info.buy_gas(gas_cost.0)?;
    info.env.evm_return_data_len = ret_len;
    info.write_u32(ret_len_ptr, ret_len)?;
    Ok(status as u8)
}

/// Execute a DELEGATECALL.
pub fn delegate_call_contract<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    contract_ptr: u32,
    calldata_ptr: u32,
    calldata_len: u32,
    gas: u64,
    ret_len_ptr: u32,
) -> Result<u8, Escape> {
    crate::trace::record_leaf(
        "delegate_call_contract",
        Default::default(),
        Default::default(),
    );
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::CALL_CONTRACT_BASE_INK)?;
    info.pay_for_read(calldata_len)?;
    info.pay_for_read(calldata_len)?; // read from geth
    let contract = Address::from_slice(&info.read_fixed::<20>(contract_ptr)?);
    let calldata = info.read_slice(calldata_ptr, calldata_len)?;
    let gas_left = info.ink_ready().map(|ink| info.pricing().ink_to_gas(ink))?;
    let gas_req = Gas(gas.min(gas_left.0));
    let (ret_len, gas_cost, status) = info
        .env
        .evm_api
        .delegate_call(contract, &calldata, gas_left, gas_req)
        .map_err(|e| Escape::Internal(e.to_string()))?;
    info.buy_gas(gas_cost.0)?;
    info.env.evm_return_data_len = ret_len;
    info.write_u32(ret_len_ptr, ret_len)?;
    Ok(status as u8)
}

/// Execute a STATICCALL.
pub fn static_call_contract<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    contract_ptr: u32,
    calldata_ptr: u32,
    calldata_len: u32,
    gas: u64,
    ret_len_ptr: u32,
) -> Result<u8, Escape> {
    crate::trace::record_leaf(
        "static_call_contract",
        Default::default(),
        Default::default(),
    );
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::CALL_CONTRACT_BASE_INK)?;
    info.pay_for_read(calldata_len)?;
    info.pay_for_read(calldata_len)?; // read from geth
    let contract = Address::from_slice(&info.read_fixed::<20>(contract_ptr)?);
    let calldata = info.read_slice(calldata_ptr, calldata_len)?;
    let gas_left = info.ink_ready().map(|ink| info.pricing().ink_to_gas(ink))?;
    let gas_req = Gas(gas.min(gas_left.0));
    let (ret_len, gas_cost, status) = info
        .env
        .evm_api
        .static_call(contract, &calldata, gas_left, gas_req)
        .map_err(|e| Escape::Internal(e.to_string()))?;
    info.buy_gas(gas_cost.0)?;
    info.env.evm_return_data_len = ret_len;
    info.write_u32(ret_len_ptr, ret_len)?;
    Ok(status as u8)
}

/// Deploy a contract via CREATE.
pub fn create1<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    code_ptr: u32,
    code_len: u32,
    endowment_ptr: u32,
    contract_ptr: u32,
    ret_len_ptr: u32,
) -> MaybeEscape {
    crate::trace::record_leaf("create1", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::CREATE1_BASE_INK)?;
    info.pay_for_read(code_len)?;
    info.pay_for_read(code_len)?; // read from geth
    let code = info.read_slice(code_ptr, code_len)?;
    let endowment = U256::from_be_bytes(info.read_fixed::<32>(endowment_ptr)?);
    let gas_left = info.ink_ready().map(|ink| info.pricing().ink_to_gas(ink))?;
    let (response, ret_len, gas_cost) = info
        .env
        .evm_api
        .create1(code, endowment, gas_left)
        .map_err(|e| Escape::Internal(e.to_string()))?;
    let address = match response {
        crate::evm_api::CreateResponse::Success(addr) => addr,
        crate::evm_api::CreateResponse::Fail(reason) => {
            return Err(Escape::Internal(reason));
        }
    };
    info.buy_gas(gas_cost.0)?;
    info.env.evm_return_data_len = ret_len;
    info.write_u32(ret_len_ptr, ret_len)?;
    info.write_slice(contract_ptr, address.as_slice())?;
    Ok(())
}

/// Deploy a contract via CREATE2.
pub fn create2<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    code_ptr: u32,
    code_len: u32,
    endowment_ptr: u32,
    salt_ptr: u32,
    contract_ptr: u32,
    ret_len_ptr: u32,
) -> MaybeEscape {
    crate::trace::record_leaf("create2", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::CREATE2_BASE_INK)?;
    info.pay_for_read(code_len)?;
    info.pay_for_read(code_len)?; // read from geth
    let code = info.read_slice(code_ptr, code_len)?;
    let endowment = U256::from_be_bytes(info.read_fixed::<32>(endowment_ptr)?);
    let salt = B256::from(info.read_fixed::<32>(salt_ptr)?);
    let gas_left = info.ink_ready().map(|ink| info.pricing().ink_to_gas(ink))?;
    let (response, ret_len, gas_cost) = info
        .env
        .evm_api
        .create2(code, endowment, salt, gas_left)
        .map_err(|e| Escape::Internal(e.to_string()))?;
    let address = match response {
        crate::evm_api::CreateResponse::Success(addr) => addr,
        crate::evm_api::CreateResponse::Fail(reason) => {
            return Err(Escape::Internal(reason));
        }
    };
    info.buy_gas(gas_cost.0)?;
    info.env.evm_return_data_len = ret_len;
    info.write_u32(ret_len_ptr, ret_len)?;
    info.write_slice(contract_ptr, address.as_slice())?;
    Ok(())
}

/// Read return data into WASM memory.
pub fn read_return_data<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    dest_ptr: u32,
    offset: u32,
    size: u32,
) -> Result<u32, Escape> {
    crate::trace::record_leaf("read_return_data", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::READ_RETURN_DATA_BASE_INK)?;
    let max = info.env.evm_return_data_len.saturating_sub(offset);
    info.pay_for_write(size.min(max))?;
    if max == 0 {
        return Ok(0);
    }
    let data = info.env.evm_api.get_return_data();
    let offset = offset as usize;
    let size = size as usize;
    let available = data.len().saturating_sub(offset);
    let copy_len = available.min(size);
    if copy_len > 0 {
        info.write_slice(dest_ptr, &data[offset..offset + copy_len])?;
    }
    Ok(copy_len as u32)
}

/// Get the size of the return data.
pub fn return_data_size<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>) -> Result<u32, Escape> {
    crate::trace::record_leaf("return_data_size", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::RETURN_DATA_SIZE_BASE_INK)?;
    Ok(info.env.evm_return_data_len)
}

/// Emit a log.
pub fn emit_log<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    data_ptr: u32,
    data_len: u32,
    topics: u32,
) -> MaybeEscape {
    crate::trace::record_leaf("emit_log", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::EMIT_LOG_BASE_INK)?;
    if topics > 4 || data_len < topics * 32 {
        return Escape::logical("bad topic data");
    }
    info.pay_for_read(data_len)?;
    info.pay_for_evm_log(topics, data_len - topics * 32)?;
    let data = info.read_slice(data_ptr, data_len)?;
    info.env
        .evm_api
        .emit_log(data, topics)
        .map_err(|e| Escape::Internal(e.to_string()))?;
    Ok(())
}

/// Get an account's balance.
pub fn account_balance<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    addr_ptr: u32,
    dest_ptr: u32,
) -> MaybeEscape {
    crate::trace::record_leaf("account_balance", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::ACCOUNT_BALANCE_BASE_INK)?;
    info.require_gas(evm_gas::COLD_ACCOUNT_GAS)?;
    let address = Address::from_slice(&info.read_fixed::<20>(addr_ptr)?);
    let (balance, gas_cost) = info
        .env
        .evm_api
        .account_balance(address)
        .map_err(|e| Escape::Internal(e.to_string()))?;
    info.buy_gas(gas_cost.0)?;
    info.write_slice(dest_ptr, &balance.to_be_bytes::<32>())?;
    Ok(())
}

/// Get an account's code.
pub fn account_code<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    addr_ptr: u32,
    offset: u32,
    size: u32,
    dest_ptr: u32,
) -> Result<u32, Escape> {
    crate::trace::record_leaf("account_code", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::ACCOUNT_CODE_BASE_INK)?;
    info.require_gas(evm_gas::COLD_ACCOUNT_GAS)?;
    let address = Address::from_slice(&info.read_fixed::<20>(addr_ptr)?);
    let gas_left = info.ink_ready().map(|ink| info.pricing().ink_to_gas(ink))?;
    let arbos_version = info.env.evm_data.arbos_version;
    let (code, gas_cost) = info
        .env
        .evm_api
        .account_code(arbos_version, address, gas_left)
        .map_err(|e| Escape::Internal(e.to_string()))?;
    info.buy_gas(gas_cost.0)?;
    info.pay_for_write(code.len() as u32)?;
    let offset = offset as usize;
    let size = size as usize;
    let available = code.len().saturating_sub(offset);
    let copy_len = available.min(size);
    if copy_len > 0 {
        info.write_slice(dest_ptr, &code[offset..offset + copy_len])?;
    }
    Ok(copy_len as u32)
}

/// Get an account's code size.
pub fn account_code_size<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    addr_ptr: u32,
) -> Result<u32, Escape> {
    crate::trace::record_leaf("account_code_size", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::ACCOUNT_CODE_SIZE_BASE_INK)?;
    info.require_gas(evm_gas::COLD_ACCOUNT_GAS)?;
    let address = Address::from_slice(&info.read_fixed::<20>(addr_ptr)?);
    let gas_left = info.ink_ready().map(|ink| info.pricing().ink_to_gas(ink))?;
    let arbos_version = info.env.evm_data.arbos_version;
    let (code, gas_cost) = info
        .env
        .evm_api
        .account_code(arbos_version, address, gas_left)
        .map_err(|e| Escape::Internal(e.to_string()))?;
    info.buy_gas(gas_cost.0)?;
    Ok(code.len() as u32)
}

/// Get an account's code hash.
pub fn account_codehash<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    addr_ptr: u32,
    dest_ptr: u32,
) -> MaybeEscape {
    crate::trace::record_leaf("account_codehash", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::ACCOUNT_CODE_HASH_BASE_INK)?;
    info.require_gas(evm_gas::COLD_ACCOUNT_GAS)?;
    let address = Address::from_slice(&info.read_fixed::<20>(addr_ptr)?);
    let (hash, gas_cost) = info
        .env
        .evm_api
        .account_codehash(address)
        .map_err(|e| Escape::Internal(e.to_string()))?;
    info.buy_gas(gas_cost.0)?;
    info.write_slice(dest_ptr, hash.as_slice())?;
    Ok(())
}

/// Get remaining EVM gas.
pub fn evm_gas_left<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>) -> Result<u64, Escape> {
    crate::trace::record_leaf("evm_gas_left", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::EVM_GAS_LEFT_BASE_INK)?;
    let ink = info.ink_ready()?;
    Ok(info.pricing().ink_to_gas(ink).0)
}

/// Get remaining ink.
pub fn evm_ink_left<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>) -> Result<u64, Escape> {
    crate::trace::record_leaf("evm_ink_left", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::EVM_INK_LEFT_BASE_INK)?;
    Ok(info.ink_ready()?.0)
}

/// Write the block base fee.
pub fn block_basefee<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>, ptr: u32) -> MaybeEscape {
    crate::trace::record_leaf("block_basefee", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::BLOCK_BASEFEE_BASE_INK)?;
    info.write_slice(ptr, info.env.evm_data.block_basefee.as_slice())?;
    Ok(())
}

/// Get the chain ID.
pub fn chainid<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>) -> Result<u64, Escape> {
    crate::trace::record_leaf("chainid", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::CHAIN_ID_BASE_INK)?;
    Ok(info.env.evm_data.chain_id)
}

/// Write the block coinbase address.
pub fn block_coinbase<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>, ptr: u32) -> MaybeEscape {
    crate::trace::record_leaf("block_coinbase", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::BLOCK_COINBASE_BASE_INK)?;
    info.write_slice(ptr, info.env.evm_data.block_coinbase.as_slice())?;
    Ok(())
}

/// Get the block gas limit.
pub fn block_gas_limit<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>) -> Result<u64, Escape> {
    crate::trace::record_leaf("block_gas_limit", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::BLOCK_GAS_LIMIT_BASE_INK)?;
    Ok(info.env.evm_data.block_gas_limit)
}

/// Get the block number.
pub fn block_number<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>) -> Result<u64, Escape> {
    crate::trace::record_leaf("block_number", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::BLOCK_NUMBER_BASE_INK)?;
    Ok(info.env.evm_data.block_number)
}

/// Get the block timestamp.
pub fn block_timestamp<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>) -> Result<u64, Escape> {
    crate::trace::record_leaf("block_timestamp", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::BLOCK_TIMESTAMP_BASE_INK)?;
    Ok(info.env.evm_data.block_timestamp)
}

/// Write the contract address.
pub fn contract_address<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    ptr: u32,
) -> MaybeEscape {
    crate::trace::record_leaf("contract_address", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::ADDRESS_BASE_INK)?;
    info.write_slice(ptr, info.env.evm_data.contract_address.as_slice())?;
    Ok(())
}

/// 256-bit division.
pub fn math_div<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    a_ptr: u32,
    b_ptr: u32,
) -> MaybeEscape {
    crate::trace::record_leaf("math_div", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::MATH_DIV_BASE_INK)?;
    let a = U256::from_be_bytes(info.read_fixed::<32>(a_ptr)?);
    let b = U256::from_be_bytes(info.read_fixed::<32>(b_ptr)?);
    let result = if b.is_zero() { U256::ZERO } else { a / b };
    info.write_slice(a_ptr, &result.to_be_bytes::<32>())?;
    Ok(())
}

/// 256-bit modulo.
pub fn math_mod<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    a_ptr: u32,
    b_ptr: u32,
) -> MaybeEscape {
    crate::trace::record_leaf("math_mod", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::MATH_MOD_BASE_INK)?;
    let a = U256::from_be_bytes(info.read_fixed::<32>(a_ptr)?);
    let b = U256::from_be_bytes(info.read_fixed::<32>(b_ptr)?);
    let result = if b.is_zero() { U256::ZERO } else { a % b };
    info.write_slice(a_ptr, &result.to_be_bytes::<32>())?;
    Ok(())
}

/// 256-bit power.
pub fn math_pow<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    base_ptr: u32,
    exp_ptr: u32,
) -> MaybeEscape {
    crate::trace::record_leaf("math_pow", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::MATH_POW_BASE_INK)?;
    let base = U256::from_be_bytes(info.read_fixed::<32>(base_ptr)?);
    let exp_bytes = info.read_fixed::<32>(exp_ptr)?;
    info.buy_ink(crate::pricing::pow_price(&exp_bytes))?;
    let exp = U256::from_be_bytes(exp_bytes);
    let result = base.pow(exp);
    info.write_slice(base_ptr, &result.to_be_bytes::<32>())?;
    Ok(())
}

/// 256-bit addmod.
pub fn math_add_mod<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    a_ptr: u32,
    b_ptr: u32,
    mod_ptr: u32,
) -> MaybeEscape {
    crate::trace::record_leaf("math_add_mod", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::MATH_ADD_MOD_BASE_INK)?;
    let a = U256::from_be_bytes(info.read_fixed::<32>(a_ptr)?);
    let b = U256::from_be_bytes(info.read_fixed::<32>(b_ptr)?);
    let modulus = U256::from_be_bytes(info.read_fixed::<32>(mod_ptr)?);
    let result = if modulus.is_zero() {
        U256::ZERO
    } else {
        a.add_mod(b, modulus)
    };
    info.write_slice(a_ptr, &result.to_be_bytes::<32>())?;
    Ok(())
}

/// 256-bit mulmod.
pub fn math_mul_mod<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    a_ptr: u32,
    b_ptr: u32,
    mod_ptr: u32,
) -> MaybeEscape {
    crate::trace::record_leaf("math_mul_mod", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::MATH_MUL_MOD_BASE_INK)?;
    let a = U256::from_be_bytes(info.read_fixed::<32>(a_ptr)?);
    let b = U256::from_be_bytes(info.read_fixed::<32>(b_ptr)?);
    let modulus = U256::from_be_bytes(info.read_fixed::<32>(mod_ptr)?);
    let result = if modulus.is_zero() {
        U256::ZERO
    } else {
        a.mul_mod(b, modulus)
    };
    info.write_slice(a_ptr, &result.to_be_bytes::<32>())?;
    Ok(())
}

/// Get the reentrant counter.
pub fn msg_reentrant<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>) -> Result<u32, Escape> {
    crate::trace::record_leaf("msg_reentrant", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::MSG_REENTRANT_BASE_INK)?;
    Ok(info.env.evm_data.reentrant)
}

/// Write the message sender address.
pub fn msg_sender<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>, ptr: u32) -> MaybeEscape {
    crate::trace::record_leaf("msg_sender", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::MSG_SENDER_BASE_INK)?;
    info.write_slice(ptr, info.env.evm_data.msg_sender.as_slice())?;
    Ok(())
}

/// Write the message value.
pub fn msg_value<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>, ptr: u32) -> MaybeEscape {
    crate::trace::record_leaf("msg_value", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::MSG_VALUE_BASE_INK)?;
    info.write_slice(ptr, info.env.evm_data.msg_value.as_slice())?;
    Ok(())
}

/// Write the transaction gas price.
pub fn tx_gas_price<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>, ptr: u32) -> MaybeEscape {
    crate::trace::record_leaf("tx_gas_price", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::TX_GAS_PRICE_BASE_INK)?;
    info.write_slice(ptr, info.env.evm_data.tx_gas_price.as_slice())?;
    Ok(())
}

/// Get the ink price.
pub fn tx_ink_price<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>) -> Result<u32, Escape> {
    crate::trace::record_leaf("tx_ink_price", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::TX_INK_PRICE_BASE_INK)?;
    Ok(info.pricing().ink_price)
}

/// Write the transaction origin address.
pub fn tx_origin<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>, ptr: u32) -> MaybeEscape {
    crate::trace::record_leaf("tx_origin", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.buy_ink(hio::TX_ORIGIN_BASE_INK)?;
    info.write_slice(ptr, info.env.evm_data.tx_origin.as_slice())?;
    Ok(())
}

/// Charge for WASM memory growth.
pub fn pay_for_memory_grow<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    pages: u16,
) -> MaybeEscape {
    crate::trace::record_leaf(
        "pay_for_memory_grow",
        Default::default(),
        Default::default(),
    );
    let mut info = hostio!(&mut env);
    if pages == 0 {
        info.buy_ink(hio::PAY_FOR_MEMORY_GROW_BASE_INK)?;
        return Ok(());
    }
    let gas_cost = info
        .env
        .evm_api
        .add_pages(pages)
        .map_err(|e| Escape::Internal(e.to_string()))?;
    info.buy_gas(gas_cost.0)?;
    Ok(())
}

/// Compute keccak256 hash.
pub fn native_keccak256<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    input_ptr: u32,
    input_len: u32,
    output_ptr: u32,
) -> MaybeEscape {
    crate::trace::record_leaf("native_keccak256", Default::default(), Default::default());
    let mut info = hostio!(&mut env);
    info.pay_for_keccak(input_len)?;
    let data = info.read_slice(input_ptr, input_len)?;
    let hash = alloy_primitives::keccak256(&data);
    info.write_slice(output_ptr, hash.as_slice())?;
    Ok(())
}

// Debug functions

/// Log text to console (debug only).
pub fn console_log_text<E: EvmApi>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    ptr: u32,
    len: u32,
) -> MaybeEscape {
    crate::trace::record_leaf("console_log_text", Default::default(), Default::default());
    let info = hostio!(&mut env);
    let text = info.read_slice(ptr, len)?;
    if let Ok(s) = std::str::from_utf8(&text) {
        tracing::debug!(target: "stylus", "{s}");
    }
    Ok(())
}

/// Log a value to console (debug only).
pub fn console_log<E: EvmApi, T: std::fmt::Display>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    value: T,
) -> MaybeEscape {
    crate::trace::record_leaf("console_log", Default::default(), Default::default());
    let _info = hostio!(&mut env);
    tracing::debug!(target: "stylus", "{value}");
    Ok(())
}

/// Log and return a value (debug only).
pub fn console_tee<E: EvmApi, T: Copy + std::fmt::Display>(
    mut env: FunctionEnvMut<'_, WasmEnv<E>>,
    value: T,
) -> Result<T, Escape> {
    crate::trace::record_leaf("console_tee", Default::default(), Default::default());
    let _info = hostio!(&mut env);
    tracing::debug!(target: "stylus", "{value}");
    Ok(value)
}

/// No-op host function (debug only).
pub fn null_host<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>) -> MaybeEscape {
    crate::trace::record_leaf("null_host", Default::default(), Default::default());
    let _info = hostio!(&mut env);
    Ok(())
}

/// Start a benchmark measurement (debug only).
pub fn start_benchmark<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>) -> MaybeEscape {
    crate::trace::record_leaf("start_benchmark", Default::default(), Default::default());
    let _info = hostio!(&mut env);
    Ok(())
}

/// End a benchmark measurement (debug only).
pub fn end_benchmark<E: EvmApi>(mut env: FunctionEnvMut<'_, WasmEnv<E>>) -> MaybeEscape {
    crate::trace::record_leaf("end_benchmark", Default::default(), Default::default());
    let _info = hostio!(&mut env);
    Ok(())
}
