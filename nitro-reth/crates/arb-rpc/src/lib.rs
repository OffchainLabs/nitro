//! Arbitrum-specific RPC types and extensions.

use alloy_primitives::{B256, U256};
use serde::{Deserialize, Serialize};

/// Arbitrum transaction receipt extension fields.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ArbReceiptFields {
    /// L1 block number when the L2 tx was batched.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub l1_block_number: Option<u64>,
    /// Gas units charged for L1 calldata.
    #[serde(skip_serializing_if = "Option::is_none")]
    pub gas_used_for_l1: Option<U256>,
}

/// Arbitrum block information returned by `arb_getBlockInfo`.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct ArbBlockInfo {
    /// The L1 block number that this L2 block was batched on.
    pub l1_block_number: u64,
    /// ArbOS format version of this block.
    pub arbos_format_version: u64,
    /// The send count (L2-to-L1 messages) as of this block.
    pub send_count: u64,
    /// The send root hash.
    pub send_root: B256,
}

/// Maintenance status for the node.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct ArbMaintenanceStatus {
    pub status: String,
}
