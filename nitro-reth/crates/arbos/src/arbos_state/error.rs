use arb_storage::StorageError;

use crate::{
    address_set::AddressSetError, address_table::AddressTableError, blockhash::BlockhashesError,
    features::FeaturesError, filtered_transactions::FilteredTxError, l1_pricing::L1PricingError,
    l2_pricing::L2PricingError, merkle_accumulator::MerkleAccumulatorError,
    programs::ProgramsError, retryables::RetryableError,
};

/// Top-level error for `arbos_state` operations that orchestrate multiple
/// subsystems.
///
/// Each subsystem owns its own error type; this enum aggregates them via
/// `#[from]` so cross-module call chains in `arbos_state` (e.g. running an
/// ArbOS version upgrade hook that touches programs, l2 pricing, and the
/// transaction filtering address set in one go) can use `?` ergonomically.
#[derive(thiserror::Error, Debug)]
pub enum ArbosStateError {
    /// Underlying storage failure.
    #[error(transparent)]
    Storage(#[from] StorageError),

    /// Surfaced from a retryable-state operation.
    #[error(transparent)]
    Retryable(#[from] RetryableError),

    /// Surfaced from an L1 pricing operation.
    #[error(transparent)]
    L1Pricing(#[from] L1PricingError),

    /// Surfaced from an L2 pricing operation.
    #[error(transparent)]
    L2Pricing(#[from] L2PricingError),

    /// Surfaced from an address-set operation.
    #[error(transparent)]
    AddressSet(#[from] AddressSetError),

    /// Surfaced from an address-table operation.
    #[error(transparent)]
    AddressTable(#[from] AddressTableError),

    /// Surfaced from a blockhashes operation.
    #[error(transparent)]
    Blockhashes(#[from] BlockhashesError),

    /// Surfaced from a features operation.
    #[error(transparent)]
    Features(#[from] FeaturesError),

    /// Surfaced from a merkle-accumulator operation.
    #[error(transparent)]
    MerkleAccumulator(#[from] MerkleAccumulatorError),

    /// Surfaced from a programs operation.
    #[error(transparent)]
    Programs(#[from] ProgramsError),

    /// Surfaced from a filtered-transactions operation.
    #[error(transparent)]
    FilteredTx(#[from] FilteredTxError),

    /// `ArbosState::open` found `version == 0` in storage. Expected only at
    /// genesis before `initialize` runs.
    #[error("ArbOS state has not been initialised (version=0)")]
    Uninitialised,

    /// `ArbosState::open` found a version word this build does not recognise.
    #[error("unsupported ArbOS version: {0}")]
    UnsupportedVersion(u64),

    /// A scheduled upgrade targets an ArbOS version this node does not
    /// support.
    #[error("scheduled ArbOS version {version} exceeds maximum supported {max}")]
    UnsupportedScheduledVersion {
        /// Version the upgrade points at.
        version: u64,
        /// Maximum version this build of arbreth supports.
        max: u64,
    },

    /// The state is currently running an ArbOS version this build does not
    /// know how to apply per-block hooks for.
    #[error("unsupported running ArbOS version: {0}")]
    UnsupportedRunningVersion(u64),

    /// Genesis initialisation found the address table already populated.
    #[error("address table must be empty during genesis initialisation")]
    AddressTableNotEmpty,

    /// During genesis initialisation, a freshly registered address received a
    /// slot index that did not match its position in the input list.
    #[error("address table slot mismatch during genesis initialisation")]
    AddressTableSlotMismatch,
}
