use std::io;
use std::io::ErrorKind::InvalidData;
use std::io::Read;
use crate::transfer::{IOResult, ANOTHER};
use crate::{BatchInfo, GoGlobalState, PreimageMap, ValidationInput};
use crate::transfer::primitives::{read_bytes, read_bytes32, read_u64, read_u8};

pub fn receive_validation_input(reader: &mut impl Read) -> IOResult<ValidationInput> {
    let start_state = receive_global_state(reader)?;
    let inbox = receive_batches(reader)?;
    let delayed_message = receive_delayed_message(reader)?.unwrap_or_default();
    let preimages = receive_preimages(reader)?;

    Ok(ValidationInput {
        id: 0,
        has_delayed_msg: delayed_message.data.is_empty(),
        delayed_msg_nr: delayed_message.number,
        preimages: Default::default(),
        batch_info: inbox,
        delayed_msg: delayed_message.data,
        start_state,
        user_wasms: Default::default(),
        debug_chain: false,
        max_user_wasm_size: 0,
    })
}

fn receive_global_state(reader: &mut impl Read) -> IOResult<GoGlobalState> {
    let inbox_position = read_u64(reader)?;
    let position_within_message = read_u64(reader)?;
    let last_block_hash = read_bytes32(reader)?;
    let last_send_root = read_bytes32(reader)?;
    Ok(GoGlobalState {
        block_hash: last_block_hash,
        send_root: last_send_root,
        batch: inbox_position,
        pos_in_batch: position_within_message,
    })
}

fn receive_batches(reader: &mut impl Read) -> IOResult<Vec<BatchInfo>> {
    let mut batches = vec![];
    while read_u8(reader)? == ANOTHER {
        let number = read_u64(reader)?;
        let data = read_bytes(reader)?;
        batches.push(BatchInfo { number, data });
    }
    Ok(batches)
}

fn receive_delayed_message(reader: &mut impl Read) -> IOResult<Option<BatchInfo>> {
    match &receive_batches(reader)?[..] {
        [] => Ok(None),
        [batch_info] => Ok(Some(batch_info.clone())),
        _ => Err(io::Error::new(InvalidData, "multiple delayed batches")),
    }
}

fn receive_preimages(reader: &mut impl Read) -> IOResult<PreimageMap> {
    // Preimages are not implemented yet.
    Ok(())
}
