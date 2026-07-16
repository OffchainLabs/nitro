//! Serializing a `SequencerInboxBatch` into the canonical batch bytes.
use alloy_consensus::Transaction;
use alloy_primitives::B256;
use alloy_rpc_types_eth::Log;
use alloy_sol_types::{SolCall, SolEvent, sol};

use crate::{Batch, DataLocation, LogsFetcher, MelError, MelResult};

/// DA header flag marking a blob-hashes batch payload (`daprovider.BlobHashesHeaderFlag`).
const BLOB_HASHES_HEADER_FLAG: u8 = 0x50;

/// Length of a Solidity ABI function selector (the leading 4 bytes of calldata).
const SELECTOR_LEN: usize = 4;

sol! {
    #[allow(missing_docs)]
    function addSequencerL2BatchFromOrigin(
        uint256 sequenceNumber,
        bytes data,
        uint256 afterDelayedMessagesRead,
        address gasRefunder
    ) external;

    #[allow(missing_docs)]
    #[derive(Debug)]
    event SequencerBatchData(uint256 indexed batchSequenceNumber, bytes data);
}

/// Serializes `batch` into its canonical byte form, caching the result.
#[allow(dead_code)] // Wired into MEL block processing in a later change.
pub(crate) fn serialize_batch<L, T>(
    batch: &mut Batch,
    tx: &T,
    logs_fetcher: &L,
) -> MelResult<Vec<u8>>
where
    L: LogsFetcher,
    T: Transaction,
{
    if let Some(serialized) = &batch.cached_serialized {
        return Ok(serialized.clone());
    }

    let header_vals = [
        batch.time_bounds.min_timestamp,
        batch.time_bounds.max_timestamp,
        batch.time_bounds.min_block_number,
        batch.time_bounds.max_block_number,
        batch.after_delayed_count,
    ];
    let mut full_data = Vec::new();
    for bound in header_vals {
        full_data.extend_from_slice(&bound.to_be_bytes());
    }

    // Append the batch data itself.
    let data = get_sequencer_batch_data(batch, tx, logs_fetcher)?;
    full_data.extend_from_slice(&data);

    batch.cached_serialized = Some(full_data.clone());
    Ok(full_data)
}

/// Fetches the raw batch data referenced by a batch, following its data location.
fn get_sequencer_batch_data<L, T>(batch: &Batch, tx: &T, logs_fetcher: &L) -> MelResult<Vec<u8>>
where
    L: LogsFetcher,
    T: Transaction,
{
    match batch.data_location {
        Some(DataLocation::TxInput) => {
            let data = tx.input();
            if data.len() < SELECTOR_LEN {
                return Err(MelError::TxDataTooShort);
            }
            let call = addSequencerL2BatchFromOriginCall::abi_decode_raw(&data[SELECTOR_LEN..])
                .map_err(|source| MelError::AbiDecode {
                    event: "addSequencerL2BatchFromOrigin",
                    source,
                })?;
            Ok(call.data.to_vec())
        }
        Some(DataLocation::SeparateEvent) => {
            // We want the last 8 bytes of a 32-byte hash to hold the sequence
            // number, matching the indexed `batchSequenceNumber` topic.
            let mut number_as_hash = [0u8; 32];
            number_as_hash[24..].copy_from_slice(&batch.sequence_number.to_be_bytes());
            let number_as_hash = B256::from(number_as_hash);

            let logs = logs_fetcher.logs_for_tx_index(
                batch.block_hash,
                batch
                    .raw_log
                    .transaction_index
                    .ok_or(MelError::SequencerBatchData(
                        "raw log missing transaction index",
                    ))?,
            )?;
            if logs.is_empty() {
                return Err(MelError::SequencerBatchData(
                    "no logs found in transaction receipt",
                ));
            }

            let matching: Vec<&Log> = logs
                .iter()
                .filter(|l| {
                    l.inner.address == batch.bridge_address
                        && l.topics().first() == Some(&SequencerBatchData::SIGNATURE_HASH)
                        && l.topics().get(1) == Some(&number_as_hash)
                })
                .collect();
            if matching.is_empty() {
                return Err(MelError::SequencerBatchData(
                    "expected to find sequencer batch data",
                ));
            }
            if matching.len() > 1 {
                return Err(MelError::SequencerBatchData(
                    "expected to find only one matching sequencer batch data",
                ));
            }

            let event = SequencerBatchData::decode_log(&matching[0].inner)
                .map_err(|source| MelError::AbiDecode {
                    event: "SequencerBatchData",
                    source,
                })?
                .data;
            Ok(event.data.to_vec())
        }
        Some(DataLocation::BlobHashes) => {
            let hashes = tx.blob_versioned_hashes().unwrap_or(&[]);
            if hashes.is_empty() {
                return Err(MelError::SequencerBatchData(
                    "blob batch transaction has no blobs",
                ));
            }
            let mut data = Vec::with_capacity(1 + hashes.len() * 32);
            data.push(BLOB_HASHES_HEADER_FLAG);
            for h in hashes {
                data.extend_from_slice(h.as_slice());
            }
            Ok(data)
        }
        // No data when in a force-inclusion batch (no recognised data location).
        None => Ok(Vec::new()),
    }
}

#[cfg(test)]
mod tests {
    use alloy_consensus::{TxEip4844, TxLegacy};
    use alloy_primitives::{Address, LogData, U256};

    use super::*;
    use crate::{
        TimeBounds,
        test_utils::{MockLogs, rpc_log},
    };

    /// The 40-byte header the serializer prepends, for the fixed time bounds and
    /// delayed count used by [`make_batch`].
    fn expected_header() -> Vec<u8> {
        let mut header = Vec::new();
        for v in [1u64, 2, 3, 4, 5] {
            header.extend_from_slice(&v.to_be_bytes());
        }
        header
    }

    fn make_batch(data_location: Option<DataLocation>, seq: u64) -> Batch {
        Batch {
            block_hash: B256::ZERO,
            parent_chain_block_number: 0,
            sequence_number: seq,
            before_inbox_acc: B256::ZERO,
            after_inbox_acc: B256::ZERO,
            after_delayed_acc: B256::ZERO,
            after_delayed_count: 5,
            time_bounds: TimeBounds {
                min_timestamp: 1,
                max_timestamp: 2,
                min_block_number: 3,
                max_block_number: 4,
            },
            data_location,
            bridge_address: Address::ZERO,
            raw_log: rpc_log(Address::ZERO, LogData::default()),
            cached_serialized: None,
        }
    }

    #[test]
    fn serializes_tx_input_batch() -> MelResult<()> {
        let payload = b"batchdata".to_vec();
        let call = addSequencerL2BatchFromOriginCall {
            sequenceNumber: U256::ZERO,
            data: payload.clone().into(),
            afterDelayedMessagesRead: U256::ZERO,
            gasRefunder: Address::ZERO,
        };
        let tx = TxLegacy {
            input: call.abi_encode().into(),
            ..Default::default()
        };
        let mut batch = make_batch(Some(DataLocation::TxInput), 0);
        let out = serialize_batch(&mut batch, &tx, &MockLogs::default())?;
        assert_eq!(&out[..40], expected_header().as_slice());
        assert_eq!(&out[40..], payload.as_slice());
        // The result is cached on the batch.
        assert_eq!(batch.cached_serialized.as_deref(), Some(out.as_slice()));
        Ok(())
    }

    #[test]
    fn serializes_blob_hashes_batch() -> MelResult<()> {
        let h1 = B256::repeat_byte(0xA1);
        let h2 = B256::repeat_byte(0xB2);
        let tx = TxEip4844 {
            blob_versioned_hashes: vec![h1, h2],
            ..Default::default()
        };
        let mut batch = make_batch(Some(DataLocation::BlobHashes), 0);
        let out = serialize_batch(&mut batch, &tx, &MockLogs::default())?;
        let mut expected_data = vec![0x50u8];
        expected_data.extend_from_slice(h1.as_slice());
        expected_data.extend_from_slice(h2.as_slice());
        assert_eq!(&out[..40], expected_header().as_slice());
        assert_eq!(&out[40..], expected_data.as_slice());
        Ok(())
    }

    #[test]
    fn serializes_separate_event_batch() -> MelResult<()> {
        let payload = b"eventdata".to_vec();
        let seq = 3u64;
        let bridge = Address::repeat_byte(0xCC);
        let ev = SequencerBatchData {
            batchSequenceNumber: U256::from(seq),
            data: payload.clone().into(),
        };
        let logs = MockLogs {
            tx_logs: vec![rpc_log(bridge, ev.encode_log_data())],
            ..Default::default()
        };
        let mut batch = make_batch(Some(DataLocation::SeparateEvent), seq);
        batch.bridge_address = bridge;
        batch.raw_log = rpc_log(bridge, LogData::default());
        batch.raw_log.transaction_index = Some(0);
        let out = serialize_batch(&mut batch, &TxLegacy::default(), &logs)?;
        assert_eq!(&out[..40], expected_header().as_slice());
        assert_eq!(&out[40..], payload.as_slice());
        Ok(())
    }

    #[test]
    fn serializes_force_inclusion_batch_with_no_data() -> MelResult<()> {
        let mut batch = make_batch(None, 0);
        let out = serialize_batch(&mut batch, &TxLegacy::default(), &MockLogs::default())?;
        assert_eq!(out, expected_header());
        Ok(())
    }

    #[test]
    fn returns_cached_serialization_without_recomputing() -> MelResult<()> {
        let mut batch = make_batch(None, 0);
        batch.cached_serialized = Some(vec![9, 9, 9]);
        let out = serialize_batch(&mut batch, &TxLegacy::default(), &MockLogs::default())?;
        assert_eq!(out, vec![9, 9, 9]);
        Ok(())
    }

    #[test]
    fn rejects_tx_input_shorter_than_selector() {
        let tx = TxLegacy {
            input: vec![1u8, 2].into(),
            ..Default::default()
        };
        let mut batch = make_batch(Some(DataLocation::TxInput), 0);
        let result = serialize_batch(&mut batch, &tx, &MockLogs::default());
        assert!(matches!(result, Err(MelError::TxDataTooShort)));
    }
}
