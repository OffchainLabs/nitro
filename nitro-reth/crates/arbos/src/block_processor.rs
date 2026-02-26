use alloy_primitives::{Address, B256, U256};

use crate::header::ArbHeaderInfo;
use crate::internal_tx::L1Info;
use crate::l2_pricing::GETH_BLOCK_GAS_LIMIT;

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

/// Parameters for creating a new block header.
#[derive(Debug, Clone)]
pub struct NewHeaderParams {
    pub parent_hash: B256,
    pub parent_number: u64,
    pub parent_timestamp: u64,
    pub parent_extra_data: Vec<u8>,
    pub parent_mix_hash: B256,
    pub coinbase: Address,
    pub timestamp: u64,
    pub base_fee: U256,
}

/// Computed header fields from `create_new_header`.
#[derive(Debug, Clone)]
pub struct NewHeaderResult {
    pub parent_hash: B256,
    pub coinbase: Address,
    pub number: u64,
    pub gas_limit: u64,
    pub timestamp: u64,
    pub extra_data: Vec<u8>,
    pub mix_hash: B256,
    pub base_fee: U256,
    pub difficulty: U256,
}

/// Create new header fields for an Arbitrum block.
///
/// Mirrors the Go `createNewHeader` function. In reth, the actual header
/// construction is done by the block builder; this computes the Arbitrum-specific fields.
pub fn create_new_header(
    l1_info: Option<&L1Info>,
    prev_hash: B256,
    prev_number: u64,
    prev_timestamp: u64,
    prev_extra: &[u8],
    prev_mix_hash: B256,
    base_fee: U256,
) -> NewHeaderResult {
    let mut timestamp = 0u64;
    let mut coinbase = Address::ZERO;

    if let Some(info) = l1_info {
        timestamp = info.l1_timestamp;
        coinbase = info.poster;
    }

    // Timestamp must be non-decreasing
    if timestamp < prev_timestamp {
        timestamp = prev_timestamp;
    }

    // Carry over extra data and mix hash from previous block
    let mut extra_data = vec![0u8; 32];
    let copy_len = prev_extra.len().min(32);
    extra_data[..copy_len].copy_from_slice(&prev_extra[..copy_len]);

    NewHeaderResult {
        parent_hash: prev_hash,
        coinbase,
        number: prev_number + 1,
        gas_limit: GETH_BLOCK_GAS_LIMIT,
        timestamp,
        extra_data,
        mix_hash: prev_mix_hash,
        base_fee,
        difficulty: U256::from(1),
    }
}

/// Compute the Arbitrum header info to finalize a block.
///
/// In Go this is `FinalizeBlock` which sets header fields from ArbOS state.
/// In reth, we derive the info and let the block assembler apply it.
pub fn finalize_block_header_info(
    send_root: B256,
    send_count: u64,
    l1_block_number: u64,
    arbos_version: u64,
) -> ArbHeaderInfo {
    ArbHeaderInfo {
        send_root,
        send_count,
        l1_block_number,
        arbos_format_version: arbos_version,
    }
}
