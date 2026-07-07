//! arb-mel

#![cfg_attr(not(test), warn(unused_crate_dependencies))]

use alloy_primitives::B256;
use alloy_rpc_types_eth::Log;

mod batch_lookup;
mod batch_messages;
mod delayed_message_lookup;
mod error;
mod message_extraction;
mod parse_sequencer_message;
mod serialize_batch;
#[cfg(test)]
mod test_utils;
mod types;

pub use error::MelError;
pub use message_extraction::{ExtractionOutput, extract_messages};
pub use parse_sequencer_message::SequencerMessage;
pub use types::*;

pub trait LogsFetcher {
    fn logs_for_block_hash(&self, block_hash: B256) -> Result<Vec<Log>, MelError>;
    fn logs_for_tx_index(&self, block_hash: B256, tx_index: u64) -> Result<Vec<Log>, MelError>;
}

pub trait TxFetcher {
    /// Parent-chain transaction type returned by the fetcher.
    type Transaction;

    /// Fetches the parent-chain transaction that emitted the given log.
    fn transaction_by_log(&self, log: &Log) -> MelResult<Self::Transaction>;
}

pub trait DelayedMessageDB {
    fn read_delayed_message(
        &self,
        mel_state: &MelState,
        index: u64,
    ) -> MelResult<DelayedInboxMessage>;
}
