//! The `ProcessingNextBlock` FSM phase: fetch the next block, detect reorgs, and
//! extract messages.
//!
//! Ports nitro's `arbnode/mel/runner/process_next_block.go`.

use std::{
    sync::{Arc, atomic::Ordering},
    time::Duration,
};

use alloy_eips::BlockNumberOrTag;
use alloy_rpc_types_eth::{Log, Transaction};
use arb_da_provider_client::DaReaderSource;
use arb_mel::{DelayedInboxMessage, DelayedMessageDB, MelError, MelResult, MelState, TxFetcher};
use arb_parent_chain_client::ParentChainReader;

use crate::{
    MelRunnerError, Result, config::ReadMode, consumer::MessageConsumer, database::Database,
    extractor::MessageExtractor, fsm::FsmState,
};

/// How far behind the parent-chain tip still counts as caught up, tolerating the
/// chain advancing between the "next block" and "latest block" reads. Mirrors
/// nitro's hard-coded 5-block tolerance.
const CAUGHT_UP_TOLERANCE_BLOCKS: u64 = 5;

impl<P, D, C, S> MessageExtractor<P, D, C, S>
where
    P: ParentChainReader,
    D: Database,
    C: MessageConsumer,
    S: DaReaderSource,
{
    /// The `ProcessingNextBlock` FSM phase: fetches the next parent-chain block,
    /// detects reorgs, runs extraction via `arb-mel`, and transitions to
    /// `SavingMessages` (or `Reorging`). Mirrors `ProcessNextBlock`.
    pub(crate) async fn process_next_block(&mut self) -> (Duration, Result<()>) {
        let retry = self.config.retry_interval;
        let (pre_number, pre_hash, pre_msg_count, prev_was_reorg) = match &self.fsm_state {
            FsmState::ProcessingNextBlock {
                mel_state,
                prev_step_was_reorg,
            } => (
                mel_state.parent_chain_block_number,
                mel_state.parent_chain_block_hash,
                mel_state.msg_count,
                *prev_step_was_reorg,
            ),
            _ => {
                return (
                    retry,
                    Err(MelRunnerError::InvalidState(
                        "expected ProcessingNextBlock".to_string(),
                    )),
                );
            }
        };

        // Read-mode gate: for safe/finalized, don't get ahead of the confirmed tip.
        if self.config.read_mode != ReadMode::Latest
            && pre_number + 1 > self.last_block_to_read.load(Ordering::Relaxed)
        {
            return (retry, Ok(()));
        }

        let header = match self
            .logs_and_headers_prefetcher
            .get_header_by_number(pre_number + 1)
            .await
        {
            Ok(Some(h)) => h,
            Ok(None) => {
                // Next block not posted yet: retry, no error. Catch-up is a
                // latest-only signal; a fresh latest read within tolerance of our
                // head guards against a transient miss for a block that exists.
                if !self.caught_up && self.config.read_mode == ReadMode::Latest {
                    match self
                        .parent_chain_reader
                        .header_by_number(BlockNumberOrTag::Latest)
                        .await
                    {
                        Ok(Some(latest)) => {
                            let behind = latest.inner.number.checked_sub(pre_number);
                            if behind.is_some_and(|behind| behind <= CAUGHT_UP_TOLERANCE_BLOCKS) {
                                self.caught_up = true;
                            }
                        }
                        Ok(None) => {}
                        Err(e) => tracing::error!(
                            error = %e,
                            "failed to fetch parent-chain latest block to determine MEL catch-up",
                        ),
                    }
                }
                return (retry, Ok(()));
            }
            Err(e) => return (retry, Err(e)),
        };

        // Reorg detection: the next block must build on our current head hash.
        if header.inner.parent_hash != pre_hash {
            let mel_state = match std::mem::replace(&mut self.fsm_state, FsmState::Start) {
                FsmState::ProcessingNextBlock { mel_state, .. } => mel_state,
                _ => unreachable!("state checked above"),
            };
            self.fsm_state = FsmState::Reorging { mel_state };
            return (Duration::ZERO, Ok(()));
        }

        // After a reorg, notify the consumer so its MEL validator can rewind
        // before we extract on the new fork.
        if prev_was_reorg
            && let Err(e) = self
                .msg_consumer
                .reorged_to_parent_chain_block(pre_number)
                .await
        {
            return (retry, Err(e));
        }

        // Prefetch this range's logs, then extract via arb-mel. The prefetcher
        // serves the logs; txs are fetched on demand by `TxByLogFetcher`. The
        // delayed-msg db is nil until a runner-DB-backed one lands.
        let result = match &self.fsm_state {
            FsmState::ProcessingNextBlock { mel_state, .. } => {
                if let Err(e) = self.logs_and_headers_prefetcher.fetch(mel_state).await {
                    return (retry, Err(e));
                }
                let tx_fetcher = TxByLogFetcher {
                    parent_chain_reader: Arc::clone(&self.parent_chain_reader),
                };
                arb_mel::extract_messages(
                    mel_state,
                    &header.inner,
                    self.data_providers.as_ref(),
                    &NilDelayedMessageDb,
                    &self.logs_and_headers_prefetcher,
                    &tx_fetcher,
                )
                .await
            }
            _ => unreachable!("state checked above"),
        };
        let output = match result {
            Ok(o) => o,
            Err(e) => return (retry, Err(e.into())),
        };

        self.fsm_state = FsmState::SavingMessages {
            pre_state_msg_count: pre_msg_count,
            post_state: output.post_state,
            messages: output.messages,
            delayed_messages: output.delayed_messages,
            batch_metas: output.batch_metas,
        };
        (Duration::ZERO, Ok(()))
    }
}

struct NilDelayedMessageDb;

impl DelayedMessageDB for NilDelayedMessageDb {
    fn read_delayed_message(
        &self,
        _state: &MelState,
        _index: u64,
    ) -> MelResult<Option<DelayedInboxMessage>> {
        Ok(None)
    }
}

/// Fetches the tx that emitted a log via a direct `transaction_by_hash` RPC,
/// mirroring nitro's `txByLogFetcher`.
struct TxByLogFetcher<P> {
    parent_chain_reader: Arc<P>,
}

#[async_trait::async_trait]
impl<P: ParentChainReader> TxFetcher for TxByLogFetcher<P> {
    type Transaction = Transaction;

    async fn transaction_by_log(&self, log: &Log) -> MelResult<Self::Transaction> {
        let tx_hash = log.transaction_hash.ok_or(MelError::UnexpectedLogType)?;
        self.parent_chain_reader
            .transaction_by_hash(tx_hash)
            .await
            .map_err(|e| MelError::TransactionFetch(e.to_string()))?
            .ok_or_else(|| MelError::TransactionFetch(format!("transaction {tx_hash} not found")))
    }
}

#[cfg(test)]
mod tests {
    use alloy_consensus::{
        SignableTransaction, TxEip1559, TxEnvelope,
        transaction::{Recovered, TransactionInfo},
    };
    use alloy_eips::eip2718::Encodable2718;
    use alloy_primitives::{Address, B256, Bytes, LogData, Signature, U256};
    use arb_parent_chain_client::MockParentChainReader;

    use super::*;

    fn mk_tx(nonce: u64) -> Transaction {
        let signed = TxEip1559 {
            nonce,
            ..Default::default()
        }
        .into_signed(Signature::new(U256::ZERO, U256::ZERO, false));
        let recovered = Recovered::new_unchecked(TxEnvelope::Eip1559(signed), Address::ZERO);
        Transaction::from_transaction(
            recovered,
            TransactionInfo {
                hash: None,
                index: None,
                block_hash: None,
                block_number: None,
                base_fee: None,
            },
        )
    }

    fn log_with_tx(tx_hash: B256) -> Log {
        Log {
            inner: alloy_primitives::Log {
                address: Address::ZERO,
                data: LogData::new_unchecked(vec![], Bytes::new()),
            },
            block_hash: Some(B256::repeat_byte(0x01)),
            block_number: Some(1),
            block_timestamp: None,
            transaction_hash: Some(tx_hash),
            transaction_index: Some(0),
            log_index: None,
            removed: false,
        }
    }

    #[tokio::test]
    async fn tx_by_log_fetches_via_rpc() {
        let tx = mk_tx(1);
        let tx_hash = tx.inner.trie_hash();
        let mut mock = MockParentChainReader::new();
        mock.with_transaction(tx);
        let fetcher = TxByLogFetcher {
            parent_chain_reader: Arc::new(mock),
        };

        let got = fetcher
            .transaction_by_log(&log_with_tx(tx_hash))
            .await
            .unwrap();
        assert_eq!(got.inner.trie_hash(), tx_hash);

        // A hash the reader doesn't know errors.
        assert!(
            fetcher
                .transaction_by_log(&log_with_tx(B256::repeat_byte(0x99)))
                .await
                .is_err()
        );
    }
}
