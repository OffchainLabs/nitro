use alloy_consensus::Header;
use alloy_primitives::{B256, keccak256};
use arbos::arbos_types::{
    L1_MESSAGE_TYPE_BATCH_POSTING_REPORT, MessageWithMetadata, get_data_stats,
    legacy_cost_for_stats, parse_batch_posting_report_fields,
};

use crate::{
    BatchMeta, DelayedInboxMessage, DelayedMessageDB, LogsFetcher, MelError, MelResult, MelState,
    TxFetcher, batch_lookup, batch_messages, delayed_message_lookup, mel_config_lookup,
    parse_sequencer_message, serialize_batch,
};

pub struct ExtractionOutput {
    pub post_state: MelState,
    pub messages: Vec<MessageWithMetadata>,
    pub delayed_messages: Vec<DelayedInboxMessage>,
    pub batch_metas: Vec<BatchMeta>,
}

pub fn extract_messages<D, L, T>(
    input_state: MelState,
    parent_chain_header: &Header,
    delayed_msg_db: &D,
    logs_fetcher: &L,
    tx_fetcher: &T,
) -> MelResult<ExtractionOutput>
where
    D: DelayedMessageDB,
    L: LogsFetcher,
    T: TxFetcher,
    T::Transaction: alloy_consensus::Transaction,
{
    // Verify parent chain header linkage.
    if input_state.parent_chain_block_hash != parent_chain_header.parent_hash {
        return Err(MelError::ParentHashMismatch {
            expected: input_state.parent_chain_block_hash,
            got: parent_chain_header.parent_hash,
        });
    }
    // TODO: reset the local_msg_accumulator field to empty after clone.
    let mut post_state = input_state.clone();
    post_state.parent_chain_block_hash = parent_chain_header.hash_slow();
    post_state.parent_chain_prev_block_hash = input_state.parent_chain_block_hash;
    post_state.parent_chain_block_number = parent_chain_header.number;

    let (mut batches, batch_txs) = batch_lookup::parse_batches_from_block(
        &post_state,
        parent_chain_header,
        tx_fetcher,
        logs_fetcher,
    )?;
    let mut delayed_messages = delayed_message_lookup::parse_delayed_messages_from_block(
        &post_state,
        parent_chain_header,
        tx_fetcher,
        logs_fetcher,
    )?;

    // Save the indices of batch posting reports for later, once batches are
    // serialized and we need to fill in each report's batch gas stats. We track
    // indices rather than references so the reports can be mutated in place
    // below (they are also returned in the extraction output).
    let batch_posting_report_indices = delayed_messages
        .iter()
        .enumerate()
        .filter(|(_, delayed)| delayed.message.header.kind == L1_MESSAGE_TYPE_BATCH_POSTING_REPORT)
        .map(|(i, _)| i)
        .collect::<Vec<usize>>();
    if batch_posting_report_indices.len() > batches.len() {
        return Err(MelError::TooManyBatchPostingReports {
            reports: batch_posting_report_indices.len(),
            batches: batches.len(),
        });
    }

    let mut batch_post_report_idx: usize = 0;
    let mut batch_post_report_batch_hash = B256::ZERO;
    let mut serialized_batches: Vec<Vec<u8>> = Vec::new();
    for (batch, tx) in batches.iter_mut().zip(batch_txs.iter()) {
        let serialized = serialize_batch::serialize_batch(batch, tx, logs_fetcher)?;
        if batch_post_report_idx < batch_posting_report_indices.len() {
            let report_idx = batch_posting_report_indices[batch_post_report_idx];
            if batch_post_report_batch_hash == B256::ZERO {
                batch_post_report_batch_hash =
                    parse_batch_posting_report(&delayed_messages[report_idx])?;
            }
            let got_hash = keccak256(&serialized);
            if got_hash == batch_post_report_batch_hash {
                // Fill in the batch gas stats into the batch posting report.
                let stats = get_data_stats(&serialized);
                let legacy_cost = legacy_cost_for_stats(&stats);
                let report = &mut delayed_messages[report_idx];
                report.message.batch_data_stats = Some(stats);
                report.message.legacy_batch_gas_cost = Some(legacy_cost);
                // Process next report.
                batch_post_report_idx += 1;
                batch_post_report_batch_hash = B256::ZERO;
            }
        }
        serialized_batches.push(serialized);
    }

    // Batch posting reports are included in the same transaction as a batch, so
    // every report should have been matched to a batch and filled in with its
    // gas stats above.
    if batch_posting_report_indices.len() != batch_post_report_idx {
        return Err(MelError::BatchPostingReportsNotProcessed {
            reports: batch_posting_report_indices.len(),
            processed: batch_post_report_idx,
        });
    }

    // Update the delayed message inbox accumulator in the MelState.
    for delayed in delayed_messages.iter() {
        post_state.accumulate_delayed_message(delayed)?;
        post_state.delayed_messages_seen += 1;
    }

    let mut messages = Vec::new();
    let mut batch_metas = Vec::new();

    // Extract L2 messages from batches.
    for (i, batch) in batches.iter().enumerate() {
        let expected_batch_seq_num = batches[0].sequence_number + i as u64;
        if batch.sequence_number != expected_batch_seq_num {
            // This should never happen if the batch fetching logic is correct.
            return Err(MelError::BatchSequenceMismatch {
                expected: expected_batch_seq_num,
                got: batch.sequence_number,
            });
        }
        let serialized = &serialized_batches[i];
        let mut raw_seq_msg = parse_sequencer_message::parse_sequencer_message(
            batch.sequence_number,
            batch.block_hash,
            serialized,
            parse_sequencer_message::DEFAULT_MAX_UNCOMPRESSED_BATCH_SIZE,
        )?;
        let messages_in_batch = batch_messages::extract_batch_messages(
            &mut post_state,
            &mut raw_seq_msg,
            delayed_msg_db,
        )?;
        for msg in messages_in_batch.into_iter() {
            post_state.accumulate_message(&msg)?;
            messages.push(msg);
            post_state.msg_count += 1;
        }
        post_state.batch_count += 1;
        batch_metas.push(BatchMeta {
            accumulator: batch.after_inbox_acc,
            message_count: post_state.msg_count,
            delayed_message_count: batch.after_delayed_count,
            parent_chain_block: batch.parent_chain_block_number,
        });
        if batch.after_delayed_count != post_state.delayed_messages_read {
            return Err(MelError::DelayedCountMismatch {
                batch_after_delayed: batch.after_delayed_count,
                state_delayed_read: post_state.delayed_messages_read,
            });
        }
    }

    // Check for MEL config events in this block.
    if let Some(mel_config) =
        mel_config_lookup::parse_mel_config_from_block(parent_chain_header, logs_fetcher)?
    {
        // Sanity check: the contract sets activation block = block.number at emission.
        // This means the event must be observed in the same parent chain block
        // it was emitted in.
        if mel_config.activation_block != parent_chain_header.number {
            return Err(MelError::MelConfigActivationMismatch {
                activation_block: mel_config.activation_block,
                parent_chain_block: parent_chain_header.number,
            });
        }
        if post_state.version == 0 {
            post_state.move_unread_delayed_messages_to_inbox_accumulator(delayed_msg_db)?;
        }
        post_state.version = mel_config.mel_version;
        post_state.delayed_message_posting_target_address = mel_config.inbox;
        post_state.batch_posting_target_address = mel_config.sequencer_inbox;
    }

    Ok(ExtractionOutput {
        post_state,
        messages,
        delayed_messages,
        batch_metas,
    })
}

/// Parses a batch posting report delayed message and returns the batch data
/// hash it references.
fn parse_batch_posting_report(report: &DelayedInboxMessage) -> MelResult<B256> {
    let fields = parse_batch_posting_report_fields(&report.message.l2_msg)
        .map_err(|e| MelError::BatchPostingReportParse(e.to_string()))?;
    Ok(fields.data_hash)
}

#[cfg(test)]
mod tests {
    use alloy_primitives::{Address, U256};
    use alloy_rpc_types_eth::Log;
    use alloy_sol_types::{SolEvent, sol};
    use arbos::arbos_types::{L1IncomingMessageHeader, L1_MESSAGE_TYPE_BATCH_POSTING_REPORT};

    use super::*;
    use crate::DelayedInboxMessage;
    use crate::test_utils::{MockDelayedDb, MockLogs, MockTx, rpc_log};

    sol! {
        #[allow(missing_docs)]
        struct TimeBoundsAbi {
            uint64 minTimestamp;
            uint64 maxTimestamp;
            uint64 minBlockNumber;
            uint64 maxBlockNumber;
        }
        #[allow(missing_docs)]
        event SequencerBatchDelivered(
            uint256 indexed batchSequenceNumber,
            bytes32 indexed beforeAcc,
            bytes32 indexed afterAcc,
            bytes32 delayedAcc,
            uint256 afterDelayedMessagesRead,
            TimeBoundsAbi timeBounds,
            uint8 dataLocation
        );
        #[allow(missing_docs)]
        event MessageDelivered(
            uint256 indexed messageIndex,
            bytes32 indexed beforeInboxAcc,
            address inbox,
            uint8 kind,
            address sender,
            bytes32 messageDataHash,
            uint256 baseFeeL1,
            uint64 timestamp
        );
        #[allow(missing_docs)]
        event InboxMessageDelivered(uint256 indexed messageNum, bytes data);
    }

    const BATCH_TARGET: Address = Address::repeat_byte(0xBA);
    const DELAYED_TARGET: Address = Address::repeat_byte(0xDD);
    const INBOX: Address = Address::repeat_byte(0xEE);

    fn wired_state() -> MelState {
        MelState {
            batch_posting_target_address: BATCH_TARGET,
            delayed_message_posting_target_address: DELAYED_TARGET,
            ..Default::default()
        }
    }

    fn force_inclusion_batch_log(seq: u64, after_delayed: u64) -> Log {
        let ev = SequencerBatchDelivered {
            batchSequenceNumber: U256::from(seq),
            beforeAcc: B256::ZERO,
            afterAcc: B256::repeat_byte(0x22),
            delayedAcc: B256::ZERO,
            afterDelayedMessagesRead: U256::from(after_delayed),
            timeBounds: TimeBoundsAbi {
                minTimestamp: 0,
                maxTimestamp: 0,
                minBlockNumber: 0,
                maxBlockNumber: 0,
            },
            dataLocation: 3,
        };
        rpc_log(BATCH_TARGET, ev.encode_log_data())
    }

    fn report_logs(index: u64, data: Vec<u8>) -> Vec<Log> {
        let data_hash = keccak256(&data);
        let delivered = MessageDelivered {
            messageIndex: U256::from(index),
            beforeInboxAcc: B256::ZERO,
            inbox: INBOX,
            kind: L1_MESSAGE_TYPE_BATCH_POSTING_REPORT,
            sender: Address::ZERO,
            messageDataHash: data_hash,
            baseFeeL1: U256::ZERO,
            timestamp: 0,
        };
        let inbox_data = InboxMessageDelivered {
            messageNum: U256::from(index),
            data: data.into(),
        };
        vec![
            rpc_log(DELAYED_TARGET, delivered.encode_log_data()),
            rpc_log(INBOX, inbox_data.encode_log_data()),
        ]
    }

    fn report_body(data_hash: B256) -> Vec<u8> {
        let mut body = Vec::new();
        body.extend_from_slice(&[0u8; 32]);
        body.extend_from_slice(&[0u8; 20]);
        body.extend_from_slice(data_hash.as_slice());
        body.extend_from_slice(&[0u8; 32]);
        body.extend_from_slice(&[0u8; 32]);
        body
    }

    fn make_delayed_msg(i: u64) -> DelayedInboxMessage {
        let request_id = B256::from(U256::from(i).to_be_bytes::<32>());
        DelayedInboxMessage {
            block_hash: B256::ZERO,
            before_inbox_acc: B256::ZERO,
            message: arbos::arbos_types::L1IncomingMessage {
                header: L1IncomingMessageHeader {
                    request_id: Some(request_id),
                    ..Default::default()
                },
                l2_msg: vec![i as u8],
                ..Default::default()
            },
            parent_chain_block_number: 0,
        }
    }

    #[test]
    fn rejects_parent_hash_mismatch() {
        // Input state's parent hash does not line up with the header's parent.
        let input_state = MelState {
            parent_chain_block_hash: B256::repeat_byte(0xAB),
            ..Default::default()
        };
        let result = extract_messages(
            input_state,
            &Header::default(),
            &MockDelayedDb::default(),
            &MockLogs::default(),
            &MockTx,
        );
        assert!(matches!(
            result,
            Err(MelError::ParentHashMismatch { expected, .. }) if expected == B256::repeat_byte(0xAB)
        ));
    }

    #[test]
    fn extracts_empty_block_and_advances_state() -> MelResult<()> {
        // A default header (number 0, zero parent hash) with no logs: linkage
        // holds, and the post-state records the header linkage while producing no
        // messages.
        let header = Header::default();
        let input_state = MelState::default();
        let out = extract_messages(
            input_state,
            &header,
            &MockDelayedDb::default(),
            &MockLogs::default(),
            &MockTx,
        )?;

        assert!(out.messages.is_empty());
        assert!(out.delayed_messages.is_empty());
        assert!(out.batch_metas.is_empty());
        assert_eq!(out.post_state.parent_chain_block_hash, header.hash_slow());
        assert_eq!(out.post_state.parent_chain_prev_block_hash, B256::ZERO);
        assert_eq!(out.post_state.parent_chain_block_number, header.number);
        Ok(())
    }

    #[test]
    fn extracts_force_inclusion_batch() -> MelResult<()> {
        let logs = MockLogs {
            block_logs: vec![force_inclusion_batch_log(0, 0)],
            ..Default::default()
        };
        let out = extract_messages(
            wired_state(),
            &Header::default(),
            &MockDelayedDb::default(),
            &logs,
            &MockTx,
        )?;
        assert_eq!(out.post_state.batch_count, 1);
        assert_eq!(out.post_state.msg_count, 1);
        assert_eq!(out.post_state.delayed_messages_seen, 0);
        assert_eq!(out.messages.len(), 1);
        assert_eq!(out.batch_metas.len(), 1);
        assert_eq!(out.batch_metas[0].accumulator, B256::repeat_byte(0x22));
        assert!(out.delayed_messages.is_empty());
        Ok(())
    }

    #[test]
    fn rejects_too_many_batch_posting_reports() {
        let logs = MockLogs {
            block_logs: report_logs(0, b"x".to_vec()),
            ..Default::default()
        };
        let result = extract_messages(
            wired_state(),
            &Header::default(),
            &MockDelayedDb::default(),
            &logs,
            &MockTx,
        );
        assert!(matches!(
            result,
            Err(MelError::TooManyBatchPostingReports {
                reports: 1,
                batches: 0
            })
        ));
    }

    #[test]
    fn rejects_unprocessed_batch_posting_report() {
        let mut block_logs = vec![force_inclusion_batch_log(0, 0)];
        block_logs.extend(report_logs(0, report_body(B256::repeat_byte(0xFF))));
        let logs = MockLogs {
            block_logs,
            ..Default::default()
        };
        let result = extract_messages(
            wired_state(),
            &Header::default(),
            &MockDelayedDb::default(),
            &logs,
            &MockTx,
        );
        assert!(matches!(
            result,
            Err(MelError::BatchPostingReportsNotProcessed {
                reports: 1,
                processed: 0
            })
        ));
    }

    #[test]
    fn move_unread_rebuilds_inbox_accumulator() -> MelResult<()> {
        let mut state = MelState {
            delayed_messages_read: 2,
            delayed_messages_seen: 5,
            ..Default::default()
        };
        let db = MockDelayedDb::with_messages([
            (2, make_delayed_msg(2)),
            (3, make_delayed_msg(3)),
            (4, make_delayed_msg(4)),
        ]);
        state.move_unread_delayed_messages_to_inbox_accumulator(&db)?;

        let mut expected = MelState {
            delayed_messages_read: 2,
            delayed_messages_seen: 5,
            ..Default::default()
        };
        for i in 2..5 {
            expected.accumulate_delayed_message(&make_delayed_msg(i))?;
        }
        assert_eq!(
            state.delayed_message_inbox_acc,
            expected.delayed_message_inbox_acc
        );
        assert_ne!(state.delayed_message_inbox_acc, B256::ZERO);
        assert_eq!(state.delayed_message_outbox_acc, B256::ZERO);
        Ok(())
    }

    #[test]
    fn move_unread_no_messages_leaves_state_unchanged() -> MelResult<()> {
        let mut state = MelState {
            delayed_messages_read: 3,
            delayed_messages_seen: 3,
            ..Default::default()
        };
        state.move_unread_delayed_messages_to_inbox_accumulator(&MockDelayedDb::default())?;
        assert_eq!(state.delayed_message_inbox_acc, B256::ZERO);
        assert_eq!(state.delayed_message_outbox_acc, B256::ZERO);
        Ok(())
    }

    #[test]
    fn move_unread_errors_when_inbox_accumulator_non_zero() {
        let preexisting = B256::repeat_byte(0xBE);
        let mut state = MelState {
            delayed_messages_read: 0,
            delayed_messages_seen: 1,
            delayed_message_inbox_acc: preexisting,
            ..Default::default()
        };
        let db = MockDelayedDb::with_messages([(0, make_delayed_msg(0))]);
        let result = state.move_unread_delayed_messages_to_inbox_accumulator(&db);
        assert!(matches!(
            result,
            Err(MelError::NonZeroDelayedAccumulator { .. })
        ));
        assert_eq!(state.delayed_message_inbox_acc, preexisting);
    }

    #[test]
    fn move_unread_errors_when_outbox_accumulator_non_zero() {
        let preexisting = B256::repeat_byte(0xFA);
        let mut state = MelState {
            delayed_messages_read: 0,
            delayed_messages_seen: 1,
            delayed_message_outbox_acc: preexisting,
            ..Default::default()
        };
        let db = MockDelayedDb::with_messages([(0, make_delayed_msg(0))]);
        let result = state.move_unread_delayed_messages_to_inbox_accumulator(&db);
        assert!(matches!(
            result,
            Err(MelError::NonZeroDelayedAccumulator { .. })
        ));
        assert_eq!(state.delayed_message_outbox_acc, preexisting);
    }

    #[test]
    fn move_unread_propagates_db_read_errors() {
        let mut state = MelState {
            delayed_messages_read: 0,
            delayed_messages_seen: 2,
            ..Default::default()
        };
        let result =
            state.move_unread_delayed_messages_to_inbox_accumulator(&MockDelayedDb::failing());
        assert!(matches!(
            result,
            Err(MelError::DelayedAccumulatorCreation(_))
        ));
    }
}
