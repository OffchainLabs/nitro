// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use crate::{
    gostack::GoStack,
    machine::{Escape, MaybeEscape, WasmEnv, WasmEnvMut},
    syscall::{DynamicObject, GoValue, JsValue, STYLUS_ID},
    user::evm::{EvmMsg, JitApi},
};
use arbutil::{heapify, Color, DebugColor};
use eyre::{bail, eyre, Result};
use parking_lot::Mutex;
use prover::{
    programs::{
        config::{EvmData, GoParams},
        prelude::*,
    },
    utils::{Bytes20, Bytes32},
};
use std::{
    mem,
    sync::{
        mpsc::{self, Sender, SyncSender},
        Arc,
    },
    thread,
    time::Duration,
};
use stylus::{
    native::{self, NativeInstance},
    run::RunProgram,
    EvmApi,
};
use wasmer::{FunctionEnv, FunctionEnvMut, StoreMut};

mod evm;

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
/// λ(mach *Machine, data []byte, params *Configs, api *GoApi, evmData: *EvmData, gas *u64, root *[32]byte)
///     -> (status byte, out *Vec<u8>)
pub fn call_user_wasm(mut env: WasmEnvMut, sp: u32) -> MaybeEscape {
    let mut sp = GoStack::simple(sp, &env);
    macro_rules! unbox {
        () => {
            unsafe { *Box::from_raw(sp.read_ptr_mut()) }
        };
    }
    use EvmMsg::*;
    use UserOutcomeKind::*;

    // move inputs
    let module: Vec<u8> = unbox!();
    let calldata = sp.read_go_slice_owned();
    let configs: (CompileConfig, StylusConfig) = unbox!();
    let evm = sp.read_go_slice_owned();
    let evm_data: EvmData = unbox!();

    // buy ink
    let pricing = configs.1.pricing;
    let gas = sp.read_go_ptr();
    let ink = pricing.gas_to_ink(sp.read_u64_raw(gas));

    // skip the root since we don't use these
    sp.skip_u64();

    let (tx, rx) = mpsc::sync_channel(0);
    let evm = JitApi::new(evm, tx.clone());

    let handle = thread::spawn(move || unsafe {
        // Safety: module came from compile_user_wasm
        let instance = NativeInstance::deserialize(&module, configs.0.clone(), evm, evm_data);
        let mut instance = match instance {
            Ok(instance) => instance,
            Err(error) => {
                let message = format!("failed to instantiate program {error:?}");
                tx.send(Panic(message.clone())).unwrap();
                panic!("{message}");
            }
        };

        let outcome = instance.run_main(&calldata, configs.1, ink);
        tx.send(Done).unwrap();

        let ink_left = match outcome.as_ref().map(|e| e.into()) {
            Ok(OutOfStack) => 0, // take all ink when out of stack
            _ => instance.ink_left().into(),
        };
        (outcome, ink_left)
    });

    loop {
        let msg = match rx.recv_timeout(Duration::from_secs(15)) {
            Ok(msg) => msg,
            Err(err) => return Escape::hostio(format!("{err}")),
        };
        match msg {
            Call(func, args, respond) => {
                let (env, mut store) = env.data_and_store_mut();
                let js = &mut env.js_state;

                let mut objects = vec![];
                let mut object_ids = vec![];
                for arg in args {
                    let id = js.pool.insert(DynamicObject::Uint8Array(arg.0));
                    objects.push(GoValue::Object(id));
                    object_ids.push(id);
                }
                println!("Ready with objects {}", object_ids.debug_pink());

                let Some(DynamicObject::FunctionWrapper(func)) = js.pool.get(func) else {
                    return Escape::hostio(format!("missing func {}", func.red()))
                };

                js.set_pending_event(*func, JsValue::Ref(STYLUS_ID), objects);
                unsafe { sp.resume(env, &mut store)? };

                let outs = vec![];
                println!("Resumed with results {}", outs.debug_pink());
                for id in object_ids {
                    env.js_state.pool.remove(id);
                }
                respond.send(outs).unwrap();
            }
            Panic(error) => return Escape::hostio(error),
            Done => break,
        }
    }

    let (outcome, ink_left) = handle.join().unwrap();

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
/// go side: λ(origin u32) *EvmData
pub fn evm_data_impl(env: WasmEnvMut, sp: u32) {
    let mut sp = GoStack::simple(sp, &env);
    let origin = sp.read_go_ptr();
    let origin = sp.read_bytes20(origin.into());
    let evm_data = EvmData::new(origin.into());
    sp.write_ptr(heapify(evm_data));
}
