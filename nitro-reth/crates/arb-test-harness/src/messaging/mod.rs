//! L1 message builders.
//!
//! Each public type (`DepositBuilder`, `RetryableBuilder`, etc.)
//! produces an [`L1Message`] that the node trait submits via
//! `nitroexecution_digestMessage`. All bytes match the on-the-wire
//! shape Nitro expects; concrete encodings are filled in by Agent B.

pub mod batch;
pub mod deposit;
pub mod kinds;
pub mod retryable;
pub mod signed_tx;

use alloy_primitives::Bytes;
use serde::{Deserialize, Serialize};

/// An incoming L1 message ready to be `digestMessage`'d. Mirrors the
/// JSON shape Nitro accepts: `{ message: { header, l2Msg }, ... }`.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct L1Message {
    pub header: L1MessageHeader,
    /// Base64-encoded L2 message body.
    #[serde(rename = "l2Msg")]
    pub l2_msg: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct L1MessageHeader {
    pub kind: u8,
    pub sender: alloy_primitives::Address,
    #[serde(rename = "blockNumber")]
    pub block_number: u64,
    pub timestamp: u64,
    #[serde(rename = "requestId", skip_serializing_if = "Option::is_none")]
    pub request_id: Option<alloy_primitives::B256>,
    #[serde(rename = "baseFeeL1")]
    pub base_fee_l1: u64,
}

/// Common builder trait. Each concrete builder implements this.
pub trait MessageBuilder {
    fn build(&self) -> crate::Result<L1Message>;
}

/// Encodes a raw `Bytes` payload as base64 for the `l2Msg` field.
pub fn b64_l2_msg(bytes: &Bytes) -> String {
    use base64::Engine;
    base64::engine::general_purpose::STANDARD.encode(bytes.as_ref())
}
