// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
use crate::transfer::{
    receive_response, receive_validation_input, send_failure_response, send_successful_response,
    send_validation_input,
};
use crate::{GoGlobalState, ValidationInput};
use arbutil::Bytes32;
use std::collections::{BTreeMap, HashMap};
use std::io::pipe;

#[test]
fn transfer_successful_response() -> Result<(), Box<dyn std::error::Error>> {
    let new_state = GoGlobalState {
        block_hash: Bytes32::from([1u8; 32]),
        send_root: Bytes32::from([2u8; 32]),
        batch: 42,
        pos_in_batch: 7,
    };
    let memory_used = 123456u64;

    let (mut reader, mut writer) = pipe()?;

    send_successful_response(&mut writer, &new_state, memory_used)?;
    let (received_state, received_memory) = receive_response(&mut reader)??;

    assert_eq!(received_state, new_state);
    assert_eq!(received_memory, memory_used);
    Ok(())
}

#[test]
fn transfer_failure_response() -> Result<(), Box<dyn std::error::Error>> {
    let error_message = "Validation failed due to some error.";

    let (mut reader, mut writer) = pipe()?;

    send_failure_response(&mut writer, error_message)?;
    let result = receive_response(&mut reader)?;

    match result {
        Err(err_msg) => assert_eq!(err_msg, error_message),
        Ok(_) => panic!("Expected failure response, but got success."),
    }
    Ok(())
}

#[test]
fn transfer_validation_input() -> Result<(), Box<dyn std::error::Error>> {
    let input = ValidationInput {
        small_globals: [42, 7],
        large_globals: [[1u8; 32], [2u8; 32]],

        sequencer_messages: BTreeMap::from([
            (10, vec![1, 2, 3]),
            (11, vec![4, 5, 6]),
            (12, vec![7, 8]),
        ]),

        delayed_messages: BTreeMap::from([(1, vec![0xAA, 0xBB, 0xCC])]),

        preimages: BTreeMap::from([
            (
                0, // Keccak256
                BTreeMap::from([
                    ([0u8; 32], vec![0xDE, 0xAD, 0xBE, 0xEF]),
                    ([1u8; 32], vec![0xBA, 0xAD, 0xF0, 0x0D]),
                ]),
            ),
            (
                3, // DACertificate
                BTreeMap::from([([2u8; 32], vec![0xFE, 0xED, 0xFA, 0xCE])]),
            ),
        ]),

        module_asms: HashMap::from([([3u8; 32], vec![20, 21, 22]), ([4u8; 32], vec![30, 31, 32])]),
    };

    let (mut reader, mut writer) = pipe()?;

    send_validation_input(&mut writer, &input)?;
    let received = receive_validation_input(&mut reader)?;

    assert_eq!(received.small_globals, input.small_globals);
    assert_eq!(received.large_globals, input.large_globals);
    assert_eq!(received.sequencer_messages, input.sequencer_messages);
    assert_eq!(received.delayed_messages, input.delayed_messages);
    assert_eq!(received.preimages, input.preimages);
    assert_eq!(received.module_asms, input.module_asms);

    Ok(())
}
