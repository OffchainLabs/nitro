// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! Endpoints related to the `ValidationSpawner` Go interface and used by the nitro's validation
//! client.
//! Some of the data structures here are counterparts of Go structs defined in the `validator`
//! package. Their serialization is configured to match the Go side (by using `PascalCase` for
//! field names).

use crate::engine::config::{TARGET_AMD_64, TARGET_ARM_64, TARGET_HOST};
use crate::engine::execution::{validate_continuous, validate_native, ValidationRequest};
use crate::{config::InputMode, ServerState};
use arbutil::Bytes32;
use axum::extract::State;
use axum::response::IntoResponse;
use axum::Json;
use serde::{Deserialize, Serialize};
use std::sync::Arc;

pub fn local_target() -> &'static str {
    if cfg!(all(target_os = "linux", target_arch = "aarch64")) {
        TARGET_ARM_64
    } else if cfg!(all(target_os = "linux", target_arch = "x86_64")) {
        TARGET_AMD_64
    } else {
        TARGET_HOST
    }
}

/// Counterpart for Go struct `validator.BatchInfo`.
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase")]
pub struct BatchInfo {
    pub number: u64,
    pub data: Vec<u8>,
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
#[derive(Clone, Debug, Deserialize, Serialize, Default)]
#[serde(rename_all = "PascalCase")]
pub struct GlobalState {
    pub block_hash: Bytes32,
    pub send_root: Bytes32,
    pub batch: u64,
    pub pos_in_batch: u64,
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

impl From<jit::GlobalState> for GlobalState {
    fn from(state: jit::GlobalState) -> Self {
        Self {
            block_hash: state.last_block_hash,
            send_root: state.last_send_root,
            batch: state.inbox_position,
            pos_in_batch: state.position_within_message,
        }
    }
}

pub async fn validate(
    State(state): State<Arc<ServerState>>,
    Json(request): Json<ValidationRequest>,
) -> Result<Json<GlobalState>, String> {
    match state.mode {
        InputMode::Native => validate_native(request).await,
        InputMode::Continuous => validate_continuous(&state, request).await,
    }
}

pub async fn capacity(State(state): State<Arc<ServerState>>) -> impl IntoResponse {
    format!("{:?}", state.available_workers)
}

pub async fn name() -> impl IntoResponse {
    "Rust JIT validator"
}

pub async fn wasm_module_roots(State(state): State<Arc<ServerState>>) -> impl IntoResponse {
    format!("[{:?}]", state.module_root)
}
