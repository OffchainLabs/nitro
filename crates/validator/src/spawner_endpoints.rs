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
use axum::extract::State;
use axum::response::IntoResponse;
use axum::Json;
use std::sync::Arc;
use validation::{GoGlobalState, ValidationInput};

pub fn local_target() -> &'static str {
    if cfg!(all(target_os = "linux", target_arch = "aarch64")) {
        TARGET_ARM_64
    } else if cfg!(all(target_os = "linux", target_arch = "x86_64")) {
        TARGET_AMD_64
    } else {
        TARGET_HOST
    }
}

pub async fn validate(
    State(state): State<Arc<ServerState>>,
    Json(request): Json<ValidationInput>,
) -> Result<Json<GoGlobalState>, String> {
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
