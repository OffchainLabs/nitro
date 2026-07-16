//! Shared helpers for the crate's unit tests.

use alloy_consensus::TxLegacy;
use alloy_primitives::{Address, B256, LogData};
use alloy_rpc_types_eth::Log;

use crate::{LogsFetcher, MelResult, TxFetcher};

/// Wraps `data` in an rpc [`Log`] emitted by `address`.
pub(crate) fn rpc_log(address: Address, data: LogData) -> Log {
    Log {
        inner: alloy_primitives::Log { address, data },
        ..Default::default()
    }
}

/// A [`LogsFetcher`] that returns canned logs for block-hash and tx-index lookups.
#[derive(Default)]
pub(crate) struct MockLogs {
    pub block_logs: Vec<Log>,
    pub tx_logs: Vec<Log>,
}

impl LogsFetcher for MockLogs {
    fn logs_for_block_hash(&self, _block_hash: B256) -> MelResult<Vec<Log>> {
        Ok(self.block_logs.clone())
    }
    fn logs_for_tx_index(&self, _block_hash: B256, _tx_index: u64) -> MelResult<Vec<Log>> {
        Ok(self.tx_logs.clone())
    }
}

/// A [`TxFetcher`] yielding a default [`TxLegacy`] for any log.
pub(crate) struct MockTx;

impl TxFetcher for MockTx {
    type Transaction = TxLegacy;
    fn transaction_by_log(&self, _log: &Log) -> MelResult<TxLegacy> {
        Ok(TxLegacy::default())
    }
}
