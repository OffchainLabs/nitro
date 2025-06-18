use arbutil::{Bytes32, PreimageType};
use rayon::str::Bytes;
use std::collections::HashMap;
use std::fs::File;
use std::io::BufReader;
use std::path::{Path, PathBuf};
use std::sync::Arc;

use crate::machine::{GlobalState, Machine};
use crate::parse_input::*;
use crate::utils::CBytes;

pub fn prepare_machine(validation_entry_file: PathBuf, machines: PathBuf) -> eyre::Result<Machine> {
    // Load the preimages file from disk.
    let file = File::open(validation_entry_file)?;
    let reader = BufReader::new(file);

    let data = FileData::from_reader(reader)?;
    let preimages = data
        .preimages_b64
        .into_iter()
        .flat_map(|preimage| preimage.1.into_iter())
        .collect::<HashMap<Bytes32, Vec<u8>>>();

    // Create a preimage resolver which is a function
    // that simply retrieves preimages by hash from a hashmap.
    let preimage_resolver = move |_: u64, _: PreimageType, hash: Bytes32| -> Option<CBytes> {
        preimages
            .get(&hash)
            .map(|data| CBytes::from(data.as_slice()))
    };
    let preimage_resolver = Arc::new(Box::new(preimage_resolver));

    let binary_path = Path::new(&machines);
    let mut mach = Machine::new_from_wavm(binary_path)?;

    let block_hash: [u8; 32] = data.start_state.block_hash.try_into().unwrap();
    let block_hash: Bytes32 = block_hash.into();
    let send_root: [u8; 32] = data.start_state.send_root.try_into().unwrap();
    let send_root: Bytes32 = send_root.into();
    let start_mel_root: [u8; 32] = data.start_state.mel_root.try_into().unwrap();
    let start_mel_root: Bytes32 = start_mel_root.into();
    let bytes32_vals: [Bytes32; 3] = [block_hash, send_root, start_mel_root];
    let u64_vals: [u64; 2] = [data.start_state.batch, data.start_state.pos_in_batch];
    let start_state = GlobalState {
        bytes32_vals,
        u64_vals,
    };

    let end_mel_root = Bytes32::try_from(data.end_mel_root)?;
    mach.set_preimage_resolver(preimage_resolver);
    mach.set_global_state(start_state);
    mach.set_end_mel_root(end_mel_root);
    Ok(mach)
}
