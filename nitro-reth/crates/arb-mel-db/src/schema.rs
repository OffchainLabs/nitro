use arb_consensus_db::{
    kv::KeyBuf,
    schema::{self, BatchMetadataAt, ConsensusDbKey},
};

/// A key accessible through [`MelDb`](crate::MelDb).
///
/// Extends [`ConsensusDbKey`] with boundary-aware key construction: values that
/// have a legacy/MEL split override [`MelDbKey::mel_key`] to dispatch on the
/// boundary; everything else uses the default, which forwards straight through to
/// the underlying consensus-db key.
pub trait MelDbKey: ConsensusDbKey {
    /// Build the key, given the legacy/MEL boundary (`initial batch/delayed count`).
    /// Default: pass straight through to the base [`ConsensusDbKey::key`].
    fn mel_key(&self, _init: u64) -> KeyBuf {
        self.key()
    }
}

/// `s`/`q`: batch metadata is stored under the legacy prefix below the boundary
/// and the MEL prefix at or above it.
impl MelDbKey for BatchMetadataAt {
    fn mel_key(&self, init: u64) -> KeyBuf {
        let prefix = if self.0 < init {
            schema::SEQUENCER_BATCH_META_PREFIX
        } else {
            MEL_SEQUENCER_BATCH_META_PREFIX
        };
        schema::key(prefix, self.0)
    }
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
