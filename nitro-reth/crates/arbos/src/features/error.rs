use arb_storage::StorageError;

/// Errors raised by the feature-flag bitmask.
#[derive(Clone, thiserror::Error, Debug)]
pub enum FeaturesError {
    /// Underlying storage failure.
    #[error(transparent)]
    Storage(#[from] StorageError),
}
