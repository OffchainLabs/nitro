// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
use crate::transfer::primitives::{write_bytes, write_bytes32, write_u32, write_u64, write_u8};
use crate::transfer::{markers, IOResult};
use crate::{local_target, BatchInfo, GoGlobalState, PreimageMap, UserWasm, ValidationInput};
use arbutil::Bytes32;
use std::collections::HashMap;
use std::io::ErrorKind::InvalidData;
use std::io::{Error, Write};

pub fn send_validation_input(writer: &mut impl Write, input: &ValidationInput) -> IOResult<()> {
    send_global_state(writer, &input.start_state)?;
    send_batches(writer, &input.batch_info)?;
    if let Some(batch) = input.delayed_msg() {
        send_batches(writer, &[batch])?;
    }
    send_preimages(writer, &input.preimages)?;
    send_user_wasms(writer, &input.user_wasms)?;
    finish_sending(writer)
}

pub fn send_successful_response(
    writer: &mut impl Write,
    new_state: &GoGlobalState,
    memory_used: u64,
) -> IOResult<()> {
    write_u8(writer, markers::SUCCESS)?;
    send_global_state(writer, new_state)?;
    write_u64(writer, memory_used)
}

pub fn send_failure_response(writer: &mut impl Write, error_message: &str) -> IOResult<()> {
    write_u8(writer, markers::FAILURE)?;
    write_bytes(writer, error_message.as_bytes())
}

fn send_global_state(writer: &mut impl Write, start_state: &GoGlobalState) -> IOResult<()> {
    write_u64(writer, start_state.batch)?;
    write_u64(writer, start_state.pos_in_batch)?;
    write_bytes32(writer, &start_state.block_hash)?;
    write_bytes32(writer, &start_state.send_root)
}

fn send_batches(writer: &mut impl Write, batch_info: &[BatchInfo]) -> IOResult<()> {
    for batch in batch_info {
        write_u8(writer, markers::ANOTHER)?;
        write_u64(writer, batch.number)?;
        write_bytes(writer, &batch.data)?;
    }
    write_u8(writer, markers::SUCCESS)
}

fn send_preimages(writer: &mut impl Write, preimages: &PreimageMap) -> IOResult<()> {
    write_u32(writer, preimages.len() as u32)?;
    for (preimage_type, preimage_map) in preimages {
        write_u8(writer, *preimage_type as u8)?;
        write_u32(writer, preimage_map.len() as u32)?;
        for (hash, preimage) in preimage_map {
            write_bytes32(writer, hash)?;
            write_bytes(writer, preimage)?;
        }
    }
    Ok(())
}

fn send_user_wasms(
    writer: &mut impl Write,
    user_wasms: &HashMap<String, HashMap<Bytes32, UserWasm>>,
) -> IOResult<()> {
    let local_target = local_target();
    let local_target_user_wasms = user_wasms.get(local_target);

    if local_target_user_wasms.is_none_or(|m| m.is_empty()) {
        for (arch, wasms) in user_wasms {
            if !wasms.is_empty() {
                return Err(Error::new(
                    InvalidData,
                    format!("bad stylus arch. got {arch}, expected {local_target}"),
                ));
            }
        }
    }

    let Some(local_target_user_wasms) = local_target_user_wasms else {
        return Ok(());
    };

    write_u32(writer, local_target_user_wasms.len() as u32)?;
    for (hash, wasm) in local_target_user_wasms {
        write_bytes32(writer, hash)?;
        write_bytes(writer, wasm.as_ref())?;
    }
    Ok(())
}

fn finish_sending(writer: &mut impl Write) -> IOResult<()> {
    write_u8(writer, markers::READY)
}
