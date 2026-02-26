use alloy_primitives::{B256, keccak256};

use super::incoming_message::L1IncomingMessage;

/// An L1 incoming message with additional metadata.
#[derive(Debug, Clone)]
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
    /// Computes a hash of the message for consensus.
    /// Only includes MEL (minimum execution layer) consensus fields.
    pub fn hash(&self) -> B256 {
        let serialized = self.message.serialize();
        let mut data = Vec::new();
        data.extend_from_slice(&serialized);
        data.extend_from_slice(&self.delayed_messages_read.to_be_bytes());
        keccak256(&data)
    }

    /// Returns a shallow copy with only consensus-relevant fields.
    pub fn with_only_mel_consensus_fields(&self) -> Self {
        MessageWithMetadata {
            message: L1IncomingMessage {
                header: self.message.header.clone(),
                l2_msg: self.message.l2_msg.clone(),
                batch_gas_left: None,
            },
            delayed_messages_read: self.delayed_messages_read,
        }
    }
}
