// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! Endpoints related to the `ValidationSpawner` Go interface and used by the nitro's validation
//! client.
//! Some of the data structures here are counterparts of Go structs defined in the `validator`
//! package. Their serialization is configured to match the Go side (by using `PascalCase` for
//! field names).

use crate::ServerState;
use axum::extract::State;
use axum::response::IntoResponse;
use axum::Json;
use std::sync::Arc;
use validation::{BatchInfo, GoGlobalState, ValidationInput};

pub async fn capacity(State(state): State<Arc<ServerState>>) -> impl IntoResponse {
    format!("{:?}", state.available_workers)
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

pub async fn validate(
    State(state): State<Arc<ServerState>>,
    Json(request): Json<ValidationInput>,
) -> Result<Json<GoGlobalState>, String> {
    let delayed_inbox = match request.has_delayed_msg {
        true => vec![BatchInfo {
            number: request.delayed_msg_nr,
            data: request.delayed_msg,
        }],
        false => vec![],
    };

    let opts = jit::Opts {
        validator: jit::ValidatorOpts {
            binary: state.binary.clone(),
            cranelift: true, // The default for JIT binary, no need for LLVM right now
            debug: false, // JIT's debug messages are using printlns, which would clutter the server logs
            require_success: false, // Relevant for JIT binary only.
        },
        input_mode: jit::InputMode::Native(jit::NativeInput {
            old_state: request.start_state.into(),
            inbox: request.batch_info,
            delayed_inbox,
            preimages: request.preimages,
            programs: request.user_wasms[stylus_archs().await].clone(),
        }),
    };

    let result = jit::run(&opts).map_err(|error| format!("{error}"))?;
    if let Some(err) = result.error {
        Err(format!("{err}"))
    } else {
        Ok(Json(GoGlobalState::from(result.new_state)))
    }
}

pub async fn wasm_module_roots(State(state): State<Arc<ServerState>>) -> impl IntoResponse {
    format!("[{:?}]", state.module_root)
}
