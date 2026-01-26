use std::collections::HashMap;

use arbutil::{Bytes32, PreimageType};
use axum::Json;
use serde::{Deserialize, Serialize};

use crate::{
    endpoints::spawner_endpoints::{stylus_archs, BatchInfo, GlobalState},
    server_jit::{
        config::JitMachineConfig, jit_machine::JitMachine, machine_locator::MachineLocator,
    },
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

pub async fn validate_native(
    request: ValidationRequest,
    locator: &MachineLocator,
) -> Result<Json<GlobalState>, String> {
    let delayed_inbox = match request.has_delayed_msg {
        true => vec![jit::SequencerMessage {
            number: request.delayed_msg_number,
            data: request.delayed_msg,
        }],
        false => vec![],
    };
    let config = JitMachineConfig::default();

    let opts = jit::Opts {
        validator: jit::ValidatorOpts {
            binary: locator
                .get_machine_path(request.module_root)
                .join(&config.prover_bin_path),
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

    let result = jit::run(&opts).map_err(|error| format!("{error:?}"))?;
    if let Some(err) = result.error {
        Err(format!("{err}"))
    } else {
        Ok(Json(GlobalState::from(result.new_state)))
    }
}

pub async fn validate_contiguous(
    request: ValidationRequest,
    locator: &MachineLocator,
) -> Result<Json<GlobalState>, String> {
    let config = JitMachineConfig::default();
    let module_root = if request.module_root == Bytes32::default() {
        None
    } else {
        Some(request.module_root)
    };

    let mut jit_machine =
        JitMachine::new(&config, locator, module_root).map_err(|error| format!("{error:?}"))?;

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
