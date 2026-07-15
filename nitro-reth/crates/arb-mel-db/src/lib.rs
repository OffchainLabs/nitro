use alloy_primitives::B256;
use arb_consensus_db::{
    ConsensusDb, ConsensusDbError, Result, kv,
    schema::{L1IncomingMessage, LegacyDelayedMessageAt, ParentChainBlockAt, RlpDelayedMessageAt},
};

pub mod schema;

/// The legacy/MEL boundary, resolved from the initial-state anchor: a position below the
/// relevant count reads the legacy prefix, at or above it reads the MEL prefix.
#[derive(Debug, Clone, Copy)]
struct Boundary {
    batch_count: u64,
    delayed_count: u64,
}

#[derive(Debug)]
pub struct MelDb<S> {
    consensus_db: ConsensusDb<S>,
    /// The legacy/MEL boundary, or `None` for a MEL-only DB (all lookups route to MEL prefixes).
    initial: Option<Boundary>,
}

impl<S: kv::KvStore> MelDb<S> {
    /// Open a MEL DB over `consensus_db`, resolving the legacy/MEL boundary from the stored
    /// initial-state anchor (mirrors nitro's `Database.loadInitialBoundary`).
    ///
    /// With no anchor (`_initialMelStateBlockNum` absent) the DB is MEL-only, so every lookup
    /// routes to the MEL prefixes.
    pub fn open(consensus_db: ConsensusDb<S>) -> Result<Self> {
        let initial = match consensus_db.get(schema::InitialMelStateBlockNum)? {
            None => None,
            Some(block_num) => {
                let state = consensus_db
                    .get(schema::MelStateAt(block_num))?
                    .ok_or(ConsensusDbError::InvalidStoredValue)?
                    .0;
                Some(Boundary {
                    batch_count: state.batch_count,
                    delayed_count: state.delayed_messages_seen,
                })
            }
        };
        Ok(MelDb {
            consensus_db,
            initial,
        })
    }

    /// Read a split key, passing the batch boundary to [`schema::mel_key`], which selects the
    /// legacy or MEL prefix. Only batch metadata (`s`/`q`) is a [`schema::MelDbKey`]; delayed
    /// messages go through [`Self::delayed_message`], since their legacy side is a multi-key
    /// reconstruction rather than a prefix swap.
    pub fn get<K: schema::MelDbKey>(&self, key: K) -> Result<Option<K::StoredValue>> {
        let boundary = self.initial.map_or(0, |b| b.batch_count);
        self.consensus_db
            .get_at_key(&schema::mel_key(&key, boundary))
    }

    /// Write a split key, passing the batch boundary to [`schema::mel_key`]. See [`Self::get`].
    pub fn put<K: schema::MelDbKey>(&mut self, key: K, value: &K::StoredValue) -> Result<()> {
        let boundary = self.initial.map_or(0, |b| b.batch_count);
        self.consensus_db
            .put_at_key(&schema::mel_key(&key, boundary), value)
    }

    /// Read a delayed message by its delayed index, dispatching across the legacy/MEL boundary.
    ///
    /// At or above the boundary it reads the MEL `y` record directly. Below it, it reconstructs a
    /// [`schema::DelayedInboxMessage`] from the pre-MEL records (mirrors nitro's
    /// `legacyFetchDelayedMessage`): the message and its parent-chain block from
    /// [`Self::legacy_message_and_parent_block`], and `before_inbox_acc` from the previous
    /// index's accumulator. The legacy format did not store `block_hash`, so it is left zero.
    pub fn delayed_message(&self, index: u64) -> Result<Option<schema::DelayedInboxMessage>> {
        if self.initial.is_none_or(|b| index >= b.delayed_count) {
            return self.consensus_db.get(schema::MelDelayedMessageAt(index));
        }
        let Some((message, parent_chain_block_number)) =
            self.legacy_message_and_parent_block(index)?
        else {
            return Ok(None);
        };
        let before_inbox_acc = if index == 0 {
            B256::ZERO
        } else {
            self.legacy_accumulator(index - 1)?
        };
        Ok(Some(schema::DelayedInboxMessage {
            block_hash: B256::ZERO,
            before_inbox_acc,
            message,
            parent_chain_block_number,
        }))
    }

    /// Read a pre-MEL delayed message and its parent-chain block number, preferring the RLP `e`
    /// form and falling back to the older wire `d` form. For `e`, the parent-chain block is stored
    /// under `p`, falling back to the message's own header block number when absent; `d` has no
    /// `p` entry and always uses the header block number (mirrors nitro).
    fn legacy_message_and_parent_block(
        &self,
        index: u64,
    ) -> Result<Option<(L1IncomingMessage, u64)>> {
        if let Some(record) = self.consensus_db.get(RlpDelayedMessageAt(index))? {
            let parent = self
                .consensus_db
                .get(ParentChainBlockAt(index))?
                .map_or(record.message.header.block_number, |block| block.0);
            return Ok(Some((record.message, parent)));
        }
        Ok(self
            .consensus_db
            .get(LegacyDelayedMessageAt(index))?
            .map(|record| {
                let parent = record.message.header.block_number;
                (record.message, parent)
            }))
    }

    /// Read the accumulator (`AfterInboxAcc`) of a pre-MEL delayed message, preferring the `e`
    /// form and falling back to `d`. Errors if the record is missing.
    fn legacy_accumulator(&self, index: u64) -> Result<B256> {
        if let Some(record) = self.consensus_db.get(RlpDelayedMessageAt(index))? {
            return Ok(record.accumulator);
        }
        self.consensus_db
            .get(LegacyDelayedMessageAt(index))?
            .map(|record| record.accumulator)
            .ok_or(ConsensusDbError::InvalidStoredValue)
    }
}
