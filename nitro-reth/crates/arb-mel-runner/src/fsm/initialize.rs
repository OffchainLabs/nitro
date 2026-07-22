//! The `Start` FSM phase: load the head MEL state and decide whether to process
//! or reorg.
//!
//! Ports nitro's `arbnode/mel/runner/initialize.go`.

use std::time::Duration;

use alloy_eips::BlockNumberOrTag;
use arb_da_provider_client::DaReaderSource;
use arb_parent_chain_client::ParentChainReader;

use crate::{
    MelRunnerError, Result, consumer::MessageConsumer, database::Database,
    extractor::MessageExtractor, fsm::FsmState,
};

impl<P, D, C, S> MessageExtractor<P, D, C, S>
where
    P: ParentChainReader,
    D: Database,
    C: MessageConsumer,
    S: DaReaderSource,
{
    /// The `Start` FSM phase: loads the head MEL state and transitions to
    /// `ProcessingNextBlock`, or to `Reorging` if the stored head hash no longer
    /// matches the parent chain. Mirrors `Initialize`.
    pub(crate) async fn initialize(&mut self) -> (Duration, Result<()>) {
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
}
