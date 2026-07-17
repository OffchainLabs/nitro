//! The message extractor: the FSM driver.
//!
//! Ports the `MessageExtractor` struct and its FSM handlers from nitro's
//! `arbnode/mel/runner/` (`mel.go`, `initialize.go`, `process_next_block.go`,
//! `save_messages.go`, `reorg.go`). In the target architecture this runner is a
//! standalone RPC provider (serving nitro's `MELNative`); the [`MessageConsumer`]
//! it pushes to is the remote nitro node.
//!
//! [`MessageExtractor::act`] ticks the FSM once (the unit tests drive it
//! directly, as nitro's do); [`MessageExtractor::run`] drives it in a loop,
//! replacing nitro's `stopwaiter`.

use std::sync::Arc;
use std::sync::atomic::{AtomicU64, Ordering};
use std::time::Duration;

use alloy_eips::BlockNumberOrTag;
use alloy_primitives::B256;
use alloy_rpc_types_eth::{Log, Transaction};
use arb_da_provider_client::DaReaderSource;
use arb_mel::{DelayedInboxMessage, DelayedMessageDB, LogsFetcher, MelResult, MelState, TxFetcher};
use arb_parent_chain_client::ParentChainReader;

use crate::{
    MelRunnerError, Result,
    batch_counter::SequencerBatchCountFetcher,
    config::MessageExtractionConfig,
    consumer::MessageConsumer,
    database::Database,
    fsm::{FsmState, FsmStateKind},
    types::RollupAddresses,
};

/// Reads parent-chain blocks one by one and turns them into L2 messages.
///
/// Holds its collaborators as `Arc<dyn ...>` trait objects so they can be mocked
/// in tests and swapped for real implementations later.
pub struct MessageExtractor {
    config: MessageExtractionConfig,
    parent_chain_reader: Arc<dyn ParentChainReader>,
    addrs: RollupAddresses,
    db: Arc<dyn Database>,
    msg_consumer: Arc<dyn MessageConsumer>,
    data_providers: Arc<dyn DaReaderSource>,
    seq_batch_counter: Option<Arc<dyn SequencerBatchCountFetcher>>,
    fsm_state: FsmState,
    stuck_count: u64,
    last_block_to_read: AtomicU64,
    caught_up: bool,
}

impl MessageExtractor {
    /// Creates an extractor in the `Start` state. Mirrors `NewMessageExtractor`.
    ///
    /// Unlike nitro (which sets the consumer separately after construction), the
    /// message consumer is a required argument here.
    pub fn new(
        config: MessageExtractionConfig,
        parent_chain_reader: Arc<dyn ParentChainReader>,
        addrs: RollupAddresses,
        db: Arc<dyn Database>,
        msg_consumer: Arc<dyn MessageConsumer>,
        data_providers: Arc<dyn DaReaderSource>,
        seq_batch_counter: Option<Arc<dyn SequencerBatchCountFetcher>>,
    ) -> Self {
        Self {
            config,
            parent_chain_reader,
            addrs,
            db,
            msg_consumer,
            data_providers,
            seq_batch_counter,
            fsm_state: FsmState::Start,
            stuck_count: 0,
            last_block_to_read: AtomicU64::new(0),
            caught_up: false,
        }
    }

    /// The current FSM state discriminant. Mirrors `CurrentFSMState`.
    pub fn current_fsm_state(&self) -> FsmStateKind {
        self.fsm_state.kind()
    }

    /// Whether the extractor has reached the parent-chain tip.
    pub fn caught_up(&self) -> bool {
        self.caught_up
    }

    /// The rollup contract addresses this extractor was configured with.
    pub fn rollup_addresses(&self) -> &RollupAddresses {
        &self.addrs
    }

    /// Whether an optional sequencer batch-count fetcher is configured.
    pub fn has_sequencer_batch_counter(&self) -> bool {
        self.seq_batch_counter.is_some()
    }

    /// The DA provider passed into extraction for recovering off-chain batch payloads.
    pub fn data_providers(&self) -> &Arc<dyn DaReaderSource> {
        &self.data_providers
    }

    /// Whether the FSM has been stuck at one state past the stall tolerance.
    pub fn is_stuck(&self) -> bool {
        self.stuck_count > self.config.stall_tolerance
    }

    /// Ticks the FSM once, performing the action for the current state. Mirrors `Act`.
    ///
    /// Returns how long to wait before the next tick, and the result of this one.
    /// A non-zero delay with `Ok` means "retry the same state later"; an `Err`
    /// leaves the state unchanged so the caller retries after the delay.
    pub async fn act(&mut self) -> (Duration, Result<()>) {
        match self.current_fsm_state() {
            FsmStateKind::Start => self.initialize().await,
            FsmStateKind::ProcessingNextBlock => self.process_next_block().await,
            FsmStateKind::SavingMessages => self.save_messages().await,
            FsmStateKind::Reorging => self.reorg().await,
        }
    }

    /// Ticks once and updates the stuck-count bookkeeping. Returns the wait delay.
    pub async fn tick(&mut self) -> Duration {
        let (delay, res) = self.act().await;
        match res {
            Ok(()) => self.stuck_count = 0,
            Err(e) => {
                // An error implies no change in the FSM state.
                self.stuck_count += 1;
                if self.is_stuck() {
                    tracing::error!(
                        state = %self.current_fsm_state(),
                        stuck_count = self.stuck_count,
                        error = %e,
                        "MEL extractor stuck at the same fsm state past the stall tolerance",
                    );
                } else {
                    tracing::warn!(error = %e, "error in MEL message extractor");
                }
            }
        }
        delay
    }

    /// Drives the FSM in a loop until the task is dropped. Replaces `stopwaiter`.
    pub async fn run(&mut self) {
        loop {
            let delay = self.tick().await;
            if !delay.is_zero() {
                tokio::time::sleep(delay).await;
            }
        }
    }

    // --- FSM handlers ---

    /// `Start`: load the head MEL state and decide whether to process or reorg.
    async fn initialize(&mut self) -> (Duration, Result<()>) {
        let retry = self.config.retry_interval;
        let head = match self.db.get_head_mel_state().await {
            Ok(s) => s,
            Err(e) => return (retry, Err(e)),
        };
        // NOTE: RebuildDelayedMsgPreimages is deferred (needs consensus hashing).
        let header = match self
            .parent_chain_reader
            .header_by_number(BlockNumberOrTag::Number(head.parent_chain_block_number))
            .await
        {
            Ok(Some(h)) => h,
            Ok(None) => {
                return (
                    retry,
                    Err(MelRunnerError::NotFound(format!(
                        "parent chain header at block {}",
                        head.parent_chain_block_number
                    ))),
                );
            }
            Err(e) => return (retry, Err(e.into())),
        };
        if head.parent_chain_block_hash != header.hash {
            self.fsm_state = FsmState::Reorging { mel_state: head };
        } else {
            self.fsm_state = FsmState::ProcessingNextBlock {
                mel_state: head,
                prev_step_was_reorg: false,
            };
        }
        (Duration::ZERO, Ok(()))
    }

    /// `ProcessingNextBlock`: fetch the next block, detect reorgs, extract messages.
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
        if self.config.read_mode != "latest"
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
            let FsmState::ProcessingNextBlock { mel_state, .. } =
                std::mem::replace(&mut self.fsm_state, FsmState::Start)
            else {
                unreachable!("state checked above");
            };
            self.fsm_state = FsmState::Reorging { mel_state };
            return (Duration::ZERO, Ok(()));
        }

        // After a reorg, tell the consumer (nitro node) so its MEL validator can
        // rewind before we extract on the new fork. In native nitro this is a
        // channel send; here it is the consumer's ReorgedToParentChainBlock call.
        if prev_was_reorg
            && let Err(e) = self
                .msg_consumer
                .reorged_to_parent_chain_block(pre_number)
                .await
        {
            return (retry, Err(e));
        }

        // Extract messages from the block via arb-mel. Logs/tx/delayed fetchers are
        // nil until the logs-and-headers fetcher lands (empty logs -> no batches ->
        // no messages); DA payloads are recovered on demand via `data_providers`.
        let pre_state = match &self.fsm_state {
            FsmState::ProcessingNextBlock { mel_state, .. } => mel_state.clone(),
            _ => unreachable!("state checked above"),
        };
        let output = match arb_mel::extract_messages(
            pre_state,
            &header.inner,
            self.data_providers.as_ref(),
            &NilDelayedMessageDb,
            &NilLogsFetcher,
            &NilTxFetcher,
        )
        .await
        {
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

    /// `SavingMessages`: persist data and push messages to the consumer.
    ///
    /// Each sub-step is retried independently; on any failure the FSM stays in
    /// `SavingMessages` (matching nitro's non-atomic save path).
    async fn save_messages(&mut self) -> (Duration, Result<()>) {
        let retry = self.config.retry_interval;
        let consumer = self.msg_consumer.clone();

        if let FsmState::SavingMessages {
            pre_state_msg_count,
            post_state,
            messages,
            delayed_messages,
            batch_metas,
        } = &self.fsm_state
        {
            if let Err(e) = self.db.save_batch_metas(post_state, batch_metas).await {
                return (retry, Err(e));
            }
            if let Err(e) = self
                .db
                .save_delayed_messages(post_state, delayed_messages)
                .await
            {
                return (retry, Err(e));
            }
            if let Err(e) = consumer.push_messages(*pre_state_msg_count, messages).await {
                return (retry, Err(e));
            }
            if let Err(e) = self.db.save_state(post_state).await {
                return (retry, Err(e));
            }
        } else {
            return (
                retry,
                Err(MelRunnerError::InvalidState(
                    "expected SavingMessages".to_string(),
                )),
            );
        }

        // Everything persisted: process the next block on top of post_state.
        let FsmState::SavingMessages { post_state, .. } =
            std::mem::replace(&mut self.fsm_state, FsmState::Start)
        else {
            unreachable!("state checked above");
        };
        self.fsm_state = FsmState::ProcessingNextBlock {
            mel_state: post_state,
            prev_step_was_reorg: false,
        };
        (Duration::ZERO, Ok(()))
    }

    /// `Reorging`: rewind one parent-chain block and resume processing.
    async fn reorg(&mut self) -> (Duration, Result<()>) {
        let retry = self.config.retry_interval;
        let number = match &self.fsm_state {
            FsmState::Reorging { mel_state } => mel_state.parent_chain_block_number,
            _ => {
                return (
                    retry,
                    Err(MelRunnerError::InvalidState(
                        "expected Reorging".to_string(),
                    )),
                );
            }
        };
        if number == 0 {
            return (retry, Err(MelRunnerError::ReorgBelowGenesis));
        }
        let previous = match self.db.state(number - 1).await {
            Ok(s) => s,
            Err(e) => return (retry, Err(e)),
        };
        self.fsm_state = FsmState::ProcessingNextBlock {
            mel_state: previous,
            prev_step_was_reorg: true,
        };
        (Duration::ZERO, Ok(()))
    }
}

// Empty ("nil") collaborators for `arb_mel::extract_messages`, used until the
// logs-and-headers fetcher is implemented. An empty logs source yields no
// batches (hence no messages) for every block.

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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{DaReaderRegistry, MelState, MockDatabase, MockMessageConsumer};
    use alloy_primitives::B256;
    use arb_parent_chain_client::MockParentChainReader;
    use arb_parent_chain_client::test_utils::header_with_parent;

    fn head_at(number: u64, hash: B256) -> MelState {
        MelState {
            parent_chain_block_number: number,
            parent_chain_block_hash: hash,
            ..Default::default()
        }
    }

    #[tokio::test]
    async fn full_fsm_cycle_including_reorg() {
        let h0 = B256::repeat_byte(0xa0);
        let h1 = B256::repeat_byte(0xa1);

        // Headers are set up front; the FSM fetches each lazily as it advances.
        // Block 1 (h0) matches the head, block 2 (h1) builds on it, and block 3
        // does NOT build on h1 — which triggers a reorg once we reach it.
        let mut reader_mock = MockParentChainReader::new();
        reader_mock
            .with_header(header_with_parent(1, h0, B256::ZERO))
            .with_header(header_with_parent(2, h1, h0))
            .with_header(header_with_parent(
                3,
                B256::repeat_byte(0xa3),
                B256::repeat_byte(0xff),
            ));
        let reader = Arc::new(reader_mock);

        let db = Arc::new(MockDatabase::new());
        let consumer = Arc::new(MockMessageConsumer::new());

        let mut ex = MessageExtractor::new(
            MessageExtractionConfig::test(),
            reader.clone(),
            RollupAddresses::default(),
            db.clone(),
            consumer.clone(),
            Arc::new(DaReaderRegistry::new()),
            None,
        );
        assert_eq!(ex.current_fsm_state(), FsmStateKind::Start);

        // Start with no head state -> error, stays in Start.
        let (_, res) = ex.act().await;
        assert!(res.unwrap_err().to_string().contains("not found"));
        assert_eq!(ex.current_fsm_state(), FsmStateKind::Start);

        // Persist a head state anchored at block 1 / hash h0.
        db.save_state(&head_at(1, h0)).await.unwrap();

        // Reader error during initialize -> error, stays in Start.
        reader.set_error(Some("oops"));
        let (_, res) = ex.act().await;
        assert!(res.unwrap_err().to_string().contains("oops"));
        assert_eq!(ex.current_fsm_state(), FsmStateKind::Start);

        // Clear the error -> transitions to ProcessingNextBlock.
        reader.set_error(None::<&str>);
        let (_, res) = ex.act().await;
        assert!(res.is_ok());
        assert_eq!(ex.current_fsm_state(), FsmStateKind::ProcessingNextBlock);

        // Reader error while processing -> error, stays in ProcessingNextBlock.
        reader.set_error(Some("oops"));
        let (_, res) = ex.act().await;
        assert!(res.unwrap_err().to_string().contains("oops"));
        assert_eq!(ex.current_fsm_state(), FsmStateKind::ProcessingNextBlock);

        // Cleared -> block 2 builds on h0 -> extract -> SavingMessages.
        reader.set_error(None::<&str>);
        let (_, res) = ex.act().await;
        assert!(res.is_ok());
        assert_eq!(ex.current_fsm_state(), FsmStateKind::SavingMessages);

        // SavingMessages -> ProcessingNextBlock (now anchored at block 2).
        let (_, res) = ex.act().await;
        assert!(res.is_ok());
        assert_eq!(ex.current_fsm_state(), FsmStateKind::ProcessingNextBlock);

        // Block 3 does NOT build on h1 -> reorg detected.
        let (_, res) = ex.act().await;
        assert!(res.is_ok());
        assert_eq!(ex.current_fsm_state(), FsmStateKind::Reorging);

        // Reorging rewinds to state(1) -> ProcessingNextBlock (prev_step_was_reorg).
        let (_, res) = ex.act().await;
        assert!(res.is_ok());
        assert_eq!(ex.current_fsm_state(), FsmStateKind::ProcessingNextBlock);

        // Resuming after the reorg notifies the consumer, then extracts block 2.
        let (_, res) = ex.act().await;
        assert!(res.is_ok());
        assert_eq!(consumer.last_reorg_block(), Some(1));
        assert_eq!(ex.current_fsm_state(), FsmStateKind::SavingMessages);
    }

    #[tokio::test]
    async fn reaching_the_tip_marks_caught_up() {
        let h5 = B256::repeat_byte(0xc5);
        let mut reader_mock = MockParentChainReader::new();
        reader_mock.with_header(header_with_parent(5, h5, B256::repeat_byte(0xc4)));
        let reader = Arc::new(reader_mock);

        let db = Arc::new(MockDatabase::new());
        db.save_state(&head_at(5, h5)).await.unwrap();

        let mut ex = MessageExtractor::new(
            MessageExtractionConfig::test(),
            reader,
            RollupAddresses::default(),
            db,
            Arc::new(MockMessageConsumer::new()),
            Arc::new(DaReaderRegistry::new()),
            None,
        );

        // Start -> ProcessingNextBlock (header at block 5 matches the head).
        assert!(ex.act().await.1.is_ok());
        assert_eq!(ex.current_fsm_state(), FsmStateKind::ProcessingNextBlock);

        // Block 6 is absent -> tip reached -> caught up, stays ProcessingNextBlock.
        assert!(!ex.caught_up());
        let (_, res) = ex.act().await;
        assert!(res.is_ok());
        assert_eq!(ex.current_fsm_state(), FsmStateKind::ProcessingNextBlock);
        assert!(ex.caught_up());
    }

    #[tokio::test]
    async fn save_messages_error_keeps_state_then_recovers() {
        let h0 = B256::repeat_byte(0xb0);
        let h1 = B256::repeat_byte(0xb1);

        let mut reader_mock = MockParentChainReader::new();
        reader_mock
            .with_header(header_with_parent(1, h0, B256::ZERO))
            .with_header(header_with_parent(2, h1, h0));
        let reader = Arc::new(reader_mock);

        let db = Arc::new(MockDatabase::new());
        let consumer = Arc::new(MockMessageConsumer::new());

        let mut ex = MessageExtractor::new(
            MessageExtractionConfig::test(),
            reader,
            RollupAddresses::default(),
            db.clone(),
            consumer.clone(),
            Arc::new(DaReaderRegistry::new()),
            None,
        );

        db.save_state(&head_at(1, h0)).await.unwrap();
        // Start -> ProcessingNextBlock -> SavingMessages.
        assert!(ex.act().await.1.is_ok());
        assert_eq!(ex.current_fsm_state(), FsmStateKind::ProcessingNextBlock);
        assert!(ex.act().await.1.is_ok());
        assert_eq!(ex.current_fsm_state(), FsmStateKind::SavingMessages);

        // PushMessages error keeps the FSM in SavingMessages.
        consumer.set_error(Some("push failed"));
        let (_, res) = ex.act().await;
        assert!(res.unwrap_err().to_string().contains("push failed"));
        assert_eq!(ex.current_fsm_state(), FsmStateKind::SavingMessages);

        // Cleared error -> retry succeeds and advances.
        consumer.set_error(None::<String>);
        let (_, res) = ex.act().await;
        assert!(res.is_ok());
        assert_eq!(ex.current_fsm_state(), FsmStateKind::ProcessingNextBlock);
    }

    #[tokio::test]
    async fn stall_is_detected_after_tolerance() {
        let mut cfg = MessageExtractionConfig::test();
        cfg.stall_tolerance = 2;

        let mut ex = MessageExtractor::new(
            cfg,
            Arc::new(MockParentChainReader::new()),
            RollupAddresses::default(),
            Arc::new(MockDatabase::new()), // no head state -> Start keeps erroring
            Arc::new(MockMessageConsumer::new()),
            Arc::new(DaReaderRegistry::new()),
            None,
        );

        assert!(!ex.is_stuck());
        // stall_tolerance = 2, so the 3rd stuck tick crosses the threshold.
        ex.tick().await;
        ex.tick().await;
        assert!(!ex.is_stuck());
        ex.tick().await;
        assert!(ex.is_stuck());
    }
}
