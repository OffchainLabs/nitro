// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(clippy::too_many_arguments)]

use crate::{
    gostack::GoStack,
    machine::{ModuleAsm, WasmEnvMut},
    syscall::WasmerJsEnv,
};
use arbutil::{
    evm::{
        api::EvmApiMethod,
        js::{ApiValue, JsCallIntoGo, JsEvmApi},
        user::{UserOutcome, UserOutcomeKind},
        EvmData,
    },
    Color,
};
use eyre::{bail, Context, Result};
use go_js::JsValue;
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
    Call(EvmApiMethod, Vec<ApiValue>, SyncSender<Vec<ApiValue>>),
    Panic(String),
    Done,
}

impl ApiCaller {
    fn new(parent: SyncSender<EvmMsg>) -> Self {
        Self { parent }
    }
}

impl JsCallIntoGo for ApiCaller {
    fn call_go(&mut self, func: EvmApiMethod, args: Vec<ApiValue>) -> Vec<ApiValue> {
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
    module: ModuleAsm,
    calldata: Vec<u8>,
    compile: CompileConfig,
    config: StylusConfig,
    api_id: u32,
    evm_data: EvmData,
    ink: u64,
) -> Result<(Result<UserOutcome>, u64)> {
    use EvmMsg::*;
    use UserOutcomeKind::*;

    let (tx, rx) = mpsc::sync_channel(0);
    let evm_api = JsEvmApi::new(ApiCaller::new(tx.clone()));

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
            Call(method, args, respond) => {
                let (env, mut store) = env.data_and_store_mut();

                let api = &format!("api{api_id}");
                let api = env.js_state.get_globals().get_path(&["stylus", api]);
                let exports = &mut env.exports;
                let js_env = &mut WasmerJsEnv::new(sp, &mut store, exports, &mut env.go_state)?;

                // get the callback into Go
                let array = match api.clone() {
                    JsValue::Array(array) => array,
                    x => bail!("bad EVM api type for {api_id}: {x:?}"),
                };
                let array = array.lock();
                let func = match array.get(method as usize) {
                    Some(JsValue::Function(func)) => func,
                    x => bail!("bad EVM api func for {method:?}, {api_id}: {x:?}"),
                };

                // call into go
                let args = args.into_iter().map(Into::into).collect();
                let outs = func.call(js_env, api, args).wrap_err("EVM api failed")?;

                // send the outputs
                let outs = match outs {
                    JsValue::Array(outs) => outs.lock().clone().into_iter(),
                    x => bail!("bad EVM api result for {method:?}: {x:?}"),
                };
                let outs = outs.map(TryInto::try_into).collect::<Result<_, _>>()?;
                respond.send(outs).unwrap();
            }
            Panic(error) => bail!(error),
            Done => break,
        }
    }

    Ok(handle.join().unwrap())
}
