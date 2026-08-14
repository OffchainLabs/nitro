//! Shared helpers for the crate's unit tests.

use std::collections::BTreeMap;

use alloy_consensus::TxLegacy;
use alloy_primitives::{Address, B256, LogData};
use alloy_rpc_types_eth::Log;

use crate::{
    DelayedInboxMessage, DelayedMessageDB, LogsFetcher, MelError, MelResult, MelState,
    SequencerMessage, TxFetcher,
};

/// Wraps `data` in an rpc [`Log`] emitted by `address`.
pub(crate) fn rpc_log(address: Address, data: LogData) -> Log {
    Log {
        inner: alloy_primitives::Log { address, data },
        ..Default::default()
    }
}

pub(crate) fn brotli_compress(raw: &[u8]) -> Vec<u8> {
    nitro_brotli::compress(raw, 9, 22, nitro_brotli::Dictionary::Empty)
        .expect("brotli fixture compress")
}

pub(crate) fn sequencer_message_with_segments(
    after_delayed_messages: u64,
    segments: Vec<Vec<u8>>,
) -> SequencerMessage {
    SequencerMessage {
        min_timestamp: 0,
        max_timestamp: 0,
        min_l1_block: 0,
        max_l1_block: 0,
        after_delayed_messages,
        segments,
    }
}

pub(crate) fn sequencer_message_with_timestamp_range(
    segments: Vec<Vec<u8>>,
    min_timestamp: u64,
    max_timestamp: u64,
) -> SequencerMessage {
    SequencerMessage {
        min_timestamp,
        max_timestamp,
        min_l1_block: 0,
        max_l1_block: 0,
        after_delayed_messages: 0,
        segments,
    }
}

/// A [`LogsFetcher`] that returns canned logs for block-hash and tx-index lookups.
#[derive(Default)]
pub(crate) struct MockLogs {
    pub block_logs: Vec<Log>,
    pub tx_logs: Vec<Log>,
    pub fail: bool,
}

impl LogsFetcher for MockLogs {
    fn logs_for_block_hash(&self, _block_hash: B256) -> MelResult<Vec<Log>> {
        if self.fail {
            return Err(MelError::Unknown);
        }
        Ok(self.block_logs.clone())
    }
    fn logs_for_tx_index(&self, _block_hash: B256, _tx_index: u64) -> MelResult<Vec<Log>> {
        if self.fail {
            return Err(MelError::Unknown);
        }
        Ok(self.tx_logs.clone())
    }
}

/// A [`TxFetcher`] yielding a default [`TxLegacy`] for any log.
pub(crate) struct MockTx;

#[async_trait::async_trait]
impl TxFetcher for MockTx {
    type Transaction = TxLegacy;
    async fn transaction_by_log(&self, _log: &Log) -> MelResult<TxLegacy> {
        Ok(TxLegacy::default())
    }
}

#[derive(Default)]
pub(crate) struct MockDelayedDb {
    pub messages: BTreeMap<u64, DelayedInboxMessage>,
    pub fail: bool,
}

impl MockDelayedDb {
    pub fn with_messages(entries: impl IntoIterator<Item = (u64, DelayedInboxMessage)>) -> Self {
        Self {
            messages: entries.into_iter().collect(),
            fail: false,
        }
    }

    pub fn failing() -> Self {
        Self {
            messages: BTreeMap::new(),
            fail: true,
        }
    }
}

impl DelayedMessageDB for MockDelayedDb {
    fn read_delayed_message(
        &self,
        _state: &MelState,
        index: u64,
    ) -> MelResult<Option<DelayedInboxMessage>> {
        if self.fail {
            return Err(MelError::Unknown);
        }
        Ok(self.messages.get(&index).cloned())
    }
}
