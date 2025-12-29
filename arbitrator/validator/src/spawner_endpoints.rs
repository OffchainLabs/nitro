// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! Endpoints related to the `ValidationSpawner` Go interface and used by the nitro's validation
//! client.

use arbutil::{Bytes32, PreimageType};
use axum::response::IntoResponse;
use axum::Json;
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

pub async fn capacity() -> impl IntoResponse {
    "1" // TODO: Figure out max number of workers (optionally, make it configurable)
}

pub async fn name() -> impl IntoResponse {
    "Rust JIT validator"
}

pub async fn stylus_archs() -> impl IntoResponse {
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
    // TODO: Implement actual validation logic
    serde_json::to_string(&request.start_state)
        .map_err(|e| format!("Failed to serialize state: {e}",))
}

pub async fn wasm_module_roots() -> impl IntoResponse {
    "[]" // TODO: Figure this out from local replay.wasm
}

type Hash = Bytes32;

/// Counterpart for Go struct `validator.ValidationInput`.
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase")] // Match Go struct field names
pub struct ValidationRequest {
    id: u64,
    has_delayed_msg: bool,
    delayed_msg_nr: u64,
    preimages: HashMap<PreimageType, Hash>,
    user_wasms: HashMap<String, HashMap<Hash, Vec<u8>>>,
    batch_info: Vec<BatchInfo>,
    delayed_msg: Vec<u8>,
    start_state: GlobalState,
    debug_chain: bool,
}

/// Counterpart for Go struct `validator.BatchInfo`.
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase")] // Match Go struct field names
pub struct BatchInfo {
    number: u64,
    data: Vec<u8>,
}

/// Counterpart for Go struct `validator.GoGlobalState`.
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase")] // Match Go struct field names
pub struct GlobalState {
    block_hash: Hash,
    send_root: Hash,
    batch: u64,
    pos_in_batch: u64,
}
