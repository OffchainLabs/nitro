// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! Endpoints related to the `ValidationSpawner` Go interface and used by the nitro's validation
//! client.
//! Some of the data structures here are counterparts of Go structs defined in the `validator`
//! package. Their serialization is configured to match the Go side (by using `PascalCase` for
//! field names).

use crate::config::ExecutionMode;
use crate::engine::execution::{validate_continuous, validate_native};
use crate::engine::ModuleRoot;
use crate::ServerState;
use axum::extract::State;
use axum::response::IntoResponse;
use axum::Json;
use serde::{Deserialize, Serialize};
use std::sync::Arc;
use validation::{GoGlobalState, ValidationInput};

/// JSON-RPC 2.0 request envelope.
#[derive(Deserialize)]
pub struct JsonRpcRequest {
    pub id: serde_json::Value,
    pub params: Vec<serde_json::Value>,
}

/// JSON-RPC 2.0 response envelope.
#[derive(Serialize)]
pub struct JsonRpcResponse<T: Serialize> {
    pub jsonrpc: &'static str,
    pub id: serde_json::Value,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub result: Option<T>,
    #[serde(skip_serializing_if = "Option::is_none")]
    pub error: Option<JsonRpcError>,
}

#[derive(Serialize)]
pub struct JsonRpcError {
    pub code: i64,
    pub message: String,
}

impl<T: Serialize> JsonRpcResponse<T> {
    fn success(id: serde_json::Value, result: T) -> Self {
        Self {
            jsonrpc: "2.0",
            id,
            result: Some(result),
            error: None,
        }
    }

    fn error(id: serde_json::Value, message: String) -> Self {
        Self {
            jsonrpc: "2.0",
            id,
            result: None,
            error: Some(JsonRpcError {
                code: -32000,
                message,
            }),
        }
    }
}

/// Validation request that includes both ValidationInput and module_root.
pub struct ValidationRequest {
    pub validation_input: ValidationInput,
    pub module_root: Option<ModuleRoot>,
}

pub async fn validate(
    State(state): State<Arc<ServerState>>,
    Json(rpc_request): Json<JsonRpcRequest>,
) -> Json<JsonRpcResponse<GoGlobalState>> {
    let id = rpc_request.id;

    let validation_input: ValidationInput = match rpc_request.params.first() {
        Some(value) => match serde_json::from_value(value.clone()) {
            Ok(input) => input,
            Err(e) => {
                return Json(JsonRpcResponse::error(
                    id,
                    format!("Failed to parse validation input: {e}"),
                ))
            }
        },
        None => {
            return Json(JsonRpcResponse::error(
                id,
                "Missing validation input in params".to_string(),
            ))
        }
    };

    let module_root: Option<ModuleRoot> = rpc_request
        .params
        .get(1)
        .and_then(|v| v.as_str())
        .and_then(|s| s.parse().ok());

    let request = ValidationRequest {
        validation_input,
        module_root,
    };

    let result = match &state.execution {
        ExecutionMode::Native { module_cache } => {
            validate_native(&state.locator, module_cache, request).await
        }
        ExecutionMode::Continuous { jit_manager } => {
            validate_continuous(&state.locator, jit_manager, request).await
        }
    };

    match result {
        Ok(Json(global_state)) => Json(JsonRpcResponse::success(id, global_state)),
        Err(e) => Json(JsonRpcResponse::error(id, e)),
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
        .map(|root_meta| root_meta.module_root.to_string())
        .collect();
    format!("[{}]", module_roots.join(", "))
}
