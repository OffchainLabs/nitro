//! The message-extraction interface and its real adapter.
//!
//! Wraps the `arb-mel` crate's [`arb_mel::extract_messages`] — the algorithm that
//! reads a parent-chain block and derives L2 messages, delayed messages, and
//! batch metadata. That function is **synchronous** and reads through sync
//! collaborator traits ([`arb_mel::LogsFetcher`], [`arb_mel::TxFetcher`],
//! [`arb_mel::DelayedMessageDB`]), while the runner's [`ParentChainReader`] is
//! async. [`RpcMessageExtraction`] bridges the two: it async-prefetches the
//! block's logs and the transactions that emitted them, then feeds sync
//! in-memory fetchers into `extract_messages` (nitro's `logsAndHeadersFetcher`
//! pattern).
//!
//! [`MockMessageExtraction`] keeps the FSM unit-testable without a parent chain.

use std::collections::HashMap;
use std::sync::{Arc, Mutex};

use alloy_primitives::B256;
use alloy_rpc_types_eth::{Filter, Header, Log, Transaction};
use arb_mel::{
    DelayedInboxMessage, DelayedMessageDB, ExtractionOutput, LogsFetcher, MelResult, MelState,
    TxFetcher,
};
use arb_parent_chain_client::ParentChainReader;
use async_trait::async_trait;

use crate::{MelError, Result};

/// Extracts messages from a single parent-chain block.
#[async_trait]
pub trait MessageExtraction: Send + Sync {
    /// Applies `parent_header` on top of `input_state`, producing the next state
    /// and the messages/delayed-messages/batch-metadata derived from the block.
    async fn extract_messages(
        &self,
        input_state: MelState,
        parent_header: &Header,
    ) -> Result<ExtractionOutput>;
}

/// The production [`MessageExtraction`]: drives `arb_mel::extract_messages` over
/// data prefetched from a [`ParentChainReader`].
pub struct RpcMessageExtraction {
    parent_chain_reader: Arc<dyn ParentChainReader>,
}

impl RpcMessageExtraction {
    /// Creates an extractor that reads from `parent_chain_reader`.
    pub fn new(parent_chain_reader: Arc<dyn ParentChainReader>) -> Self {
        Self {
            parent_chain_reader,
        }
    }
}

#[async_trait]
impl MessageExtraction for RpcMessageExtraction {
    async fn extract_messages(
        &self,
        input_state: MelState,
        parent_header: &Header,
    ) -> Result<ExtractionOutput> {
        let block_hash = parent_header.hash;

        // Prefetch every log in the block, plus the transaction that emitted each
        // one, so the sync fetchers below can serve `extract_messages` without IO.
        let logs = self
            .parent_chain_reader
            .filter_logs(&Filter::new().at_block_hash(block_hash))
            .await?;
        let mut txs_by_hash: HashMap<B256, Transaction> = HashMap::new();
        for log in &logs {
            let Some(tx_hash) = log.transaction_hash else {
                continue;
            };
            if txs_by_hash.contains_key(&tx_hash) {
                continue;
            }
            if let Some(tx) = self
                .parent_chain_reader
                .transaction_by_hash(tx_hash)
                .await?
            {
                txs_by_hash.insert(tx_hash, tx);
            }
        }

        let logs_fetcher = PrefetchedLogs { block_hash, logs };
        let tx_fetcher = PrefetchedTxs { txs_by_hash };
        let out = arb_mel::extract_messages(
            input_state,
            &parent_header.inner,
            &DelayedMsgDbStub,
            &logs_fetcher,
            &tx_fetcher,
        )?;
        Ok(out)
    }
}

/// Serves prefetched block logs to `arb_mel::extract_messages`.
struct PrefetchedLogs {
    block_hash: B256,
    logs: Vec<Log>,
}

impl LogsFetcher for PrefetchedLogs {
    fn logs_for_block_hash(&self, block_hash: B256) -> MelResult<Vec<Log>> {
        Ok(if block_hash == self.block_hash {
            self.logs.clone()
        } else {
            Vec::new()
        })
    }

    fn logs_for_tx_index(&self, block_hash: B256, tx_index: u64) -> MelResult<Vec<Log>> {
        if block_hash != self.block_hash {
            return Ok(Vec::new());
        }
        Ok(self
            .logs
            .iter()
            .filter(|log| log.transaction_index == Some(tx_index))
            .cloned()
            .collect())
    }
}

/// Serves prefetched transactions, keyed by the emitting log's tx hash.
struct PrefetchedTxs {
    txs_by_hash: HashMap<B256, Transaction>,
}

impl TxFetcher for PrefetchedTxs {
    type Transaction = Transaction;

    fn transaction_by_log(&self, log: &Log) -> MelResult<Self::Transaction> {
        let tx_hash = log.transaction_hash.ok_or(arb_mel::MelError::Unknown)?;
        self.txs_by_hash
            .get(&tx_hash)
            .cloned()
            .ok_or(arb_mel::MelError::Unknown)
    }
}

/// Placeholder [`DelayedMessageDB`]: `extract_messages` does not read delayed
/// messages in the current `arb-mel` happy path (its accumulate/move steps are
/// stubs). Back this with [`crate::Database`] once `arb-mel` wires delayed-message
/// accumulation.
struct DelayedMsgDbStub;

impl DelayedMessageDB for DelayedMsgDbStub {
    fn read_delayed_message(
        &self,
        _state: &MelState,
        _index: u64,
    ) -> MelResult<Option<DelayedInboxMessage>> {
        Ok(None)
    }
}

/// A [`MessageExtraction`] for tests.
///
/// Advances the state by one block (setting the block number and hashes from
/// `parent_header`) and produces no messages — enough to drive the FSM's happy
/// path. Inject an error with [`Self::set_error`].
#[derive(Debug, Default)]
pub struct MockMessageExtraction {
    error: Mutex<Option<String>>,
}

impl MockMessageExtraction {
    /// Creates a mock with default (state-advancing) behavior.
    pub fn new() -> Self {
        Self::default()
    }

    /// Injects (or clears) an error returned by [`MessageExtraction::extract_messages`].
    pub fn set_error(&self, err: Option<impl Into<String>>) {
        *self.error.lock().unwrap() = err.map(Into::into);
    }
}

#[async_trait]
impl MessageExtraction for MockMessageExtraction {
    async fn extract_messages(
        &self,
        input_state: MelState,
        parent_header: &Header,
    ) -> Result<ExtractionOutput> {
        if let Some(msg) = self.error.lock().unwrap().clone() {
            return Err(MelError::Extraction(msg));
        }
        let mut post_state = input_state;
        post_state.parent_chain_prev_block_hash = post_state.parent_chain_block_hash;
        post_state.parent_chain_block_number = parent_header.inner.number;
        post_state.parent_chain_block_hash = parent_header.hash;
        Ok(ExtractionOutput {
            post_state,
            messages: Vec::new(),
            delayed_messages: Vec::new(),
            batch_metas: Vec::new(),
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use arb_parent_chain_client::MockParentChainReader;

    #[tokio::test]
    async fn rpc_extraction_advances_state_for_empty_block() {
        // Drives the full async-prefetch -> sync-fetcher -> arb_mel::extract_messages
        // path. arb-mel's mel-config lookup is currently a stub that only validates
        // at block 0, so the end-to-end wiring is exercised against a genesis-style
        // empty block (default header: number 0, zero parent hash; the default input
        // state links to it via a zero parent-chain block hash).
        let header = Header::default();
        let input_state = MelState::default();

        // No logs registered -> empty block.
        let reader = Arc::new(MockParentChainReader::new());
        let extraction = RpcMessageExtraction::new(reader);

        let out = extraction
            .extract_messages(input_state, &header)
            .await
            .unwrap();

        assert!(out.messages.is_empty());
        assert!(out.delayed_messages.is_empty());
        assert!(out.batch_metas.is_empty());
        assert_eq!(out.post_state.parent_chain_block_number, 0);
        // arb-mel records the canonical (computed) hash of the consensus header.
        assert_eq!(
            out.post_state.parent_chain_block_hash,
            header.inner.hash_slow()
        );
    }

    #[tokio::test]
    async fn mock_extraction_error_injection() {
        let m = MockMessageExtraction::new();
        m.set_error(Some("boom"));
        // `ExtractionOutput` isn't `Debug`, so match rather than `unwrap_err`.
        let err = match m
            .extract_messages(MelState::default(), &Header::default())
            .await
        {
            Err(e) => e,
            Ok(_) => panic!("expected an injected error"),
        };
        assert!(err.to_string().contains("boom"));
        m.set_error(None::<String>);
        assert!(
            m.extract_messages(MelState::default(), &Header::default())
                .await
                .is_ok()
        );
    }
}
