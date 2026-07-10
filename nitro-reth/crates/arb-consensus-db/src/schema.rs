use alloy_primitives::{Address, B256, Bytes, U256};
use alloy_rlp::{RlpDecodable, RlpEncodable};

use crate::{
    Result,
    kv::KeyBuf,
    rlp::{NilList, NilString},
};

pub const CURRENT_VERSION: u64 = 2;

/// Stored value in the key-value store.
///
/// RLP encoding may be implemented using the [`rlp_value`] macro.
pub trait StoredValue: Sized {
    const PREFIX: &[u8];
    fn encode(&self) -> Vec<u8>;
    fn decode(bytes: &[u8]) -> Result<Self>;
}

#[derive(Debug, RlpEncodable, RlpDecodable)]
pub struct BatchMetadata {
    pub accumulator: B256,
    pub message_count: u64,
    pub delayed_message_count: u64,
    pub parent_chain_block: u64,
}

#[derive(Debug, RlpEncodable, RlpDecodable)]
pub struct MessageWithMetadata {
    pub message: L1IncomingMessage,
    pub delayed_messages_read: u64,
}

#[derive(Debug, RlpEncodable, RlpDecodable)]
#[rlp(trailing)]
pub struct L1IncomingMessage {
    pub header: L1IncomingMessageHeader,
    pub l2msg: Bytes,
    pub legacy_batch_gas_cost: Option<u64>,
    pub batch_data_stats: Option<BatchDataStats>,
}

#[derive(Debug, RlpEncodable, RlpDecodable)]
pub struct L1IncomingMessageHeader {
    pub kind: u8,
    pub poster: Address,
    pub block_number: u64,
    pub timestamp: u64,
    // TODO: can omit NilList if byte-compatibility is not needed
    pub request_id: NilList<B256>,
    pub l1_base_fee: U256,
}

#[derive(Debug, RlpEncodable, RlpDecodable)]
pub struct BatchDataStats {
    pub length: u64,
    pub non_zeros: u64,
}

#[derive(Debug, RlpEncodable, RlpDecodable)]
pub struct MessageResult {
    pub block_hash: B256,
    pub send_root: B256,
}

#[derive(Debug, RlpEncodable, RlpDecodable)]
pub struct BlockHashDbValue {
    pub block_hash: NilString<B256>,
}

macro_rules! rlp_value {
    ($ty:ty => $prefix:expr) => {
        impl StoredValue for $ty {
            const PREFIX: &[u8] = $prefix;
            fn encode(&self) -> Vec<u8> {
                alloy_rlp::encode(self)
            }
            fn decode(bytes: &[u8]) -> Result<Self> {
                Ok(alloy_rlp::decode_exact(bytes)?)
            }
        }
    };
}

rlp_value!(BatchMetadata => SEQUENCER_BATCH_META_PREFIX);
rlp_value!(MessageWithMetadata => MESSAGE_PREFIX);
rlp_value!(MessageResult => MESSAGE_RESULT_PREFIX);
rlp_value!(BlockHashDbValue => BLOCK_HASH_INPUT_FEED_PREFIX);

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
/// Maps a delayed message count to the first sequencer batch sequence number with this delayed count
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

/// Build a database key: `prefix` followed by `pos` as 8 big-endian bytes.
pub fn key(prefix: &[u8], pos: u64) -> KeyBuf {
    let pos_bytes = pos.to_be_bytes();
    let mut key = Vec::with_capacity(prefix.len() + pos_bytes.len());
    key.extend_from_slice(prefix);
    key.extend_from_slice(&pos_bytes);
    key
}
