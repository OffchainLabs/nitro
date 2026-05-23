use alloc::boxed::Box;

/// Result alias for [`ArbError`].
pub type ArbResult<T> = Result<T, ArbError>;

/// Top-level error for arbreth.
#[derive(Debug, thiserror::Error)]
pub enum ArbError {
    /// Storage-layer failure surfaced via arb-storage.
    #[error(transparent)]
    Storage(#[from] arb_storage_errors::StorageError),

    /// Database-level failure surfaced from the underlying state DB.
    #[error(transparent)]
    Database(#[from] arb_storage_errors::DatabaseError),

    /// Any other error.
    #[error(transparent)]
    Other(Box<dyn core::error::Error + Send + Sync>),
}
