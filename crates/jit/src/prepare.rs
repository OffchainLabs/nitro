// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::machine::WasmEnv;
use arbutil::{Bytes32, PreimageType};
use eyre::Ok;
use std::fs::File;
use std::io::BufReader;
use std::path::Path;
use validation::{local_target, ValidationInput, ValidationRequest};

pub fn prepare_env_from_json(json_inputs: &Path, debug: bool) -> eyre::Result<WasmEnv> {
    let file = File::open(json_inputs)?;
    let reader = BufReader::new(file);

    let req = ValidationRequest::from_reader(reader)?;
    let input = ValidationInput::from_request(&req, local_target());

    let mut env = WasmEnv::default();
    env.process.already_has_input = true;
    env.process.debug = debug;

    env.small_globals = input.small_globals;
    env.large_globals = input.large_globals.map(Bytes32);

    for (num, data) in input.sequencer_messages {
        env.sequencer_messages.insert(num, data);
    }
    for (num, data) in input.delayed_messages {
        env.delayed_messages.insert(num, data);
    }

    for (preimage_ty, inner_map) in input.preimages {
        let preimage_ty = PreimageType::try_from(preimage_ty)
            .unwrap_or_else(|_| panic!("unknown preimage type: {preimage_ty}"));
        let map = env.preimages.entry(preimage_ty).or_default();
        for (hash, preimage) in inner_map {
            map.insert(Bytes32(hash), preimage);
        }
    }

    for (module_hash, asm) in input.module_asms {
        env.module_asms.insert(Bytes32(module_hash), asm.into());
    }

    Ok(env)
}
