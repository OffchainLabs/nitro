use alloc::{boxed::Box, string::ToString};
use core::fmt::Display;

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

impl ArbError {
    /// Create a new `ArbError` from a given error.
    pub fn other<E>(error: E) -> Self
    where
        E: core::error::Error + Send + Sync + 'static,
    {
        Self::Other(Box::new(error))
    }

    /// Create a new `ArbError` from a given message.
    pub fn msg(msg: impl Display) -> Self {
        Self::Other(msg.to_string().into())
    }
}
