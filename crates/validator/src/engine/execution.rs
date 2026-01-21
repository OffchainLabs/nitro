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
//! 2. **Continuous Mode (`validate_contiguous`):** Orchestrates an external "JIT Machine"
//!    process (via `JitMachine`). This mode spawns a separate binary to handle
//!    validation, isolating the execution environment and allowing for specific
//!    binary version targeting.

use std::collections::HashMap;

use arbutil::{Bytes32, PreimageType};
use axum::Json;
use serde::{Deserialize, Serialize};

use crate::{
    engine::{config::JitMachineConfig, machine::JitMachine},
    spawner_endpoints::{stylus_archs, BatchInfo, GlobalState},
};

/// Counterpart for Go struct `validator.ValidationInput`.
#[derive(Clone, Debug, Deserialize, Serialize)]
#[serde(rename_all = "PascalCase")]
pub struct ValidationRequest {
    id: u64,
    pub has_delayed_msg: bool,
    #[serde(rename = "DelayedMsgNr")]
    pub delayed_msg_number: u64,
    pub preimages: HashMap<PreimageType, HashMap<Bytes32, Vec<u8>>>,
    pub user_wasms: HashMap<String, HashMap<Bytes32, Vec<u8>>>,
    pub batch_info: Vec<BatchInfo>,
    pub delayed_msg: Vec<u8>,
    pub start_state: GlobalState,
    pub module_root: Bytes32,
    debug_chain: bool,
}

pub async fn validate_native(request: ValidationRequest) -> Result<Json<GlobalState>, String> {
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

    let result = jit::run(&opts).map_err(|error| format!("{error}"))?;
    if let Some(err) = result.error {
        Err(format!("{err}"))
    } else {
        Ok(Json(GlobalState::from(result.new_state)))
    }
}

pub async fn validate_continuous(request: ValidationRequest) -> Result<Json<GlobalState>, String> {
    let config = JitMachineConfig::default();
    let module_root = if request.module_root == Bytes32::default() {
        None
    } else {
        Some(request.module_root)
    };

    let mut jit_machine =
        JitMachine::new(&config, module_root).map_err(|error| format!("{error:?}"))?;

    let new_state = jit_machine
        .feed_machine(&request)
        .await
        .map_err(|error| format!("{error:?}"))?;

    // Make sure JIT validator binary is done
    jit_machine
        .complete_machine()
        .await
        .map_err(|error| format!("{error:?}"))?;

    Ok(Json(new_state))
}
