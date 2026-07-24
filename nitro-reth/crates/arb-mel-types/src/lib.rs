//! Core MEL data types, shared between the extraction logic (`arb-mel`), the runner,
//! and the storage layer (`arb-mel-db`, `arb-consensus-db`). These mirror nitro's
//! `arbnode/mel` types and are byte-faithful to its `arbitrumdata` RLP encoding.

use alloy_primitives::{Address, B256};
use alloy_rlp::{RlpDecodable, RlpEncodable};
use arbos_types::L1IncomingMessage;

/// A computed MEL state.
///
/// The field set and order mirror nitro's `mel.State` exported fields exactly, so the
/// RLP encoding is byte-compatible with nitro's `arbitrumdata` on-disk representation
/// (nitro's unexported runtime-only fields are not serialized and so are omitted here).
#[derive(Debug, Default, Clone, RlpEncodable, RlpDecodable)]
pub struct MelState {
    pub version: u16,
    pub parent_chain_id: u64,
    pub parent_chain_block_number: u64,
    pub batch_posting_target_address: Address,
    pub delayed_message_posting_target_address: Address,
    pub parent_chain_block_hash: B256,
    pub parent_chain_prev_block_hash: B256,
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

/// Metadata for a sequencer batch: its accumulator and message/delayed/parent-chain counts.
/// Field order mirrors nitro's `mel.BatchMetadata`.
#[derive(Debug, Clone, RlpEncodable, RlpDecodable)]
pub struct BatchMetadata {
    pub accumulator: B256,
    pub message_count: u64,
    pub delayed_message_count: u64,
    pub parent_chain_block: u64,
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
}
