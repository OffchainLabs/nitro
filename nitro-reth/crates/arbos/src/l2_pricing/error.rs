use arb_storage::StorageError;

/// Errors raised by the L2 pricing subsystem.
#[derive(Clone, thiserror::Error, Debug)]
pub enum L2PricingError {
    /// Underlying storage failure.
    #[error(transparent)]
    Storage(#[from] StorageError),
}
