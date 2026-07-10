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

    pub fn get<T: schema::MelStoredValue>(&self, pos: u64) -> Result<Option<T>> {
        self.consensus_db
            .get_at_key(&schema::key::<T>(pos, self.initial_batch_count))
    }
}
