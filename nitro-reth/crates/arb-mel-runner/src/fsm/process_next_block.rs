//! The `ProcessingNextBlock` FSM phase: fetch the next block, detect reorgs, and
//! extract messages.
//!
//! Ports nitro's `arbnode/mel/runner/process_next_block.go`.

use std::{sync::atomic::Ordering, time::Duration};

use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use alloy_rpc_types_eth::{Log, Transaction};
use arb_da_provider_client::DaReaderSource;
use arb_mel::{DelayedInboxMessage, DelayedMessageDB, LogsFetcher, MelResult, MelState, TxFetcher};
use arb_parent_chain_client::ParentChainReader;

use crate::{
    MelRunnerError, Result, config::ReadMode, consumer::MessageConsumer, database::Database,
    extractor::MessageExtractor, fsm::FsmState,
};

/// The `ProcessingNextBlock` FSM phase.
#[async_trait::async_trait]
pub(crate) trait ProcessingNextBlock {
    /// Fetches the next parent-chain block, detects reorgs, runs extraction via
    /// `arb-mel`, and transitions to `SavingMessages` (or `Reorging`). Mirrors
    /// `ProcessNextBlock`.
    async fn process_next_block(&mut self) -> (Duration, Result<()>);
}

#[async_trait::async_trait]
impl<P, D, C, S> ProcessingNextBlock for MessageExtractor<P, D, C, S>
where
    P: ParentChainReader,
    D: Database,
    C: MessageConsumer,
    S: DaReaderSource,
{
    async fn process_next_block(&mut self) -> (Duration, Result<()>) {
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
            .parent_chain_reader
            .header_by_number(BlockNumberOrTag::Number(pre_number + 1))
            .await
        {
            Ok(Some(h)) => h,
            Ok(None) => {
                // The next block isn't available yet: we've reached the tip.
                self.caught_up = true;
                return (retry, Ok(()));
            }
            Err(e) => return (retry, Err(e.into())),
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

        // Extract via arb-mel. Logs/tx/delayed fetchers are nil until the
        // logs-and-headers fetcher lands (empty logs -> no batches -> no messages).
        let result = match &self.fsm_state {
            FsmState::ProcessingNextBlock { mel_state, .. } => {
                arb_mel::extract_messages(
                    mel_state,
                    &header.inner,
                    self.data_providers.as_ref(),
                    &NilDelayedMessageDb,
                    &NilLogsFetcher,
                    &NilTxFetcher,
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

// Empty ("nil") collaborators for `arb_mel::extract_messages`, until the
// logs-and-headers fetcher lands.

struct NilLogsFetcher;

impl LogsFetcher for NilLogsFetcher {
    fn logs_for_block_hash(&self, _block_hash: B256) -> MelResult<Vec<Log>> {
        Ok(Vec::new())
    }

    fn logs_for_tx_index(&self, _block_hash: B256, _tx_index: u64) -> MelResult<Vec<Log>> {
        Ok(Vec::new())
    }
}

/// Never invoked while `NilLogsFetcher` yields no logs.
struct NilTxFetcher;

impl TxFetcher for NilTxFetcher {
    type Transaction = Transaction;

    fn transaction_by_log(&self, _log: &Log) -> MelResult<Self::Transaction> {
        Err(arb_mel::MelError::Unknown)
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
