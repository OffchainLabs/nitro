use arb_consensus_db::{ConsensusDb, Result, kv};

pub mod schema;

#[derive(Debug)]
pub struct MelDb<S> {
    consensus_db: ConsensusDb<S>,
    initial_batch_count: u64,
}

impl<S: kv::KvStore> MelDb<S> {
    pub fn open(consensus_db: ConsensusDb<S>) -> Result<Self> {
        let initial_batch_count = 0; // TODO(NIT-5118): lookup value
        Ok(MelDb {
            consensus_db,
            initial_batch_count,
        })
    }

    pub fn get<K: schema::MelDbKey>(&self, key: K) -> Result<Option<K::StoredValue>> {
        self.consensus_db
            .get_at_key(&key.mel_key(self.initial_batch_count))
    }

    pub fn put<K: schema::MelDbKey>(&mut self, key: K, value: &K::StoredValue) -> Result<()> {
        self.consensus_db
            .put_at_key(&key.mel_key(self.initial_batch_count), value)
    }
}
