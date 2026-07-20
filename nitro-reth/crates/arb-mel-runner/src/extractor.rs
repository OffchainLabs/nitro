//! The message extractor: the FSM driver.
//!
//! Ports nitro's `arbnode/mel/runner/mel.go`. Each FSM phase's handler lives in
//! its own module (`initialize`, `process_next_block`, `save_messages`, `reorg`)
//! as a trait this struct implements.
//!
//! [`MessageExtractor::act`] ticks the FSM once; [`MessageExtractor::run`] drives
//! it in a loop.

use std::{
    sync::{Arc, atomic::AtomicU64},
    time::Duration,
};

use arb_da_provider_client::DaReaderSource;
use arb_parent_chain_client::ParentChainReader;

use crate::{
    Result,
    config::MessageExtractionConfig,
    consumer::MessageConsumer,
    database::Database,
    fsm::{FsmState, FsmStateKind},
    initialize::Initializing,
    process_next_block::ProcessingNextBlock,
    reorg::Reorging,
    save_messages::SavingMessages,
    types::RollupAddresses,
};

/// Reads parent-chain blocks one by one and turns them into L2 messages.
///
/// Collaborators are generic `Arc<_>` handles.
pub struct MessageExtractor<P, D, C, S>
where
    P: ParentChainReader,
    D: Database,
    C: MessageConsumer,
    S: DaReaderSource,
{
    pub(crate) config: MessageExtractionConfig,
    pub(crate) parent_chain_reader: Arc<P>,
    addrs: RollupAddresses,
    pub(crate) db: Arc<D>,
    pub(crate) msg_consumer: Arc<C>,
    pub(crate) data_providers: Arc<S>,
    pub(crate) fsm_state: FsmState,
    stuck_count: u64,
    pub(crate) last_block_to_read: AtomicU64,
    pub(crate) caught_up: bool,
}

impl<P, D, C, S> MessageExtractor<P, D, C, S>
where
    P: ParentChainReader,
    D: Database,
    C: MessageConsumer,
    S: DaReaderSource,
{
    /// Creates an extractor in the `Start` state. Mirrors `NewMessageExtractor`.
    pub fn new(
        config: MessageExtractionConfig,
        parent_chain_reader: Arc<P>,
        addrs: RollupAddresses,
        db: Arc<D>,
        msg_consumer: Arc<C>,
        data_providers: Arc<S>,
    ) -> Self {
        Self {
            config,
            parent_chain_reader,
            addrs,
            db,
            msg_consumer,
            data_providers,
            fsm_state: FsmState::Start,
            stuck_count: 0,
            last_block_to_read: AtomicU64::new(0),
            caught_up: false,
        }
    }
}

impl<P, D, C, S> MessageExtractor<P, D, C, S>
where
    P: ParentChainReader,
    D: Database,
    C: MessageConsumer,
    S: DaReaderSource,
{
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

    /// The DA provider passed into extraction for recovering off-chain batch payloads.
    pub fn data_providers(&self) -> &Arc<S> {
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
}

#[cfg(test)]
mod tests {
    use alloy_primitives::B256;
    use arb_parent_chain_client::{MockParentChainReader, test_utils::header_with_parent};

    use super::*;
    use crate::{DaReaderRegistry, MelState, MockDatabase, MockMessageConsumer};

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
