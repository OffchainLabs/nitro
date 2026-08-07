//! The `Reorging` FSM phase: rewind one parent-chain block and resume
//! processing.
//!
//! Ports nitro's `arbnode/mel/runner/reorg.go`.

use std::time::Duration;

use arb_da_provider::DaReaderSource;
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
    /// The `Reorging` FSM phase: rewinds to the MEL state one parent-chain block
    /// back and transitions to `ProcessingNextBlock` with `prev_step_was_reorg`
    /// set. Mirrors `Reorg`.
    pub(crate) async fn reorg(&mut self) -> (Duration, Result<()>) {
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
