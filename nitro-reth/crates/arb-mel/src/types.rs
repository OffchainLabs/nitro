use alloy_primitives::{Address, B256, keccak256};
use alloy_rlp::{RlpDecodable, RlpEncodable};
use arbos::arbos_types::{L1IncomingMessage, MessageWithMetadata};

use crate::{DelayedMessageDB, MelError};

pub type MelResult<T> = Result<T, MelError>;

/// A computed MEL state.
///
/// The field set and order mirror nitro's `mel.State` exported fields exactly, so the
/// RLP encoding is byte-compatible with nitro's `arbitrumdata` on-disk representation
/// (nitro's unexported runtime-only fields are not serialized and so are omitted here).
#[derive(Default, Clone, RlpEncodable, RlpDecodable)]
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

fn chain_accumulator(prev: B256, msg_hash: B256) -> B256 {
    let mut preimage = [0u8; 64];
    preimage[..32].copy_from_slice(prev.as_slice());
    preimage[32..].copy_from_slice(msg_hash.as_slice());
    keccak256(preimage)
}

impl MelState {
    // TODO: not yet at parity with Nitro's AccumulateDelayedMessage. Missing
    // initMsg capture (when delayed_messages_seen == 0) and preimage recording
    // that the pour/pop FIFO and MEL validation depend on.
    pub fn accumulate_delayed_message(&mut self, message: &DelayedInboxMessage) -> MelResult<()> {
        self.delayed_message_inbox_acc =
            chain_accumulator(self.delayed_message_inbox_acc, message.abi_hash());
        Ok(())
    }

    // TODO: not yet at parity with Nitro's AccumulateMessage. Missing preimage
    // recording required for MEL validation-mode replay.
    pub fn accumulate_message(&mut self, message: &MessageWithMetadata) -> MelResult<()> {
        self.local_msg_accumulator =
            chain_accumulator(self.local_msg_accumulator, message.abi_hash());
        Ok(())
    }

    pub fn move_unread_delayed_messages_to_inbox_accumulator(
        &mut self,
        delayed_msg_db: &impl DelayedMessageDB,
    ) -> MelResult<()> {
        let mut unread = Vec::new();
        for i in self.delayed_messages_read..self.delayed_messages_seen {
            let msg = delayed_msg_db
                .read_delayed_message(self, i)
                .map_err(|e| MelError::DelayedAccumulatorCreation(e.to_string()))?
                .ok_or_else(|| {
                    MelError::DelayedAccumulatorCreation(format!(
                        "no delayed message in db at index {i}"
                    ))
                })?;
            unread.push(msg);
        }
        if self.delayed_message_inbox_acc != B256::ZERO
            || self.delayed_message_outbox_acc != B256::ZERO
        {
            return Err(MelError::NonZeroDelayedAccumulator {
                inbox: self.delayed_message_inbox_acc,
                outbox: self.delayed_message_outbox_acc,
            });
        }
        for msg in &unread {
            self.accumulate_delayed_message(msg)?;
        }
        Ok(())
    }
}

/// A delayed inbox message reconstructed from a `MessageDelivered` event and
/// its corresponding inbox-message data.
#[derive(Clone)]
pub struct DelayedInboxMessage {
    pub block_hash: B256,
    pub before_inbox_acc: B256,
    pub message: L1IncomingMessage,
    pub parent_chain_block_number: u64,
}

impl DelayedInboxMessage {
    /// ABI-style commitment hash fed into the MEL delayed-inbox accumulator.
    ///
    /// TODO: packed concatenation, not yet reconciled with the on-chain
    /// `abi.encode` layout (Nitro's `DelayedInboxMessage.Hash()` uses RLP).
    pub fn abi_hash(&self) -> B256 {
        let mut data = Vec::new();
        data.extend_from_slice(self.block_hash.as_slice());
        data.extend_from_slice(self.before_inbox_acc.as_slice());
        data.extend_from_slice(&self.message.serialize());
        data.extend_from_slice(&self.parent_chain_block_number.to_be_bytes());
        keccak256(&data)
    }
}

/// Time bounds for a sequencer batch.
#[derive(Debug, Clone, Copy, Default)]
pub struct TimeBounds {
    pub min_timestamp: u64,
    pub max_timestamp: u64,
    pub min_block_number: u64,
    pub max_block_number: u64,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum DataLocation {
    TxInput,
    SeparateEvent,
    BlobHashes,
}

impl DataLocation {
    /// Decodes the on-chain `dataLocation` byte. Unknown values (e.g. a
    /// force-inclusion batch that carries no data) map to `None`.
    pub fn from_byte(byte: u8) -> Option<Self> {
        match byte {
            0 => Some(DataLocation::TxInput),
            1 => Some(DataLocation::SeparateEvent),
            2 => Some(DataLocation::BlobHashes),
            _ => None,
        }
    }
}

/// A batch parsed from a `SequencerBatchDelivered` log.
pub struct Batch {
    pub block_hash: B256,
    pub parent_chain_block_number: u64,
    pub sequence_number: u64,
    pub before_inbox_acc: B256,
    pub after_inbox_acc: B256,
    pub after_delayed_acc: B256,
    pub after_delayed_count: u64,
    pub time_bounds: TimeBounds,
    pub data_location: Option<DataLocation>,
    pub bridge_address: Address,
    pub raw_log: alloy_rpc_types_eth::Log,
    pub cached_serialized: Option<Vec<u8>>,
}

pub struct BatchMeta {
    pub accumulator: B256,
    pub message_count: u64,
    pub delayed_message_count: u64,
    pub parent_chain_block: u64,
}

#[derive(Debug, PartialEq)]
pub enum BatchSegmentKind {
    L2Message,
    L2MessageBrotli,
    DelayedMessages,
    AdvanceTimestamp,
    AdvanceL1BlockNumber,
    Unknown,
}

impl From<BatchSegmentKind> for u8 {
    fn from(value: BatchSegmentKind) -> Self {
        use BatchSegmentKind::*;
        match value {
            L2Message => 0,
            L2MessageBrotli => 1,
            DelayedMessages => 2,
            AdvanceTimestamp => 3,
            AdvanceL1BlockNumber => 4,
            Unknown => 5,
        }
    }
}

impl From<u8> for BatchSegmentKind {
    fn from(value: u8) -> Self {
        use BatchSegmentKind::*;
        match value {
            0 => L2Message,
            1 => L2MessageBrotli,
            2 => DelayedMessages,
            3 => AdvanceTimestamp,
            4 => AdvanceL1BlockNumber,
            _ => Unknown,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn data_location_from_byte() {
        assert_eq!(DataLocation::from_byte(0), Some(DataLocation::TxInput));
        assert_eq!(
            DataLocation::from_byte(1),
            Some(DataLocation::SeparateEvent)
        );
        assert_eq!(DataLocation::from_byte(2), Some(DataLocation::BlobHashes));
        // Unknown bytes (e.g. a force-inclusion batch with no data) decode to None.
        assert_eq!(DataLocation::from_byte(3), None);
        assert_eq!(DataLocation::from_byte(255), None);
    }

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
