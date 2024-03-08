// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::callerenv::CallerEnv;
use crate::machine::{Escape, MaybeEscape, WasmEnvMut};
use crate::stylus_backend::exec_wasm;
use arbutil::Bytes32;
use arbutil::{evm::EvmData, format::DebugBytes, heapify};
use eyre::eyre;
use prover::programs::prelude::StylusConfig;
use prover::{
    machine::Module,
    programs::{config::PricingParams, prelude::*},
};

type Uptr = u32;

/// activates a user program
pub fn activate(
    mut env: WasmEnvMut,
    wasm_ptr: Uptr,
    wasm_size: u32,
    pages_ptr: Uptr,
    asm_estimate_ptr: Uptr,
    init_gas_ptr: Uptr,
    version: u16,
    debug: u32,
    module_hash_ptr: Uptr,
    gas_ptr: Uptr,
    err_buf: Uptr,
    err_buf_len: u32,
) -> Result<u32, Escape> {
    let mut caller_env = CallerEnv::new(&mut env);
    let wasm = caller_env.caller_read_slice(wasm_ptr, wasm_size);
    let debug = debug != 0;

    let page_limit = caller_env.caller_read_u16(pages_ptr);
    let gas_left = &mut caller_env.caller_read_u64(gas_ptr);
    match Module::activate(&wasm, version, page_limit, debug, gas_left) {
        Ok((module, data)) => {
            caller_env.caller_write_u64(gas_ptr, *gas_left);
            caller_env.caller_write_u16(pages_ptr, data.footprint);
            caller_env.caller_write_u32(asm_estimate_ptr, data.asm_estimate);
            caller_env.caller_write_u32(init_gas_ptr, data.init_gas);
            caller_env.caller_write_bytes32(module_hash_ptr, module.hash());
            Ok(0)
        }
        Err(error) => {
            let mut err_bytes = error.wrap_err("failed to activate").debug_bytes();
            err_bytes.truncate(err_buf_len as usize);
            caller_env.caller_write_slice(err_buf, &err_bytes);
            caller_env.caller_write_u64(gas_ptr, 0);
            caller_env.caller_write_u16(pages_ptr, 0);
            caller_env.caller_write_bytes32(module_hash_ptr, Bytes32::default());
            Ok(err_bytes.len() as u32)
        }
    }
}

/// Links and creates user program (in jit starts it as well)
/// consumes both evm_data_handler and config_handler
/// returns module number
pub fn new_program(
    mut env: WasmEnvMut,
    compiled_hash_ptr: Uptr,
    calldata_ptr: Uptr,
    calldata_size: u32,
    stylus_config_handler: u64,
    evm_data_handler: u64,
    gas: u64,
) -> Result<u32, Escape> {
    let mut caller_env = CallerEnv::new(&mut env);
    let compiled_hash = caller_env.caller_read_bytes32(compiled_hash_ptr);
    let calldata = caller_env.caller_read_slice(calldata_ptr, calldata_size);
    let evm_data: EvmData = unsafe { *Box::from_raw(evm_data_handler as *mut EvmData) };
    let config: JitConfig = unsafe { *Box::from_raw(stylus_config_handler as *mut JitConfig) };

    // buy ink
    let pricing = config.stylus.pricing;
    let ink = pricing.gas_to_ink(gas);

    let Some(module) = caller_env.wenv.module_asms.get(&compiled_hash).cloned() else {
        return Err(Escape::Failure(format!(
            "module hash {:?} not found in {:?}",
            compiled_hash,
            caller_env.wenv.module_asms.keys()
        )));
    };

    let cothread = exec_wasm(
        module,
        calldata,
        config.compile,
        config.stylus,
        evm_data,
        ink,
    )
    .unwrap();

    caller_env.wenv.threads.push(cothread);

    Ok(caller_env.wenv.threads.len() as u32)
}

/// starts the program (in jit waits for first request)
/// module MUST match last module number returned from new_program
/// returns request_id for the first request from the program
pub fn start_program(mut env: WasmEnvMut, module: u32) -> Result<u32, Escape> {
    let caller_env = CallerEnv::new(&mut env);

    if caller_env.wenv.threads.len() as u32 != module || module == 0 {
        return Escape::hostio(format!(
            "got request for thread {module} but len is {}",
            caller_env.wenv.threads.len()
        ));
    }
    let thread = caller_env.wenv.threads.last_mut().unwrap();
    thread.wait_next_message()?;
    let msg = thread.last_message()?;
    Ok(msg.1)
}

// gets information about request according to id
// request_id MUST be last request id returned from start_program or send_response
pub fn get_request(mut env: WasmEnvMut, id: u32, len_ptr: u32) -> Result<u32, Escape> {
    let mut caller_env = CallerEnv::new(&mut env);
    let thread = caller_env.wenv.threads.last_mut().unwrap();
    let msg = thread.last_message()?;
    if msg.1 != id {
        return Escape::hostio("get_request id doesn't match");
    };
    caller_env.caller_write_u32(len_ptr, msg.0.req_data.len() as u32);
    Ok(msg.0.req_type)
}

// gets data associated with last request.
// request_id MUST be last request receieved
// data_ptr MUST point to a buffer of at least the length returned by get_request
pub fn get_request_data(mut env: WasmEnvMut, id: u32, data_ptr: u32) -> MaybeEscape {
    let caller_env = CallerEnv::new(&mut env);
    let thread = caller_env.wenv.threads.last_mut().unwrap();
    let msg = thread.last_message()?;
    if msg.1 != id {
        return Escape::hostio("get_request id doesn't match");
    };
    caller_env.caller_write_slice(data_ptr, &msg.0.req_data);
    Ok(())
}

// sets response for the next request made
// id MUST be the id of last request made
pub fn set_response(
    mut env: WasmEnvMut,
    id: u32,
    gas: u64,
    result_ptr: Uptr,
    result_len: u32,
    raw_data_ptr: Uptr,
    raw_data_len: u32,
) -> MaybeEscape {
    let caller_env = CallerEnv::new(&mut env);
    let result = caller_env.caller_read_slice(result_ptr, result_len);
    let raw_data = caller_env.caller_read_slice(raw_data_ptr, raw_data_len);

    let thread = caller_env.wenv.threads.last_mut().unwrap();
    thread.set_response(id, result, raw_data, gas)
}

// sends previos response
// MUST be called right after set_response to the same id
// returns request_id for the next request
pub fn send_response(mut env: WasmEnvMut, req_id: u32) -> Result<u32, Escape> {
    let caller_env = CallerEnv::new(&mut env);
    let thread = caller_env.wenv.threads.last_mut().unwrap();
    let msg = thread.last_message()?;
    if msg.1 != req_id {
        return Escape::hostio("get_request id doesn't match");
    };
    thread.wait_next_message()?;
    let msg = thread.last_message()?;
    Ok(msg.1)
}

// removes the last created program
pub fn pop(mut env: WasmEnvMut) -> MaybeEscape {
    let caller_env = CallerEnv::new(&mut env);

    match caller_env.wenv.threads.pop() {
        None => Err(Escape::Child(eyre!("no child"))),
        Some(mut thread) => thread.wait_done(),
    }
}

pub struct JitConfig {
    stylus: StylusConfig,
    compile: CompileConfig,
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

/// Creates an `EvmData` handler from its component parts.
///
pub fn create_evm_data(
    mut env: WasmEnvMut,
    block_basefee_ptr: Uptr,
    chainid: u64,
    block_coinbase_ptr: Uptr,
    block_gas_limit: u64,
    block_number: u64,
    block_timestamp: u64,
    contract_address_ptr: Uptr,
    msg_sender_ptr: Uptr,
    msg_value_ptr: Uptr,
    tx_gas_price_ptr: Uptr,
    tx_origin_ptr: Uptr,
    reentrant: u32,
) -> Result<u64, Escape> {
    let mut caller_env = CallerEnv::new(&mut env);

    let evm_data = EvmData {
        block_basefee: caller_env.caller_read_bytes32(block_basefee_ptr),
        chainid,
        block_coinbase: caller_env.caller_read_bytes20(block_coinbase_ptr),
        block_gas_limit,
        block_number,
        block_timestamp,
        contract_address: caller_env.caller_read_bytes20(contract_address_ptr),
        msg_sender: caller_env.caller_read_bytes20(msg_sender_ptr),
        msg_value: caller_env.caller_read_bytes32(msg_value_ptr),
        tx_gas_price: caller_env.caller_read_bytes32(tx_gas_price_ptr),
        tx_origin: caller_env.caller_read_bytes20(tx_origin_ptr),
        reentrant,
        return_data_len: 0,
        tracing: false,
    };
    let res = heapify(evm_data);
    Ok(res as u64)
}
