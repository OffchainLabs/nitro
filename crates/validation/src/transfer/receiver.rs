use crate::transfer::primitives::{read_bytes, read_bytes32, read_u32, read_u64, read_u8};
use crate::transfer::{IOResult, ANOTHER, READY};
use crate::{local_target, BatchInfo, GoGlobalState, PreimageMap, UserWasm, ValidationInput};
use arbutil::{Bytes32, PreimageType};
use io::Error;
use std::collections::HashMap;
use std::io;
use std::io::ErrorKind::InvalidData;
use std::io::Read;

pub fn receive_validation_input(reader: &mut impl Read) -> IOResult<ValidationInput> {
    let start_state = receive_global_state(reader)?;
    let inbox = receive_batches(reader)?;
    let delayed_message = receive_delayed_message(reader)?.unwrap_or_default();
    let preimages = receive_preimages(reader)?;
    let user_wasms = receive_user_wasms(reader)?;
    ensure_readiness(reader)?;

    Ok(ValidationInput {
        has_delayed_msg: delayed_message.data.is_empty(),
        delayed_msg_nr: delayed_message.number,
        preimages,
        batch_info: inbox,
        delayed_msg: delayed_message.data,
        start_state,
        user_wasms: HashMap::from([(local_target(), user_wasms)]),
        ..Default::default()
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
        _ => Err(Error::new(InvalidData, "multiple delayed batches")),
    }
}

fn receive_preimages(reader: &mut impl Read) -> IOResult<PreimageMap> {
    let preimage_types = read_u32(reader)?;
    let mut preimages = PreimageMap::with_capacity(preimage_types as usize);
    for _ in 0..preimage_types {
        let preimage_ty = PreimageType::try_from(read_u8(reader)?)
            .map_err(|e| Error::new(InvalidData, e.to_string()))?;
        let map = preimages.entry(preimage_ty).or_default();
        let preimage_count = read_u32(reader)?;
        for _ in 0..preimage_count {
            let hash = read_bytes32(reader)?;
            let preimage = read_bytes(reader)?;
            map.insert(hash, preimage);
        }
    }
    Ok(preimages)
}

fn receive_user_wasms(reader: &mut impl Read) -> IOResult<HashMap<Bytes32, UserWasm>> {
    let programs_count = read_u32(reader)?;
    let mut user_wasms = HashMap::with_capacity(programs_count as usize);
    for _ in 0..programs_count {
        let module_hash = read_bytes32(reader)?;
        let module_asm = read_bytes(reader)?;
        user_wasms.insert(module_hash, UserWasm(module_asm));
    }
    Ok(user_wasms)
}

fn ensure_readiness(reader: &mut impl Read) -> IOResult<()> {
    let byte = read_u8(reader)?;
    if byte == READY {
        Ok(())
    } else {
        Err(Error::new(
            InvalidData,
            format!("expected READY byte, got {byte}"),
        ))
    }
}
