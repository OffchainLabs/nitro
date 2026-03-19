//! Nitro execution RPC namespace (`nitroexecution_*`).
//!
//! Implements the RPC interface that the Nitro consensus layer uses to drive
//! block production on this execution client. The critical method is
//! `digestMessage`, which takes an L1 incoming message with metadata and
//! produces a block, returning the block hash and send root.

use alloy_primitives::{Address, B256, U256};
use jsonrpsee::{core::RpcResult, proc_macros::rpc};
use serde::{Deserialize, Serialize};

/// Deserializer that accepts U256 as hex string ("0x..."), decimal string ("12345"),
/// or bare JSON number (12345). The canonical `*big.Int` marshals to a bare JSON number.
#[allow(dead_code)]
mod u256_dec_or_hex {
    use alloy_primitives::U256;
    use serde::{self, Deserialize, Deserializer, Serializer};

    pub fn serialize<S>(value: &U256, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        serde::Serialize::serialize(value, serializer)
    }

    pub fn deserialize<'de, D>(deserializer: D) -> Result<U256, D::Error>
    where
        D: Deserializer<'de>,
    {
        let v = serde_json::Value::deserialize(deserializer)?;
        match v {
            serde_json::Value::Number(n) => {
                if let Some(u) = n.as_u64() {
                    Ok(U256::from(u))
                } else {
                    // Large number: parse from string representation
                    U256::from_str_radix(&n.to_string(), 10).map_err(serde::de::Error::custom)
                }
            }
            serde_json::Value::String(s) => {
                if let Some(hex) = s.strip_prefix("0x").or_else(|| s.strip_prefix("0X")) {
                    U256::from_str_radix(hex, 16).map_err(serde::de::Error::custom)
                } else {
                    U256::from_str_radix(&s, 10).map_err(serde::de::Error::custom)
                }
            }
            _ => Err(serde::de::Error::custom(
                "expected number or string for U256",
            )),
        }
    }
}

/// Optional variant of u256_dec_or_hex.
mod opt_u256_dec_or_hex {
    use alloy_primitives::U256;
    use serde::{self, Deserialize, Deserializer, Serializer};

    pub fn serialize<S>(value: &Option<U256>, serializer: S) -> Result<S::Ok, S::Error>
    where
        S: Serializer,
    {
        match value {
            Some(v) => serde::Serialize::serialize(v, serializer),
            None => serializer.serialize_none(),
        }
    }

    pub fn deserialize<'de, D>(deserializer: D) -> Result<Option<U256>, D::Error>
    where
        D: Deserializer<'de>,
    {
        let v = serde_json::Value::deserialize(deserializer)?;
        match v {
            serde_json::Value::Null => Ok(None),
            serde_json::Value::Number(n) => {
                if let Some(u) = n.as_u64() {
                    Ok(Some(U256::from(u)))
                } else {
                    let val = U256::from_str_radix(&n.to_string(), 10)
                        .map_err(serde::de::Error::custom)?;
                    Ok(Some(val))
                }
            }
            serde_json::Value::String(s) if s.is_empty() => Ok(None),
            serde_json::Value::String(s) => {
                let val = if let Some(hex) = s.strip_prefix("0x").or_else(|| s.strip_prefix("0X")) {
                    U256::from_str_radix(hex, 16).map_err(serde::de::Error::custom)?
                } else {
                    U256::from_str_radix(&s, 10).map_err(serde::de::Error::custom)?
                };
                Ok(Some(val))
            }
            _ => Err(serde::de::Error::custom(
                "expected number, string, or null for U256",
            )),
        }
    }
}

// ---------------------------------------------------------------------------
// RPC data types (JSON-serializable, matching the canonical JSON tags)
// ---------------------------------------------------------------------------

/// L1 incoming message header.
/// Fields have explicit JSON tags (camelCase).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RpcL1IncomingMessageHeader {
    pub kind: u8,
    pub sender: Address,
    #[serde(rename = "blockNumber")]
    pub block_number: u64,
    pub timestamp: u64,
    #[serde(default, skip_serializing_if = "Option::is_none", rename = "requestId")]
    pub request_id: Option<B256>,
    #[serde(
        default,
        skip_serializing_if = "Option::is_none",
        rename = "baseFeeL1",
        with = "opt_u256_dec_or_hex"
    )]
    pub base_fee_l1: Option<U256>,
}

/// Batch data statistics for L1 cost estimation.
/// Fields have explicit JSON tags (lowercase).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RpcBatchDataStats {
    pub length: u64,
    pub nonzeros: u64,
}

/// L1 incoming message containing header and L2 payload.
/// Fields have explicit JSON tags (camelCase).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RpcL1IncomingMessage {
    pub header: RpcL1IncomingMessageHeader,
    /// Base64-encoded L2 message bytes.
    #[serde(default, skip_serializing_if = "Option::is_none", rename = "l2Msg")]
    pub l2_msg: Option<String>,
    /// Legacy batch gas cost (for older batch posting reports).
    #[serde(
        default,
        skip_serializing_if = "Option::is_none",
        rename = "batchGasCost"
    )]
    pub batch_gas_cost: Option<u64>,
    /// Batch data statistics (for newer batch posting reports).
    #[serde(
        default,
        skip_serializing_if = "Option::is_none",
        rename = "batchDataTokens"
    )]
    pub batch_data_tokens: Option<RpcBatchDataStats>,
}

/// Message with metadata, sent by the consensus layer to the execution client.
/// Fields have explicit JSON tags (camelCase).
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RpcMessageWithMetadata {
    pub message: RpcL1IncomingMessage,
    #[serde(rename = "delayedMessagesRead")]
    pub delayed_messages_read: u64,
}

/// Extended message info including block hash and metadata.
/// Uses PascalCase JSON serialization (no explicit JSON tags).
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
/// Uses PascalCase JSON serialization (no explicit JSON tags).
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct RpcMessageResult {
    pub block_hash: B256,
    pub send_root: B256,
}

/// Finality data pushed from consensus.
/// Uses PascalCase JSON serialization (no explicit JSON tags).
#[derive(Debug, Clone, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct RpcFinalityData {
    #[serde(default)]
    pub msg_idx: u64,
    #[serde(default)]
    pub block_hash: B256,
}

/// Consensus sync data pushed from consensus.
/// Uses PascalCase JSON serialization (no explicit JSON tags).
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
/// Uses PascalCase JSON serialization (no explicit JSON tags).
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
    fn set_consensus_sync_data(&self, sync_data: RpcConsensusSyncData) -> RpcResult<()>;

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
