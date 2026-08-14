//! Supporting MEL types owned by the runner.
//!
//! The core MEL types (`MelState`, `DelayedInboxMessage`, `BatchMetadata`) live in
//! the `arb-mel` crate and are re-exported from the crate root. This module holds
//! the runner-only types that `arb-mel` doesn't provide.

use alloy_primitives::Address;

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
