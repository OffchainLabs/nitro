// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
use crate::transfer::primitives::{write_bytes, write_u32, write_u64, write_u8};
use crate::transfer::{markers, IOResult};
use crate::{GoGlobalState, Inbox, Preimages, ValidationInput};
use std::collections::BTreeMap;
use std::io::Write;

pub fn send_validation_input(writer: &mut impl Write, input: &ValidationInput) -> IOResult<()> {
    send_globals(writer, &input.small_globals, &input.large_globals)?;
    send_inbox(writer, &input.sequencer_messages)?;
    send_inbox(writer, &input.delayed_messages)?;
    send_preimages(writer, &input.preimages)?;
    send_module_asms(writer, &input.module_asms)?;
    finish_sending(writer)?;
    writer.write_all(&input.end_parent_chain_block_hash)
}

pub fn send_successful_response(
    writer: &mut impl Write,
    new_state: &GoGlobalState,
    memory_used: u64,
) -> IOResult<()> {
    write_u8(writer, markers::SUCCESS)?;
    send_globals(
        writer,
        &[new_state.batch, new_state.pos_in_batch],
        &[
            new_state.block_hash.0,
            new_state.send_root.0,
            new_state.mel_state_hash.0,
            new_state.mel_msg_hash.0,
        ],
    )?;
    write_u64(writer, memory_used)
}

pub fn send_failure_response(writer: &mut impl Write, error_message: &str) -> IOResult<()> {
    write_u8(writer, markers::FAILURE)?;
    write_bytes(writer, error_message.as_bytes())
}

fn send_globals(
    writer: &mut impl Write,
    small_globals: &[u64; 2],
    large_globals: &[[u8; 32]; 4],
) -> IOResult<()> {
    write_u64(writer, small_globals[0])?;
    write_u64(writer, small_globals[1])?;
    writer.write_all(&large_globals[0])?;
    writer.write_all(&large_globals[1])?;
    writer.write_all(&large_globals[2])?;
    writer.write_all(&large_globals[3])
}

fn send_inbox(writer: &mut impl Write, inbox: &Inbox) -> IOResult<()> {
    for (number, data) in inbox {
        write_u8(writer, markers::ANOTHER)?;
        write_u64(writer, *number)?;
        write_bytes(writer, data)?;
    }
    write_u8(writer, markers::SUCCESS)
}

fn send_preimages(writer: &mut impl Write, preimages: &Preimages) -> IOResult<()> {
    write_u32(writer, preimages.len() as u32)?;
    for (preimage_type, preimage_map) in preimages {
        write_u8(writer, *preimage_type)?;
        write_u32(writer, preimage_map.len() as u32)?;
        for (hash, preimage) in preimage_map {
            writer.write_all(hash)?;
            write_bytes(writer, preimage)?;
        }
    }
    Ok(())
}

fn send_module_asms(
    writer: &mut impl Write,
    module_asms: &BTreeMap<[u8; 32], Vec<u8>>,
) -> IOResult<()> {
    write_u32(writer, module_asms.len() as u32)?;
    for (hash, asm) in module_asms {
        writer.write_all(hash)?;
        write_bytes(writer, asm)?;
    }
    Ok(())
}

fn finish_sending(writer: &mut impl Write) -> IOResult<()> {
    write_u8(writer, markers::READY)
}
