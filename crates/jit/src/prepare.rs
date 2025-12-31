// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::machine::WasmEnv;
use arbutil::{Bytes32, PreimageType};
use eyre::Ok;
use prover::parse_input::FileData;
use std::env;
use std::fs::File;
use std::io::BufReader;
use std::path::PathBuf;

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

pub fn prepare_env(json_inputs: PathBuf, debug: bool) -> eyre::Result<WasmEnv> {
    let file = File::open(json_inputs)?;
    let reader = BufReader::new(file);

    let data = FileData::from_reader(reader)?;

    let mut env = WasmEnv::default();
    env.process.forks = false; // Should be set to false when using json_inputs
    env.process.debug = debug;

    let block_hash: [u8; 32] = data.start_state.block_hash.try_into().unwrap();
    let block_hash: Bytes32 = block_hash.into();
    let send_root: [u8; 32] = data.start_state.send_root.try_into().unwrap();
    let send_root: Bytes32 = send_root.into();
    let bytes32_vals: [Bytes32; 2] = [block_hash, send_root];
    let u64_vals: [u64; 2] = [data.start_state.batch, data.start_state.pos_in_batch];
    env.small_globals = u64_vals;
    env.large_globals = bytes32_vals;

    for batch_info in data.batch_info.iter() {
        env.sequencer_messages
            .insert(batch_info.number, batch_info.data_b64.clone());
    }

    if data.delayed_msg_nr != 0 && !data.delayed_msg_b64.is_empty() {
        env.delayed_messages
            .insert(data.delayed_msg_nr, data.delayed_msg_b64.clone());
    }

    for (ty, inner_map) in data.preimages_b64 {
        let preimage_ty = PreimageType::try_from(ty as u8)?;
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
