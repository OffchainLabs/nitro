//! [`Database`] adapter over the libmdbx-backed [`MelDb`].

use std::sync::RwLock;

use arb_consensus_db::kv::LibmdbxKvStore;
use arb_mel_runner::{Database, MelRunnerError, Result};
use arb_mel_types::{BatchMetadata, DelayedInboxMessage, MelState};
use async_trait::async_trait;

type MelDb = arb_mel_db::MelDb<LibmdbxKvStore>;

/// Adapts the synchronous, `&mut`-writing [`MelDb`] to the runner's async,
/// shared-reference [`Database`] trait.
///
/// Calls block briefly on mdbx I/O instead of `spawn_blocking`: reads are
/// microseconds and the FSM is the only driver, so there's no executor to starve.
pub struct MelDatabase(RwLock<MelDb>);

impl MelDatabase {
    pub fn new(db: MelDb) -> Self {
        Self(RwLock::new(db))
    }
}

fn db_err(err: arb_mel_db::MelDbError) -> MelRunnerError {
    MelRunnerError::Database(err.to_string())
}

#[async_trait]
impl Database for MelDatabase {
    async fn get_head_mel_state(&self) -> Result<MelState> {
        self.0
            .read()
            .unwrap()
            .head_state()
            .map_err(db_err)?
            .ok_or_else(|| MelRunnerError::NotFound("head mel state".to_string()))
    }

    async fn get_head_mel_state_block_num(&self) -> Result<u64> {
        self.0
            .read()
            .unwrap()
            .head_state_block_num()
            .map_err(db_err)?
            .ok_or_else(|| MelRunnerError::NotFound("head mel state block number".to_string()))
    }

    async fn state(&self, parent_chain_block_number: u64) -> Result<MelState> {
        self.0
            .read()
            .unwrap()
            .state(parent_chain_block_number)
            .map_err(db_err)?
            .ok_or_else(|| {
                MelRunnerError::NotFound(format!("mel state at block {parent_chain_block_number}"))
            })
    }

    async fn fetch_delayed_message(&self, index: u64) -> Result<DelayedInboxMessage> {
        self.0
            .read()
            .unwrap()
            .delayed_message(index)
            .map_err(db_err)?
            .ok_or_else(|| MelRunnerError::NotFound(format!("delayed message {index}")))
    }

    async fn save_batch_metas(
        &self,
        state: &MelState,
        batch_metas: &[BatchMetadata],
    ) -> Result<()> {
        self.0
            .write()
            .unwrap()
            .save_batch_metas(state, batch_metas)
            .map_err(db_err)
    }

    async fn save_delayed_messages(
        &self,
        state: &MelState,
        delayed_messages: &[DelayedInboxMessage],
    ) -> Result<()> {
        self.0
            .write()
            .unwrap()
            .save_delayed_messages(state, delayed_messages)
            .map_err(db_err)
    }

    async fn save_state(&self, state: &MelState) -> Result<()> {
        self.0.write().unwrap().save_state(state).map_err(db_err)
    }

    async fn set_head_mel_state_block_num(&self, parent_chain_block_number: u64) -> Result<()> {
        self.0
            .write()
            .unwrap()
            .set_head_state_block_num(parent_chain_block_number)
            .map_err(db_err)
    }
}
