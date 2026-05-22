//! Error types for the arb-node launcher and genesis flows.
//!
//! Replaces the historical `Result<_, String>` plumbing through the
//! persistence-thread channel, the parallel-state-root callback, and the
//! genesis initializer with concrete typed variants.

use arbos::arbos_state::ArbosStateError;
use reth_storage_errors::provider::ProviderError;
use reth_trie_parallel::root::ParallelStateRootError;

/// Errors surfaced by the `arb-node` launcher infrastructure.
#[derive(Debug, thiserror::Error)]
pub enum LauncherError {
    /// Failure originating in the wider arbreth error chain.
    #[error(transparent)]
    Arb(#[from] arb_errors::ArbError),

    /// A reth provider operation (open RW handle, save blocks, commit,
    /// rollback) failed against the on-disk database.
    #[error(transparent)]
    Provider(#[from] ProviderError),

    /// Parallel state-root computation failed.
    #[error(transparent)]
    ParallelStateRoot(#[from] ParallelStateRootError),

    /// The parallel state-root callback was queried before the launcher
    /// finished wiring it up. This indicates a startup-ordering bug.
    #[error("parallel state root callback not initialized")]
    ParallelStateRootNotInitialized,

    /// ArbOS genesis initialization failed.
    #[error(transparent)]
    Genesis(#[from] GenesisError),
}

/// Errors surfaced while initializing ArbOS genesis state.
#[derive(Debug, thiserror::Error)]
pub enum GenesisError {
    /// `initialize_arbos_state` was called against a database that already
    /// contains a non-zero ArbOS version.
    #[error("ArbOS state already initialized")]
    AlreadyInitialized,

    /// A backing-storage write performed during the slot-bootstrap phase
    /// (initial version, chain ID, network-fee account, chain config) failed.
    #[error("failed to write genesis slot ({what}): {source}")]
    StorageWrite {
        /// Short label describing which slot was being written.
        what: &'static str,
        /// The underlying storage failure.
        #[source]
        source: arb_storage_errors::StorageError,
    },

    /// An ArbOS subsystem refused to initialize cleanly during bootstrap.
    #[error("failed to initialize {subsystem}: {source}")]
    InitSubsystem {
        /// Subsystem name (e.g. `"retryables"`, `"chain owners"`).
        subsystem: &'static str,
        /// The underlying ArbOS error.
        #[source]
        source: ArbosStateError,
    },

    /// `ArbosState::open` failed after the initial slot writes were
    /// persisted. Signals a structural inconsistency in the bootstrap path.
    #[error("failed to open ArbOS state after initial setup: {0}")]
    OpenArbosState(#[source] ArbosStateError),

    /// Stepping the ArbOS version upward from `1` to the target failed.
    #[error("failed to upgrade ArbOS to version {target}: {source}")]
    Upgrade {
        /// Target ArbOS version requested by the chain spec.
        target: u64,
        /// The underlying upgrade failure.
        #[source]
        source: ArbosStateError,
    },
}
