// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! Endpoints related to the `ValidationSpawner` Go interface and used by the nitro's validation
//! client.
//! Some of the data structures here are counterparts of Go structs defined in the `validator`
//! package. Their serialization is configured to match the Go side (by using `PascalCase` for
//! field names).

use crate::ServerState;
use arbutil::{Bytes32, PreimageType};
use axum::extract::State;
use axum::response::IntoResponse;
use axum::Json;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;
use std::sync::Arc;

pub async fn capacity() -> impl IntoResponse {
    "1" // TODO: Figure out max number of workers (optionally, make it configurable)
}

pub async fn name() -> impl IntoResponse {
    "Rust JIT validator"
}

pub async fn stylus_archs() -> &'static str {
    if cfg!(target_os = "linux") {
        if cfg!(target_arch = "aarch64") {
            return "arm64";
        } else if cfg!(target_arch = "x86_64") {
            return "amd64";
        }
    }
    "host"
}

pub async fn validate(Json(request): Json<ValidationRequest>) -> impl IntoResponse {
    let delayed_inbox = match request.has_delayed_msg {
        true => vec![jit::SequencerMessage {
            number: request.delayed_msg_number,
            data: request.delayed_msg,
        }],
        false => vec![],
    };

    let opts = jit::Opts {
        validator: jit::ValidatorOpts {
            binary: Default::default(),
            cranelift: true, // The default for JIT binary, no need for LLVM right now
            debug: false, // JIT's debug messages are using printlns, which would clutter the server logs
            require_success: false, // Relevant for JIT binary only.
        },
        input_mode: jit::InputMode::Native(jit::NativeInput {
            old_state: request.start_state.into(),
            inbox: request.batch_info.into_iter().map(Into::into).collect(),
            delayed_inbox,
            preimages: request.preimages,
            programs: request.user_wasms[stylus_archs().await].clone(),
        }),
    };


}

pub async fn wasm_module_roots(State(state): State<Arc<ServerState>>) -> impl IntoResponse {
    format!("[{:?}]", state.module_root)
}

/// Counterpart for Go struct `validator.ValidationInput`.
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase")]
pub struct ValidationRequest {
    id: u64,
    has_delayed_msg: bool,
    #[serde(rename = "DelayedMsgNr")]
    delayed_msg_number: u64,
    preimages: HashMap<PreimageType, HashMap<Bytes32, Vec<u8>>>,
    user_wasms: HashMap<String, HashMap<Bytes32, Vec<u8>>>,
    batch_info: Vec<BatchInfo>,
    delayed_msg: Vec<u8>,
    start_state: GlobalState,
    debug_chain: bool,
}

/// Counterpart for Go struct `validator.BatchInfo`.
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase")]
pub struct BatchInfo {
    number: u64,
    data: Vec<u8>,
}

impl From<BatchInfo> for jit::SequencerMessage {
    fn from(batch: BatchInfo) -> Self {
        Self {
            number: batch.number,
            data: batch.data,
        }
    }
}

/// Counterpart for Go struct `validator.GoGlobalState`.
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase")]
pub struct GlobalState {
    block_hash: Bytes32,
    send_root: Bytes32,
    batch: u64,
    pos_in_batch: u64,
}

impl From<GlobalState> for jit::GlobalState {
    fn from(state: GlobalState) -> Self {
        Self {
            last_block_hash: state.block_hash,
            last_send_root: state.send_root,
            inbox_position: state.batch,
            position_within_message: state.pos_in_batch,
        }
    }
}
