//! The Message Extraction Layer (MEL) runner.
//!
//! MEL reads parent-chain (L1) blocks one at a time and transforms them into L2
//! messages for the execution layer. This crate is the orchestration layer that
//! drives a finite state machine over the parent chain and calls into the
//! `arb-mel` crate's [`extract_messages`](arb_mel::extract_messages) for the
//! per-block extraction algorithm.
//!
//! The collaborators the runner depends on — the database, the message consumer,
//! the DA provider, the sequencer-batch counter, and message extraction itself —
//! are expressed as **traits**, each with a hand-written mock, so the FSM control
//! flow can be unit-tested without their real implementations. Parent-chain reads
//! reuse [`arb_parent_chain_client::ParentChainReader`]; the canonical MEL types
//! (`MelState`, `DelayedInboxMessage`, `BatchMeta`) come from `arb-mel`.
//!
//! The entry point is [`MessageExtractor`]; [`MessageExtractor::act`] ticks the
//! FSM once, and [`MessageExtractor::run`] drives it in a loop.

mod config;
mod consumer;
mod database;
mod extractor;
mod fsm;
mod logs_and_headers_fetcher;
mod types;

// The DA provider is the real `arb-da-provider-client`; re-export for convenience.
pub use arb_da_provider_client::{DaReaderRegistry, DaReaderSource};
// Canonical MEL types are owned by `arb-mel`; re-export for convenience.
pub use arb_mel::{BatchMeta, DelayedInboxMessage, ExtractionOutput, MelState};
pub use config::{MessageExtractionConfig, ReadMode};
pub use consumer::{MessageConsumer, MockMessageConsumer};
pub use database::{Database, MockDatabase};
pub use extractor::MessageExtractor;
pub use fsm::{FsmState, FsmStateKind};
pub use types::{MessageSyncProgress, RollupAddresses};

/// Something went wrong while running the message extractor.
#[derive(Debug, thiserror::Error)]
pub enum MelRunnerError {
    /// A requested item was not present in the database.
    #[error("{0}: not found")]
    NotFound(String),
    /// A database read or write failed.
    #[error("database error: {0}")]
    Database(String),
    /// The message-extraction algorithm returned an error.
    #[error("message extraction error: {0}")]
    Extraction(String),
    /// A `arb_mel::extract_messages` call failed.
    #[error(transparent)]
    Mel(#[from] arb_mel::MelError),
    /// Pushing extracted messages to the consumer failed.
    #[error("message consumer error: {0}")]
    Consumer(String),
    /// A parent-chain read failed.
    #[error("parent chain error: {0}")]
    ParentChain(#[from] arb_parent_chain_client::ParentChainError),
    /// The FSM was asked to act on a state it cannot handle.
    #[error("invalid fsm state: {0}")]
    InvalidState(String),
    /// The configuration failed validation.
    #[error("invalid config: {0}")]
    Config(String),
    /// A reorg was requested at the genesis block, which cannot be rewound.
    #[error("cannot reorg below genesis block")]
    ReorgBelowGenesis,
}

/// Return type used throughout the crate.
pub type Result<T, E = MelRunnerError> = std::result::Result<T, E>;
