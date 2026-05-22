use arb_storage::StorageError;

use crate::address_set::AddressSetError;

/// Errors raised by the L1 pricing subsystem (batch poster table and update
/// model).
#[derive(thiserror::Error, Debug)]
pub enum L1PricingError {
    /// Underlying storage failure.
    #[error(transparent)]
    Storage(#[from] StorageError),

    /// The batch-poster table is backed by an [`AddressSet`], so its lookup
    /// failures surface here.
    #[error(transparent)]
    AddressSet(#[from] AddressSetError),

    /// `open_poster(create_if_not_exist=false)` was called with an unknown
    /// address.
    #[error("batch poster not found")]
    BatchPosterNotFound,

    /// `add_poster` was called with an address that is already a poster.
    #[error("batch poster already exists")]
    BatchPosterAlreadyExists,

    /// `set_parent_gas_floor_per_token` requires ArbOS v50 or later.
    #[error("parent gas floor is unsupported before ArbOS v50")]
    ParentGasFloorUnsupportedVersion,

    /// `update_for_batch_poster_spending` was called with a time outside the
    /// monotonic window `[last_update_time, current_time]`.
    #[error("invalid update time")]
    InvalidUpdateTime,
}
