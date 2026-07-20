//! Parsing `SequencerBatchDelivered` logs from a parent-chain block.

use alloy_consensus::Header;
use alloy_sol_types::{SolEvent, sol};

use crate::{
    Batch, DataLocation, LogsFetcher, MelError, MelResult, MelState, TimeBounds, TxFetcher,
};

sol! {
    #[allow(missing_docs)]
    #[derive(Debug)]
    struct TimeBoundsAbi {
        uint64 minTimestamp;
        uint64 maxTimestamp;
        uint64 minBlockNumber;
        uint64 maxBlockNumber;
    }
    #[allow(missing_docs)]
    #[derive(Debug)]
    event SequencerBatchDelivered(
        uint256 indexed batchSequenceNumber,
        bytes32 indexed beforeAcc,
        bytes32 indexed afterAcc,
        bytes32 delayedAcc,
        uint256 afterDelayedMessagesRead,
        TimeBoundsAbi timeBounds,
        uint8 dataLocation
    );
}

impl From<TimeBoundsAbi> for TimeBounds {
    fn from(tb: TimeBoundsAbi) -> Self {
        TimeBounds {
            min_timestamp: tb.minTimestamp,
            max_timestamp: tb.maxTimestamp,
            min_block_number: tb.minBlockNumber,
            max_block_number: tb.maxBlockNumber,
        }
    }
}

/// Parses all `SequencerBatchDelivered` batches (and their txs) from the logs
/// of a single parent-chain block.
#[allow(dead_code)] // Wired into MEL block processing in a later change.
pub(crate) fn parse_batches_from_block<L, T>(
    mel_state: &MelState,
    parent_chain_header: &Header,
    tx_fetcher: &T,
    logs_fetcher: &L,
) -> MelResult<(Vec<Batch>, Vec<T::Transaction>)>
where
    L: LogsFetcher,
    T: TxFetcher,
{
    let block_hash = parent_chain_header.hash_slow();
    let logs = logs_fetcher.logs_for_block_hash(block_hash)?;

    let mut batches = Vec::with_capacity(logs.len());
    let mut batch_txs = Vec::with_capacity(logs.len());
    let mut last_seq_num: Option<u64> = None;

    for log in &logs {
        // Only consider batches posted to the configured sequencer inbox.
        if log.inner.address != mel_state.batch_posting_target_address {
            continue;
        }
        if log.topics().first() != Some(&SequencerBatchDelivered::SIGNATURE_HASH) {
            continue;
        }

        let event = SequencerBatchDelivered::decode_log(&log.inner)
            .map_err(|source| MelError::AbiDecode {
                event: "SequencerBatchDelivered",
                source,
            })?
            .data;

        let seq_num: u64 = event
            .batchSequenceNumber
            .try_into()
            .map_err(|_| MelError::NonU64("sequence number"))?;
        let after_delayed_count: u64 = event
            .afterDelayedMessagesRead
            .try_into()
            .map_err(|_| MelError::NonU64("delayed messages read"))?;

        if let Some(last) = last_seq_num
            && seq_num != last + 1
        {
            return Err(MelError::BatchesOutOfOrder {
                after: last,
                got: seq_num,
            });
        }
        last_seq_num = Some(seq_num);

        let tx = tx_fetcher.transaction_by_log(log)?;

        batches.push(Batch {
            block_hash,
            parent_chain_block_number: parent_chain_header.number,
            sequence_number: seq_num,
            before_inbox_acc: event.beforeAcc,
            after_inbox_acc: event.afterAcc,
            after_delayed_acc: event.delayedAcc,
            after_delayed_count,
            time_bounds: event.timeBounds.into(),
            data_location: DataLocation::from_byte(event.dataLocation),
            bridge_address: log.inner.address,
            raw_log: log.clone(),
            cached_serialized: None,
        });
        batch_txs.push(tx);
    }

    Ok((batches, batch_txs))
}

#[cfg(test)]
mod tests {
    use alloy_primitives::{Address, B256, LogData, U256};
    use alloy_rpc_types_eth::Log;

    use super::*;
    use crate::test_utils::{MockLogs, MockTx, rpc_log};

    /// Parent-chain address the tests post their batches from.
    fn target() -> Address {
        Address::repeat_byte(0xAB)
    }

    /// A `MelState` configured to accept batches from [`target`].
    fn state() -> MelState {
        MelState {
            batch_posting_target_address: target(),
            ..Default::default()
        }
    }

    /// Builds a `SequencerBatchDelivered` log with the given sequence number and
    /// data-location byte, posted from [`target`].
    fn batch_delivered_log(seq: U256, data_location: u8) -> Log {
        let ev = SequencerBatchDelivered {
            batchSequenceNumber: seq,
            beforeAcc: B256::repeat_byte(0x11),
            afterAcc: B256::repeat_byte(0x22),
            delayedAcc: B256::repeat_byte(0x33),
            afterDelayedMessagesRead: U256::from(9u64),
            timeBounds: TimeBoundsAbi {
                minTimestamp: 1,
                maxTimestamp: 2,
                minBlockNumber: 3,
                maxBlockNumber: 4,
            },
            dataLocation: data_location,
        };
        rpc_log(target(), ev.encode_log_data())
    }

    #[test]
    fn parses_sequential_batches() -> MelResult<()> {
        let logs = MockLogs {
            block_logs: vec![
                batch_delivered_log(U256::from(5u64), 0),
                batch_delivered_log(U256::from(6u64), 1),
            ],
            ..Default::default()
        };
        let (batches, txs) =
            parse_batches_from_block(&state(), &Header::default(), &MockTx, &logs)?;
        assert_eq!(batches.len(), 2);
        assert_eq!(txs.len(), 2);
        assert_eq!(batches[0].sequence_number, 5);
        assert_eq!(batches[1].sequence_number, 6);
        assert_eq!(batches[0].after_delayed_count, 9);
        assert_eq!(batches[0].before_inbox_acc, B256::repeat_byte(0x11));
        assert_eq!(batches[0].after_inbox_acc, B256::repeat_byte(0x22));
        assert_eq!(batches[0].time_bounds.max_block_number, 4);
        assert_eq!(batches[0].data_location, Some(DataLocation::TxInput));
        assert_eq!(batches[1].data_location, Some(DataLocation::SeparateEvent));
        Ok(())
    }

    #[test]
    fn rejects_out_of_order_batches() {
        let logs = MockLogs {
            block_logs: vec![
                batch_delivered_log(U256::from(5u64), 0),
                batch_delivered_log(U256::from(7u64), 0),
            ],
            ..Default::default()
        };
        let result = parse_batches_from_block(&state(), &Header::default(), &MockTx, &logs);
        assert!(matches!(
            result,
            Err(MelError::BatchesOutOfOrder { after: 5, got: 7 })
        ));
    }

    #[test]
    fn ignores_logs_from_other_addresses_and_signatures() -> MelResult<()> {
        // A different signature at the target address, and a batch log from an
        // unrelated address: both are skipped.
        let wrong_sig = rpc_log(
            target(),
            LogData::new_unchecked(vec![B256::repeat_byte(0xEE)], Default::default()),
        );
        let wrong_address = {
            let mut log = batch_delivered_log(U256::from(0u64), 0);
            log.inner.address = Address::repeat_byte(0x01);
            log
        };
        let logs = MockLogs {
            block_logs: vec![
                wrong_sig,
                wrong_address,
                batch_delivered_log(U256::from(0u64), 0),
            ],
            ..Default::default()
        };
        let (batches, _) = parse_batches_from_block(&state(), &Header::default(), &MockTx, &logs)?;
        assert_eq!(batches.len(), 1);
        assert_eq!(batches[0].sequence_number, 0);
        Ok(())
    }

    #[test]
    fn rejects_non_u64_sequence_number() {
        let logs = MockLogs {
            block_logs: vec![batch_delivered_log(U256::MAX, 0)],
            ..Default::default()
        };
        let result = parse_batches_from_block(&state(), &Header::default(), &MockTx, &logs);
        assert!(matches!(result, Err(MelError::NonU64("sequence number"))));
    }

    #[test]
    fn propagates_logs_fetcher_error() {
        let logs = MockLogs {
            fail: true,
            ..Default::default()
        };
        let result = parse_batches_from_block(&state(), &Header::default(), &MockTx, &logs);
        assert!(result.is_err());
    }

    #[test]
    fn propagates_decode_error() {
        let malformed = rpc_log(
            target(),
            LogData::new_unchecked(
                vec![SequencerBatchDelivered::SIGNATURE_HASH],
                Default::default(),
            ),
        );
        let logs = MockLogs {
            block_logs: vec![malformed],
            ..Default::default()
        };
        let result = parse_batches_from_block(&state(), &Header::default(), &MockTx, &logs);
        assert!(matches!(
            result,
            Err(MelError::AbiDecode {
                event: "SequencerBatchDelivered",
                ..
            })
        ));
    }
}
