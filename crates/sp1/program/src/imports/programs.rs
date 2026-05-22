//! This module should implement programs related APIs, mainly for
//! launching stylus programs.
//! TODO: for now, we are focusing on getting replay.wasm to run first,
//! so this module only contains dummy impls that serve as a placeholder,
//! and will throw errors when called.
//! They are expected to follow implementations in:
//! https://github.com/OffchainLabs/nitro/blob/d2dba175c037c47e68cf3038f0d4b06b54983644/arbitrator/jit/src/program.rs

#![allow(clippy::too_many_arguments)]

use arbutil::evm::{EvmData, api::Gas};
use caller_env::MemAccess;
use prover::programs::config::{CompileConfig, PricingParams, StylusConfig};
use wasmer::{FunctionEnvMut, WasmPtr};

use crate::{
    Escape, JitConfig, MaybeEscape, Ptr,
    replay::CustomEnvData,
    state::{gp, sp1_env},
    stylus::MessageToCothread,
};

/// Hardcoded message ID used by the Arbitrator protocol for program communication.
const ARBITRATOR_MSG_ID: u32 = 0x33333333;

pub fn new_program(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    compiled_hash_ptr: Ptr,
    calldata_ptr: Ptr,
    calldata_size: u32,
    stylus_config_handler: u64,
    evm_data_handler: u64,
    gas: u64,
) -> Result<u32, Escape> {
    let (mem, data) = sp1_env(&mut ctx);
    let compiled_hash = mem.read_bytes32(gp(compiled_hash_ptr));
    let calldata = mem.read_slice(gp(calldata_ptr), calldata_size as usize);
    let evm_data: EvmData = unsafe { *Box::from_raw(evm_data_handler as *mut EvmData) };
    let config: JitConfig = unsafe { *Box::from_raw(stylus_config_handler as *mut JitConfig) };

    data.launch_program(&compiled_hash, calldata, config, evm_data, gas)
}

pub fn pop(mut ctx: FunctionEnvMut<CustomEnvData>) {
    let data = ctx.data_mut();

    // FIXME: this is wrong, when poping, we should yield from replay.wasm coroutine, and keep
    // running the poped program till it sends the last message and terminates.
    data.pop_last_program();
}

pub fn set_response(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    id: u32,
    gas: u64,
    result_ptr: Ptr,
    result_len: u32,
    raw_data_ptr: Ptr,
    raw_data_len: u32,
) -> MaybeEscape {
    let (mem, data) = sp1_env(&mut ctx);

    // Arbitrator for now only uses hardcoded id, we can ignore
    // ids safely.
    assert_eq!(id, ARBITRATOR_MSG_ID);

    let result = mem.read_slice(gp(result_ptr), result_len as usize);
    let raw_data = mem.read_slice(gp(raw_data_ptr), raw_data_len as usize);

    data.send_to_cothread(MessageToCothread {
        result,
        raw_data,
        cost: Gas(gas),
    });

    Ok(())
}

pub fn get_request(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    id: u32,
    len_ptr: Ptr,
) -> Result<u32, Escape> {
    let (mut mem, data) = sp1_env(&mut ctx);

    // Arbitrator for now only uses hardcoded id, we can ignore
    // ids safely.
    assert_eq!(id, ARBITRATOR_MSG_ID);

    let msg = data.get_last_msg();
    let len: u32 = msg.req_data.len().try_into().expect("req_data length exceeds u32::MAX");
    mem.write_u32(gp(len_ptr), len);

    Ok(msg.req_type)
}

pub fn get_request_data(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    id: u32,
    data_ptr: Ptr,
) -> MaybeEscape {
    let (mut mem, data) = sp1_env(&mut ctx);

    // Arbitrator for now only uses hardcoded id, we can ignore
    // ids safely.
    assert_eq!(id, ARBITRATOR_MSG_ID);

    let msg = data.get_last_msg();
    mem.write_slice(gp(data_ptr), &msg.req_data);

    Ok(())
}

pub fn start_program(mut ctx: FunctionEnvMut<CustomEnvData>, module: u32) -> u32 {
    let data = ctx.data_mut();

    data.wait_next_message(Some(module));
    let _ = data.get_last_msg();

    // Arbitrator for now only uses hardcoded id, we can ignore
    // ids safely.
    ARBITRATOR_MSG_ID
}

pub fn send_response(mut ctx: FunctionEnvMut<CustomEnvData>, req_id: u32) -> u32 {
    let data = ctx.data_mut();

    // Arbitrator for now only uses hardcoded id, we can ignore
    // ids safely.
    assert_eq!(req_id, ARBITRATOR_MSG_ID);

    data.wait_next_message(None);
    let _ = data.get_last_msg();

    ARBITRATOR_MSG_ID
}

pub fn create_stylus_config(
    _ctx: FunctionEnvMut<CustomEnvData>,
    version: u16,
    max_depth: u32,
    ink_price: u32,
    debug: u32,
) -> u64 {
    let stylus = StylusConfig {
        version,
        max_depth,
        pricing: PricingParams { ink_price },
    };
    let compile = CompileConfig::version(version, debug != 0);
    let res = heapify(JitConfig { stylus, compile });
    res as u64
}

const DEFAULT_STYLUS_ARBOS_VERSION: u64 = 31;

pub fn create_evm_data(
    ctx: FunctionEnvMut<CustomEnvData>,
    block_basefee_ptr: Ptr,
    chainid: u64,
    block_coinbase_ptr: Ptr,
    block_gas_limit: u64,
    block_number: u64,
    block_timestamp: u64,
    contract_address_ptr: Ptr,
    module_hash_ptr: Ptr,
    msg_sender_ptr: Ptr,
    msg_value_ptr: Ptr,
    tx_gas_price_ptr: Ptr,
    tx_origin_ptr: Ptr,
    cached: u32,
    reentrant: u32,
) -> Result<u64, Escape> {
    create_evm_data_v2(
        ctx,
        DEFAULT_STYLUS_ARBOS_VERSION,
        block_basefee_ptr,
        chainid,
        block_coinbase_ptr,
        block_gas_limit,
        block_number,
        block_timestamp,
        contract_address_ptr,
        module_hash_ptr,
        msg_sender_ptr,
        msg_value_ptr,
        tx_gas_price_ptr,
        tx_origin_ptr,
        cached,
        reentrant,
    )
}

pub fn create_evm_data_v2(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    arbos_version: u64,
    block_basefee_ptr: Ptr,
    chainid: u64,
    block_coinbase_ptr: Ptr,
    block_gas_limit: u64,
    block_number: u64,
    block_timestamp: u64,
    contract_address_ptr: Ptr,
    module_hash_ptr: Ptr,
    msg_sender_ptr: Ptr,
    msg_value_ptr: Ptr,
    tx_gas_price_ptr: Ptr,
    tx_origin_ptr: Ptr,
    cached: u32,
    reentrant: u32,
) -> Result<u64, Escape> {
    let (mem, _) = sp1_env(&mut ctx);

    let evm_data = EvmData {
        arbos_version,
        block_basefee: mem.read_bytes32(gp(block_basefee_ptr)),
        cached: cached != 0,
        chainid,
        block_coinbase: mem.read_bytes20(gp(block_coinbase_ptr)),
        block_gas_limit,
        block_number,
        block_timestamp,
        contract_address: mem.read_bytes20(gp(contract_address_ptr)),
        module_hash: mem.read_bytes32(gp(module_hash_ptr)),
        msg_sender: mem.read_bytes20(gp(msg_sender_ptr)),
        msg_value: mem.read_bytes32(gp(msg_value_ptr)),
        tx_gas_price: mem.read_bytes32(gp(tx_gas_price_ptr)),
        tx_origin: mem.read_bytes20(gp(tx_origin_ptr)),
        reentrant,
        return_data_len: 0,
        tracing: false,
    };

    let res = heapify(evm_data);
    Ok(res as u64)
}

pub fn activate(
    ctx: FunctionEnvMut<CustomEnvData>,
    wasm_ptr: Ptr,
    wasm_size: u32,
    pages_ptr: WasmPtr<u16>,
    asm_estimate_ptr: Ptr,
    init_cost_ptr: WasmPtr<u16>,
    cached_init_cost_ptr: WasmPtr<u16>,
    stylus_version: u16,
    debug: u32,
    codehash: Ptr,
    module_hash_ptr: Ptr,
    gas_ptr: WasmPtr<u64>,
    err_buf: Ptr,
    err_buf_len: u32,
) -> Result<u32, Escape> {
    activate_v2(
        ctx,
        wasm_ptr,
        wasm_size,
        pages_ptr,
        asm_estimate_ptr,
        init_cost_ptr,
        cached_init_cost_ptr,
        stylus_version,
        DEFAULT_STYLUS_ARBOS_VERSION,
        debug,
        codehash,
        module_hash_ptr,
        gas_ptr,
        err_buf,
        err_buf_len,
    )
}

pub fn activate_v2(
    _ctx: FunctionEnvMut<CustomEnvData>,
    _wasm_ptr: Ptr,
    _wasm_size: u32,
    _pages_ptr: WasmPtr<u16>,
    _asm_estimate_ptr: Ptr,
    _init_cost_ptr: WasmPtr<u16>,
    _cached_init_cost_ptr: WasmPtr<u16>,
    _stylus_version: u16,
    _arbos_version_for_gas: u64,
    _debug: u32,
    _codehash: Ptr,
    _module_hash_ptr: Ptr,
    _gas_ptr: WasmPtr<u64>,
    _err_buf: Ptr,
    _err_buf_len: u32,
) -> Result<u32, Escape> {
    // TODO: per offline discussion with the Arbitrum team, we will call
    // into a separate WASM module to calculate WAVM hash for each stylus
    // program. We won't aim to pull in WAVM logic here.
    todo!("Implement activate_v2!");
}

fn heapify<T>(value: T) -> *mut T {
    Box::into_raw(Box::new(value))
}

/// program_requires_prepare
pub fn program_requires_prepare(
    mut _env: FunctionEnvMut<CustomEnvData>,
    _module_hash_ptr: WasmPtr<u16>,
) -> Result<u32, Escape> {
    Ok(0)
}

/// program_prepare
pub fn program_prepare(
    mut _env: FunctionEnvMut<CustomEnvData>,
    _wasm_ptr: WasmPtr<u16>,
    _wasm_size: u64,
    _module_hash_ptr: WasmPtr<u16>,
    _code_hash_ptr: WasmPtr<u16>,
    _max_wasm_size: u32,
    _page_limit: u32,
    _debug_mode: u32,
    _stylus_version: u32,
) -> MaybeEscape {
    Ok(())
}
