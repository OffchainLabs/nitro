// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{evm_api::ApiCaller, Program, PROGRAMS};
use arbutil::{
    evm::{js::JsEvmApi, user::UserOutcomeKind, EvmData},
    format::DebugBytes,
    heapify, wavm,
};
use go_abi::GoStack;
use prover::{
    programs::config::{PricingParams, StylusConfig},
    Machine,
};
use std::mem;

// these hostio methods allow the replay machine to modify itself
#[link(wasm_import_module = "hostio")]
extern "C" {
    fn wavm_link_module(hash: *const MemoryLeaf) -> u32;
    fn wavm_unlink_module();
}

// these dynamic hostio methods allow introspection into user modules
#[link(wasm_import_module = "hostio")]
extern "C" {
    fn program_set_ink(module: u32, internals: u32, ink: u64);
    fn program_set_stack(module: u32, internals: u32, stack: u32);
    fn program_ink_left(module: u32, internals: u32) -> u64;
    fn program_ink_status(module: u32, internals: u32) -> u32;
    fn program_stack_left(module: u32, internals: u32) -> u32;
    fn program_call_main(module: u32, main: u32, args_len: usize) -> u32;
}

#[repr(C, align(256))]
struct MemoryLeaf([u8; 32]);

/// Compiles and instruments a user wasm.
///
/// # Safety
///
/// The Go compiler expects the call to take the form
///     λ(wasm []byte, pageLimit, version u16, debug u32) (mach *Machine, info WasmInfo, err *Vec<u8>)
///
/// These values are placed on the stack as follows
///     stack:  || wasm... || pageLimit | version | debug || mach ptr || info... || err ptr ||
///     info:   || footprint | 2 pad | size ||
///
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_compileUserWasmRustImpl(
    sp: usize,
) {
    let mut sp = GoStack::new(sp);
    let wasm = sp.read_go_slice_owned();
    let page_limit = sp.read_u16();
    let version = sp.read_u16();
    let debug = sp.read_bool32();

    match Machine::new_user_stub(&wasm, page_limit, version, debug) {
        Ok((machine, info)) => {
            let footprint = info.footprint;
            let size = info.size;
            sp.write_ptr(heapify(machine));
            sp.write_u16(footprint).skip_u16().write_u32(size); // wasm info
            sp.write_nullptr();
        }
        Err(error) => {
            sp.write_nullptr();
            sp.skip_space(); // skip wasm info
            sp.write_ptr(heapify(error.debug_bytes()));
        }
    }
}

/// Links and executes a user wasm.
///
/// # Safety
///
/// The Go compiler expects the call to take the form
///     λ(
///           mach *Machine, calldata []byte, params *Configs, evmApi []byte, evmData: *EvmData,
///           gas *u64, root *[32]byte
///     ) -> (status byte, out *Vec<u8>)
///
/// These values are placed on the stack as follows
///     || mach || calldata... || params || evmApi... || evmData || gas || root || status | 3 pad | out ptr ||
///
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_callUserWasmRustImpl(
    sp: usize,
) {
    let mut sp = GoStack::new(sp);
    let machine: Machine = sp.unbox();
    let calldata = sp.read_go_slice_owned();
    let config: StylusConfig = sp.unbox();
    let evm_api = JsEvmApi::new(sp.read_go_slice_owned(), ApiCaller::new());
    let evm_data: EvmData = sp.unbox();

    // buy ink
    let pricing = config.pricing;
    let gas = sp.read_go_ptr();
    let ink = pricing.gas_to_ink(wavm::caller_load64(gas));

    // compute the module root, or accept one from the caller
    let root = sp.read_go_ptr();
    let root = (root != 0).then(|| wavm::read_bytes32(root));
    let module = root.unwrap_or_else(|| machine.main_module_hash());
    let (main, internals) = machine.program_info();

    // link the program and ready its instrumentation
    let module = wavm_link_module(&MemoryLeaf(module.0));
    program_set_ink(module, internals, ink);
    program_set_stack(module, internals, config.max_depth);

    // provide arguments
    let args_len = calldata.len();
    PROGRAMS.push(Program::new(calldata, evm_api, evm_data, config));

    // call the program
    let go_stack = sp.save_stack();
    let status = program_call_main(module, main, args_len);
    let outs = PROGRAMS.pop().unwrap().into_outs();
    sp.restore_stack(go_stack);

    /// cleans up and writes the output
    macro_rules! finish {
        ($status:expr, $ink_left:expr) => {
            finish!($status, std::ptr::null::<u8>(), $ink_left);
        };
        ($status:expr, $outs:expr, $ink_left:expr) => {{
            sp.write_u8($status as u8).skip_space();
            sp.write_ptr($outs);
            wavm::caller_store64(gas, pricing.ink_to_gas($ink_left));
            wavm_unlink_module();
            return;
        }};
    }

    // check if instrumentation stopped the program
    use UserOutcomeKind::*;
    if program_ink_status(module, internals) != 0 {
        finish!(OutOfInk, 0);
    }
    if program_stack_left(module, internals) == 0 {
        finish!(OutOfStack, 0);
    }

    // the program computed a final result
    let ink_left = program_ink_left(module, internals);
    finish!(status, heapify(outs), ink_left)
}

/// Reads the length of a rust `Vec`
/// Safety: λ(vec *Vec<u8>) (len u32)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_readRustVecLenImpl(
    sp: usize,
) {
    let mut sp = GoStack::new(sp);
    let vec: &Vec<u8> = &*sp.read_ptr();
    sp.write_u32(vec.len() as u32);
}

/// Copies the contents of a rust `Vec` into a go slice, dropping it in the process
/// Safety: λ(vec *Vec<u8>, dest []byte)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_rustVecIntoSliceImpl(
    sp: usize,
) {
    let mut sp = GoStack::new(sp);
    let vec: Vec<u8> = sp.unbox();
    let ptr: *mut u8 = sp.read_ptr_mut();
    wavm::write_slice(&vec, ptr as u64);
    mem::drop(vec)
}

/// Drops a `Machine`.
///
/// # Safety
///
/// The Go compiler expects the call to take the form
///     λ(mach *Machine)
///
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_rustMachineDropImpl(
    sp: usize,
) {
    let mut sp = GoStack::new(sp);
    if let Some(mach) = sp.unbox_option::<Machine>() {
        mem::drop(mach);
    }
}

/// Creates a `StylusConfig` from its component parts.
///
/// # Safety
///
/// The Go compiler expects the call to take the form
///     λ(version u16, maxDepth, inkPrice u32, debugMode u32) *StylusConfig
///
/// The values are placed on the stack as follows
///     || version | 2 garbage bytes | max_depth || ink_price | debugMode || result ptr ||
///
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_rustConfigImpl(
    sp: usize,
) {
    let mut sp = GoStack::new(sp);

    let config = StylusConfig {
        version: sp.read_u16(),
        max_depth: sp.skip_u16().read_u32(),
        pricing: PricingParams {
            ink_price: sp.read_u32(),
        },
    };
    sp.skip_u32(); // skip debugMode
    sp.write_ptr(heapify(config));
}

/// Creates an `EvmData` from its component parts.
///
/// # Safety
///
/// The Go compiler expects the call to take the form
///     λ(
///         blockBasefee *[32]byte, chainid u64, blockCoinbase *[20]byte, blockGasLimit,
///         blockNumber, blockTimestamp u64, contractAddress, msgSender *[20]byte,
///         msgValue, txGasPrice *[32]byte, txOrigin *[20]byte, reentrant u32,
///     ) -> *EvmData
///
/// These values are placed on the stack as follows
///     || baseFee || chainid || coinbase || gas limit || block number || timestamp || address ||
///     || sender || value || gas price || origin || reentrant | 4 pad || data ptr ||
///
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_rustEvmDataImpl(
    sp: usize,
) {
    use wavm::{read_bytes20, read_bytes32};
    let mut sp = GoStack::new(sp);
    let evm_data = EvmData {
        block_basefee: read_bytes32(sp.read_go_ptr()),
        chainid: sp.read_u64(),
        block_coinbase: read_bytes20(sp.read_go_ptr()),
        block_gas_limit: sp.read_u64(),
        block_number: sp.read_u64(),
        block_timestamp: sp.read_u64(),
        contract_address: read_bytes20(sp.read_go_ptr()),
        msg_sender: read_bytes20(sp.read_go_ptr()),
        msg_value: read_bytes32(sp.read_go_ptr()),
        tx_gas_price: read_bytes32(sp.read_go_ptr()),
        tx_origin: read_bytes20(sp.read_go_ptr()),
        reentrant: sp.read_u32(),
        return_data_len: 0,
    };
    sp.skip_space();
    sp.write_ptr(heapify(evm_data));
}
