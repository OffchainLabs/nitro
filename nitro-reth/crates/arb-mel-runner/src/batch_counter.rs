//! The sequencer batch-count fetcher interface.
//!
//! Ports `melrunner.SequencerBatchCountFetcher`: an optional collaborator that
//! queries the on-chain sequencer-inbox batch count at a given parent-chain
//! block. Used by the (deferred) sync-progress read API; kept here so the
//! collaborator exists as an interface and can be mocked.

use std::sync::Mutex;

use async_trait::async_trait;

use crate::{MelError, Result};

/// Queries the sequencer-inbox batch count at a parent-chain block.
#[async_trait]
pub trait SequencerBatchCountFetcher: Send + Sync {
    /// Returns the batch count as of parent-chain block `block_num`.
    async fn get_batch_count(&self, block_num: u64) -> Result<u64>;
}

/// A [`SequencerBatchCountFetcher`] for tests returning a fixed count.
#[derive(Debug, Default)]
pub struct MockSequencerBatchCountFetcher {
    count: u64,
    error: Mutex<Option<String>>,
}

impl MockSequencerBatchCountFetcher {
    /// Creates a fetcher that returns `count`.
    pub fn new(count: u64) -> Self {
        Self {
            count,
            error: Mutex::new(None),
        }
    }

    /// Injects (or clears) an error returned by [`SequencerBatchCountFetcher::get_batch_count`].
    pub fn set_error(&self, err: Option<impl Into<String>>) {
        *self.error.lock().unwrap() = err.map(Into::into);
    }
}

#[async_trait]
impl SequencerBatchCountFetcher for MockSequencerBatchCountFetcher {
    async fn get_batch_count(&self, _block_num: u64) -> Result<u64> {
        match self.error.lock().unwrap().clone() {
            Some(msg) => Err(MelError::Database(msg)),
            None => Ok(self.count),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn returns_fixed_count() {
        let f = MockSequencerBatchCountFetcher::new(5);
        assert_eq!(f.get_batch_count(100).await.unwrap(), 5);
    }

    #[tokio::test]
    async fn honors_injected_error() {
        let f = MockSequencerBatchCountFetcher::new(5);
        f.set_error(Some("rpc down"));
        assert!(f.get_batch_count(100).await.is_err());
    }
}
