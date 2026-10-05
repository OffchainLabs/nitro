//! The MEL database interface.
//!
//! In nitro this is a concrete `*Database` over an `ethdb.KeyValueStore`. Here it
//! is a trait so the runner can be driven against a mock; the real
//! (KV/schema/RLP) implementation is ported separately. Only the methods the
//! runner actually calls are included. Types come from `arb-mel`.

use std::{collections::HashMap, sync::Mutex};

use arb_mel::{BatchMetadata, DelayedInboxMessage, MelState};
use async_trait::async_trait;

use crate::{MelRunnerError, Result};

/// Persistence for MEL state, delayed messages, and batch metadata.
///
/// A lookup that finds nothing returns [`MelRunnerError::NotFound`].
#[async_trait]
pub trait Database: Send + Sync {
    /// Returns the current head MEL state.
    async fn get_head_mel_state(&self) -> Result<MelState>;
    /// Returns the parent-chain block number of the head MEL state.
    async fn get_head_mel_state_block_num(&self) -> Result<u64>;
    /// Returns the MEL state at the given parent-chain block number.
    async fn state(&self, parent_chain_block_number: u64) -> Result<MelState>;
    /// Returns the delayed message at the given index.
    async fn fetch_delayed_message(&self, index: u64) -> Result<DelayedInboxMessage>;
    /// Persists batch metadata produced for `state`.
    async fn save_batch_metas(&self, state: &MelState, batch_metas: &[BatchMetadata])
    -> Result<()>;
    /// Persists delayed messages observed for `state`.
    async fn save_delayed_messages(
        &self,
        state: &MelState,
        delayed_messages: &[DelayedInboxMessage],
    ) -> Result<()>;
    /// Persists `state` as the new head state.
    async fn save_state(&self, state: &MelState) -> Result<()>;
    /// Sets the head MEL state block number (used when reorging to a batch).
    async fn set_head_mel_state_block_num(&self, parent_chain_block_number: u64) -> Result<()>;
}

/// An in-memory [`Database`] for tests.
///
/// Behaves like a small MEL store: [`Self::save_state`] records the state (by
/// block number), makes it the head, and updates the head block number, so a
/// test can mutate the store at runtime through a shared handle exactly as
/// nitro's tests call `melDB.SaveState`. Data can also be pre-loaded with the
/// `with_*` builders. Set an error with [`Self::set_error`] to make every method
/// fail (mirroring the Go mocks' `returnErr` knob).
///
/// Not `Debug`: `arb_mel::MelState` doesn't implement it.
#[derive(Default)]
pub struct MockDatabase {
    inner: Mutex<Inner>,
    error: Mutex<Option<String>>,
}

#[derive(Default)]
struct Inner {
    head: Option<MelState>,
    head_block_num: Option<u64>,
    states: HashMap<u64, MelState>,
}

impl MockDatabase {
    /// Creates an empty database.
    pub fn new() -> Self {
        Self::default()
    }

    /// Pre-loads `state` as the head state (also indexed by its block number).
    pub fn with_head(&mut self, state: MelState) -> &mut Self {
        let inner = self.inner.get_mut().unwrap();
        inner.head_block_num = Some(state.parent_chain_block_number);
        inner
            .states
            .insert(state.parent_chain_block_number, state.clone());
        inner.head = Some(state);
        self
    }

    /// Pre-loads a state at `parent_chain_block_number` without touching the head.
    pub fn with_state_at(&mut self, parent_chain_block_number: u64, state: MelState) -> &mut Self {
        self.inner
            .get_mut()
            .unwrap()
            .states
            .insert(parent_chain_block_number, state);
        self
    }

    /// Injects (or clears) an error returned by every method.
    pub fn set_error(&self, err: Option<impl Into<String>>) {
        *self.error.lock().unwrap() = err.map(Into::into);
    }

    fn check_error(&self) -> Result<()> {
        match self.error.lock().unwrap().clone() {
            Some(msg) => Err(MelRunnerError::Database(msg)),
            None => Ok(()),
        }
    }
}

#[async_trait]
impl Database for MockDatabase {
    async fn get_head_mel_state(&self) -> Result<MelState> {
        self.check_error()?;
        self.inner
            .lock()
            .unwrap()
            .head
            .clone()
            .ok_or_else(|| MelRunnerError::NotFound("head mel state".to_string()))
    }

    async fn get_head_mel_state_block_num(&self) -> Result<u64> {
        self.check_error()?;
        self.inner
            .lock()
            .unwrap()
            .head_block_num
            .ok_or_else(|| MelRunnerError::NotFound("head mel state block number".to_string()))
    }

    async fn state(&self, parent_chain_block_number: u64) -> Result<MelState> {
        self.check_error()?;
        self.inner
            .lock()
            .unwrap()
            .states
            .get(&parent_chain_block_number)
            .cloned()
            .ok_or_else(|| {
                MelRunnerError::NotFound(format!("mel state at block {parent_chain_block_number}"))
            })
    }

    async fn fetch_delayed_message(&self, index: u64) -> Result<DelayedInboxMessage> {
        // `arb_mel::DelayedInboxMessage` isn't `Clone`, so the mock doesn't store
        // them; the FSM tests don't exercise this path (preimage rebuild is deferred).
        self.check_error()?;
        Err(MelRunnerError::NotFound(format!("delayed message {index}")))
    }

    async fn save_batch_metas(
        &self,
        _state: &MelState,
        _batch_metas: &[BatchMetadata],
    ) -> Result<()> {
        self.check_error()
    }

    async fn save_delayed_messages(
        &self,
        _state: &MelState,
        _delayed_messages: &[DelayedInboxMessage],
    ) -> Result<()> {
        self.check_error()
    }

    async fn save_state(&self, state: &MelState) -> Result<()> {
        self.check_error()?;
        let mut inner = self.inner.lock().unwrap();
        inner.head_block_num = Some(state.parent_chain_block_number);
        inner
            .states
            .insert(state.parent_chain_block_number, state.clone());
        inner.head = Some(state.clone());
        Ok(())
    }

    async fn set_head_mel_state_block_num(&self, parent_chain_block_number: u64) -> Result<()> {
        self.check_error()?;
        self.inner.lock().unwrap().head_block_num = Some(parent_chain_block_number);
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn state_at(n: u64) -> MelState {
        MelState {
            parent_chain_block_number: n,
            ..Default::default()
        }
    }

    #[tokio::test]
    async fn empty_head_is_not_found() {
        let db = MockDatabase::new();
        assert!(matches!(
            db.get_head_mel_state().await,
            Err(MelRunnerError::NotFound(_))
        ));
        assert!(matches!(
            db.get_head_mel_state_block_num().await,
            Err(MelRunnerError::NotFound(_))
        ));
    }

    #[tokio::test]
    async fn save_state_becomes_head_and_is_indexed() {
        let db = MockDatabase::new();
        db.save_state(&state_at(7)).await.unwrap();
        assert_eq!(db.get_head_mel_state_block_num().await.unwrap(), 7);
        // MelState isn't PartialEq; check the identifying field.
        assert_eq!(
            db.get_head_mel_state()
                .await
                .unwrap()
                .parent_chain_block_number,
            7
        );
        assert_eq!(db.state(7).await.unwrap().parent_chain_block_number, 7);
    }

    #[tokio::test]
    async fn with_state_at_does_not_set_head() {
        let mut db = MockDatabase::new();
        db.with_state_at(3, state_at(3));
        assert_eq!(db.state(3).await.unwrap().parent_chain_block_number, 3);
        assert!(matches!(
            db.get_head_mel_state().await,
            Err(MelRunnerError::NotFound(_))
        ));
    }

    #[tokio::test]
    async fn injected_error_fails_every_read() {
        let db = MockDatabase::new();
        db.save_state(&state_at(1)).await.unwrap();
        db.set_error(Some("boom"));
        assert!(matches!(
            db.get_head_mel_state().await,
            Err(MelRunnerError::Database(_))
        ));
        db.set_error(None::<String>);
        assert!(db.get_head_mel_state().await.is_ok());
    }
}
