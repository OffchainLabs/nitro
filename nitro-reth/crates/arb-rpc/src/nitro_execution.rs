//! Nitro execution RPC namespace (`nitroexecution_*`).
//!
//! Implements the RPC interface that the Nitro consensus layer uses to drive
//! block production on this execution client. The critical method is
//! `digestMessage`, which takes an L1 incoming message with metadata and
//! produces a block, returning the block hash and send root.

use alloy_primitives::{Address, B256, U256};
use jsonrpsee::{core::RpcResult, proc_macros::rpc};
use serde::{Deserialize, Serialize};

// ---------------------------------------------------------------------------
// RPC data types (JSON-serializable, matching Go's JSON tags)
// ---------------------------------------------------------------------------

/// L1 incoming message header.
/// Go fields have explicit JSON tags (camelCase).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RpcL1IncomingMessageHeader {
    pub kind: u8,
    pub sender: Address,
    #[serde(rename = "blockNumber")]
    pub block_number: u64,
    pub timestamp: u64,
    #[serde(default, skip_serializing_if = "Option::is_none", rename = "requestId")]
    pub request_id: Option<B256>,
    #[serde(default, skip_serializing_if = "Option::is_none", rename = "baseFeeL1")]
    pub base_fee_l1: Option<U256>,
}

/// Batch data statistics for L1 cost estimation.
/// Go fields have explicit JSON tags (lowercase).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RpcBatchDataStats {
    pub length: u64,
    pub nonzeros: u64,
}

/// L1 incoming message containing header and L2 payload.
/// Go fields have explicit JSON tags (camelCase).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RpcL1IncomingMessage {
    pub header: RpcL1IncomingMessageHeader,
    /// Base64-encoded L2 message bytes.
    #[serde(default, skip_serializing_if = "Option::is_none", rename = "l2Msg")]
    pub l2_msg: Option<String>,
    /// Legacy batch gas cost (for older batch posting reports).
    #[serde(default, skip_serializing_if = "Option::is_none", rename = "batchGasCost")]
    pub batch_gas_cost: Option<u64>,
    /// Batch data statistics (for newer batch posting reports).
    #[serde(default, skip_serializing_if = "Option::is_none", rename = "batchDataTokens")]
    pub batch_data_tokens: Option<RpcBatchDataStats>,
}

/// Message with metadata, sent by Nitro consensus to the execution client.
/// Go fields have explicit JSON tags (camelCase).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RpcMessageWithMetadata {
    pub message: RpcL1IncomingMessage,
    #[serde(rename = "delayedMessagesRead")]
    pub delayed_messages_read: u64,
}

/// Extended message info including block hash and metadata.
/// Go struct has NO JSON tags, uses PascalCase.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct RpcMessageWithMetadataAndBlockInfo {
    #[serde(rename = "MessageWithMeta")]
    pub message: RpcMessageWithMetadata,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub block_hash: Option<B256>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub block_metadata: Option<Vec<u8>>,
}

/// Result of block production: block hash and send root.
/// Go struct has NO JSON tags, uses PascalCase.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct RpcMessageResult {
    pub block_hash: B256,
    pub send_root: B256,
}

/// Finality data pushed from consensus.
/// Go struct has NO JSON tags, uses PascalCase.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct RpcFinalityData {
    #[serde(default)]
    pub msg_idx: u64,
    #[serde(default)]
    pub block_hash: B256,
}

/// Consensus sync data pushed from consensus.
/// Go struct has NO JSON tags, uses PascalCase.
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct RpcConsensusSyncData {
    pub synced: bool,
    pub max_message_count: u64,
    #[serde(default)]
    pub sync_progress_map: Option<serde_json::Value>,
    #[serde(default)]
    pub updated_at: Option<String>,
}

/// Maintenance status.
/// Go struct has NO JSON tags, uses PascalCase.
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct RpcMaintenanceStatus {
    pub is_running: bool,
}

// ---------------------------------------------------------------------------
// RPC trait definition
// ---------------------------------------------------------------------------

/// Nitro execution RPC namespace.
///
/// This is the interface that Nitro's consensus layer calls to drive
/// block production on the execution client.
#[rpc(server, namespace = "nitroexecution")]
pub trait NitroExecutionApi {
    /// Process a message and produce a block.
    #[method(name = "digestMessage")]
    async fn digest_message(
        &self,
        msg_idx: u64,
        message: RpcMessageWithMetadata,
        message_for_prefetch: Option<RpcMessageWithMetadata>,
    ) -> RpcResult<RpcMessageResult>;

    /// Handle a chain reorg by rolling back and replaying messages.
    #[method(name = "reorg")]
    async fn reorg(
        &self,
        msg_idx_of_first_msg_to_add: u64,
        new_messages: Vec<RpcMessageWithMetadataAndBlockInfo>,
        old_messages: Vec<RpcMessageWithMetadata>,
    ) -> RpcResult<Vec<RpcMessageResult>>;

    /// Returns the current head message index.
    #[method(name = "headMessageIndex")]
    async fn head_message_index(&self) -> RpcResult<u64>;

    /// Returns the block hash and send root for a given message index.
    #[method(name = "resultAtMessageIndex")]
    async fn result_at_message_index(&self, msg_idx: u64) -> RpcResult<RpcMessageResult>;

    /// Updates finality information.
    #[method(name = "setFinalityData")]
    fn set_finality_data(
        &self,
        safe: Option<RpcFinalityData>,
        finalized: Option<RpcFinalityData>,
        validated: Option<RpcFinalityData>,
    ) -> RpcResult<()>;

    /// Updates consensus sync data.
    #[method(name = "setConsensusSyncData")]
    fn set_consensus_sync_data(
        &self,
        sync_data: RpcConsensusSyncData,
    ) -> RpcResult<()>;

    /// Marks the feed start position.
    #[method(name = "markFeedStart")]
    fn mark_feed_start(&self, to: u64) -> RpcResult<()>;

    /// Triggers maintenance operations.
    #[method(name = "triggerMaintenance")]
    async fn trigger_maintenance(&self) -> RpcResult<()>;

    /// Checks if maintenance should be triggered.
    #[method(name = "shouldTriggerMaintenance")]
    async fn should_trigger_maintenance(&self) -> RpcResult<bool>;

    /// Returns current maintenance status.
    #[method(name = "maintenanceStatus")]
    async fn maintenance_status(&self) -> RpcResult<RpcMaintenanceStatus>;

    /// Returns the ArbOS version for a given message index.
    #[method(name = "arbOSVersionForMessageIndex")]
    async fn arbos_version_for_message_index(&self, msg_idx: u64) -> RpcResult<u64>;
}
