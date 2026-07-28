//! arb-mel

#![cfg_attr(not(test), warn(unused_crate_dependencies))]

use alloy_primitives::B256;
use alloy_rpc_types_eth::Log;

mod batch_lookup;
mod batch_messages;
mod delayed_message_lookup;
mod error;
mod mel_config_lookup;
mod message_extraction;
mod parse_sequencer_message;
mod serialize_batch;
mod state;
#[cfg(test)]
mod test_utils;
mod types;

// Parent-chain event types, re-exported so downstream crates (the runner's log
// prefetcher) build filters from the same `SIGNATURE_HASH` the extractor matches.
pub use arb_mel_types::{BatchMetadata, DelayedInboxMessage, MelState};
pub use batch_lookup::SequencerBatchDelivered;
pub use delayed_message_lookup::{
    InboxMessageDelivered, InboxMessageDeliveredFromOrigin, MessageDelivered,
};
pub use error::MelError;
pub use mel_config_lookup::{MELConfigSet, MelConfig};
pub use message_extraction::{ExtractionOutput, extract_messages};
pub use parse_sequencer_message::SequencerMessage;
pub use serialize_batch::SequencerBatchData;
pub use state::*;
pub use types::*;

pub trait LogsFetcher {
    fn logs_for_block_hash(&self, block_hash: B256) -> Result<Vec<Log>, MelError>;
    fn logs_for_tx_index(&self, block_hash: B256, tx_index: u64) -> Result<Vec<Log>, MelError>;
}

#[async_trait::async_trait]
pub trait TxFetcher {
    /// Parent-chain transaction type returned by the fetcher.
    type Transaction;

    /// Fetches the parent-chain transaction that emitted the given log.
    async fn transaction_by_log(&self, log: &Log) -> MelResult<Self::Transaction>;
}

pub trait DelayedMessageDB {
    fn read_delayed_message(
        &self,
        mel_state: &MelState,
        index: u64,
    ) -> MelResult<Option<DelayedInboxMessage>>;
}
