// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

#![allow(clippy::too_many_arguments)]

use crate::machine::{Escape, MaybeEscape};
use arbutil::evm::api::VecReader;
use arbutil::evm::{
    api::{EvmApiMethod, EVM_API_METHOD_REQ_OFFSET},
    req::EvmApiRequestor,
    req::RequestHandler,
    user::UserOutcome,
    EvmData,
};
use eyre::{eyre, Result};
use prover::programs::prelude::*;
use std::thread;
use std::time::Duration;
use std::{
    sync::{
        mpsc::{self, Receiver, SyncSender},
        Arc,
    },
    thread::JoinHandle,
};
use stylus::{native::NativeInstance, run::RunProgram};
use wasmer::Target;

struct MessageToCothread {
    result: Vec<u8>,
    raw_data: Vec<u8>,
    cost: u64,
}

#[derive(Clone)]
pub struct MessageFromCothread {
    pub req_type: u32,
    pub req_data: Vec<u8>,
}

struct CothreadRequestor {
    tx: SyncSender<MessageFromCothread>,
    rx: Receiver<MessageToCothread>,
}

impl RequestHandler<VecReader> for CothreadRequestor {
    fn request(
        &mut self,
        req_type: EvmApiMethod,
        req_data: impl AsRef<[u8]>,
    ) -> (Vec<u8>, VecReader, u64) {
        let msg = MessageFromCothread {
            req_type: req_type as u32 + EVM_API_METHOD_REQ_OFFSET,
            req_data: req_data.as_ref().to_vec(),
        };

        if let Err(error) = self.tx.send(msg) {
            panic!("failed sending request from cothread: {error}");
        }
        match self.rx.recv_timeout(Duration::from_secs(5)) {
            Ok(response) => (
                response.result,
                VecReader::new(response.raw_data),
                response.cost,
            ),
            Err(_) => panic!("no response from main thread"),
        }
    }
}

pub struct CothreadHandler {
    tx: SyncSender<MessageToCothread>,
    rx: Receiver<MessageFromCothread>,
    thread: Option<JoinHandle<MaybeEscape>>,
    last_request: Option<(MessageFromCothread, u32)>,
}

impl CothreadHandler {
    pub fn wait_next_message(&mut self) -> MaybeEscape {
        let msg = self.rx.recv_timeout(Duration::from_secs(10));
        let Ok(msg) = msg else {
            return Escape::hostio("did not receive message");
        };
        self.last_request = Some((msg, 0x33333333)); // TODO: Ids
        Ok(())
    }

    pub fn wait_done(&mut self) -> MaybeEscape {
        let error = || Escape::Child(eyre!("no child"));
        let status = self.thread.take().ok_or_else(error)?.join();
        match status {
            Ok(res) => res,
            Err(_) => Escape::hostio("failed joining child process"),
        }
    }

    pub fn last_message(&self) -> Result<(MessageFromCothread, u32), Escape> {
        self.last_request
            .clone()
            .ok_or_else(|| Escape::HostIO("no message waiting".to_string()))
    }

    pub fn set_response(
        &mut self,
        id: u32,
        result: Vec<u8>,
        raw_data: Vec<u8>,
        cost: u64,
    ) -> MaybeEscape {
        let Some(msg) = self.last_request.clone() else {
            return Escape::hostio("trying to set response but no message pending");
        };
        if msg.1 != id {
            return Escape::hostio("trying to set response for wrong message id");
        };
        let msg = MessageToCothread {
            result,
            raw_data,
            cost,
        };
        if let Err(err) = self.tx.send(msg) {
            return Escape::hostio(format!("failed to send response to stylus thread: {err:?}"));
        };
        Ok(())
    }
}

/// Executes a wasm on a new thread
pub fn exec_wasm(
    module: Arc<[u8]>,
    calldata: Vec<u8>,
    compile: CompileConfig,
    config: StylusConfig,
    evm_data: EvmData,
    ink: u64,
) -> Result<CothreadHandler> {
    let (tothread_tx, tothread_rx) = mpsc::sync_channel::<MessageToCothread>(0);
    let (fromthread_tx, fromthread_rx) = mpsc::sync_channel::<MessageFromCothread>(0);

    let cothread = CothreadRequestor {
        tx: fromthread_tx,
        rx: tothread_rx,
    };

    let evm_api = EvmApiRequestor::new(cothread);

    let mut instance = unsafe {
        NativeInstance::deserialize(
            &module,
            compile.clone(),
            evm_api,
            evm_data,
            Target::default(),
        )
    }?;

    let thread = thread::spawn(move || {
        let outcome = instance.run_main(&calldata, config, ink);

        let ink_left = match outcome.as_ref() {
            Ok(UserOutcome::OutOfStack) => 0, // take all ink when out of stack
            _ => instance.ink_left().into(),
        };

        let outcome = match outcome {
            Err(e) | Ok(UserOutcome::Failure(e)) => UserOutcome::Failure(e.wrap_err("call failed")),
            Ok(outcome) => outcome,
        };

        let (out_kind, data) = outcome.into_data();
        let gas_left = config.pricing.ink_to_gas(ink_left);

        let mut output = Vec::with_capacity(8 + data.len());
        output.extend(gas_left.to_be_bytes());
        output.extend(data);

        let msg = MessageFromCothread {
            req_data: output,
            req_type: out_kind as u32,
        };
        instance
            .env_mut()
            .evm_api
            .request_handler()
            .tx
            .send(msg)
            .or_else(|_| Escape::hostio("failed sending messaage to thread"))
    });

    Ok(CothreadHandler {
        tx: tothread_tx,
        rx: fromthread_rx,
        thread: Some(thread),
        last_request: None,
    })
}
