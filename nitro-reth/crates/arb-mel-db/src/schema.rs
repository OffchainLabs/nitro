use arb_consensus_db::{
    codecs::rlp::Rlp,
    fixed_key,
    kv::KeyBuf,
    prefix_key,
    schema::{self, BatchMetadataAt, ConsensusDbKey},
};
use arb_mel_types::{BatchMetadata, DelayedInboxMessage, MelState};

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
/// Maps a delayed sequence number to an accumulator and an RLP encoded message [TODO(NIT-4209):
/// might need to replace or be replaced by RLP_DELAYED_MESSAGE_PREFIX]
pub const MEL_DELAYED_MESSAGE_PREFIX: &[u8] = b"y";
/// Maps a batch sequence number to BatchMetadata [TODO(NIT-4209): might need to replace or be
/// replaced by SEQUENCER_BATCH_META_PREFIX]
pub const MEL_SEQUENCER_BATCH_META_PREFIX: &[u8] = b"q";

/// Contains the latest computed MEL state's parent chain block number
pub const HEAD_MEL_STATE_BLOCK_NUM_KEY: &[u8] = b"_headMelStateBlockNum";
/// Contains the initial MEL state's parent chain block number (legacy/MEL boundary)
pub const INITIAL_MEL_STATE_BLOCK_NUM_KEY: &[u8] = b"_initialMelStateBlockNum";

/// Key under the `l` prefix: a computed MEL state, keyed by parent-chain block number. Internal
/// target, go through [`MelDb::state`] and [`MelDb::save_state`] for public access.
#[derive(Debug)]
pub(crate) struct MelStateAt(pub(crate) u64);

prefix_key!(MelStateAt[MEL_STATE_PREFIX] => Rlp<MelState>);

/// Key under the `y` prefix: a delayed inbox message by delayed index. Internal target, go
/// through [`MelDb::delayed_message`] and [`MelDb::save_delayed_messages`] for public access.
#[derive(Debug)]
pub(crate) struct MelDelayedMessageAt(pub(crate) u64);

prefix_key!(MelDelayedMessageAt[MEL_DELAYED_MESSAGE_PREFIX] => Rlp<DelayedInboxMessage>);

/// Key under the `q` prefix: the MEL-era write target for sequencer batch metadata. Internal
/// target, go through [`MelDb::save_batch_metas`] for public access. Reads go through
/// [`BatchMetadataAt`] with boundary dispatch; new batches are always written MEL-side.
#[derive(Debug)]
pub(crate) struct MelBatchMetaAt(pub(crate) u64);

prefix_key!(MelBatchMetaAt[MEL_SEQUENCER_BATCH_META_PREFIX] => BatchMetadata);

/// Fixed key for the latest computed MEL state's parent-chain block number.
#[derive(Debug)]
pub struct HeadMelStateBlockNum;

/// Fixed key for the initial MEL state's parent-chain block number (legacy/MEL boundary).
#[derive(Debug)]
pub struct InitialMelStateBlockNum;

fixed_key!(HeadMelStateBlockNum[HEAD_MEL_STATE_BLOCK_NUM_KEY] => u64);
fixed_key!(InitialMelStateBlockNum[INITIAL_MEL_STATE_BLOCK_NUM_KEY] => u64);

#[cfg(test)]
mod tests {
    use arb_consensus_db::schema::key;

    use super::*;

    /// Each positional MEL key is wired to its prefix and lays out as
    /// `prefix ++ big-endian(position)` (byte-compatible with nitro's `arbitrumdata`).
    #[test]
    fn mel_keys_lay_out_as_prefix_plus_big_endian_position() {
        let pos = 1u64.to_be_bytes();
        assert_eq!(key(&MelStateAt(1)), [b"l".as_slice(), &pos].concat());
        assert_eq!(
            key(&MelDelayedMessageAt(1)),
            [b"y".as_slice(), &pos].concat()
        );
        assert_eq!(key(&MelBatchMetaAt(1)), [b"q".as_slice(), &pos].concat());
    }
}
