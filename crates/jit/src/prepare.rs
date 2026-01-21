// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::machine::WasmEnv;
use eyre::Ok;
use nitro_api::validator;
use std::env;
use std::fs::File;
use std::io::BufReader;
use std::path::Path;

// local_target matches rawdb.LocalTarget() on the go side.
// While generating json_inputs file, one should make sure user_wasms map
// has entry for the system's arch that jit validation is being run on
pub fn local_target() -> String {
    if env::consts::OS == "linux" {
        match env::consts::ARCH {
            "aarch64" => "arm64".to_string(),
            "x86_64" => "amd64".to_string(),
            _ => "host".to_string(),
        }
    } else {
        "host".to_string()
    }
}

pub fn prepare_env_from_json(json_inputs: &Path, debug: bool) -> eyre::Result<WasmEnv> {
    let file = File::open(json_inputs)?;
    let reader = BufReader::new(file);

    let data = validator::ValidationInput::from_reader(reader)?;

    let mut env = WasmEnv::default();
    env.process.already_has_input = true;
    env.process.debug = debug;

    env.small_globals = [data.start_state.batch, data.start_state.pos_in_batch];
    env.large_globals = [data.start_state.block_hash, data.start_state.send_root];

    for batch_info in data.batch_info.iter() {
        env.sequencer_messages
            .insert(batch_info.number, batch_info.data.clone());
    }

    if data.delayed_msg_nr != 0 && !data.delayed_msg.is_empty() {
        env.delayed_messages
            .insert(data.delayed_msg_nr, data.delayed_msg.clone());
    }

    for (preimage_ty, inner_map) in data.preimages {
        let map = env.preimages.entry(preimage_ty).or_default();
        for (hash, preimage) in inner_map {
            map.insert(hash, preimage);
        }
    }

    if let Some(user_wasms) = data.user_wasms.get(&local_target()) {
        for (module_hash, module_asm) in user_wasms.iter() {
            env.module_asms
                .insert(*module_hash, module_asm.as_vec().into());
        }
    }

    Ok(env)
}
