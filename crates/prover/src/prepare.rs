// Copyright 2024-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
use crate::machine::{argument_data_to_inbox, GlobalState, Machine};
use crate::utils::CBytes;
use arbutil::{Bytes32, PreimageType};
use nitro_api::validator::ValidationInput;
use std::collections::HashMap;
use std::fs::File;
use std::io::BufReader;
use std::path::{Path, PathBuf};
use std::sync::Arc;

pub fn prepare_machine(preimages: PathBuf, machines: PathBuf) -> eyre::Result<Machine> {
    let file = File::open(preimages)?;
    let reader = BufReader::new(file);

    let data = ValidationInput::from_reader(reader)?;
    let preimages = data
        .preimages
        .into_iter()
        .flat_map(|preimage| preimage.1.into_iter())
        .collect::<HashMap<Bytes32, Vec<u8>>>();
    let preimage_resolver = move |_: u64, _: PreimageType, hash: Bytes32| -> Option<CBytes> {
        preimages
            .get(&hash)
            .map(|data| CBytes::from(data.as_slice()))
    };
    let preimage_resolver = Arc::new(Box::new(preimage_resolver));

    let binary_path = Path::new(&machines);
    let mut mach = Machine::new_from_wavm(binary_path)?;

    let start_state = GlobalState {
        bytes32_vals: [data.start_state.block_hash, data.start_state.send_root],
        u64_vals: [data.start_state.batch, data.start_state.pos_in_batch],
    };

    for (arch, wasm) in data.user_wasms.iter() {
        if arch != "wavm" {
            continue;
        }
        for (id, wasm) in wasm.iter() {
            mach.add_stylus_module(*id, wasm.as_vec());
        }
    }

    mach.set_global_state(start_state);

    mach.set_preimage_resolver(preimage_resolver);

    let identifier = argument_data_to_inbox(0).unwrap();
    for batch_info in data.batch_info.iter() {
        mach.add_inbox_msg(identifier, batch_info.number, batch_info.data.clone());
    }

    let identifier = argument_data_to_inbox(1).unwrap();
    mach.add_inbox_msg(identifier, data.delayed_msg_nr, data.delayed_msg);

    Ok(mach)
}
