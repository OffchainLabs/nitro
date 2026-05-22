use crate::AnyError;
use alloc::{boxed::Box, string::String};
use core::error::Error;

/// Errors surfaced by the underlying state database when arb-storage reads or
/// writes a slot.
///
/// The underlying `Database::Error` is erased here so that callers do not
/// have to propagate a generic `<DBErr>` parameter through every layer.
#[derive(Clone, Debug, thiserror::Error)]
pub enum DatabaseError {
    /// The database returned an error while reading a slot.
    #[error("failed to read storage slot: {0}")]
    Read(DatabaseErrorInfo),

    /// The database returned an error while writing a slot.
    #[error("failed to write storage slot: {0}")]
    Write(DatabaseErrorInfo),

    /// Catch-all wrapper for implementation-specific errors that do not fit
    /// `Read` or `Write`.
    #[error(transparent)]
    Custom(AnyError),
}

impl DatabaseError {
    /// Wraps any [`Error`] as a [`DatabaseError::Custom`].
    pub fn custom<E>(error: E) -> Self
    where
        E: Error + Send + Sync + 'static,
    {
        Self::Custom(AnyError::new(error))
    }
}

/// Implementation-agnostic information about a database failure.
///
/// Stores only the rendered error message, since the concrete `Database::Error`
/// type is erased at the leaf and there is no general way to extract a numeric
/// code from it. Implementations that do carry a code can encode it into the
/// message string.
#[derive(Debug, Clone, PartialEq, Eq, thiserror::Error)]
#[error("{message}")]
pub struct DatabaseErrorInfo {
    /// Human-readable error message rendered from the underlying database
    /// error via `Display`.
    pub message: Box<str>,
}

impl DatabaseErrorInfo {
    pub fn new(message: impl Into<String>) -> Self {
        Self {
            message: message.into().into_boxed_str(),
        }
    }
}
