use alloy_primitives::{Address, B256};

use crate::internal_tx::L1Info;

/// Conditional options that may be attached to a transaction.
#[derive(Debug, Clone, Default)]
pub struct ConditionalOptions {
    pub known_accounts: Vec<(Address, Option<B256>)>,
    pub block_number_min: Option<u64>,
    pub block_number_max: Option<u64>,
    pub timestamp_min: Option<u64>,
    pub timestamp_max: Option<u64>,
}

/// Hooks for the sequencer to control block production.
pub trait SequencingHooks {
    /// Returns the next transaction to include, or None if the block is complete.
    fn next_tx_to_sequence(&mut self) -> Option<Vec<u8>>;

    /// Filters a transaction before execution.
    fn pre_tx_filter(&self, tx: &[u8]) -> Result<(), String>;

    /// Filters a transaction after execution.
    fn post_tx_filter(&self, tx: &[u8], result: &[u8]) -> Result<(), String>;

    /// Determines whether to discard invalid txs early.
    fn discard_invalid_txs_early(&self) -> bool;

    /// Block-level filter.
    fn block_filter(&self) -> Result<(), String> {
        Ok(())
    }

    /// Inserts the error for the last tx.
    fn insert_last_tx_error(&mut self, _err: String) {}
}

/// Default no-op implementation for sequencing hooks.
pub struct NoopSequencingHooks;

impl SequencingHooks for NoopSequencingHooks {
    fn next_tx_to_sequence(&mut self) -> Option<Vec<u8>> {
        None
    }

    fn pre_tx_filter(&self, _tx: &[u8]) -> Result<(), String> {
        Ok(())
    }

    fn post_tx_filter(&self, _tx: &[u8], _result: &[u8]) -> Result<(), String> {
        Ok(())
    }

    fn discard_invalid_txs_early(&self) -> bool {
        false
    }
}

/// The result of block production.
#[derive(Debug, Clone)]
pub struct BlockProductionResult {
    pub l1_info: L1Info,
    pub num_txs: usize,
    pub gas_used: u64,
}
