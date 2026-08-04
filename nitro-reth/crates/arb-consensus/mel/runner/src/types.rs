//! Supporting MEL types owned by the runner.
//!
//! The core MEL types (`MelState`, `DelayedInboxMessage`, `BatchMetadata`) live in
//! the `arb-mel` crate and are re-exported from the crate root. This module holds
//! the runner-only types that `arb-mel` doesn't provide.

use alloy_primitives::Address;

/// Progress of message synchronization.
///
/// Mirrors `mel.MessageSyncProgress`.
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct MessageSyncProgress {
    /// Highest batch count seen on the parent chain.
    pub batch_seen: u64,
    /// Batch count processed into state.
    pub batch_processed: u64,
    /// L2 messages produced.
    pub msg_count: u64,
}

/// On-chain rollup contract addresses the extractor needs.
///
/// A trimmed port of `chaininfo.RollupAddresses` (only the fields MEL uses).
#[derive(Debug, Clone, Default, PartialEq, Eq)]
pub struct RollupAddresses {
    /// The bridge contract address.
    pub bridge: Address,
    /// The delayed-inbox contract address.
    pub inbox: Address,
    /// The sequencer-inbox contract address.
    pub sequencer_inbox: Address,
    /// The rollup contract address.
    pub rollup: Address,
}
