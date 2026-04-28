// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
use std::{
    collections::BTreeMap,
    io,
    io::{ErrorKind::InvalidData, Read},
};

use io::Error;

use crate::{
    GoGlobalState, Inbox, Preimages, ValidationInput,
    transfer::{
        IOResult, markers,
        primitives::{read_bytes, read_u8, read_u32, read_u64},
    },
};

pub fn receive_validation_input(reader: &mut impl Read) -> IOResult<ValidationInput> {
    let (small_globals, large_globals) = receive_globals(reader)?;
    let sequencer_messages = receive_inbox(reader)?;
    let delayed_messages = receive_inbox(reader)?;
    let preimages = receive_preimages(reader)?;
    let module_asms = receive_module_asms(reader)?;
    ensure_readiness(reader)?;
    let mut end_parent_chain_block_hash = [0u8; 32];
    reader.read_exact(&mut end_parent_chain_block_hash)?;

    Ok(ValidationInput {
        small_globals,
        large_globals,
        preimages,
        sequencer_messages,
        delayed_messages,
        module_asms,
        end_parent_chain_block_hash,
    })
}

pub fn receive_response(reader: &mut impl Read) -> IOResult<Result<(GoGlobalState, u64), String>> {
    match read_u8(reader)? {
        markers::SUCCESS => {
            let (small, large) = receive_globals(reader)?;
            let new_state = GoGlobalState {
                batch: small[0],
                pos_in_batch: small[1],
                block_hash: arbutil::Bytes32(large[0]),
                send_root: arbutil::Bytes32(large[1]),
                mel_state_hash: arbutil::Bytes32(large[2]),
                mel_msg_hash: arbutil::Bytes32(large[3]),
            };
            let memory_used = read_u64(reader)?;
            Ok(Ok((new_state, memory_used)))
        }
        markers::FAILURE => {
            let error_bytes = read_bytes(reader)?;
            let error_message = String::from_utf8_lossy(&error_bytes).to_string();
            Ok(Err(error_message))
        }
        other => Ok(Err(format!("unexpected response byte: {other}"))),
    }
}

fn receive_globals(reader: &mut impl Read) -> IOResult<([u64; 2], [[u8; 32]; 4])> {
    let small_globals = [read_u64(reader)?, read_u64(reader)?];
    let mut large_globals = [[0u8; 32]; 4];
    reader.read_exact(&mut large_globals[0])?;
    reader.read_exact(&mut large_globals[1])?;
    reader.read_exact(&mut large_globals[2])?;
    reader.read_exact(&mut large_globals[3])?;
    Ok((small_globals, large_globals))
}

fn receive_inbox(reader: &mut impl Read) -> IOResult<Inbox> {
    let mut inbox = Inbox::new();
    while read_u8(reader)? == markers::ANOTHER {
        let number = read_u64(reader)?;
        let data = read_bytes(reader)?;
        inbox.insert(number, data);
    }
    Ok(inbox)
}

fn receive_preimages(reader: &mut impl Read) -> IOResult<Preimages> {
    let preimage_types = read_u32(reader)?;
    let mut preimages = Preimages::new();
    for _ in 0..preimage_types {
        let preimage_ty = read_u8(reader)?;
        let map = preimages.entry(preimage_ty).or_default();
        let preimage_count = read_u32(reader)?;
        for _ in 0..preimage_count {
            let mut hash = [0u8; 32];
            reader.read_exact(&mut hash)?;
            let preimage = read_bytes(reader)?;
            map.insert(hash, preimage);
        }
    }
    Ok(preimages)
}

fn receive_module_asms(reader: &mut impl Read) -> IOResult<BTreeMap<[u8; 32], Vec<u8>>> {
    let count = read_u32(reader)?;
    let mut module_asms = BTreeMap::new();
    for _ in 0..count {
        let mut hash = [0u8; 32];
        reader.read_exact(&mut hash)?;
        let asm = read_bytes(reader)?;
        module_asms.insert(hash, asm);
    }
    Ok(module_asms)
}

fn ensure_readiness(reader: &mut impl Read) -> IOResult<()> {
    let byte = read_u8(reader)?;
    if byte == markers::READY {
        Ok(())
    } else {
        Err(Error::new(
            InvalidData,
            format!("expected READY byte, got {byte}"),
        ))
    }
}
