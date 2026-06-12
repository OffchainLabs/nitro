//! Programs-related host APIs for launching and driving stylus programs.
//! These mirror the JIT implementations in `crates/jit/src/program.rs` and
//! must keep behaving identically to them.

#![allow(clippy::too_many_arguments)]

use arbutil::{
    Bytes32,
    evm::{EvmData, api::Gas},
    format::DebugBytes,
};
use caller_env::{GuestPtr, MemAccess};
use prover::{
    machine::Module,
    programs::config::{CompileConfig, PricingParams, StylusConfig},
};
use wasmer::FunctionEnvMut;

use crate::{
    Escape, JitConfig, MaybeEscape, replay::CustomEnvData, state::sp1_env,
    stylus::MessageToCothread,
};

/// Hardcoded message ID used by the Arbitrator protocol for program communication.
const ARBITRATOR_MSG_ID: u32 = 0x33333333;

pub fn new_program(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    compiled_hash_ptr: GuestPtr,
    calldata_ptr: GuestPtr,
    calldata_size: u32,
    stylus_config_handler: u64,
    evm_data_handler: u64,
    gas: u64,
) -> Result<u32, Escape> {
    let (mem, data) = sp1_env(&mut ctx);
    let compiled_hash = mem.read_bytes32(compiled_hash_ptr);
    let calldata = mem.read_slice(calldata_ptr, calldata_size as usize);
    let evm_data: EvmData = unsafe { *Box::from_raw(evm_data_handler as *mut EvmData) };
    let config: JitConfig = unsafe { *Box::from_raw(stylus_config_handler as *mut JitConfig) };

    data.launch_program(&compiled_hash, calldata, config, evm_data, gas)
}

/// Removes the last created program
pub fn pop(mut ctx: FunctionEnvMut<CustomEnvData>) -> MaybeEscape {
    ctx.data_mut().pop_last_program()
}

pub fn set_response(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    id: u32,
    gas: u64,
    result_ptr: GuestPtr,
    result_len: u32,
    raw_data_ptr: GuestPtr,
    raw_data_len: u32,
) -> MaybeEscape {
    let (mem, data) = sp1_env(&mut ctx);

    // Arbitrator for now only uses hardcoded id, we can ignore
    // ids safely.
    assert_eq!(id, ARBITRATOR_MSG_ID);

    let result = mem.read_slice(result_ptr, result_len as usize);
    let raw_data = mem.read_slice(raw_data_ptr, raw_data_len as usize);

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
    len_ptr: GuestPtr,
) -> Result<u32, Escape> {
    let (mut mem, data) = sp1_env(&mut ctx);

    // Arbitrator for now only uses hardcoded id, we can ignore
    // ids safely.
    assert_eq!(id, ARBITRATOR_MSG_ID);

    let msg = data.get_last_msg();
    let len: u32 = msg
        .req_data
        .len()
        .try_into()
        .expect("req_data length exceeds u32::MAX");
    mem.write_u32(len_ptr, len);

    Ok(msg.req_type)
}

pub fn get_request_data(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    id: u32,
    data_ptr: GuestPtr,
) -> MaybeEscape {
    let (mut mem, data) = sp1_env(&mut ctx);

    // Arbitrator for now only uses hardcoded id, we can ignore
    // ids safely.
    assert_eq!(id, ARBITRATOR_MSG_ID);

    let msg = data.get_last_msg();
    mem.write_slice(data_ptr, &msg.req_data);

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
    let (mem, _) = sp1_env(&mut ctx);

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

pub fn activate(
    ctx: FunctionEnvMut<CustomEnvData>,
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

/// Activates a user program, mirroring `activate_v2` in `crates/jit/src/program.rs`.
///
/// NOTE: `Module::activate` is the same code the JIT and the arbitrator's
/// wasm32 build (`user-host/src/link.rs`) run, so the module hash matches
/// them bit-for-bit. Running it natively in the guest also keeps keccak on
/// SP1's syscall-patched `tiny-keccak`. If activation cycles in the main
/// proof ever become a problem, the escalation path is a separate provable
/// activation program (see `stylus-compiler-program`), not an in-guest wasm
/// module (which would lose the keccak syscall acceleration).
pub fn activate_v2(
    mut ctx: FunctionEnvMut<CustomEnvData>,
    wasm_ptr: GuestPtr,
    wasm_size: u32,
    pages_ptr: GuestPtr,
    asm_estimate_ptr: GuestPtr,
    init_cost_ptr: GuestPtr,
    cached_init_cost_ptr: GuestPtr,
    stylus_version: u16,
    arbos_version_for_activation: u64,
    debug: u32,
    codehash: GuestPtr,
    module_hash_ptr: GuestPtr,
    gas_ptr: GuestPtr,
    err_buf: GuestPtr,
    err_buf_len: u32,
) -> Result<u32, Escape> {
    let (mut mem, _) = sp1_env(&mut ctx);
    let wasm = mem.read_slice(wasm_ptr, wasm_size as usize);
    let codehash = &mem.read_bytes32(codehash);
    let debug = debug != 0;

    let page_limit = mem.read_u16(pages_ptr);
    let gas_left = &mut mem.read_u64(gas_ptr);
    match Module::activate(
        &wasm,
        codehash,
        stylus_version,
        arbos_version_for_activation,
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

fn heapify<T>(value: T) -> *mut T {
    Box::into_raw(Box::new(value))
}

/// program_requires_prepare
pub fn program_requires_prepare(
    mut _env: FunctionEnvMut<CustomEnvData>,
    _module_hash_ptr: GuestPtr,
) -> Result<u32, Escape> {
    Ok(0)
}

/// program_prepare
pub fn program_prepare(
    mut _env: FunctionEnvMut<CustomEnvData>,
    _wasm_ptr: GuestPtr,
    _wasm_size: u64,
    _module_hash_ptr: GuestPtr,
    _code_hash_ptr: GuestPtr,
    _max_wasm_size: u32,
    _page_limit: u32,
    _debug_mode: u32,
    _stylus_version: u32,
) -> MaybeEscape {
    Ok(())
}
