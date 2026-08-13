use alloc::sync::Arc;
use core::{error::Error, fmt};

/// Cloneable wrapper around any [`Error`] type.
///
/// Used at the storage-errors boundary to erase the underlying
/// `Database::Error` while still allowing the wrapped error to be inspected
/// via [`AnyError::as_error`].
#[derive(Clone)]
pub struct AnyError {
    inner: Arc<dyn Error + Send + Sync + 'static>,
}

impl AnyError {
    /// Wraps `error` in an [`AnyError`].
    pub fn new<E>(error: E) -> Self
    where
        E: Error + Send + Sync + 'static,
    {
        Self {
            inner: Arc::new(error),
        }
    }

    /// Returns the inner [`Error`] trait object.
    pub fn as_error(&self) -> &(dyn Error + Send + Sync + 'static) {
        self.inner.as_ref()
    }
}

impl fmt::Debug for AnyError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        fmt::Debug::fmt(&self.inner, f)
    }
}

impl fmt::Display for AnyError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        fmt::Display::fmt(&self.inner, f)
    }
}

impl Error for AnyError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        self.inner.source()
    }
}
