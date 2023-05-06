// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(clippy::too_many_arguments)]

use crate::{
    gostack::GoStack,
    machine::WasmEnvMut,
    syscall::{DynamicObject, GoValue, JsValue, STYLUS_ID},
};
use arbutil::{
    evm::{
        js::{ApiValue, JsCallIntoGo, JsEvmApi},
        user::{UserOutcome, UserOutcomeKind},
        EvmData,
    },
    Color,
};
use eyre::{bail, Result};
use prover::programs::prelude::*;
use std::{
    sync::mpsc::{self, SyncSender},
    thread,
};
use stylus::{native::NativeInstance, run::RunProgram};

struct ApiCaller {
    parent: SyncSender<EvmMsg>,
}

enum EvmMsg {
    Call(u32, Vec<ApiValue>, SyncSender<Vec<ApiValue>>),
    Panic(String),
    Done,
}

impl ApiCaller {
    fn new(parent: SyncSender<EvmMsg>) -> Self {
        Self { parent }
    }
}

impl JsCallIntoGo for ApiCaller {
    fn call_go(&mut self, func: u32, args: Vec<ApiValue>) -> Vec<ApiValue> {
        let (tx, rx) = mpsc::sync_channel(0);
        let msg = EvmMsg::Call(func, args, tx);
        self.parent.send(msg).unwrap();
        rx.recv().unwrap()
    }
}

/// Executes a wasm on a new thread
pub(super) fn exec_wasm(
    sp: &mut GoStack,
    mut env: WasmEnvMut,
    module: Vec<u8>,
    calldata: Vec<u8>,
    compile: CompileConfig,
    config: StylusConfig,
    evm_api: Vec<u8>,
    evm_data: EvmData,
    ink: u64,
) -> Result<(Result<UserOutcome>, u64)> {
    use EvmMsg::*;
    use UserOutcomeKind::*;

    let (tx, rx) = mpsc::sync_channel(0);
    let evm_api = JsEvmApi::new(evm_api, ApiCaller::new(tx.clone()));

    let handle = thread::spawn(move || unsafe {
        // Safety: module came from compile_user_wasm
        let instance = NativeInstance::deserialize(&module, compile.clone(), evm_api, evm_data);
        let mut instance = match instance {
            Ok(instance) => instance,
            Err(error) => {
                let message = format!("failed to instantiate program {error:?}");
                tx.send(Panic(message.clone())).unwrap();
                panic!("{message}");
            }
        };

        let outcome = instance.run_main(&calldata, config, ink);
        tx.send(Done).unwrap();

        let ink_left = match outcome.as_ref().map(|e| e.into()) {
            Ok(OutOfStack) => 0, // take all ink when out of stack
            _ => instance.ink_left().into(),
        };
        (outcome, ink_left)
    });

    loop {
        let msg = match rx.recv_timeout(env.data().process.child_timeout) {
            Ok(msg) => msg,
            Err(err) => bail!("{}", err.red()),
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

                let Some(DynamicObject::FunctionWrapper(func)) = js.pool.get(func).cloned() else {
                    bail!("missing func {}", func.red())
                };

                js.set_pending_event(func, JsValue::Ref(STYLUS_ID), objects);
                unsafe { sp.resume(env, &mut store)? };

                let js = &mut env.js_state;
                let Some(JsValue::Ref(output)) = js.stylus_result.take() else {
                    bail!("no return value for func {}", func.red())
                };
                let Some(DynamicObject::ValueArray(output)) = js.pool.remove(output) else {
                    bail!("bad return value for func {}", func.red())
                };

                let mut outs = vec![];
                for out in output {
                    let id = out.assume_id()?;
                    let Some(DynamicObject::Uint8Array(x)) = js.pool.remove(id) else {
                        bail!("bad inner return value for func {}", func.red())
                    };
                    outs.push(ApiValue(x));
                }

                for id in object_ids {
                    env.js_state.pool.remove(id);
                }
                respond.send(outs).unwrap();
            }
            Panic(error) => bail!(error),
            Done => break,
        }
    }

    Ok(handle.join().unwrap())
}
