use alloy_consensus::Header;
use alloy_primitives::{B256, keccak256};
use arb_da_provider_client::DaReaderSource;
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

pub async fn extract_messages<R, D, L, T>(
    input_state: &MelState,
    parent_chain_header: &Header,
    da_reader_source: &R,
    delayed_msg_db: &D,
    logs_fetcher: &L,
    tx_fetcher: &T,
) -> MelResult<ExtractionOutput>
where
    R: DaReaderSource,
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
    let mut post_state = input_state.clone();
    // LocalMsgAccumulator restarts per block (mirrors nitro's State.Clone).
    post_state.local_msg_accumulator = B256::ZERO;
    post_state.parent_chain_block_hash = parent_chain_header.hash_slow();
    post_state.parent_chain_prev_block_hash = input_state.parent_chain_block_hash;
    post_state.parent_chain_block_number = parent_chain_header.number;

    let (mut batches, batch_txs) = batch_lookup::parse_batches_from_block(
        &post_state,
        parent_chain_header,
        tx_fetcher,
        logs_fetcher,
    )
    .await?;
    let mut delayed_messages = delayed_message_lookup::parse_delayed_messages_from_block(
        &post_state,
        parent_chain_header,
        tx_fetcher,
        logs_fetcher,
    )
    .await?;

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
            da_reader_source,
        )
        .await?;
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
    use arb_da_provider_client::DaReaderRegistry;

    use super::*;
    use crate::test_utils::{MockDelayedDb, MockLogs, MockTx};

    #[tokio::test]
    async fn rejects_parent_hash_mismatch() {
        // Input state's parent hash does not line up with the header's parent.
        let input_state = MelState {
            parent_chain_block_hash: B256::repeat_byte(0xAB),
            ..Default::default()
        };
        let result = extract_messages(
            &input_state,
            &Header::default(),
            &DaReaderRegistry::new(),
            &MockDelayedDb,
            &MockLogs::default(),
            &MockTx,
        )
        .await;
        assert!(matches!(
            result,
            Err(MelError::ParentHashMismatch { expected, .. }) if expected == B256::repeat_byte(0xAB)
        ));
    }

    #[tokio::test]
    async fn extracts_empty_block_and_advances_state() -> MelResult<()> {
        // A default header (number 0, zero parent hash) with no logs: linkage
        // holds, and the post-state records the header linkage while producing no
        // messages.
        let header = Header::default();
        let input_state = MelState::default();
        let out = extract_messages(
            &input_state,
            &header,
            &DaReaderRegistry::new(),
            &MockDelayedDb,
            &MockLogs::default(),
            &MockTx,
        )
        .await?;

        assert!(out.messages.is_empty());
        assert!(out.delayed_messages.is_empty());
        assert!(out.batch_metas.is_empty());
        assert_eq!(out.post_state.parent_chain_block_hash, header.hash_slow());
        assert_eq!(out.post_state.parent_chain_prev_block_hash, B256::ZERO);
        assert_eq!(out.post_state.parent_chain_block_number, header.number);
        Ok(())
    }
}
