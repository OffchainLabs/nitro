//! Core MEL data types, shared between the extraction logic (`arb-mel`), the runner,
//! and the storage layer (`arb-mel-db`, `arb-consensus-db`). These mirror nitro's
//! `arbnode/mel` types and are byte-faithful to its `arbitrumdata` RLP encoding.
//!
//! [`MelState`] and [`BatchMetadata`] additionally derive serde and serialize to the
//! `meldataprovider` JSON-RPC wire shape (PascalCase), matching nitro's tagless
//! `mel.State` / `mel.BatchMetadata` JSON so a Nitro node can consume them directly.

use alloy_primitives::{Address, B256, Keccak256, keccak256};
use alloy_rlp::{RlpDecodable, RlpEncodable};
use arbos_types::L1IncomingMessage;
use serde::{Deserialize, Serialize};

/// A computed MEL state.
///
/// The field set and order mirror nitro's `mel.State` exported fields exactly, so the
/// RLP encoding is byte-compatible with nitro's `arbitrumdata` on-disk representation
/// (nitro's unexported runtime-only fields are not serialized and so are omitted here).
/// The serde `PascalCase` serialization matches nitro's default (tagless) `mel.State`
/// JSON, so this type doubles as the `meldataprovider` RPC wire type.
#[derive(Debug, Default, Clone, Serialize, Deserialize, RlpEncodable, RlpDecodable)]
#[serde(rename_all = "PascalCase")]
pub struct MelState {
    pub version: u16,
    pub parent_chain_id: u64,
    pub parent_chain_block_number: u64,
    pub batch_posting_target_address: Address,
    pub delayed_message_posting_target_address: Address,
    pub parent_chain_block_hash: B256,
    pub parent_chain_previous_block_hash: B256,
    pub batch_count: u64,
    pub msg_count: u64,
    pub local_msg_accumulator: B256, /* starts at zero hash for each clone; updated only by
                                      * AccumulateMessage; represents messages accumulated
                                      * during processing of this specific parent chain block */
    pub delayed_messages_read: u64,
    pub delayed_messages_seen: u64,
    pub delayed_message_inbox_acc: B256,
    pub delayed_message_outbox_acc: B256,
}

/// A delayed inbox message reconstructed from a `MessageDelivered` event and
/// its corresponding inbox-message data. Field order mirrors nitro's
/// `mel.DelayedInboxMessage` so the RLP encoding matches `arbitrumdata`.
#[derive(Debug, Clone, RlpEncodable, RlpDecodable)]
pub struct DelayedInboxMessage {
    pub block_hash: B256,
    pub before_inbox_acc: B256,
    pub message: L1IncomingMessage,
    pub parent_chain_block_number: u64,
}

impl DelayedInboxMessage {
    /// RLP commitment hash fed into the MEL delayed-inbox accumulator.
    pub fn rlp_hash(&self) -> B256 {
        keccak256(alloy_rlp::encode(self))
    }

    /// The delayed-inbox accumulator value after this message, byte-compatible
    /// with nitro's `DelayedInboxMessage.AfterInboxAcc()` and with the on-chain
    /// `Bridge`/`Inbox` accumulator.
    ///
    /// Hashes the message header fields together with `keccak256(l2_msg)` into a
    /// per-message commitment, then chains it onto [`Self::before_inbox_acc`]:
    /// `keccak256(before_inbox_acc || commitment)`. This is the value returned by
    /// the `meldataprovider_getDelayedAcc` RPC.
    ///
    /// Distinct from the RLP-based [`Self::rlp_hash`] (nitro's `Hash()`) that the MEL
    /// state's `delayed_message_inbox_acc` chain uses. The two accumulators are
    /// not interchangeable.
    pub fn after_inbox_acc(&self) -> B256 {
        let header = &self.message.header;
        let mut hasher = Keccak256::new();
        hasher.update([header.kind]);
        hasher.update(header.poster.as_slice());
        hasher.update(header.block_number.to_be_bytes());
        hasher.update(header.timestamp.to_be_bytes());
        hasher.update(header.request_id.unwrap_or_default().as_slice());
        hasher.update(header.l1_base_fee.unwrap_or_default().to_be_bytes::<32>());
        hasher.update(keccak256(&self.message.l2_msg).as_slice());
        let inner = hasher.finalize();

        let mut chain = Keccak256::new();
        chain.update(self.before_inbox_acc.as_slice());
        chain.update(inner.as_slice());
        chain.finalize()
    }
}

/// Metadata for a sequencer batch: its accumulator and message/delayed/parent-chain counts.
/// Field order mirrors nitro's `mel.BatchMetadata`. Serde `PascalCase` serialization
/// matches nitro's `mel.BatchMetadata` JSON (the `meldataprovider` RPC wire form).
#[derive(Debug, Clone, Serialize, Deserialize, RlpEncodable, RlpDecodable)]
#[serde(rename_all = "PascalCase")]
pub struct BatchMetadata {
    pub accumulator: B256,
    pub message_count: u64,
    pub delayed_message_count: u64,
    pub parent_chain_block: u64,
}

/// Progress of message synchronization.
///
/// Mirrors `mel.MessageSyncProgress`.
#[derive(Debug, Clone, Default, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "PascalCase")]
pub struct MessageSyncProgress {
    /// Highest batch count seen on the parent chain.
    pub batch_seen: u64,
    /// Batch count processed into state.
    pub batch_processed: u64,
    /// L2 messages produced.
    pub msg_count: u64,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn default_mel_state_is_zeroed() {
        let state = MelState::default();
        assert_eq!(state.parent_chain_block_hash, B256::ZERO);
        assert_eq!(state.batch_count, 0);
        assert_eq!(state.msg_count, 0);
        assert_eq!(state.delayed_messages_seen, 0);
        assert_eq!(state.version, 0);
    }

    #[test]
    fn after_inbox_acc_matches_layout() {
        use alloy_primitives::{Address, U256};
        use arbos_types::{L1IncomingMessage, L1IncomingMessageHeader};

        let msg = DelayedInboxMessage {
            block_hash: B256::ZERO,
            before_inbox_acc: B256::repeat_byte(0xAB),
            message: L1IncomingMessage {
                header: L1IncomingMessageHeader {
                    kind: 3,
                    poster: Address::repeat_byte(0x11),
                    block_number: 0x0102_0304,
                    timestamp: 0x0506,
                    request_id: Some(B256::repeat_byte(0x22)),
                    l1_base_fee: Some(U256::from(0x777u64)),
                },
                l2_msg: vec![1, 2, 3, 4].into(),
                legacy_batch_gas_cost: None,
                batch_data_stats: None,
            },
            parent_chain_block_number: 9,
        };

        // Independent reconstruction of the expected two-stage hash.
        let mut inner = Vec::new();
        inner.push(3u8);
        inner.extend_from_slice(Address::repeat_byte(0x11).as_slice());
        inner.extend_from_slice(&0x0102_0304u64.to_be_bytes());
        inner.extend_from_slice(&0x0506u64.to_be_bytes());
        inner.extend_from_slice(B256::repeat_byte(0x22).as_slice());
        inner.extend_from_slice(&U256::from(0x777u64).to_be_bytes::<32>());
        inner.extend_from_slice(keccak256([1u8, 2, 3, 4]).as_slice());
        let inner = keccak256(&inner);
        let mut chain = [0u8; 64];
        chain[..32].copy_from_slice(B256::repeat_byte(0xAB).as_slice());
        chain[32..].copy_from_slice(inner.as_slice());

        assert_eq!(msg.after_inbox_acc(), keccak256(chain));
    }

    #[test]
    fn mel_state_json_keys_are_pascal_case() {
        let v = serde_json::to_value(MelState::default()).unwrap();
        let o = v.as_object().unwrap();
        assert!(o.contains_key("ParentChainId"));
        assert!(o.contains_key("ParentChainPreviousBlockHash"));
        assert!(o.contains_key("DelayedMessageInboxAcc"));
    }
}
