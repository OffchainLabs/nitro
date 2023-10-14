// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    gostack::GoStack,
    machine::{Escape, MaybeEscape, WasmEnvMut},
    user::evm_api::exec_wasm,
    wavmio::Bytes32,
};
use arbutil::{
    evm::{user::UserOutcome, EvmData},
    format::DebugBytes,
    heapify,
};
use prover::programs::{config::PricingParams, prelude::*};
use std::mem;
use stylus::native;

mod evm_api;

/// Instruments and "activates" a user wasm, producing a unique module hash.
///
/// Note that this operation costs gas and is limited by the amount supplied via the `gas` pointer.
/// The amount left is written back at the end of the call.
///
/// # Go side
///
/// The `modHash` and `gas` pointers must not be null.
///
/// The Go compiler expects the call to take the form
///     λ(wasm []byte, pageLimit, version u16, debug u32, modHash *hash, gas *u64) (footprint u16, err *Vec<u8>)
///
/// These values are placed on the stack as follows
///     || wasm... || pageLimit | version | debug || modhash ptr || gas ptr || footprint | 6 pad || err ptr ||
///
pub fn activate_wasm(env: WasmEnvMut, sp: u32) {
    let mut sp = GoStack::simple(sp, &env);
    let wasm = sp.read_go_slice_owned();
    let page_limit = sp.read_u16();
    let version = sp.read_u16();
    let debug = sp.read_bool32();
    let module_hash = sp.read_go_ptr();
    let gas = sp.read_go_ptr();

    macro_rules! error {
        ($error:expr) => {{
            let error = $error.wrap_err("failed to activate").debug_bytes();
            sp.write_u64_raw(gas, 0);
            sp.write_slice(module_hash, &Bytes32::default());
            sp.skip_space();
            sp.write_ptr(heapify(error));
            return;
        }};
    }

    let gas_left = &mut sp.read_u64_raw(gas);
    let (_, module, pages) = match native::activate(&wasm, version, page_limit, debug, gas_left) {
        Ok(result) => result,
        Err(error) => error!(error),
    };
    sp.write_u64_raw(gas, *gas_left);
    sp.write_slice(module_hash, &module.hash().0);
    sp.write_u16(pages).skip_space();
    sp.write_nullptr();
}

/// Links and executes a user wasm.
///
/// # Go side
///
/// The Go compiler expects the call to take the form
///     λ(moduleHash *[32]byte, calldata []byte, params *Configs, evmApi []byte, evmData: *EvmData, gas *u64) (
///         status byte, out *Vec<u8>,
///     )
///
/// These values are placed on the stack as follows
///     || modHash || calldata... || params || evmApi... || evmData || gas || status | 7 pad | out ptr ||
///
pub fn call_user_wasm(env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let sp = &mut GoStack::simple(sp, &env);
    use UserOutcome::*;

    // move inputs
    let module_hash = sp.read_bytes32();
    let calldata = sp.read_go_slice_owned();
    let (compile, config): (CompileConfig, StylusConfig) = sp.unbox();
    let evm_api = sp.read_go_slice_owned();
    let evm_data: EvmData = sp.unbox();
    let gas = sp.read_go_ptr();

    // buy ink
    let pricing = config.pricing;
    let ink = pricing.gas_to_ink(sp.read_u64_raw(gas));

    let Some(module) = env.data().module_asms.get(&module_hash).cloned() else {
        return Escape::failure(format!(
            "module hash {module_hash:?} not found in {:?}",
            env.data().module_asms.keys()
        ));
    };

    let result = exec_wasm(
        sp, env, module, calldata, compile, config, evm_api, evm_data, ink,
    );
    let (outcome, ink_left) = result.map_err(Escape::Child)?;

    let outcome = match outcome {
        Err(e) | Ok(Failure(e)) => Failure(e.wrap_err("call failed")),
        Ok(outcome) => outcome,
    };
    let (kind, outs) = outcome.into_data();
    sp.write_u8(kind.into()).skip_space();
    sp.write_ptr(heapify(outs));
    sp.write_u64_raw(gas, pricing.ink_to_gas(ink_left));
    Ok(())
}

/// Reads the length of a rust `Vec`
///
/// # Go side
///
/// The Go compiler expects the call to take the form
///     λ(vec *Vec<u8>) (len u32)
///
/// These values are placed on the stack as follows
///     || vec ptr || len u32 | pad 4 ||
///
pub fn read_rust_vec_len(env: WasmEnvMut, sp: u32) {
    let mut sp = GoStack::simple(sp, &env);
    let vec: &Vec<u8> = unsafe { sp.read_ref() };
    sp.write_u32(vec.len() as u32);
}

/// Copies the contents of a rust `Vec` into a go slice, dropping it in the process
///
/// # Go Side
///
/// The Go compiler expects the call to take the form
///     λ(vec *Vec<u8>, dest []byte)
///
/// These values are placed on the stack as follows
///     || vec ptr || dest... ||
///
pub fn rust_vec_into_slice(env: WasmEnvMut, sp: u32) {
    let mut sp = GoStack::simple(sp, &env);
    let vec: Vec<u8> = sp.unbox();
    let ptr: *mut u8 = sp.read_ptr_mut();
    sp.write_slice(ptr as u64, &vec);
    mem::drop(vec)
}

/// Creates a `StylusConfig` from its component parts.
///
/// # Go side
///
/// The Go compiler expects the call to take the form
///     λ(version u16, maxDepth, inkPrice u32, debugMode: u32) *(CompileConfig, StylusConfig)
///
/// The values are placed on the stack as follows
///     || version | 2 garbage bytes | max_depth || ink_price | debugMode || result ptr ||
///
pub fn rust_config_impl(env: WasmEnvMut, sp: u32) {
    let mut sp = GoStack::simple(sp, &env);

    let config = StylusConfig {
        version: sp.read_u16(),
        max_depth: sp.skip_u16().read_u32(),
        pricing: PricingParams {
            ink_price: sp.read_u32(),
        },
    };
    let compile = CompileConfig::version(config.version, sp.read_u32() != 0);
    sp.write_ptr(heapify((compile, config)));
}

/// Creates an `EvmData` from its component parts.
///
/// # Go side
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
pub fn evm_data_impl(env: WasmEnvMut, sp: u32) {
    let mut sp = GoStack::simple(sp, &env);
    let evm_data = EvmData {
        block_basefee: sp.read_bytes32().into(),
        chainid: sp.read_u64(),
        block_coinbase: sp.read_bytes20().into(),
        block_gas_limit: sp.read_u64(),
        block_number: sp.read_u64(),
        block_timestamp: sp.read_u64(),
        contract_address: sp.read_bytes20().into(),
        msg_sender: sp.read_bytes20().into(),
        msg_value: sp.read_bytes32().into(),
        tx_gas_price: sp.read_bytes32().into(),
        tx_origin: sp.read_bytes20().into(),
        reentrant: sp.read_u32(),
        return_data_len: 0,
        tracing: false,
    };
    sp.skip_space();
    sp.write_ptr(heapify(evm_data));
}
