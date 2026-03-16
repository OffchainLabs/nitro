// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use crate::machine::WasmEnv;
use arbutil::Bytes32;
use eyre::Ok;
use std::fs::File;
use std::io::BufReader;
use std::path::Path;
use validation::{local_target, ValidationInput, ValidationRequest};

pub fn prepare_env_from_json(json_inputs: &Path, debug: bool) -> eyre::Result<WasmEnv> {
    let file = File::open(json_inputs)?;
    let reader = BufReader::new(file);

    let req = ValidationRequest::from_reader(reader)?;
    let mut input = ValidationInput::from_request(&req, local_target());

    let mut env = WasmEnv::default();
    env.process.already_has_input = true;
    env.process.debug = debug;

    for (module_hash, asm) in input.module_asms.drain() {
        env.module_asms.insert(Bytes32(module_hash), asm.into());
    }

    env.input = input;

    Ok(env)
}
