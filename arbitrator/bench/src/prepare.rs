use arbutil::{Bytes32, PreimageType};
use prover::machine::{argument_data_to_inbox, GlobalState, Machine};
use prover::utils::CBytes;
use std::collections::HashMap;
use std::fs::File;
use std::io::BufReader;
use std::path::{Path, PathBuf};
use std::sync::Arc;

use crate::parse_input::*;

pub fn prepare_machine(preimages: PathBuf, machines: PathBuf) -> eyre::Result<Machine> {
    let file = File::open(preimages)?;
    let reader = BufReader::new(file);

    let data = FileData::from_reader(reader)?;
    let preimages = data
        .preimages_b64
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
    // println!("Creating machine from binary_path");
    let mut mach = Machine::new_from_wavm(binary_path)?;

    let block_hash: [u8; 32] = data.start_state.block_hash.try_into().unwrap();
    let block_hash: Bytes32 = block_hash.into();
    let send_root: [u8; 32] = data.start_state.send_root.try_into().unwrap();
    let send_root: Bytes32 = send_root.into();
    let bytes32_vals: [Bytes32; 2] = [block_hash, send_root];
    let u64_vals: [u64; 2] = [data.start_state.batch, data.start_state.pos_in_batch];
    let start_state = GlobalState {
        bytes32_vals,
        u64_vals,
    };

    //println!("Setting global state");
    mach.set_global_state(start_state);
    // println!("After setting global state: {:?}", mach.get_global_state());

    // println!("Setting preimage resolver");
    mach.set_preimage_resolver(preimage_resolver);

    // println!("Adding sequencer inbox message");
    let identifier = argument_data_to_inbox(0).unwrap();
    for batch_info in data.batch_info.iter() {
        mach.add_inbox_msg(identifier, batch_info.number, batch_info.data_b64.clone());
    }

    // println!("Adding delayed inbox message");
    let identifier = argument_data_to_inbox(1).unwrap();
    mach.add_inbox_msg(identifier, data.delayed_msg_nr, data.delayed_msg_b64);

    Ok(mach)
}
