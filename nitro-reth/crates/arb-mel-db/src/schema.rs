use arb_consensus_db::{
    kv::KeyBuf,
    schema::{self, BatchMetadataAt, ConsensusDbKey},
};

/// A key accessible through [`MelDb`](crate::MelDb).
///
/// Extends [`ConsensusDbKey`] with boundary-aware key construction. A value with a
/// legacy/MEL split declares `MEL_PREFIX`; the default `mel_key` then dispatches on
/// the boundary (legacy prefix below `init`, MEL prefix at/above). Keys without a
/// split leave `MEL_PREFIX` unset and forward straight through to the base key.
pub trait MelDbKey: ConsensusDbKey {
    /// The MEL-era prefix, if this value is stored under a legacy/MEL split.
    const MEL_PREFIX: Option<&[u8]> = None;
}

/// Build the key for `key` given the legacy/MEL boundary (`initial batch/delayed
/// count`): the legacy prefix below `init`, the MEL prefix at or above it. Keys
/// without a MEL split forward straight through to the base [`schema::key`].
pub fn mel_key<K: MelDbKey>(key: &K, init: u64) -> KeyBuf {
    match K::MEL_PREFIX {
        Some(mel) => {
            let pos = key.position().expect("MEL keys are positional");
            let prefix = if pos < init { K::PREFIX } else { mel };
            let mut keybuf = prefix.to_vec();
            keybuf.extend_from_slice(&pos.to_be_bytes());
            keybuf
        }
        None => schema::key(key),
    }
}

/// `s`/`q`: batch metadata is stored under the legacy prefix below the boundary and
/// the MEL prefix at or above it.
impl MelDbKey for BatchMetadataAt {
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
