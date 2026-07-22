//! The `SavingMessages` FSM phase: persist data and push messages to the
//! consumer.
//!
//! Ports nitro's `arbnode/mel/runner/save_messages.go`.

use std::time::Duration;

use arb_da_provider_client::DaReaderSource;
use arb_parent_chain_client::ParentChainReader;

use crate::{
    MelRunnerError, Result, consumer::MessageConsumer, database::Database,
    extractor::MessageExtractor, fsm::FsmState,
};

/// The `SavingMessages` FSM phase.
#[async_trait::async_trait]
pub(crate) trait SavingMessages {
    /// Persists batch metas, delayed messages, and state, and pushes messages to
    /// the consumer, then transitions to `ProcessingNextBlock`. Each sub-step is
    /// retried independently; on failure the FSM stays in `SavingMessages`.
    /// Mirrors `SaveMessages`.
    async fn save_messages(&mut self) -> (Duration, Result<()>);
}

#[async_trait::async_trait]
impl<P, D, C, S> SavingMessages for MessageExtractor<P, D, C, S>
where
    P: ParentChainReader,
    D: Database,
    C: MessageConsumer,
    S: DaReaderSource,
{
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
        let post_state = match std::mem::replace(&mut self.fsm_state, FsmState::Start) {
            FsmState::SavingMessages { post_state, .. } => post_state,
            _ => unreachable!("state checked above"),
        };
        self.fsm_state = FsmState::ProcessingNextBlock {
            mel_state: post_state,
            prev_step_was_reorg: false,
        };
        (Duration::ZERO, Ok(()))
    }
}
