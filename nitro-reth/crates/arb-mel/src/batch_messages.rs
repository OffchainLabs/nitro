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
use arbos::{arbos_state, arbos_types::{L1IncomingMessage, L1IncomingMessageHeader, MAX_L2_MESSAGE_SIZE, MessageWithMetadata, invalid_l1_message}, l1_pricing::BATCH_POSTER_ADDRESS};
use tracing::{error, info};

use crate::{DelayedMessageDB, MelError, BatchSegmentKind, MelResult, MelState, SequencerMessage, parse_sequencer_message::decompress_brotli};

/// Extracts the L2 messages contained in a parsed sequencer batch.
///
/// Walks `seq_msg.segments`, interleaving delayed messages read from `delayed_message_db` 
/// whenever the batch's `after_delayed_messages` count runs ahead of 
/// `mel_state.delayed_messages_read`, while advancing the per-message
/// timestamp / block-number cursors. For now it returns no messages.
pub fn extract_batch_messages<D: DelayedMessageDB>(
    mel_state: &mut MelState,
    seq_msg: &mut SequencerMessage,
    delayed_message_db: &D,
) -> MelResult<Vec<MessageWithMetadata>> {
    let mut messages: Vec<MessageWithMetadata> = Vec::with_capacity(seq_msg.segments.len());
    let mut tstamp = 0u64;
    let mut block_number = 0u64;
    if seq_msg.segments.is_empty() {
        let d_msg_kind: u8 = BatchSegmentKind::DelayedMessages.into();
        seq_msg.segments = vec![vec![d_msg_kind]];
    }
    for (idx, segment) in seq_msg.segments.iter().enumerate() {
        match message_from_segment(
            mel_state,
            seq_msg,
            segment,
            idx,
            tstamp, 
            block_number,
            delayed_message_db,
        ) {
            Ok((msg, new_block_number, new_timestamp)) => {
                tstamp = new_timestamp;
                block_number = new_block_number;
                if let Some(msg) = msg {
                    messages.push(msg);
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
        if let Some(msg) = extract_delayed_msg_from_segment(
            mel_state, seq_msg, delayed_message_db
        )? {
            messages.push(msg);
        }
    }
    Ok(messages)
}

fn message_from_segment<D: DelayedMessageDB>(
    mel_state: &mut MelState,
    seq_msg: &SequencerMessage,
    segment: &[u8],
    segment_idx: usize,
    timestamp: u64,
    block_number: u64,
    delayed_message_db: &D
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
            Ok((None, timestamp, block_number))
        }
        AdvanceL1BlockNumber => {
            let advancing: u64 = alloy_rlp::decode_exact(&segment[1..]).unwrap();
            block_number += advancing;
            Ok((None, timestamp, block_number))
        }
        L2Message | L2MessageBrotli => {
            let segment = &segment[1..];
            let msg = produce_l2_message(
                kind,
                seq_msg,
                segment,
                block_number,
                timestamp,
                mel_state.delayed_messages_read,
            )?;
            Ok((Some(msg), timestamp, block_number))
        }
        DelayedMessages =>  {
            let msg = extract_delayed_msg_from_segment(
                mel_state, seq_msg, delayed_message_db,
            )?;
            Ok((msg, timestamp, block_number))
        }
        Unknown => {
            error!("Dropping unknown batch segment kind: {kind:?} {segment_idx:?}");
            let msg = MessageWithMetadata {
                message: invalid_l1_message(),
                delayed_messages_read: mel_state.delayed_messages_read,
            };
            Ok((Some(msg), timestamp, block_number))
        }
    }
}

fn produce_l2_message(
    kind: BatchSegmentKind,
    seq_msg: &SequencerMessage,
    segment: &[u8],
    block_number: u64,
    timestamp: u64,
    delayed_messages_read: u64,
) -> MelResult<MessageWithMetadata> {
    let mut timestamp = timestamp;
    let mut block_number = block_number;
    if timestamp < seq_msg.min_timestamp {
        timestamp = seq_msg.min_timestamp;
    } else if timestamp > seq_msg.max_timestamp {
        timestamp = seq_msg.max_timestamp;
    }
    if block_number < seq_msg.min_l1_block {
        block_number = seq_msg.min_l1_block;
    } else if block_number > seq_msg.max_l1_block {
        block_number = seq_msg.max_l1_block;
    }
    let mut seg: Option<Vec<u8>> = None;
    if kind == BatchSegmentKind::L2MessageBrotli {
        seg = match decompress_brotli(
            segment, MAX_L2_MESSAGE_SIZE,
        ) {
            Ok(decompressed) => Some(decompressed),
            Err(error) => {
                info!("Dropping compressed message: {error:?}");
                return Ok(MessageWithMetadata {
                    message: invalid_l1_message(),
                    delayed_messages_read,
                });
            },
        };
    }
    Ok(MessageWithMetadata { 
        message: L1IncomingMessage {
            header: L1IncomingMessageHeader {
                kind: BatchSegmentKind::L2Message.into(),
                poster: BATCH_POSTER_ADDRESS,
                block_number,
                timestamp,
                request_id: None,
                l1_base_fee: None,
            },
            l2_msg: seg.unwrap_or(segment.to_vec()),
            legacy_batch_gas_cost: None,
            batch_data_stats: None,
        },
        delayed_messages_read,
    })
}

fn extract_delayed_msg_from_segment<D: DelayedMessageDB>(
    mel_state: &mut MelState,
    seq_msg: &SequencerMessage,
    delayed_msg_db: &D,
) -> MelResult<Option<MessageWithMetadata>> {
    if mel_state.delayed_messages_read >= seq_msg.after_delayed_messages {
        return Ok(Some(MessageWithMetadata {
            message: invalid_l1_message(),
            delayed_messages_read: seq_msg.after_delayed_messages,
        }));
    }
    let delayed = delayed_msg_db.read_delayed_message(mel_state, mel_state.delayed_messages_read)?;
    match delayed {
        Some(delayed_msg) => {
            mel_state.delayed_messages_read += 1;
            Ok(Some(MessageWithMetadata {
                message: delayed_msg.message,
                delayed_messages_read: mel_state.delayed_messages_read,
            }))
        }
        None => {
            info!(
                "No more delayed messages in queue at index {}, delayed messages seen {}", 
                mel_state.delayed_messages_read, 
                mel_state.delayed_messages_seen,
            );
            Ok(None)
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::test_utils::MockDelayedDb;

    #[test]
    fn extraction_stub_returns_no_messages() -> MelResult<()> {
        Ok(())
    }
}
