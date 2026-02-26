//! Arbitrum payload builder types.
//!
//! Defines the payload attributes and built payload types used
//! by the engine API and block construction pipeline.

use alloy_primitives::{Address, Bytes, B256, U256};
use alloy_rpc_types_engine::PayloadAttributes;
use serde::{Deserialize, Serialize};

/// Arbitrum-specific payload attributes extending the standard engine API.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ArbPayloadAttributes {
    /// Standard Ethereum payload attributes.
    #[serde(flatten)]
    pub inner: PayloadAttributes,
    /// Sequencer message to be processed in this block.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub transactions: Option<Vec<Bytes>>,
    /// Whether the node is running as a sequencer.
    #[serde(default)]
    pub no_tx_pool: bool,
}

impl ArbPayloadAttributes {
    /// Returns the timestamp for this payload.
    pub fn timestamp(&self) -> u64 {
        self.inner.timestamp
    }

    /// Returns the parent beacon block root, if set.
    pub fn parent_beacon_block_root(&self) -> Option<B256> {
        self.inner.parent_beacon_block_root
    }

    /// Returns the suggested fee recipient.
    pub fn suggested_fee_recipient(&self) -> Address {
        self.inner.suggested_fee_recipient
    }

    /// Returns the previous randao value.
    pub fn prev_randao(&self) -> B256 {
        self.inner.prev_randao
    }
}

/// A built Arbitrum payload ready to be sealed.
#[derive(Debug, Clone)]
pub struct ArbBuiltPayload {
    /// The block hash.
    pub block_hash: B256,
    /// The total fees collected.
    pub fees: U256,
}

impl ArbBuiltPayload {
    /// Create a new built payload.
    pub fn new(block_hash: B256, fees: U256) -> Self {
        Self { block_hash, fees }
    }
}
