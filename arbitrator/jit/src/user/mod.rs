// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    gostack::GoStack,
    machine::{Escape, MaybeEscape, WasmEnvMut},
    user::evm_api::exec_wasm,
};
use arbutil::{
    evm::{user::UserOutcome, EvmData},
    heapify,
};
use prover::{
    binary::WasmBinary,
    machine as prover_machine,
    programs::{config::PricingParams, prelude::*},
};
use std::mem;
use stylus::native;

mod evm_api;

/// Compiles and instruments user wasm.
/// go side: λ(wasm []byte, version, debug u32, pageLimit u16, machineHash []byte) (module *Module, footprint u16, err *Vec<u8>)

pub fn compile_user_wasm(env: WasmEnvMut, sp: u32) {
    let mut sp = GoStack::simple(sp, &env);
    let wasm = sp.read_go_slice_owned();
    let compile = CompileConfig::version(sp.read_u32(), sp.read_u32() != 0);
    let page_limit = sp.read_u16();
    sp.skip_space();
    let (out_hash_ptr, out_hash_len) = sp.read_go_slice();

    macro_rules! error {
        ($error:expr) => {{
            let error = $error.wrap_err("failed to compile");
            let error = format!("{:?}", error).as_bytes().to_vec();
            sp.write_nullptr();
            sp.skip_space(); // skip footprint
            sp.write_ptr(heapify(error));
            return;
        }};
    }

    // ensure the wasm compiles during proving
    let (bin, stylus_data, footprint) = match WasmBinary::parse_user(&wasm, page_limit, &compile) {
        Ok(result) => result,
        Err(error) => error!(error),
    };

    let prover_module =
        match prover_machine::Module::from_user_binary(&bin, compile.debug.debug_funcs, Some(stylus_data)) {
            Ok(prover_module) => prover_module,
            Err(error) => error!(error),
        };

    let module = match native::module(&wasm, compile) {
        Ok(module) => module,
        Err(error) => error!(error),
    };

    if out_hash_len != 32 {
        error!(eyre::eyre!(
            "Go attempting to read compiled machine hash into bad buffer length: {out_hash_len}"
        ));
    }

    let canonical_hash = prover_module.hash();
    sp.write_slice(out_hash_ptr, canonical_hash.as_slice());
    sp.write_ptr(heapify(module));
    sp.write_u16(footprint).skip_space();
    sp.write_nullptr();
}

/// Links and executes a user wasm.
/// λ(mach *Machine, calldata []byte, params *Configs, evmApi []byte, evmData: *EvmData, gas *u64, root *[32]byte)
///     -> (status byte, out *Vec<u8>)
pub fn call_user_wasm(env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let sp = &mut GoStack::simple(sp, &env);
    use UserOutcome::*;

    // move inputs
    let module: Vec<u8> = sp.unbox();
    let calldata = sp.read_go_slice_owned();
    let (compile, config): (CompileConfig, StylusConfig) = sp.unbox();
    let evm_api = sp.read_go_slice_owned();
    let evm_data: EvmData = sp.unbox();

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

    let config = StylusConfig {
        version: sp.read_u32(),
        max_depth: sp.read_u32(),
        pricing: PricingParams {
            ink_price: sp.read_u64(),
            hostio_ink: sp.read_u64(),
        },
    };
    let compile = CompileConfig::version(config.version, sp.read_u32() != 0);
    sp.skip_space().write_ptr(heapify((compile, config)));
}

/// Creates an `EvmData` from its component parts.
/// go side: λ(
///     blockBasefee, blockChainid *[32]byte, blockCoinbase *[20]byte,
///     blockGasLimit u64, blockNumber *[32]byte, blockTimestamp u64, contractAddress, msgSender *[20]byte,
///     msgValue, txGasPrice *[32]byte, txOrigin *[20]byte,
///) *EvmData
pub fn evm_data_impl(env: WasmEnvMut, sp: u32) {
    let mut sp = GoStack::simple(sp, &env);
    let evm_data = EvmData {
        block_basefee: sp.read_bytes32().into(),
        block_chainid: sp.read_bytes32().into(),
        block_coinbase: sp.read_bytes20().into(),
        block_gas_limit: sp.read_u64(),
        block_number: sp.read_bytes32().into(),
        block_timestamp: sp.read_u64(),
        contract_address: sp.read_bytes20().into(),
        msg_sender: sp.read_bytes20().into(),
        msg_value: sp.read_bytes32().into(),
        tx_gas_price: sp.read_bytes32().into(),
        tx_origin: sp.read_bytes20().into(),
        return_data_len: 0,
    };
    sp.write_ptr(heapify(evm_data));
}
