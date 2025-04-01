// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(clippy::too_many_arguments)]

use crate::caller_env::JitEnv;
use crate::machine::{Escape, MaybeEscape, WasmEnv, WasmEnvMut};
use crate::stylus_backend::{exec_wasm, MessageFromCothread};
use arbutil::evm::api::Gas;
use arbutil::Bytes32;
use arbutil::{evm::EvmData, format::DebugBytes, heapify};
use caller_env::{GuestPtr, MemAccess};
use eyre::eyre;
use prover::programs::prelude::StylusConfig;
use prover::{
    machine::Module,
    programs::{config::PricingParams, prelude::*},
};
use std::sync::Arc;

const DEFAULT_STYLUS_ARBOS_VERSION: u64 = 31;

pub fn activate(
    env: WasmEnvMut,
    wasm_ptr: GuestPtr,
    wasm_size: u32,
    pages_ptr: GuestPtr,
    asm_estimate_ptr: GuestPtr,
    init_cost_ptr: GuestPtr,
    cached_init_cost_ptr: GuestPtr,
    stylus_version: u16,
    debug: u32,
    codehash: GuestPtr,
    module_hash_ptr: GuestPtr,
    gas_ptr: GuestPtr,
    err_buf: GuestPtr,
    err_buf_len: u32,
) -> Result<u32, Escape> {
    activate_v2(
        env,
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

/// activates a user program
pub fn activate_v2(
    mut env: WasmEnvMut,
    wasm_ptr: GuestPtr,
    wasm_size: u32,
    pages_ptr: GuestPtr,
    asm_estimate_ptr: GuestPtr,
    init_cost_ptr: GuestPtr,
    cached_init_cost_ptr: GuestPtr,
    stylus_version: u16,
    arbos_version_for_gas: u64,
    debug: u32,
    codehash: GuestPtr,
    module_hash_ptr: GuestPtr,
    gas_ptr: GuestPtr,
    err_buf: GuestPtr,
    err_buf_len: u32,
) -> Result<u32, Escape> {
    let (mut mem, _) = env.jit_env();
    let wasm = mem.read_slice(wasm_ptr, wasm_size as usize);
    let codehash = &mem.read_bytes32(codehash);
    let debug = debug != 0;

    let page_limit = mem.read_u16(pages_ptr);
    let gas_left = &mut mem.read_u64(gas_ptr);
    match Module::activate(
        &wasm,
        codehash,
        stylus_version,
        arbos_version_for_gas,
        page_limit,
        debug,
        gas_left,
    ) {
        Ok((module, data)) => {
            mem.write_u64(gas_ptr, *gas_left);
            mem.write_u16(pages_ptr, data.footprint);
            mem.write_u32(asm_estimate_ptr, data.asm_estimate);
            mem.write_u16(init_cost_ptr, data.init_cost);
            mem.write_u16(cached_init_cost_ptr, data.cached_init_cost);
            mem.write_bytes32(module_hash_ptr, module.hash());
            Ok(0)
        }
        Err(error) => {
            let mut err_bytes = error.wrap_err("failed to activate").debug_bytes();
            err_bytes.truncate(err_buf_len as usize);
            mem.write_slice(err_buf, &err_bytes);
            mem.write_u64(gas_ptr, 0);
            mem.write_u16(pages_ptr, 0);
            mem.write_u32(asm_estimate_ptr, 0);
            mem.write_u16(init_cost_ptr, 0);
            mem.write_u16(cached_init_cost_ptr, 0);
            mem.write_bytes32(module_hash_ptr, Bytes32::default());
            Ok(err_bytes.len() as u32)
        }
    }
}

/// Links and creates user program (in jit starts it as well)
/// consumes both evm_data_handler and config_handler
/// returns module number
pub fn new_program(
    mut env: WasmEnvMut,
    compiled_hash_ptr: GuestPtr,
    calldata_ptr: GuestPtr,
    calldata_size: u32,
    stylus_config_handler: u64,
    evm_data_handler: u64,
    gas: u64,
) -> Result<u32, Escape> {
    let (mut mem, exec) = env.jit_env();
    let compiled_hash = mem.read_bytes32(compiled_hash_ptr);
    let calldata = mem.read_slice(calldata_ptr, calldata_size as usize);
    let evm_data: EvmData = unsafe { *Box::from_raw(evm_data_handler as *mut EvmData) };
    let config: JitConfig = unsafe { *Box::from_raw(stylus_config_handler as *mut JitConfig) };

    let Some(module) = exec.module_asms.get(&compiled_hash).cloned() else {
        return Err(Escape::Failure(format!(
            "module hash {:?} not found in {:?}",
            compiled_hash,
            exec.module_asms.keys()
        )));
    };

    launch_program_thread(exec, module, calldata, config, evm_data, gas)
}

pub fn launch_program_thread(
    exec: &mut WasmEnv,
    module: Arc<[u8]>,
    calldata: Vec<u8>,
    config: JitConfig,
    evm_data: EvmData,
    gas: u64,
) -> Result<u32, Escape> {
    // buy ink
    let pricing = config.stylus.pricing;
    let ink = pricing.gas_to_ink(Gas(gas));

    let cothread = exec_wasm(
        module,
        calldata,
        config.compile,
        config.stylus,
        evm_data,
        ink,
    )
    .unwrap();

    exec.threads.push(cothread);

    Ok(exec.threads.len() as u32)
}

/// starts the program (in jit waits for first request)
/// module MUST match last module number returned from new_program
/// returns request_id for the first request from the program
pub fn start_program(mut env: WasmEnvMut, module: u32) -> Result<u32, Escape> {
    let (_, exec) = env.jit_env();
    start_program_with_wasm_env(exec, module)
}

pub fn start_program_with_wasm_env(exec: &mut WasmEnv, module: u32) -> Result<u32, Escape> {
    if exec.threads.len() as u32 != module || module == 0 {
        return Escape::hostio(format!(
            "got request for thread {module} but len is {}",
            exec.threads.len()
        ));
    }
    let thread = exec.threads.last_mut().unwrap();
    thread.wait_next_message()?;
    let msg = thread.last_message()?;
    Ok(msg.1)
}

/// gets information about request according to id
/// request_id MUST be last request id returned from start_program or send_response
pub fn get_request(mut env: WasmEnvMut, id: u32, len_ptr: GuestPtr) -> Result<u32, Escape> {
    let (mut mem, exec) = env.jit_env();
    let msg = get_last_msg(exec, id)?;
    mem.write_u32(len_ptr, msg.req_data.len() as u32);
    Ok(msg.req_type)
}

pub fn get_last_msg(exec: &mut WasmEnv, id: u32) -> Result<MessageFromCothread, Escape> {
    let thread = exec.threads.last_mut().unwrap();
    let msg = thread.last_message()?;
    if msg.1 != id {
        return Escape::hostio("get_request id doesn't match");
    };
    Ok(msg.0)
}

// gets data associated with last request.
// request_id MUST be last request receieved
// data_ptr MUST point to a buffer of at least the length returned by get_request
pub fn get_request_data(mut env: WasmEnvMut, id: u32, data_ptr: GuestPtr) -> MaybeEscape {
    let (mut mem, exec) = env.jit_env();
    let msg = get_last_msg(exec, id)?;
    mem.write_slice(data_ptr, &msg.req_data);
    Ok(())
}

/// sets response for the next request made
/// id MUST be the id of last request made
pub fn set_response(
    mut env: WasmEnvMut,
    id: u32,
    gas: u64,
    result_ptr: GuestPtr,
    result_len: u32,
    raw_data_ptr: GuestPtr,
    raw_data_len: u32,
) -> MaybeEscape {
    let (mem, exec) = env.jit_env();
    let result = mem.read_slice(result_ptr, result_len as usize);
    let raw_data = mem.read_slice(raw_data_ptr, raw_data_len as usize);

    set_response_with_wasm_env(exec, id, gas, result, raw_data)
}

pub fn set_response_with_wasm_env(
    exec: &mut WasmEnv,
    id: u32,
    gas: u64,
    result: Vec<u8>,
    raw_data: Vec<u8>,
) -> MaybeEscape {
    let thread = exec.threads.last_mut().unwrap();
    thread.set_response(id, result, raw_data, Gas(gas))
}

/// sends previous response
/// MUST be called right after set_response to the same id
/// returns request_id for the next request
pub fn send_response(mut env: WasmEnvMut, req_id: u32) -> Result<u32, Escape> {
    let (_, exec) = env.jit_env();
    let thread = exec.threads.last_mut().unwrap();
    let msg = thread.last_message()?;
    if msg.1 != req_id {
        return Escape::hostio("get_request id doesn't match");
    };
    thread.wait_next_message()?;
    let msg = thread.last_message()?;
    Ok(msg.1)
}

/// removes the last created program
pub fn pop(mut env: WasmEnvMut) -> MaybeEscape {
    let (_, exec) = env.jit_env();
    pop_with_wasm_env(exec)
}

pub fn pop_with_wasm_env(exec: &mut WasmEnv) -> MaybeEscape {
    match exec.threads.pop() {
        None => Err(Escape::Child(eyre!("no child"))),
        Some(mut thread) => thread.wait_done(),
    }
}

pub struct JitConfig {
    pub stylus: StylusConfig,
    pub compile: CompileConfig,
}

/// Creates a `StylusConfig` from its component parts.
pub fn create_stylus_config(
    mut _env: WasmEnvMut,
    version: u16,
    max_depth: u32,
    ink_price: u32,
    debug: u32,
) -> Result<u64, Escape> {
    let stylus = StylusConfig {
        version,
        max_depth,
        pricing: PricingParams { ink_price },
    };
    let compile = CompileConfig::version(version, debug != 0);
    let res = heapify(JitConfig { stylus, compile });
    Ok(res as u64)
}

pub fn create_evm_data(
    env: WasmEnvMut,
    block_basefee_ptr: GuestPtr,
    chainid: u64,
    block_coinbase_ptr: GuestPtr,
    block_gas_limit: u64,
    block_number: u64,
    block_timestamp: u64,
    contract_address_ptr: GuestPtr,
    module_hash_ptr: GuestPtr,
    msg_sender_ptr: GuestPtr,
    msg_value_ptr: GuestPtr,
    tx_gas_price_ptr: GuestPtr,
    tx_origin_ptr: GuestPtr,
    cached: u32,
    reentrant: u32,
) -> Result<u64, Escape> {
    create_evm_data_v2(
        env,
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

/// Creates an `EvmData` handler from its component parts.
pub fn create_evm_data_v2(
    mut env: WasmEnvMut,
    arbos_version: u64,
    block_basefee_ptr: GuestPtr,
    chainid: u64,
    block_coinbase_ptr: GuestPtr,
    block_gas_limit: u64,
    block_number: u64,
    block_timestamp: u64,
    contract_address_ptr: GuestPtr,
    module_hash_ptr: GuestPtr,
    msg_sender_ptr: GuestPtr,
    msg_value_ptr: GuestPtr,
    tx_gas_price_ptr: GuestPtr,
    tx_origin_ptr: GuestPtr,
    cached: u32,
    reentrant: u32,
) -> Result<u64, Escape> {
    let (mut mem, _) = env.jit_env();

    let evm_data = EvmData {
        arbos_version,
        block_basefee: mem.read_bytes32(block_basefee_ptr),
        cached: cached != 0,
        chainid,
        block_coinbase: mem.read_bytes20(block_coinbase_ptr),
        block_gas_limit,
        block_number,
        block_timestamp,
        contract_address: mem.read_bytes20(contract_address_ptr),
        module_hash: mem.read_bytes32(module_hash_ptr),
        msg_sender: mem.read_bytes20(msg_sender_ptr),
        msg_value: mem.read_bytes32(msg_value_ptr),
        tx_gas_price: mem.read_bytes32(tx_gas_price_ptr),
        tx_origin: mem.read_bytes20(tx_origin_ptr),
        reentrant,
        return_data_len: 0,
        tracing: false,
    };
    let res = heapify(evm_data);
    Ok(res as u64)
}
