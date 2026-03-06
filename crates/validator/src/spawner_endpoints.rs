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
use axum::Json;
use serde::{Deserialize, Serialize};
use serde_json::{json, Value};
use std::sync::Arc;
use validation::ValidationInput;

/// JSON-RPC 2.0 response envelope.
#[derive(Serialize)]
pub struct JsonRpcResponse<T: Serialize> {
    pub jsonrpc: &'static str,
    pub id: Value,
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
    fn success(id: Value, result: T) -> Self {
        Self {
            jsonrpc: "2.0",
            id,
            result: Some(result),
            error: None,
        }
    }

    fn error(id: Value, message: String) -> Self {
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

/// JSON-RPC 2.0 dispatch request with `method` field.
#[derive(Deserialize)]
pub struct JsonRpcRequest {
    pub id: Value,
    pub method: String,
    #[serde(default)]
    pub params: Vec<Value>,
}

/// Standard JSON-RPC 2.0 dispatch endpoint (`POST /`).
///
/// go-ethereum's `rpc.Client` sends all requests to the base URL with the
/// `method` field in the JSON body. This handler dispatches to the appropriate
/// logic based on the method name.
pub async fn jsonrpc_dispatch(
    State(state): State<Arc<ServerState>>,
    Json(req): Json<JsonRpcRequest>,
) -> Json<JsonRpcResponse<Value>> {
    let id = req.id.clone();

    let result = match req.method.as_str() {
        "validation_name" => Ok(json!("Rust JIT validator")),
        "validation_stylusArchs" => Ok(json!([validation::local_target()])),
        "validation_wasmModuleRoots" => Ok(json!(module_roots(state))),
        "validation_capacity" => Ok(json!(state.available_workers)),
        "validation_validate" => validate(&state, &req.params).await,
        method => Err(format!("Method not found: {method}")),
    };

    match result {
        Ok(value) => Json(JsonRpcResponse::success(id, value)),
        Err(msg) => Json(JsonRpcResponse::error(id, msg)),
    }
}

fn module_roots(state: Arc<ServerState>) -> Vec<String> {
    state
        .locator
        .module_roots()
        .iter()
        .map(|root_meta| root_meta.module_root.to_string())
        .collect()
}

async fn validate(state: &Arc<ServerState>, params: &[Value]) -> Result<Value, String> {
    let validation_input: ValidationInput = params
        .first()
        .ok_or_else(|| "Missing params".to_string())
        .and_then(|v| {
            serde_json::from_value(v.clone())
                .map_err(|e| format!("Failed to parse validation input: {e}"))
        })?;

    let module_root: Option<ModuleRoot> = params
        .get(1)
        .and_then(|v| v.as_str())
        .and_then(|s| s.parse().ok());

    let request = ValidationRequest {
        validation_input,
        module_root,
    };

    let Json(gs) = match &state.execution {
        ExecutionMode::Native { module_cache } => {
            validate_native(&state.locator, module_cache, request).await
        }
        ExecutionMode::Continuous { jit_manager } => {
            validate_continuous(&state.locator, jit_manager, request).await
        }
    }?;

    serde_json::to_value(gs).map_err(|e| format!("Serialization error: {e}"))
}
