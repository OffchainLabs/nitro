pub const CURRENT_VERSION: u64 = 2;

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
/// Maps a parent chain block number to its computed MEL state
pub const MEL_STATE_PREFIX: &[u8] = b"l";
/// Maps a delayed sequence number to an accumulator and an RLP encoded message [TODO(NIT-4209): might need to replace or be replaced by RlpDelayedMessagePrefix]
pub const MEL_DELAYED_MESSAGE_PREFIX: &[u8] = b"y";
/// Maps a batch sequence number to BatchMetadata [TODO(NIT-4209): might need to replace or be replaced by SequencerBatchMetaPrefix]
pub const MEL_SEQUENCER_BATCH_META_PREFIX: &[u8] = b"q";

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
/// Contains the latest computed MEL state's parent chain block number
pub const HEAD_MEL_STATE_BLOCK_NUM_KEY: &[u8] = b"_headMelStateBlockNum";
/// Contains the initial MEL state's parent chain block number (legacy/MEL boundary)
pub const INITIAL_MEL_STATE_BLOCK_NUM_KEY: &[u8] = b"_initialMelStateBlockNum";
