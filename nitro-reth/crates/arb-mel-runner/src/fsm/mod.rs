//! The message-extraction finite state machine.
//!
//! Ports `arbnode/mel/runner/fsm.go`. Go models the FSM as a separate `FSMState`
//! plus a payload-carrying "action" (`SourceEvent`); in Rust we merge them into a
//! single payload-carrying [`FsmState`] enum, with a lightweight [`FsmStateKind`]
//! discriminant for the `CurrentFSMState`-style assertions.
//!
//! Each state's handler is an inherent method on [`crate::MessageExtractor`],
//! defined in its own submodule (`initialize`, `process_next_block`,
//! `save_messages`, `reorg`).

mod initialize;
mod process_next_block;
mod reorg;
mod save_messages;

use arb_mel::{BatchMeta, DelayedInboxMessage, MelState};
use arbos::arbos_types::MessageWithMetadata;

/// The state of the extraction FSM, carrying the data each state needs.
///
/// The handlers in [`crate::MessageExtractor`] are the sole authority on
/// transitions and only ever produce valid ones; see
/// [`FsmStateKind::can_transition_to`] for the encoded validity table (mirroring
/// `newFSM`'s transition list).
///
/// Not `Debug`: the payload types from `arb-mel` don't implement it.
pub enum FsmState {
    /// Initial state: load the head MEL state and decide where to begin.
    Start,
    /// Process the next parent-chain block and extract messages from it.
    ProcessingNextBlock {
        /// The state the next block is applied on top of.
        mel_state: MelState,
        /// Whether the previous step was a reorg (guards continuous rewinding).
        prev_step_was_reorg: bool,
    },
    /// Rewind one parent-chain block after a reorg was detected.
    Reorging {
        /// The (dirty) state to rewind from.
        mel_state: MelState,
    },
    /// Persist extracted data and push messages to the consumer.
    SavingMessages {
        /// The message count before this block (the first index to push).
        pre_state_msg_count: u64,
        /// The state after applying the block.
        post_state: MelState,
        /// Messages extracted from the block.
        messages: Vec<MessageWithMetadata>,
        /// Delayed messages observed in the block.
        delayed_messages: Vec<DelayedInboxMessage>,
        /// Batch metadata recorded for the block.
        batch_metas: Vec<BatchMeta>,
    },
}

impl FsmState {
    /// The discriminant of this state, for external inspection.
    pub fn kind(&self) -> FsmStateKind {
        match self {
            FsmState::Start => FsmStateKind::Start,
            FsmState::ProcessingNextBlock { .. } => FsmStateKind::ProcessingNextBlock,
            FsmState::Reorging { .. } => FsmStateKind::Reorging,
            FsmState::SavingMessages { .. } => FsmStateKind::SavingMessages,
        }
    }
}

/// A payload-free discriminant of [`FsmState`], mirroring nitro's `FSMState`.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum FsmStateKind {
    /// See [`FsmState::Start`].
    Start,
    /// See [`FsmState::ProcessingNextBlock`].
    ProcessingNextBlock,
    /// See [`FsmState::Reorging`].
    Reorging,
    /// See [`FsmState::SavingMessages`].
    SavingMessages,
}

impl FsmStateKind {
    /// Whether a transition from `self` to `to` is permitted, encoding the
    /// transition table from `newFSM` (`fsm.go`).
    pub fn can_transition_to(self, to: FsmStateKind) -> bool {
        use FsmStateKind::*;
        match to {
            // `processNextBlock` event
            ProcessingNextBlock => matches!(
                self,
                Start | ProcessingNextBlock | SavingMessages | Reorging
            ),
            // `reorgToOldBlock` event
            Reorging => matches!(self, Start | ProcessingNextBlock),
            // `saveMessages` event
            SavingMessages => matches!(self, ProcessingNextBlock | SavingMessages),
            // `backToStart` event
            Start => matches!(self, Start | ProcessingNextBlock),
        }
    }
}

impl std::fmt::Display for FsmStateKind {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let s = match self {
            FsmStateKind::Start => "start",
            FsmStateKind::ProcessingNextBlock => "processing_next_block",
            FsmStateKind::Reorging => "reorging",
            FsmStateKind::SavingMessages => "saving_messages",
        };
        f.write_str(s)
    }
}

#[cfg(test)]
mod tests {
    use super::FsmStateKind::*;

    #[test]
    fn allowed_transitions_match_the_go_table() {
        // processNextBlock: from {Start, ProcessingNextBlock, SavingMessages, Reorging}
        for from in [Start, ProcessingNextBlock, SavingMessages, Reorging] {
            assert!(from.can_transition_to(ProcessingNextBlock));
        }
        // reorgToOldBlock: from {Start, ProcessingNextBlock}
        assert!(Start.can_transition_to(Reorging));
        assert!(ProcessingNextBlock.can_transition_to(Reorging));
        assert!(!SavingMessages.can_transition_to(Reorging));
        assert!(!Reorging.can_transition_to(Reorging));
        // saveMessages: from {ProcessingNextBlock, SavingMessages}
        assert!(ProcessingNextBlock.can_transition_to(SavingMessages));
        assert!(SavingMessages.can_transition_to(SavingMessages));
        assert!(!Start.can_transition_to(SavingMessages));
        assert!(!Reorging.can_transition_to(SavingMessages));
    }

    #[test]
    fn state_names_match_go() {
        assert_eq!(Start.to_string(), "start");
        assert_eq!(ProcessingNextBlock.to_string(), "processing_next_block");
        assert_eq!(Reorging.to_string(), "reorging");
        assert_eq!(SavingMessages.to_string(), "saving_messages");
    }
}
