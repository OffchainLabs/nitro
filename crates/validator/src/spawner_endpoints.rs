// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! Endpoints related to the `ValidationSpawner` Go interface and used by the nitro's validation
//! client.
//! Some of the data structures here are counterparts of Go structs defined in the `validator`
//! package. Their serialization is configured to match the Go side (by using `PascalCase` for
//! field names).

use crate::engine::execution::{validate_continuous, validate_native};
use crate::engine::ModuleRoot;
use crate::{config::InputMode, ServerState};
use axum::extract::State;
use axum::response::IntoResponse;
use axum::Json;
use serde::Deserialize;
use serde_with::{As, DisplayFromStr};
use std::sync::Arc;
use validation::{GoGlobalState, ValidationInput};

/// Extended validation request that includes both ValidationInput and module_root.
/// This struct allows adding module_root to the request without modifying ValidationInput.
#[derive(Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct ValidationRequest {
    #[serde(flatten)]
    pub validation_input: ValidationInput,
    #[serde(default, with = "As::<Option<DisplayFromStr>>")]
    pub module_root: Option<ModuleRoot>,
}

pub async fn validate(
    State(state): State<Arc<ServerState>>,
    Json(request): Json<ValidationRequest>,
) -> Result<Json<GoGlobalState>, String> {
    match state.mode {
        InputMode::Native => validate_native(&state, request).await,
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
    let module_roots: Vec<String> = state
        .locator
        .module_roots()
        .iter()
        .map(|root_meta| format!("0x{}", root_meta.module_root))
        .collect();
    format!("[{}]", module_roots.join(", "))
}
