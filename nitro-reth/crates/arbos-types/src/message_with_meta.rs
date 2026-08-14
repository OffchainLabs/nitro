use alloy_primitives::{B256, keccak256};
use alloy_rlp::{RlpDecodable, RlpEncodable};

use super::incoming_message::L1IncomingMessage;

/// An L1 incoming message with additional metadata.
#[derive(Debug, Clone, Default, RlpEncodable, RlpDecodable)]
pub struct MessageWithMetadata {
    pub message: L1IncomingMessage,
    pub delayed_messages_read: u64,
}

/// Extended message info including block hash and metadata.
#[derive(Debug, Clone)]
pub struct MessageWithMetadataAndBlockInfo {
    pub message_with_meta: MessageWithMetadata,
    pub block_hash: Option<B256>,
    pub block_metadata: Option<Vec<u8>>,
}

impl MessageWithMetadata {
    /// RLP commitment hash fed into the MEL local message accumulator.
    pub fn rlp_hash(&self) -> B256 {
        keccak256(alloy_rlp::encode(self))
    }

    /// Returns a shallow copy with only consensus-relevant fields.
    pub fn with_only_mel_consensus_fields(&self) -> Self {
        MessageWithMetadata {
            message: L1IncomingMessage {
                header: self.message.header.clone(),
                l2_msg: self.message.l2_msg.clone(),
                legacy_batch_gas_cost: None,
                batch_data_stats: None,
            },
            delayed_messages_read: self.delayed_messages_read,
        }
    }
}
