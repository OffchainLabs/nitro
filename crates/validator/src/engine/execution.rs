// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

//! Validation Execution Logic and Request Models.
//!
//! This module serves as the central entry point for running validation tasks.
//! It defines the standard `ValidationTask` structure used by the API and
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

use std::collections::HashMap;

use axum::Json;
use jit::CompiledModule;
use tracing::info;
use validation::{local_target, BatchInfo, GoGlobalState, ValidationInput};

use crate::{
    engine::{
        machine::JitProcessManager, machine_locator::MachineLocator, replay_binary, ModuleRoot,
        DEFAULT_JIT_CRANELIFT,
    },
};

/// A single validation task: the input data paired with an optional module root.
pub struct ValidationTask {
    pub validation_input: ValidationInput,
    pub module_root: Option<ModuleRoot>,
}

pub async fn validate_native(
    locator: &MachineLocator,
    module_cache: &HashMap<ModuleRoot, CompiledModule>,
    task: ValidationTask,
) -> Result<Json<GoGlobalState>, String> {
    let delayed_inbox = match task.validation_input.has_delayed_msg {
        true => vec![BatchInfo {
            number: task.validation_input.delayed_msg_nr,
            data: task.validation_input.delayed_msg,
        }],
        false => vec![],
    };

    let module_root = task
        .module_root
        .unwrap_or(locator.latest_wasm_module_root().module_root);

    let binary_path = locator.get_machine_path(module_root)?;
    let binary = replay_binary(&binary_path);
    info!("validate native serving request with module root {module_root}");

    let opts = jit::Opts {
        validator: jit::ValidatorOpts {
            binary: binary.clone(),
            cranelift: DEFAULT_JIT_CRANELIFT,
            debug: false, // JIT's debug messages are using printlns, which would clutter the server logs
            require_success: false, // Relevant for JIT binary only.
        },
        input_mode: jit::InputMode::Native(jit::NativeInput {
            old_state: task.validation_input.start_state.into(),
            inbox: task.validation_input.batch_info,
            delayed_inbox,
            preimages: task.validation_input.preimages,
            programs: task.validation_input.user_wasms[local_target()].clone(),
        }),
    };

    let result = match module_cache.get(&module_root) {
        Some(compiled) => {
            jit::run_with_module(compiled, &opts).map_err(|error| format!("{error}"))?
        }
        None => return Err(format!("module root {module_root} not in cache")),
    };

    if let Some(err) = result.error {
        Err(format!("{err}"))
    } else {
        Ok(Json(GoGlobalState::from(result.new_state)))
    }
}

pub async fn validate_continuous(
    locator: &MachineLocator,
    jit_manager: &JitProcessManager,
    task: ValidationTask,
) -> Result<Json<GoGlobalState>, String> {
    let module_root = task
        .module_root
        .unwrap_or_else(|| locator.latest_wasm_module_root().module_root);

    info!("validate continuous serving request with module_root {module_root}");

    let new_state = jit_manager
        .feed_machine_with_root(&task.validation_input, module_root)
        .await
        .map_err(|error| format!("{error:?}"))?;

    Ok(Json(new_state))
}
