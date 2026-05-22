use arb_storage::StorageError;

/// Errors raised by the feature-flag bitmask.
#[derive(thiserror::Error, Debug)]
pub enum FeaturesError {
    /// Underlying storage failure.
    #[error(transparent)]
    Storage(#[from] StorageError),
}
