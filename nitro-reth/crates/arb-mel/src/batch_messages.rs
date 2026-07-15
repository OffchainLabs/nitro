// `extract_batch_messages` and its helpers are a work-in-progress port of the
// `arbstate` batch decoder. The timestamp/block cursors, the delayed-message
// loop, and several parameters are already wired up but not yet consumed by the
// stubbed-out body; allow the resulting lints until the logic is filled in.
#![allow(
    unused_imports,
    unused_variables,
    unused_assignments,
    clippy::unnecessary_unwrap,
    clippy::while_immutable_condition
)]
use core::time;

use alloy_primitives::Address;
use arbos::{arbos_state, arbos_types::{L1IncomingMessage, L1IncomingMessageHeader, MessageWithMetadata}};

use crate::{DelayedMessageDB, MelError, MelResult, MelState, SequencerMessage};

// const BatchSegmentKindL2Message uint8 = 0
// const BatchSegmentKindL2MessageBrotli uint8 = 1
// const BatchSegmentKindDelayedMessages uint8 = 2
// const BatchSegmentKindAdvanceTimestamp uint8 = 3
// const BatchSegmentKindAdvanceL1BlockNumber uint8 = 4
enum BatchSegmentKind {
    L2Message,
    L2MessageBrotli,
    DelayedMessages,
    AdvanceTimestamp,
    AdvanceL1BlockNumber,
    Unknown,
}

impl From<BatchSegmentKind> for u8 {
    fn from(value: BatchSegmentKind) -> Self {
        use BatchSegmentKind::*;
        match value {
            L2Message => 0,
            L2MessageBrotli => 1,
            DelayedMessages => 2,
            AdvanceTimestamp => 3,
            AdvanceL1BlockNumber => 4,
            Unknown => unreachable!(),
        }
    }
}

impl From<u8> for BatchSegmentKind {
    fn from(value: u8) -> Self {
        use BatchSegmentKind::*;
        match value {
            0 => L2Message,
            1 => L2MessageBrotli,
            2 => DelayedMessages,
            3 => AdvanceTimestamp,
            4 => AdvanceL1BlockNumber,
            _ => Unknown,
        }
    }
}

/// Extracts the L2 messages contained in a parsed sequencer batch.
///
/// TODO: not yet implemented. The full version must walk `seq_msg.segments`,
/// interleaving delayed messages read from `delayed_message_db` whenever the
/// batch's `after_delayed_messages` count runs ahead of
/// `mel_state.delayed_messages_read`, while advancing the per-message
/// timestamp / block-number cursors. For now it returns no messages.
pub fn extract_batch_messages<D: DelayedMessageDB>(
    mel_state: &MelState,
    seq_msg: SequencerMessage,
    _delayed_message_db: &D,
) -> MelResult<Vec<MessageWithMetadata>> {
    let mut messages: Vec<MessageWithMetadata> = Vec::with_capacity(seq_msg.segments.len());
    let mut tstamp = 0u64;
    let mut block_number = 0u64;
    let mut segments = seq_msg.segments;
    if segments.is_empty() {
        let d_msg_kind: u8 = BatchSegmentKind::DelayedMessages.into();
        segments = vec![vec![d_msg_kind]];
    }
    for (idx, segment) in segments.into_iter().enumerate() {
        match message_from_segment(&segment) {
            Ok((msg, new_block_number, new_timestamp)) => {
                tstamp = new_timestamp;
                block_number = new_block_number;
                if msg.is_some() {
                    messages.push(msg.unwrap());
                }
            },
            Err(MelError::ParsingAdvancingSegmentFailed) => {
                continue;
            },
            Err(e) => {
                return Err(e);
            }
        }
    }

    while mel_state.delayed_messages_read < seq_msg.after_delayed_messages {
        messages.push(extract_delayed_msg_from_segment(mel_state)?);
    }
    Ok(messages)
}

fn message_from_segment(
    segment: &[u8],
    timestamp: u64,
    block_number: u64,
) -> MelResult<(Option<MessageWithMetadata>, u64, u64)> {
    if segment.is_empty() {
        return Ok((None, 0, 0));
    }
    let mut timestamp = timestamp;
    let mut block_number = block_number;
    let kind: BatchSegmentKind = segment[0].into();
    use BatchSegmentKind::*;
    match kind {
        AdvanceTimestamp => {
            let advancing: u64 = alloy_rlp::decode_exact(&segment[1..]).unwrap();
            timestamp += advancing;
            return Ok((None, timestamp, block_number));
        }
        AdvanceL1BlockNumber => {
            let advancing: u64 = alloy_rlp::decode_exact(&segment[1..]).unwrap();
            block_number += advancing;
            return Ok((None, timestamp, block_number));
        }
        L2Message | L2MessageBrotli => {
            let segment = &segment[1..];
            let msg = produce_l2_message();
        }
        DelayedMessages =>  {

        }
        Unknown => {

        }
    }
    Ok((Some(MessageWithMetadata::default()), timestamp, block_number))
}

fn extract_delayed_msg_from_segment(
    mel_state: &MelState,
) -> MelResult<MessageWithMetadata> {
    Ok(MessageWithMetadata::default())
}

fn produce_l2_message(
    kind: u8,
    seq_msg: SequencerMessage,
    segment: &[u8],
    block_number: u64,
    timestamp: u64,
    delayed_messages_read: u64,
) -> MelResult<MessageWithMetadata> {
    let mut timestamp = timestamp;
    let mut block_number = block_number;
    if timestamp < seq_msg.min_timestamp {
        timestamp = seq_msg.min_timestamp;
    }
    Ok(MessageWithMetadata { 
        message: L1IncomingMessage {
            header: L1IncomingMessageHeader {
                kind: 0,
                poster: Address::default(),
                block_number: 0,
                timestamp: 0,
                request_id: None,
                l1_base_fee: None,
            },
            l2_msg: vec![],
            batch_gas_left: None,
        },
        delayed_messages_read,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::MockDelayedDb;

    #[test]
    fn extraction_stub_returns_no_messages() -> MelResult<()> {
        // The extractor is a stub; it yields no messages regardless of input.
        let out = extract_batch_messages(
            &MelState::default(),
            SequencerMessage::default(),
            &MockDelayedDb,
        )?;
        assert!(out.is_empty());
        Ok(())
    }
}
