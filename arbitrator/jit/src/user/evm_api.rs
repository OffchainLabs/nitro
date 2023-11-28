// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(clippy::too_many_arguments)]

use crate::{gostack::GoStack, machine::WasmEnv, syscall::WasmerJsEnv, wavmio::Bytes32};
use arbutil::{
    evm::{
        api::EvmApiMethod,
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
    fn call_go(&mut self, method: EvmApiMethod, args: Vec<ApiValue>) -> Vec<ApiValue> {
        let (tx, rx) = mpsc::sync_channel(0);
        let msg = EvmMsg::Call(method, args, tx);
        self.parent.send(msg).unwrap();
        rx.recv().unwrap()
    }
}

/// Executes a wasm on a new thread
pub(super) fn exec_wasm(
    sp: &mut GoStack,
    env: &mut WasmEnv,
    module: Bytes32,
    calldata: Vec<u8>,
    compile: CompileConfig,
    config: StylusConfig,
    api_id: u32,
    evm_data: EvmData,
    ink: u64,
) -> Result<(Result<UserOutcome>, u64)> {
    use EvmMsg::*;
    use UserOutcomeKind::*;

    let Some(module) = env.module_asms.get(&module).cloned() else {
        bail!(
            "module hash {module:?} not found in {:?}",
            env.module_asms.keys()
        );
    };

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
        let msg = match rx.recv_timeout(env.process.child_timeout) {
            Ok(msg) => msg,
            Err(err) => bail!("{}", err.red()),
        };
        match msg {
            Call(method, args, respond) => {
                let js_state = &mut env.js_state;
                let exports = &mut env.exports;

                let js_env = &mut WasmerJsEnv::new(sp, exports, &mut env.go_state)?;
                let outs = js_state.call_stylus_func(api_id, method, args, js_env)?;
                respond.send(outs).unwrap();
            }
            Panic(error) => bail!(error),
            Done => break,
        }
    }

    Ok(handle.join().unwrap())
}
