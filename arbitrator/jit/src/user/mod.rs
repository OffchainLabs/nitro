// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    gostack::GoStack,
    machine::{Escape, MaybeEscape, WasmEnvMut},
    user::evm_api::exec_wasm,
};
use arbutil::{
    evm::{
        user::{UserOutcome, UserOutcomeKind},
        EvmData,
    },
    heapify,
};
use eyre::eyre;
use prover::programs::{config::GoParams, prelude::*};
use std::mem;
use stylus::native;

mod evm_api;

/// Compiles and instruments user wasm.
/// go side: λ(wasm []byte, version, debug u32) (machine *Machine, err *Vec<u8>)
pub fn compile_user_wasm(env: WasmEnvMut, sp: u32) {
    let mut sp = GoStack::simple(sp, &env);
    let wasm = sp.read_go_slice_owned();
    let compile = CompileConfig::version(sp.read_u32(), sp.read_u32() != 0);

    match native::module(&wasm, compile) {
        Ok(module) => {
            sp.write_ptr(heapify(module));
            sp.write_nullptr();
        }
        Err(error) => {
            let error = format!("failed to compile: {error:?}").as_bytes().to_vec();
            sp.write_nullptr();
            sp.write_ptr(heapify(error));
        }
    }
}

/// Links and executes a user wasm.
/// λ(mach *Machine, calldata []byte, params *Configs, evmApi []byte, evmData: *EvmData, gas *u64, root *[32]byte)
///     -> (status byte, out *Vec<u8>)
pub fn call_user_wasm(env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let sp = &mut GoStack::simple(sp, &env);
    macro_rules! unbox {
        () => {
            unsafe { *Box::from_raw(sp.read_ptr_mut()) }
        };
    }
    use UserOutcomeKind::*;

    // move inputs
    let module: Vec<u8> = unbox!();
    let calldata = sp.read_go_slice_owned();
    let (compile, config): (CompileConfig, StylusConfig) = unbox!();
    let evm_api = sp.read_go_slice_owned();
    let evm_data: EvmData = unbox!();

    // buy ink
    let pricing = config.pricing;
    let gas = sp.read_go_ptr();
    let ink = pricing.gas_to_ink(sp.read_u64_raw(gas));

    // skip the root since we don't use these
    sp.skip_u64();

    let result = exec_wasm(
        sp, env, module, calldata, compile, config, evm_api, evm_data, ink,
    );
    let (outcome, ink_left) = result.map_err(Escape::Child)?;

    match outcome {
        Err(err) | Ok(UserOutcome::Failure(err)) => {
            let outs = format!("{:?}", err.wrap_err(eyre!("failed to execute program")));
            sp.write_u8(Failure.into()).skip_space();
            sp.write_ptr(heapify(outs.into_bytes()));
        }
        Ok(outcome) => {
            let (status, outs) = outcome.into_data();
            sp.write_u8(status.into()).skip_space();
            sp.write_ptr(heapify(outs));
        }
    }
    sp.write_u64_raw(gas, pricing.ink_to_gas(ink_left));
    Ok(())
}

/// Reads the length of a rust `Vec`
/// go side: λ(vec *Vec<u8>) (len u32)
pub fn read_rust_vec_len(env: WasmEnvMut, sp: u32) {
    let mut sp = GoStack::simple(sp, &env);
    let vec: &Vec<u8> = unsafe { &*sp.read_ptr() };
    sp.write_u32(vec.len() as u32);
}

/// Copies the contents of a rust `Vec` into a go slice, dropping it in the process
/// go side: λ(vec *Vec<u8>, dest []byte)
pub fn rust_vec_into_slice(env: WasmEnvMut, sp: u32) {
    let mut sp = GoStack::simple(sp, &env);
    let vec: Vec<u8> = unsafe { *Box::from_raw(sp.read_ptr_mut()) };
    let ptr: *mut u8 = sp.read_ptr_mut();
    sp.write_slice(ptr as u64, &vec);
    mem::drop(vec)
}

/// Creates a `StylusConfig` from its component parts.
/// go side: λ(version, maxDepth u32, inkPrice, hostioInk u64, debugMode: u32) *(CompileConfig, StylusConfig)
pub fn rust_config_impl(env: WasmEnvMut, sp: u32) {
    let mut sp = GoStack::simple(sp, &env);
    let params = GoParams {
        version: sp.read_u32(),
        max_depth: sp.read_u32(),
        ink_price: sp.read_u64(),
        hostio_ink: sp.read_u64(),
        debug_mode: sp.read_u32(),
    };
    sp.skip_space().write_ptr(heapify(params.configs()));
}

/// Creates an `EvmData` from its component parts.
/// go side: λ(block_basefee u32, block_chainid u32, block_coinbase u32, block_difficulty u32,
///            block_gas_limit u32, block_number u32, block_timestamp u32, contract_address u32,
///            msg_sender u32, msg_value u32, tx_gas_price u32, tx_origin u32) *EvmData
pub fn evm_data_impl(env: WasmEnvMut, sp: u32) {
    let mut sp = GoStack::simple(sp, &env);
    let block_basefee = sp.read_go_ptr().into();
    let block_basefee = sp.read_bytes32(block_basefee);
    let block_chainid = sp.read_go_ptr().into();
    let block_chainid = sp.read_bytes32(block_chainid);
    let block_coinbase = sp.read_go_ptr().into();
    let block_coinbase = sp.read_bytes20(block_coinbase);
    let block_difficulty = sp.read_go_ptr().into();
    let block_difficulty = sp.read_bytes32(block_difficulty);
    let block_gas_limit = sp.read_u64();
    let block_number = sp.read_go_ptr().into();
    let block_number = sp.read_bytes32(block_number);
    let block_timestamp = sp.read_go_ptr().into();
    let block_timestamp = sp.read_bytes32(block_timestamp);
    let contract_address = sp.read_go_ptr().into();
    let contract_address = sp.read_bytes20(contract_address);
    let msg_sender = sp.read_go_ptr().into();
    let msg_sender = sp.read_bytes20(msg_sender);
    let msg_value = sp.read_go_ptr().into();
    let msg_value = sp.read_bytes32(msg_value);
    let tx_gas_price = sp.read_go_ptr().into();
    let tx_gas_price = sp.read_bytes32(tx_gas_price);
    let tx_origin = sp.read_go_ptr().into();
    let tx_origin = sp.read_bytes20(tx_origin);
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
