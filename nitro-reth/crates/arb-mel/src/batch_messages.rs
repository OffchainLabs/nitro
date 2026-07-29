use alloy_primitives::U256;
use arbos::{
    l1_pricing::BATCH_POSTER_ADDRESS,
    types::{
        L1_MESSAGE_TYPE_L2_MESSAGE, L1IncomingMessage, L1IncomingMessageHeader,
        MAX_L2_MESSAGE_SIZE, MessageWithMetadata, invalid_l1_message,
    },
};
use tracing::{error, info, warn};

use crate::{
    BatchSegmentKind, DelayedMessageDB, MelError, MelResult, MelState, SequencerMessage,
    parse_sequencer_message::decompress_brotli,
};

/// The result of parsing a single batch segment: an optional message together
/// with the timestamp and block-number cursors after the segment was applied.
struct MessageFromSegment {
    message: Option<MessageWithMetadata>,
    timestamp: u64,
    block_number: u64,
}

/// Extracts the L2 messages contained in a parsed sequencer batch.
///
/// Walks `seq_msg.segments`, interleaving delayed messages read from `delayed_message_db`
/// whenever the batch's `after_delayed_messages` count runs ahead of
/// `mel_state.delayed_messages_read`, while advancing the per-message
/// timestamp / block-number cursors.
pub fn extract_batch_messages<D: DelayedMessageDB>(
    mel_state: &mut MelState,
    seq_msg: &mut SequencerMessage,
    delayed_message_db: &D,
) -> MelResult<Vec<MessageWithMetadata>> {
    let mut messages: Vec<MessageWithMetadata> = Vec::with_capacity(seq_msg.segments.len());
    let mut timestamp = 0u64;
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
            timestamp,
            block_number,
            delayed_message_db,
        ) {
            Ok(parsed) => {
                timestamp = parsed.timestamp;
                block_number = parsed.block_number;
                if let Some(msg) = parsed.message {
                    messages.push(msg);
                }
            }
            // An unparseable advance segment is skipped, matching the Go decoder.
            Err(MelError::ParsingAdvancingSegmentFailed) => continue,
            Err(e) => return Err(e),
        }
    }

    // Read any remaining delayed messages the batch claims but the segments did
    // not cover.
    while mel_state.delayed_messages_read < seq_msg.after_delayed_messages {
        let msg = extract_delayed_msg_from_segment(mel_state, seq_msg, delayed_message_db)?;
        messages.push(msg);
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
    delayed_message_db: &D,
) -> MelResult<MessageFromSegment> {
    if segment.is_empty() {
        warn!("empty segment in sequencer message");
        return Ok(MessageFromSegment {
            message: None,
            timestamp,
            block_number,
        });
    }
    let mut timestamp = timestamp;
    let mut block_number = block_number;
    let kind: BatchSegmentKind = segment[0].into();
    use BatchSegmentKind::*;
    match kind {
        AdvanceTimestamp | AdvanceL1BlockNumber => {
            let advancing: u64 = alloy_rlp::decode_exact(&segment[1..])
                .map_err(|_| MelError::ParsingAdvancingSegmentFailed)?;
            if kind == AdvanceTimestamp {
                timestamp += advancing;
            } else {
                block_number += advancing;
            }
            Ok(MessageFromSegment {
                message: None,
                timestamp,
                block_number,
            })
        }
        L2Message | L2MessageBrotli => {
            let msg = produce_l2_message(
                kind,
                seq_msg,
                &segment[1..],
                block_number,
                timestamp,
                mel_state.delayed_messages_read,
            )?;
            Ok(MessageFromSegment {
                message: Some(msg),
                timestamp,
                block_number,
            })
        }
        DelayedMessages => {
            let msg = extract_delayed_msg_from_segment(mel_state, seq_msg, delayed_message_db)?;
            Ok(MessageFromSegment {
                message: Some(msg),
                timestamp,
                block_number,
            })
        }
        Unknown => {
            error!("Dropping unknown batch segment kind: {kind:?} {segment_idx:?}");
            Ok(MessageFromSegment {
                message: Some(MessageWithMetadata {
                    message: invalid_l1_message(),
                    delayed_messages_read: mel_state.delayed_messages_read,
                }),
                timestamp,
                block_number,
            })
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
        seg = match decompress_brotli(segment, MAX_L2_MESSAGE_SIZE) {
            Ok(decompressed) => Some(decompressed),
            Err(error) => {
                info!("Dropping compressed message: {error:?}");
                return Ok(MessageWithMetadata {
                    message: invalid_l1_message(),
                    delayed_messages_read,
                });
            }
        };
    }
    Ok(MessageWithMetadata {
        message: L1IncomingMessage {
            header: L1IncomingMessageHeader {
                kind: L1_MESSAGE_TYPE_L2_MESSAGE,
                poster: BATCH_POSTER_ADDRESS,
                block_number,
                timestamp,
                request_id: None,
                l1_base_fee: Some(U256::ZERO),
            },
            l2_msg: seg.unwrap_or(segment.to_vec()).into(),
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
) -> MelResult<MessageWithMetadata> {
    if mel_state.delayed_messages_read >= seq_msg.after_delayed_messages {
        return Ok(MessageWithMetadata {
            message: invalid_l1_message(),
            delayed_messages_read: seq_msg.after_delayed_messages,
        });
    }
    let delayed =
        delayed_msg_db.read_delayed_message(mel_state, mel_state.delayed_messages_read)?;
    match delayed {
        Some(delayed_msg) => {
            mel_state.delayed_messages_read += 1;
            Ok(MessageWithMetadata {
                message: delayed_msg.message,
                delayed_messages_read: mel_state.delayed_messages_read,
            })
        }
        None => {
            error!(
                "No more delayed messages in queue at index {}, delayed messages seen {}",
                mel_state.delayed_messages_read, mel_state.delayed_messages_seen,
            );
            Err(MelError::NoMoreDelayedMessages)
        }
    }
}

#[cfg(test)]
mod tests {
    use alloy_primitives::B256;
    use arbos::types::{L1_MESSAGE_TYPE_INVALID, L1IncomingMessage};

    use super::*;
    use crate::{
        DelayedInboxMessage,
        test_utils::{
            MockDelayedDb, brotli_compress, sequencer_message_with_segments,
            sequencer_message_with_timestamp_range,
        },
    };

    fn delayed_with_l2(l2: &[u8]) -> DelayedInboxMessage {
        DelayedInboxMessage {
            block_hash: B256::ZERO,
            before_inbox_acc: B256::ZERO,
            message: L1IncomingMessage {
                l2_msg: l2.to_vec().into(),
                ..Default::default()
            },
            parent_chain_block_number: 0,
        }
    }

    fn advance_timestamp(delta: u64) -> Vec<u8> {
        let mut seg = vec![BatchSegmentKind::AdvanceTimestamp.into()];
        seg.extend_from_slice(&alloy_rlp::encode(delta));
        seg
    }

    fn advance_blocknum(delta: u64) -> Vec<u8> {
        let mut seg = vec![BatchSegmentKind::AdvanceL1BlockNumber.into()];
        seg.extend_from_slice(&alloy_rlp::encode(delta));
        seg
    }

    fn brotli_l2(raw: &[u8]) -> Vec<u8> {
        let mut seg = vec![BatchSegmentKind::L2MessageBrotli.into()];
        seg.extend_from_slice(&brotli_compress(raw));
        seg
    }

    #[test]
    fn field_bounds_simple() -> MelResult<()> {
        let segments = vec![
            advance_timestamp(7),
            advance_timestamp(3),
            brotli_l2(b"foobar"),
        ];
        let mut seq_msg = sequencer_message_with_timestamp_range(segments, 10, 20);
        let mut state = MelState::default();
        let msgs = extract_batch_messages(&mut state, &mut seq_msg, &MockDelayedDb::default())?;
        assert_eq!(msgs.len(), 1);
        assert_eq!(msgs[0].message.l2_msg.as_ref(), b"foobar");
        assert_eq!(msgs[0].message.header.timestamp, 10);
        Ok(())
    }

    #[test]
    fn field_bounds_complex() -> MelResult<()> {
        let segments = vec![
            advance_timestamp(7),
            brotli_l2(b"foobar"),
            advance_timestamp(3),
            brotli_l2(b"foobar"),
        ];
        let mut seq_msg = sequencer_message_with_timestamp_range(segments, 10, 20);
        let mut state = MelState::default();
        let msgs = extract_batch_messages(&mut state, &mut seq_msg, &MockDelayedDb::default())?;
        assert_eq!(msgs.len(), 2);
        assert_eq!(msgs[0].message.l2_msg.as_ref(), b"foobar");
        assert_eq!(msgs[0].message.header.timestamp, 10);
        assert_eq!(msgs[1].message.l2_msg.as_ref(), b"foobar");
        assert_eq!(msgs[1].message.header.timestamp, 10);
        Ok(())
    }

    #[test]
    fn reads_trailing_delayed_messages() -> MelResult<()> {
        let mut seq_msg = sequencer_message_with_segments(2, vec![]);
        let mut state = MelState::default();
        let db = MockDelayedDb::with_messages([
            (0, delayed_with_l2(b"foobar")),
            (1, delayed_with_l2(b"barfoo")),
        ]);
        let msgs = extract_batch_messages(&mut state, &mut seq_msg, &db)?;
        assert_eq!(msgs.len(), 2);
        assert_eq!(msgs[0].message.l2_msg.as_ref(), b"foobar");
        assert_eq!(msgs[1].message.l2_msg.as_ref(), b"barfoo");
        Ok(())
    }

    #[test]
    fn segment_advances_timestamp() -> MelResult<()> {
        let segments = vec![advance_timestamp(50), brotli_l2(b"foobar")];
        let mut seq_msg = sequencer_message_with_timestamp_range(segments, 0, 1_000_000);
        let mut state = MelState::default();
        let msgs = extract_batch_messages(&mut state, &mut seq_msg, &MockDelayedDb::default())?;
        assert_eq!(msgs.len(), 1);
        assert_eq!(msgs[0].message.l2_msg.as_ref(), b"foobar");
        assert_eq!(msgs[0].message.header.timestamp, 50);
        Ok(())
    }

    #[test]
    fn segment_advances_blocknum() -> MelResult<()> {
        let segments = vec![advance_blocknum(20), brotli_l2(b"foobar")];
        let mut seq_msg = SequencerMessage {
            min_timestamp: 0,
            max_timestamp: 1_000_000,
            min_l1_block: 0,
            max_l1_block: 1_000_000,
            after_delayed_messages: 0,
            segments,
        };
        let mut state = MelState::default();
        let msgs = extract_batch_messages(&mut state, &mut seq_msg, &MockDelayedDb::default())?;
        assert_eq!(msgs.len(), 1);
        assert_eq!(msgs[0].message.l2_msg.as_ref(), b"foobar");
        assert_eq!(msgs[0].message.header.block_number, 20);
        Ok(())
    }

    #[test]
    fn brotli_message_with_unparseable_advance() -> MelResult<()> {
        let mut advance = vec![BatchSegmentKind::AdvanceTimestamp.into()];
        advance.extend_from_slice(&50u64.to_be_bytes());
        let segments = vec![advance, brotli_l2(b"foobar")];
        let mut seq_msg = sequencer_message_with_timestamp_range(segments, 0, 1_000_000);
        let mut state = MelState::default();
        let msgs = extract_batch_messages(&mut state, &mut seq_msg, &MockDelayedDb::default())?;
        assert_eq!(msgs.len(), 1);
        assert_eq!(msgs[0].message.l2_msg.as_ref(), b"foobar");
        Ok(())
    }

    #[test]
    fn delayed_segment_beyond_read_is_invalid() -> MelResult<()> {
        let segments = vec![vec![BatchSegmentKind::DelayedMessages.into()]];
        let mut seq_msg = sequencer_message_with_segments(1, segments);
        let mut state = MelState {
            delayed_messages_read: 1,
            ..Default::default()
        };
        let msgs = extract_batch_messages(&mut state, &mut seq_msg, &MockDelayedDb::default())?;
        assert_eq!(msgs.len(), 1);
        assert_eq!(msgs[0].message.header.kind, L1_MESSAGE_TYPE_INVALID);
        Ok(())
    }

    #[test]
    fn delayed_db_error_propagates() {
        let mut seq_msg = sequencer_message_with_segments(1, vec![vec![]]);
        let mut state = MelState::default();
        let result = extract_batch_messages(&mut state, &mut seq_msg, &MockDelayedDb::failing());
        assert!(result.is_err());
    }

    #[test]
    fn delayed_missing_from_db_errors() {
        let mut seq_msg = sequencer_message_with_segments(1, vec![vec![]]);
        let mut state = MelState::default();
        let result = extract_batch_messages(&mut state, &mut seq_msg, &MockDelayedDb::default());
        assert!(matches!(result, Err(MelError::NoMoreDelayedMessages)));
    }

    #[test]
    fn reading_delayed_message_ok() -> MelResult<()> {
        let mut seq_msg = sequencer_message_with_segments(1, vec![vec![]]);
        let mut state = MelState::default();
        let db = MockDelayedDb::with_messages([(0, delayed_with_l2(b"foobar"))]);
        let msgs = extract_batch_messages(&mut state, &mut seq_msg, &db)?;
        assert_eq!(msgs.len(), 1);
        assert_eq!(msgs[0].message.l2_msg.as_ref(), b"foobar");
        Ok(())
    }
}
