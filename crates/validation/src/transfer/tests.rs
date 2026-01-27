use crate::transfer::{
    receive_response, receive_validation_input, send_failure_response, send_successful_response,
    send_validation_input,
};
use crate::{local_target, BatchInfo, GoGlobalState, UserWasm, ValidationInput};
use arbutil::{Bytes32, PreimageType};
use std::collections::HashMap;
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
fn transfer_input() -> Result<(), Box<dyn std::error::Error>> {
    let input = ValidationInput {
        start_state: Default::default(),

        batch_info: vec![
            BatchInfo {
                number: 10,
                data: vec![1, 2, 3],
            },
            BatchInfo {
                number: 11,
                data: vec![4, 5, 6],
            },
            BatchInfo {
                number: 12,
                data: vec![7, 8],
            },
        ],

        has_delayed_msg: true,
        delayed_msg_nr: 1,
        delayed_msg: vec![0xAA, 0xBB, 0xCC],

        preimages: HashMap::from([
            (
                PreimageType::Keccak256,
                HashMap::from([
                    (Bytes32::from([0u8; 32]), vec![0xDE, 0xAD, 0xBE, 0xEF]),
                    (Bytes32::from([1u8; 32]), vec![0xBA, 0xAD, 0xF0, 0x0D]),
                ]),
            ),
            (
                PreimageType::DACertificate,
                HashMap::from([(Bytes32::from([2u8; 32]), vec![0xFE, 0xED, 0xFA, 0xCE])]),
            ),
        ]),
        user_wasms: HashMap::from([(
            local_target(),
            HashMap::from([
                (Bytes32::from([3u8; 32]), UserWasm(vec![20, 21, 22])),
                (Bytes32::from([4u8; 32]), UserWasm(vec![30, 31, 32])),
            ]),
        )]),

        ..Default::default()
    };

    let (mut reader, mut writer) = pipe()?;

    send_validation_input(&mut writer, &input)?;
    let received_input = receive_validation_input(&mut reader)?;

    assert_eq!(received_input, input);

    Ok(())
}
