use arb_consensus_db::{
    kv,
    schema::{BatchMetadata, KeyPrefix},
};

pub trait MelKeyPrefix: KeyPrefix {
    const MEL_PREFIX: Option<&[u8]> = None;
}

impl MelKeyPrefix for BatchMetadata {
    const MEL_PREFIX: Option<&[u8]> = Some(MEL_SEQUENCER_BATCH_META_PREFIX);
}

/// Maps a parent chain block number to its computed MEL state
pub const MEL_STATE_PREFIX: &[u8] = b"l";
/// Maps a delayed sequence number to an accumulator and an RLP encoded message [TODO(NIT-4209): might need to replace or be replaced by RLP_DELAYED_MESSAGE_PREFIX]
pub const MEL_DELAYED_MESSAGE_PREFIX: &[u8] = b"y";
/// Maps a batch sequence number to BatchMetadata [TODO(NIT-4209): might need to replace or be replaced by SEQUENCER_BATCH_META_PREFIX]
pub const MEL_SEQUENCER_BATCH_META_PREFIX: &[u8] = b"q";

/// Contains the latest computed MEL state's parent chain block number
pub const HEAD_MEL_STATE_BLOCK_NUM_KEY: &[u8] = b"_headMelStateBlockNum";
/// Contains the initial MEL state's parent chain block number (legacy/MEL boundary)
pub const INITIAL_MEL_STATE_BLOCK_NUM_KEY: &[u8] = b"_initialMelStateBlockNum";

pub fn key<T: MelKeyPrefix>(pos: u64, init: u64) -> kv::KeyBuf {
    let prefix = if pos < init {
        T::PREFIX
    } else {
        T::MEL_PREFIX.unwrap_or(T::PREFIX)
    };
    arb_consensus_db::schema::key(prefix, pos)
}
