// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! Validation Execution Logic and Request Models.
//!
//! This module serves as the central entry point for running validation tasks.
//! It defines the standard `ValidationRequest` structure used by the API and
//! implements the two primary validation strategies:
//!
//! 1. **Native Mode (`validate_native`):** Runs validation in-process using the
//!    embedded `jit` crate. This utilizes the `jit::InputMode::Native` configuration
//!    and is typically used for direct, low-overhead validation.
//!
//! 2. **Continuous Mode (`validate_continuous`):** Orchestrates an external "JIT Machine"
//!    process (via `JitMachine`). This mode spawns a separate binary to handle
//!    validation, isolating the execution environment and allowing for specific
//!    binary version targeting.

use axum::Json;
use validation::{BatchInfo, GoGlobalState, ValidationInput};

use crate::{
    config::ServerState,
    engine::config::DEFAULT_JIT_CRANELIFT,
    spawner_endpoints::{local_target, ValidationRequest},
};

pub async fn validate_native(
    server_state: &ServerState,
    request: ValidationInput,
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
            binary: server_state.binary.clone(), // wasm binary
            cranelift: DEFAULT_JIT_CRANELIFT,
            debug: false, // JIT's debug messages are using printlns, which would clutter the server logs
            require_success: false, // Relevant for JIT binary only.
        },
        input_mode: jit::InputMode::Native(jit::NativeInput {
            old_state: request.start_state.into(),
            inbox: request.batch_info,
            delayed_inbox,
            preimages: request.preimages,
            programs: request.user_wasms[local_target()].clone(),
        }),
    };

    let result = jit::run(&opts).map_err(|error| format!("{error}"))?;
    if let Some(err) = result.error {
        Err(format!("{err}"))
    } else {
        Ok(Json(GoGlobalState::from(result.new_state)))
    }
}

pub async fn validate_continuous(
    server_state: &ServerState,
    request: ValidationRequest,
) -> Result<Json<GoGlobalState>, String> {
    if server_state.jit_machine.is_none() {
        return Err(format!(
            "Jit machine is required continuous mode. Requested module root: {}",
            server_state.module_root
        ));
    }

    if request.module_root.is_none() {
        return Err(
            "Validation request contains no module root (or empty) when one is required."
                .to_owned(),
        );
    }

    let module_root = request.module_root.unwrap();

    let jit_machine = server_state.jit_machine.as_ref().unwrap();

    if !jit_machine.is_machine_active(module_root).await {
        return Err(format!(
            "Jit machine is not active. Maybe it received a shutdown signal? Requested module root: {}",
            server_state.module_root
        ));
    }

    let new_state = jit_machine
        .feed_machine_with_root(&request.validation_input, module_root)
        .await
        .map_err(|error| format!("{error:?}"))?;

    Ok(Json(new_state))
}
