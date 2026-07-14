//! The consensus DB schema: key prefixes, fixed keys, and the typed value stored under
//! each prefix, byte-compatible with Nitro's `arbnode/db/schema`.
//!
//! Each entry is described by a [`ConsensusDbKey`] descriptor (a prefix plus, for
//! positional keys, a `u64` position) whose associated [`ConsensusDbValue`] defines the
//! on-disk encoding. Keys are laid out as `prefix ++ big-endian(position)` (see [`key`]).

use alloy_primitives::{Address, B256, Bytes, U256};
use alloy_rlp::{RlpDecodable, RlpDecodableWrapper, RlpEncodable, RlpEncodableWrapper};

use crate::{
    ConsensusDbError, Result,
    codecs::{
        legacy::{decode_l1_message_wire, encode_l1_message_wire},
        rlp::{NilList, NilString},
        strip_accumulator,
    },
    kv::KeyBuf,
};

pub const CURRENT_VERSION: u64 = 2;

/// Describes where a value lives in the key-value store and which value type is
/// stored there. Positional keys carry a `u64` position; fixed keys carry none.
pub trait ConsensusDbKey {
    type StoredValue: ConsensusDbValue;
    const PREFIX: &[u8];
    /// The position within the prefix, or `None` for a fixed (singleton) key.
    fn position(&self) -> Option<u64>;
}

/// Marker for keys that have a position (via [`prefix_key`]), so the whole prefix
/// can be range-iterated. Fixed (singleton) keys do not implement this.
pub trait PositionalKey: ConsensusDbKey {}

/// A value stored in the key-value store, along with how it (de)serializes.
///
/// RLP encoding may be implemented using the [`rlp_value`] macro.
pub trait ConsensusDbValue: Sized {
    fn encode(&self) -> Vec<u8>;
    fn decode(bytes: &[u8]) -> Result<Self>;
}

/// Implements [`ConsensusDbKey`] for a positional key: `prefix ++ big-endian(pos)`,
/// where `pos` is the key's single `u64` field.
macro_rules! prefix_key {
    ($key:ty[$prefix:expr] => $val:ty) => {
        impl ConsensusDbKey for $key {
            type StoredValue = $val;

            const PREFIX: &[u8] = $prefix;

            fn position(&self) -> Option<u64> {
                Some(self.0)
            }
        }
        impl PositionalKey for $key {}
    };
}

/// Implements [`ConsensusDbKey`] for a fixed (non-positional) key.
macro_rules! fixed_key {
    ($key:ty[$const_key:expr] => $val:ty) => {
        impl ConsensusDbKey for $key {
            type StoredValue = $val;

            const PREFIX: &[u8] = $const_key;

            fn position(&self) -> Option<u64> {
                None
            }
        }
    };
}

/// Implements [`ConsensusDbValue`] via RLP (the default encoding).
macro_rules! rlp_value {
    ($ty:ty) => {
        impl ConsensusDbValue for $ty {
            fn encode(&self) -> Vec<u8> {
                alloy_rlp::encode(self)
            }

            fn decode(bytes: &[u8]) -> Result<Self> {
                Ok(alloy_rlp::decode_exact(bytes)?)
            }
        }
    };
}

/// Key under the `s` prefix: sequencer batch metadata by batch number.
#[derive(Debug)]
pub struct BatchMetadataAt(pub u64);

/// Metadata for a sequencer batch: its accumulator and message/delayed/parent-chain counts.
#[derive(Debug, RlpEncodable, RlpDecodable)]
pub struct BatchMetadata {
    pub accumulator: B256,
    pub message_count: u64,
    pub delayed_message_count: u64,
    pub parent_chain_block: u64,
}

prefix_key!(BatchMetadataAt[SEQUENCER_BATCH_META_PREFIX] => BatchMetadata);
rlp_value!(BatchMetadata);

/// Key under the `m` prefix: an L2 message by sequence number.
#[derive(Debug)]
pub struct MessageWithMetadataAt(pub u64);

/// An L2 message together with the count of delayed messages read before it.
#[derive(Debug, RlpEncodable, RlpDecodable)]
pub struct MessageWithMetadata {
    pub message: L1IncomingMessage,
    pub delayed_messages_read: u64,
}

prefix_key!(MessageWithMetadataAt[MESSAGE_PREFIX] => MessageWithMetadata);
rlp_value!(MessageWithMetadata);

/// A message posted to the inbox on the parent chain, plus its L2 payload.
#[derive(Debug, RlpEncodable, RlpDecodable)]
#[rlp(trailing)]
pub struct L1IncomingMessage {
    pub header: L1IncomingMessageHeader,
    pub l2msg: Bytes,
    pub legacy_batch_gas_cost: Option<u64>,
    pub batch_data_stats: Option<BatchDataStats>,
}

/// Header of an [`L1IncomingMessage`].
#[derive(Debug, RlpEncodable, RlpDecodable)]
pub struct L1IncomingMessageHeader {
    pub kind: u8,
    pub poster: Address,
    pub block_number: u64,
    pub timestamp: u64,
    pub request_id: NilList<B256>,
    pub l1_base_fee: U256,
}

/// Size statistics for batch data, used in L1 cost accounting.
#[derive(Debug, RlpEncodable, RlpDecodable)]
pub struct BatchDataStats {
    pub length: u64,
    pub non_zeros: u64,
}

/// Key under the `r` prefix: the execution result of a message.
#[derive(Debug)]
pub struct MessageResultAt(pub u64);

/// The execution result of a message: the resulting block hash and send root.
#[derive(Debug, RlpEncodable, RlpDecodable)]
pub struct MessageResult {
    pub block_hash: B256,
    pub send_root: B256,
}

prefix_key!(MessageResultAt[MESSAGE_RESULT_PREFIX] => MessageResult);
rlp_value!(MessageResult);

/// Key under the `b` prefix: a block hash received through the input feed.
#[derive(Debug)]
pub struct BlockHashDbValueAt(pub u64);

/// A block hash received through the input feed (absent hashes encode as an empty string).
#[derive(Debug, RlpEncodable, RlpDecodable)]
pub struct BlockHashDbValue {
    pub block_hash: NilString<B256>,
}

prefix_key!(BlockHashDbValueAt[BLOCK_HASH_INPUT_FEED_PREFIX] => BlockHashDbValue);
rlp_value!(BlockHashDbValue);

/// Key under the `t` prefix: a block-metadata byte array from the input feed.
#[derive(Debug)]
pub struct BlockMetadataAt(pub u64);

/// A block-metadata byte array, stored verbatim (no RLP).
#[derive(Debug)]
pub struct BlockMetadata(pub Vec<u8>);

prefix_key!(BlockMetadataAt[BLOCK_METADATA_INPUT_FEED_PREFIX] => BlockMetadata);

impl ConsensusDbValue for BlockMetadata {
    fn encode(&self) -> Vec<u8> {
        self.0.clone()
    }

    fn decode(bytes: &[u8]) -> Result<Self> {
        Ok(Self(bytes.to_vec()))
    }
}

/// Key under the `x` prefix: marks a message whose block metadata is missing.
#[derive(Debug)]
pub struct MissingBlockMetadataAt(pub u64);

/// Presence marker for a message with missing block metadata (stored as an empty value).
#[derive(Debug)]
pub struct MissingBlockMetadata;

prefix_key!(MissingBlockMetadataAt[MISSING_BLOCK_METADATA_INPUT_FEED_PREFIX] => MissingBlockMetadata);

impl ConsensusDbValue for MissingBlockMetadata {
    fn encode(&self) -> Vec<u8> {
        Vec::new()
    }

    fn decode(_bytes: &[u8]) -> Result<Self> {
        Ok(Self)
    }
}

/// Key under the `p` prefix: the parent chain block number for a delayed message.
#[derive(Debug)]
pub struct ParentChainBlockAt(pub u64);

/// A parent chain block number, stored as a raw big-endian `u64` (not RLP).
#[derive(Debug)]
pub struct ParentChainBlock(pub u64);

prefix_key!(ParentChainBlockAt[PARENT_CHAIN_BLOCK_NUMBER_PREFIX] => ParentChainBlock);

impl ConsensusDbValue for ParentChainBlock {
    fn encode(&self) -> Vec<u8> {
        self.0.to_be_bytes().to_vec()
    }

    fn decode(bytes: &[u8]) -> Result<Self> {
        Ok(Self(u64::from_be_bytes(
            bytes
                .try_into()
                .map_err(|_| ConsensusDbError::InvalidStoredValue)?,
        )))
    }
}

/// Key under the `a` prefix, indexed by delayed message count.
#[derive(Debug)]
pub struct DelayedSequencedAt(pub u64);

/// The first sequencer batch sequence number at a given delayed count (RLP `u64`).
#[derive(Debug, RlpEncodableWrapper, RlpDecodableWrapper)]
pub struct DelayedSequenced(pub u64);

prefix_key!(DelayedSequencedAt[DELAYED_SEQUENCED_PREFIX] => DelayedSequenced);
rlp_value!(DelayedSequenced);

/// Key under the `e` prefix: a delayed message by delayed sequence number.
#[derive(Debug)]
pub struct RlpDelayedMessageAt(pub u64);

/// A delayed message: a 32-byte accumulator (AfterInboxAcc) followed by an RLP-encoded
/// L1 message. This is the current format (`d` is the legacy predecessor).
#[derive(Debug)]
pub struct RlpDelayedMessage {
    pub accumulator: B256,
    pub message: L1IncomingMessage,
}

prefix_key!(RlpDelayedMessageAt[RLP_DELAYED_MESSAGE_PREFIX] => RlpDelayedMessage);

impl ConsensusDbValue for RlpDelayedMessage {
    fn encode(&self) -> Vec<u8> {
        let mut bytes = self.accumulator.to_vec();
        bytes.extend_from_slice(&alloy_rlp::encode(&self.message));
        bytes
    }

    fn decode(bytes: &[u8]) -> Result<Self> {
        let (accumulator, rest) = strip_accumulator(bytes)?;
        let message = alloy_rlp::decode_exact(rest)?;
        Ok(Self {
            accumulator,
            message,
        })
    }
}

/// Key under the `d` prefix: a legacy delayed message by delayed sequence number.
#[derive(Debug)]
pub struct LegacyDelayedMessageAt(pub u64);

/// A legacy delayed message: a 32-byte accumulator (AfterInboxAcc) followed by the
/// message in Arbitrum's L1 incoming-message wire format (predates the RLP `e` format).
#[derive(Debug)]
pub struct LegacyDelayedMessage {
    pub accumulator: B256,
    pub message: L1IncomingMessage,
}

prefix_key!(LegacyDelayedMessageAt[LEGACY_DELAYED_MESSAGE_PREFIX] => LegacyDelayedMessage);

impl ConsensusDbValue for LegacyDelayedMessage {
    fn encode(&self) -> Vec<u8> {
        let mut bytes = self.accumulator.to_vec();
        bytes.extend_from_slice(&encode_l1_message_wire(&self.message));
        bytes
    }

    fn decode(bytes: &[u8]) -> Result<Self> {
        let (accumulator, rest) = strip_accumulator(bytes)?;
        let message = decode_l1_message_wire(rest)?;
        Ok(Self {
            accumulator,
            message,
        })
    }
}

// Fixed (non-positional) keys. Each stores an RLP-encoded `u64`.

/// Fixed key for the current message count.
#[derive(Debug)]
pub struct MessageCount;

/// Fixed key for the current delayed message count.
#[derive(Debug)]
pub struct DelayedMessageCount;

/// Fixed key for the current sequencer batch count.
#[derive(Debug)]
pub struct SequencerBatchCount;

/// Fixed key for the last pruned message.
#[derive(Debug)]
pub struct LastPrunedMessage;

/// Fixed key for the last pruned delayed message.
#[derive(Debug)]
pub struct LastPrunedDelayedMessage;

fixed_key!(MessageCount[MESSAGE_COUNT_KEY] => u64);
fixed_key!(DelayedMessageCount[DELAYED_MESSAGE_COUNT_KEY] => u64);
fixed_key!(SequencerBatchCount[SEQUENCER_BATCH_COUNT_KEY] => u64);
fixed_key!(LastPrunedMessage[LAST_PRUNED_MESSAGE_KEY] => u64);
fixed_key!(LastPrunedDelayedMessage[LAST_PRUNED_DELAYED_MESSAGE_KEY] => u64);
rlp_value!(u64);

/// Maps a message sequence number to a message
pub const MESSAGE_PREFIX: &[u8] = b"m";
/// Maps a message sequence number to a block hash received through the input feed
pub const BLOCK_HASH_INPUT_FEED_PREFIX: &[u8] = b"b";
/// Maps a message sequence number to a blockMetaData byte array received through the input feed
pub const BLOCK_METADATA_INPUT_FEED_PREFIX: &[u8] = b"t";
/// Maps a message sequence number whose blockMetaData byte array is missing to nil
pub const MISSING_BLOCK_METADATA_INPUT_FEED_PREFIX: &[u8] = b"x";
/// Maps a message sequence number to a message result
pub const MESSAGE_RESULT_PREFIX: &[u8] = b"r";
/// Maps a delayed sequence number to an accumulator and a message as serialized on L1
pub const LEGACY_DELAYED_MESSAGE_PREFIX: &[u8] = b"d";
/// Maps a delayed sequence number to an accumulator and an RLP encoded message
pub const RLP_DELAYED_MESSAGE_PREFIX: &[u8] = b"e";
/// Maps a delayed sequence number to a parent chain block number
pub const PARENT_CHAIN_BLOCK_NUMBER_PREFIX: &[u8] = b"p";
/// Maps a batch sequence number to BatchMetadata
pub const SEQUENCER_BATCH_META_PREFIX: &[u8] = b"s";
/// Maps a delayed message count to the first sequencer batch sequence number with this delayed
/// count
pub const DELAYED_SEQUENCED_PREFIX: &[u8] = b"a";

/// Contains the current message count
pub const MESSAGE_COUNT_KEY: &[u8] = b"_messageCount";
/// Contains the last pruned message key
pub const LAST_PRUNED_MESSAGE_KEY: &[u8] = b"_lastPrunedMessageKey";
/// Contains the last pruned RLP delayed message key
pub const LAST_PRUNED_DELAYED_MESSAGE_KEY: &[u8] = b"_lastPrunedDelayedMessageKey";
/// Contains the current delayed message count
pub const DELAYED_MESSAGE_COUNT_KEY: &[u8] = b"_delayedMessageCount";
/// Contains the current sequencer message count
pub const SEQUENCER_BATCH_COUNT_KEY: &[u8] = b"_sequencerBatchCount";
/// Contains a uint64 representing the database schema version
pub const DB_SCHEMA_VERSION: &[u8] = b"_schemaVersion";

/// Build the full key for `key`: its [`ConsensusDbKey::PREFIX`] followed by its
/// position as 8 big-endian bytes (or nothing for a fixed key).
pub fn key<K: ConsensusDbKey>(key: &K) -> KeyBuf {
    let mut keybuf = K::PREFIX.to_vec();
    if let Some(pos) = key.position() {
        keybuf.extend_from_slice(&pos.to_be_bytes());
    }
    keybuf
}
