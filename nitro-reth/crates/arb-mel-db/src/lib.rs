use alloy_primitives::B256;
use arb_consensus_db::{
    ConsensusDb, ConsensusDbBatch, ConsensusDbError, Result,
    codecs::rlp::Rlp,
    kv,
    schema::{
        BatchMetadata, L1IncomingMessage, LegacyDelayedMessageAt, ParentChainBlockAt,
        RlpDelayedMessageAt,
    },
};
use arb_mel::MelState;

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

    /// Read the MEL state stored at `parent_chain_block_number`, if any (mirrors nitro's `State`).
    pub fn state(&self, parent_chain_block_number: u64) -> Result<Option<MelState>> {
        Ok(self
            .consensus_db
            .get(schema::MelStateAt(parent_chain_block_number))?
            .map(|state| state.0))
    }

    /// Read the parent-chain block number of the current head MEL state, or `None` if no head has
    /// been set (mirrors nitro's `GetHeadMelStateBlockNum`; the MEL runner uses this to resume).
    pub fn head_state_block_num(&self) -> Result<Option<u64>> {
        self.consensus_db.get(schema::HeadMelStateBlockNum)
    }

    /// Read the current head MEL state, following the `_headMelStateBlockNum` pointer to its `l`
    /// record. Returns `None` if no head has been set (mirrors nitro's `GetHeadMelState`).
    pub fn head_state(&self) -> Result<Option<MelState>> {
        Ok(self
            .head_state_block_num()?
            .map(|block_num| self.state(block_num))
            .transpose()?
            .flatten())
    }

    /// Save `state` as the new head: writes it under `l` and advances the head pointer
    /// (`_headMelStateBlockNum`) to its block number, atomically (mirrors nitro's `SaveState`).
    pub fn save_state(&mut self, state: &MelState) -> Result<()> {
        let block_num = state.parent_chain_block_number;
        let mut batch = ConsensusDbBatch::new();
        batch.put(schema::MelStateAt(block_num), &Rlp(state.clone()));
        batch.put(schema::HeadMelStateBlockNum, &block_num);
        self.consensus_db.write_batch(batch)
    }

    /// Establish the legacy/MEL boundary at `initial_state`: atomically writes it under `l`, sets
    /// the head pointer, and records the `_initialMelStateBlockNum` anchor, then caches the
    /// boundary counts so later lookups below them route to the legacy schema (mirrors nitro's
    /// `SaveInitialMelState`).
    pub fn save_initial_mel_state(&mut self, initial_state: &MelState) -> Result<()> {
        let block_num = initial_state.parent_chain_block_number;
        let mut batch = ConsensusDbBatch::new();
        batch.put(schema::InitialMelStateBlockNum, &block_num);
        batch.put(schema::MelStateAt(block_num), &Rlp(initial_state.clone()));
        batch.put(schema::HeadMelStateBlockNum, &block_num);
        self.consensus_db.write_batch(batch)?;
        self.initial = Some(Boundary {
            batch_count: initial_state.batch_count,
            delayed_count: initial_state.delayed_messages_seen,
        });
        Ok(())
    }

    /// Save a run of newly-seen delayed messages under the MEL `y` prefix, atomically. They are
    /// the last `messages.len()` of `state.delayed_messages_seen`, so the first is written at
    /// `delayed_messages_seen - messages.len()` (mirrors nitro's `SaveDelayedMessages`).
    ///
    /// Panics if `state.delayed_messages_seen < messages.len()`, a caller invariant violation
    /// (the state was not advanced to cover these messages) that cannot occur in correct flow.
    pub fn save_delayed_messages(
        &mut self,
        state: &MelState,
        messages: &[schema::DelayedInboxMessage],
    ) -> Result<()> {
        let first = state
            .delayed_messages_seen
            .checked_sub(messages.len() as u64)
            .expect("delayed_messages_seen < batch len: state and messages are inconsistent");
        let mut batch = ConsensusDbBatch::new();
        for (i, message) in messages.iter().enumerate() {
            batch.put(schema::MelDelayedMessageAt(first + i as u64), message);
        }
        self.consensus_db.write_batch(batch)
    }

    /// Save a run of newly-computed batch metadata under the MEL `q` prefix, atomically. They are
    /// the last `metas.len()` of `state.batch_count`, so the first is written at
    /// `batch_count - metas.len()` (mirrors nitro's `SaveBatchMetas`).
    ///
    /// Panics if `state.batch_count < metas.len()`, a caller invariant violation that cannot
    /// occur in correct flow.
    pub fn save_batch_metas(&mut self, state: &MelState, metas: &[BatchMetadata]) -> Result<()> {
        let first = state
            .batch_count
            .checked_sub(metas.len() as u64)
            .expect("batch_count < metas len: state and batch metas are inconsistent");
        let mut batch = ConsensusDbBatch::new();
        for (i, meta) in metas.iter().enumerate() {
            batch.put(schema::MelBatchMetaAt(first + i as u64), meta);
        }
        self.consensus_db.write_batch(batch)
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

#[cfg(test)]
mod tests {
    use alloy_primitives::{Address, U256};
    use arb_consensus_db::{
        codecs::rlp::NilList,
        kv::MemoryKvStore,
        schema::{
            BatchMetadataAt, L1IncomingMessageHeader, LegacyDelayedMessage, ParentChainBlock,
            RlpDelayedMessage,
        },
    };

    use super::*;

    fn mel_db() -> MelDb<MemoryKvStore> {
        MelDb::open(ConsensusDb::open(MemoryKvStore::new()).unwrap()).unwrap()
    }

    fn mel_state(block: u64, batch_count: u64, delayed_seen: u64) -> MelState {
        MelState {
            parent_chain_block_number: block,
            batch_count,
            delayed_messages_seen: delayed_seen,
            ..Default::default()
        }
    }

    fn l1_msg(kind: u8) -> L1IncomingMessage {
        L1IncomingMessage {
            header: L1IncomingMessageHeader {
                kind,
                poster: Address::repeat_byte(kind),
                block_number: 100 + kind as u64,
                timestamp: 200,
                request_id: NilList(Some(B256::repeat_byte(kind))),
                l1_base_fee: U256::from(300),
            },
            l2msg: vec![kind, kind, kind].into(),
            legacy_batch_gas_cost: None,
            batch_data_stats: None,
        }
    }

    fn delayed(kind: u8) -> schema::DelayedInboxMessage {
        schema::DelayedInboxMessage {
            block_hash: B256::repeat_byte(0x10 + kind),
            before_inbox_acc: B256::repeat_byte(0x20 + kind),
            message: l1_msg(kind),
            parent_chain_block_number: 400 + kind as u64,
        }
    }

    fn batch_meta(n: u8) -> BatchMetadata {
        BatchMetadata {
            accumulator: B256::repeat_byte(n),
            message_count: n as u64 * 10,
            delayed_message_count: n as u64,
            parent_chain_block: n as u64 * 100,
        }
    }

    #[test]
    fn fresh_db_reads_are_none() {
        let db = mel_db();
        assert!(db.head_state().unwrap().is_none());
        assert!(db.head_state_block_num().unwrap().is_none());
        assert!(db.state(0).unwrap().is_none());
        assert!(db.delayed_message(0).unwrap().is_none());
        assert!(db.get(BatchMetadataAt(0)).unwrap().is_none());
    }

    #[test]
    fn save_state_roundtrips_and_sets_head() {
        let mut db = mel_db();
        let s = mel_state(50, 3, 2);
        db.save_state(&s).unwrap();

        assert_eq!(db.head_state_block_num().unwrap(), Some(50));
        assert_eq!(
            alloy_rlp::encode(db.head_state().unwrap().unwrap()),
            alloy_rlp::encode(&s)
        );
        assert_eq!(
            alloy_rlp::encode(db.state(50).unwrap().unwrap()),
            alloy_rlp::encode(&s)
        );
        assert!(db.state(51).unwrap().is_none());
    }

    #[test]
    fn delayed_messages_roundtrip_on_mel_side() {
        let mut db = mel_db();
        let msgs = [delayed(0), delayed(1), delayed(2)];
        db.save_delayed_messages(&mel_state(0, 0, 3), &msgs)
            .unwrap();

        for (i, msg) in msgs.iter().enumerate() {
            let got = db.delayed_message(i as u64).unwrap().expect("delayed");
            assert_eq!(alloy_rlp::encode(&got), alloy_rlp::encode(msg));
        }
        assert!(db.delayed_message(3).unwrap().is_none());
    }

    #[test]
    fn batch_metas_roundtrip_on_mel_side() {
        let mut db = mel_db();
        let metas = [batch_meta(0), batch_meta(1)];
        db.save_batch_metas(&mel_state(0, 2, 0), &metas).unwrap();

        for (i, meta) in metas.iter().enumerate() {
            let got = db.get(BatchMetadataAt(i as u64)).unwrap().expect("meta");
            assert_eq!(alloy_rlp::encode(&got), alloy_rlp::encode(meta));
        }
        assert!(db.get(BatchMetadataAt(2)).unwrap().is_none());
    }

    #[test]
    fn save_initial_mel_state_sets_boundary() {
        // Seed a legacy `s[0]` record, then open with no anchor yet.
        let mut cdb = ConsensusDb::open(MemoryKvStore::new()).unwrap();
        cdb.put(BatchMetadataAt(0), &batch_meta(7)).unwrap();
        let mut db = MelDb::open(cdb).unwrap();

        // No boundary yet: seq 0 routes to MEL `q` (empty), the legacy `s[0]` is ignored.
        assert!(db.get(BatchMetadataAt(0)).unwrap().is_none());

        // Setting the initial state at batch_count 1 flips seq 0 below the boundary -> legacy `s`.
        db.save_initial_mel_state(&mel_state(50, 1, 1)).unwrap();
        assert_eq!(
            alloy_rlp::encode(db.get(BatchMetadataAt(0)).unwrap().expect("legacy meta")),
            alloy_rlp::encode(batch_meta(7))
        );

        // seq 1 is at the boundary -> MEL `q`.
        db.save_batch_metas(&mel_state(0, 2, 0), &[batch_meta(1)])
            .unwrap();
        assert_eq!(
            alloy_rlp::encode(db.get(BatchMetadataAt(1)).unwrap().expect("mel meta")),
            alloy_rlp::encode(batch_meta(1))
        );
    }

    #[test]
    fn open_loads_boundary_from_disk() {
        let mut cdb = ConsensusDb::open(MemoryKvStore::new()).unwrap();
        cdb.put(schema::InitialMelStateBlockNum, &50u64).unwrap();
        cdb.put(schema::MelStateAt(50), &Rlp(mel_state(50, 1, 1)))
            .unwrap();
        cdb.put(BatchMetadataAt(0), &batch_meta(7)).unwrap(); // legacy `s[0]`
        let db = MelDb::open(cdb).unwrap();

        // Boundary was loaded from the anchor: seq 0 below it -> legacy `s`, seq 1 -> empty MEL
        // `q`.
        assert_eq!(
            alloy_rlp::encode(db.get(BatchMetadataAt(0)).unwrap().expect("legacy meta")),
            alloy_rlp::encode(batch_meta(7))
        );
        assert!(db.get(BatchMetadataAt(1)).unwrap().is_none());
    }

    #[test]
    fn reconstructs_legacy_delayed_message() {
        let mut cdb = ConsensusDb::open(MemoryKvStore::new()).unwrap();
        cdb.put(schema::InitialMelStateBlockNum, &50u64).unwrap();
        cdb.put(schema::MelStateAt(50), &Rlp(mel_state(50, 0, 1)))
            .unwrap();
        cdb.put(
            RlpDelayedMessageAt(0),
            &RlpDelayedMessage {
                accumulator: B256::repeat_byte(0xEE),
                message: l1_msg(0),
            },
        )
        .unwrap();
        cdb.put(ParentChainBlockAt(0), &ParentChainBlock(4242))
            .unwrap();
        let db = MelDb::open(cdb).unwrap();

        let got = db.delayed_message(0).unwrap().expect("reconstructed");
        assert_eq!(got.block_hash, B256::ZERO); // legacy did not store it
        assert_eq!(got.before_inbox_acc, B256::ZERO); // index 0 -> zero
        assert_eq!(got.parent_chain_block_number, 4242); // from `p`
        assert_eq!(
            alloy_rlp::encode(&got.message),
            alloy_rlp::encode(l1_msg(0))
        );
    }

    #[test]
    fn reconstructs_legacy_delayed_at_nonzero_index() {
        let mut cdb = ConsensusDb::open(MemoryKvStore::new()).unwrap();
        cdb.put(schema::InitialMelStateBlockNum, &50u64).unwrap();
        cdb.put(schema::MelStateAt(50), &Rlp(mel_state(50, 0, 2)))
            .unwrap();
        cdb.put(
            RlpDelayedMessageAt(0),
            &RlpDelayedMessage {
                accumulator: B256::repeat_byte(0xA0),
                message: l1_msg(0),
            },
        )
        .unwrap();
        cdb.put(
            RlpDelayedMessageAt(1),
            &RlpDelayedMessage {
                accumulator: B256::repeat_byte(0xA1),
                message: l1_msg(1),
            },
        )
        .unwrap();
        cdb.put(ParentChainBlockAt(1), &ParentChainBlock(777))
            .unwrap();
        let db = MelDb::open(cdb).unwrap();

        let got = db.delayed_message(1).unwrap().expect("reconstructed");
        // before_inbox_acc is the *previous* record's accumulator.
        assert_eq!(got.before_inbox_acc, B256::repeat_byte(0xA0));
        assert_eq!(got.parent_chain_block_number, 777); // from `p[1]`
        assert_eq!(got.block_hash, B256::ZERO);
        assert_eq!(
            alloy_rlp::encode(&got.message),
            alloy_rlp::encode(l1_msg(1))
        );
    }

    #[test]
    fn reconstructs_legacy_delayed_from_d_prefix() {
        let mut cdb = ConsensusDb::open(MemoryKvStore::new()).unwrap();
        cdb.put(schema::InitialMelStateBlockNum, &50u64).unwrap();
        cdb.put(schema::MelStateAt(50), &Rlp(mel_state(50, 0, 1)))
            .unwrap();
        // Only the older `d` (wire-format) record exists; no `e`, no `p`.
        cdb.put(
            LegacyDelayedMessageAt(0),
            &LegacyDelayedMessage {
                accumulator: B256::repeat_byte(0xDD),
                message: l1_msg(5),
            },
        )
        .unwrap();
        let db = MelDb::open(cdb).unwrap();

        let got = db.delayed_message(0).unwrap().expect("reconstructed");
        // `d` has no separate `p`, so the parent-chain block is the header's block number.
        assert_eq!(got.parent_chain_block_number, l1_msg(5).header.block_number);
        assert_eq!(
            alloy_rlp::encode(&got.message),
            alloy_rlp::encode(l1_msg(5))
        );
    }

    #[test]
    fn reconstructs_legacy_delayed_without_parent_chain_block() {
        let mut cdb = ConsensusDb::open(MemoryKvStore::new()).unwrap();
        cdb.put(schema::InitialMelStateBlockNum, &50u64).unwrap();
        cdb.put(schema::MelStateAt(50), &Rlp(mel_state(50, 0, 1)))
            .unwrap();
        // `e` present but no `p`: parent-chain block falls back to the header block number.
        cdb.put(
            RlpDelayedMessageAt(0),
            &RlpDelayedMessage {
                accumulator: B256::repeat_byte(0xEE),
                message: l1_msg(6),
            },
        )
        .unwrap();
        let db = MelDb::open(cdb).unwrap();

        let got = db.delayed_message(0).unwrap().expect("reconstructed");
        assert_eq!(got.parent_chain_block_number, l1_msg(6).header.block_number);
    }

    #[test]
    fn legacy_delayed_reconstruction_errors_on_missing_prev_record() {
        let mut cdb = ConsensusDb::open(MemoryKvStore::new()).unwrap();
        cdb.put(schema::InitialMelStateBlockNum, &50u64).unwrap();
        cdb.put(schema::MelStateAt(50), &Rlp(mel_state(50, 0, 2)))
            .unwrap();
        // index 1 is present, but index 0 (needed for before_inbox_acc) is missing.
        cdb.put(
            RlpDelayedMessageAt(1),
            &RlpDelayedMessage {
                accumulator: B256::repeat_byte(0xA1),
                message: l1_msg(1),
            },
        )
        .unwrap();
        cdb.put(ParentChainBlockAt(1), &ParentChainBlock(777))
            .unwrap();
        let db = MelDb::open(cdb).unwrap();

        assert!(db.delayed_message(1).is_err());
    }

    #[test]
    #[should_panic(expected = "batch_count < metas len")]
    fn save_batch_metas_guards_against_count_mismatch() {
        let mut db = mel_db();
        db.save_batch_metas(&mel_state(0, 1, 0), &[batch_meta(0), batch_meta(1)])
            .unwrap();
    }

    #[test]
    #[should_panic(expected = "delayed_messages_seen < batch len")]
    fn save_delayed_messages_guards_against_count_mismatch() {
        let mut db = mel_db();
        db.save_delayed_messages(&mel_state(0, 0, 1), &[delayed(0), delayed(1)])
            .unwrap();
    }
}
