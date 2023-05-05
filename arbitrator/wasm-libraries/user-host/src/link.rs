// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{evm_api::ApiCaller, Program, PROGRAMS};
use arbutil::{
    evm::{js::JsEvmApi, user::UserOutcomeKind, EvmData},
    heapify, wavm,
};
use fnv::FnvHashMap as HashMap;
use go_abi::GoStack;
use prover::{
    programs::config::{CompileConfig, GoParams, StylusConfig},
    Machine,
};
use std::{mem, path::Path, sync::Arc};

// these hostio methods allow the replay machine to modify itself
#[link(wasm_import_module = "hostio")]
extern "C" {
    fn link_module(hash: *const MemoryLeaf) -> u32;
    fn unlink_module();
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

/// Compiles and instruments user wasm.
/// Safety: λ(wasm []byte, version, debug u32) (machine *Machine, err *Vec<u8>)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_compileUserWasmRustImpl(
    sp: usize,
) {
    let mut sp = GoStack::new(sp);
    let wasm = sp.read_go_slice_owned();
    let compile = CompileConfig::version(sp.read_u32(), sp.read_u32() != 0);

    macro_rules! error {
        ($msg:expr, $error:expr) => {{
            let error = format!("{}: {:?}", $msg, $error).as_bytes().to_vec();
            sp.write_nullptr();
            sp.write_ptr(heapify(error));
            return;
        }};
    }

    let mut bin = match prover::binary::parse(&wasm, Path::new("user")) {
        Ok(bin) => bin,
        Err(err) => error!("failed to parse user program", err),
    };
    let stylus_data = match bin.instrument(&compile) {
        Ok(stylus_data) => stylus_data,
        Err(err) => error!("failed to instrument user program", err),
    };

    let forward = include_bytes!("../../../../target/machines/latest/forward_stub.wasm");
    let forward = prover::binary::parse(forward, Path::new("forward")).unwrap();

    let machine = Machine::from_binaries(
        &[forward],
        bin,
        false,
        false,
        false,
        compile.debug.debug_funcs,
        prover::machine::GlobalState::default(),
        HashMap::default(),
        Arc::new(|_, _| panic!("user program tried to read preimage")),
        Some(stylus_data),
    );
    let machine = match machine {
        Ok(machine) => machine,
        Err(err) => error!("failed to instrument user program", err),
    };
    sp.write_ptr(heapify(machine));
    sp.write_nullptr();
}

/// Links and executes a user wasm.
/// λ(mach *Machine, calldata []byte, params *Config, evmApi []byte, evmData *EvmData, gas *u64, root *[32]byte)
///     -> (status byte, out *Vec<u8>)
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_callUserWasmRustImpl(
    sp: usize,
) {
    let mut sp = GoStack::new(sp);
    macro_rules! unbox {
        () => {
            *Box::from_raw(sp.read_ptr_mut())
        };
    }
    let machine: Machine = unbox!();
    let calldata = sp.read_go_slice_owned();
    let config: StylusConfig = unbox!();
    let evm_api = JsEvmApi::new(sp.read_go_slice_owned(), ApiCaller::new());
    let evm_data: EvmData = unbox!();

    // buy ink
    let pricing = config.pricing;
    let gas = sp.read_go_ptr();
    let ink = pricing.gas_to_ink(wavm::caller_load64(gas));

    // compute the module root, or accept one from the caller
    let root = sp.read_go_ptr();
    let root = (root != 0).then(|| wavm::read_bytes32(root));
    let module = root.unwrap_or_else(|| machine.main_module_hash().0);
    let (main, internals) = machine.program_info();

    // link the program and ready its instrumentation
    let module = link_module(&MemoryLeaf(module));
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
            unlink_module();
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
    let vec: Vec<u8> = *Box::from_raw(sp.read_ptr_mut());
    let ptr: *mut u8 = sp.read_ptr_mut();
    wavm::write_slice(&vec, ptr as u64);
    mem::drop(vec)
}

/// Creates a `StylusConfig` from its component parts.
/// Safety: λ(version, maxDepth u32, inkGasPrice, hostioInk u64, debugMode u32) *StylusConfig
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_rustConfigImpl(
    sp: usize,
) {
    let mut sp = GoStack::new(sp);
    let params = GoParams {
        version: sp.read_u32(),
        max_depth: sp.read_u32(),
        ink_price: sp.read_u64(),
        hostio_ink: sp.read_u64(),
        debug_mode: sp.read_u32(),
    };
    sp.skip_space().write_ptr(heapify(params.configs().1));
}

/// Creates an `EvmData` from its component parts.
/// Safety: λ(block_basefee u32, block_chainid u32, block_coinbase u32, block_difficulty u32,
///           block_gas_limit u32, block_number u32, block_timestamp u32, contract_address u32,
///           msg_sender u32, msg_value u32, tx_gas_price u32, tx_origin u32) *EvmData
#[no_mangle]
pub unsafe extern "C" fn go__github_com_offchainlabs_nitro_arbos_programs_rustEvmDataImpl(
    sp: usize,
) {
    let mut sp = GoStack::new(sp);
    let block_basefee = wavm::read_bytes32(sp.read_go_ptr().into());
    let block_chainid = wavm::read_bytes32(sp.read_go_ptr().into());
    let block_coinbase = wavm::read_bytes20(sp.read_go_ptr().into());
    let block_difficulty = wavm::read_bytes32(sp.read_go_ptr().into());
    let block_gas_limit = wavm::caller_load64(sp.read_go_ptr().into());
    let block_number = wavm::read_bytes32(sp.read_go_ptr().into());
    let block_timestamp = wavm::read_bytes32(sp.read_go_ptr().into());
    let contract_address = wavm::read_bytes20(sp.read_go_ptr().into());
    let msg_sender = wavm::read_bytes20(sp.read_go_ptr().into());
    let msg_value = wavm::read_bytes32(sp.read_go_ptr().into());
    let tx_gas_price = wavm::read_bytes32(sp.read_go_ptr().into());
    let tx_origin = wavm::read_bytes20(sp.read_go_ptr());
    let evm_data = EvmData::new(
        block_basefee.into(),
        block_chainid.into(),
        block_coinbase.into(),
        block_difficulty.into(),
        block_gas_limit.into(),
        block_number.into(),
        block_timestamp.into(),
        contract_address.into(),
        msg_sender.into(),
        msg_value.into(),
        tx_gas_price.into(),
        tx_origin.into(),
    );
    sp.write_ptr(heapify(evm_data));
}
