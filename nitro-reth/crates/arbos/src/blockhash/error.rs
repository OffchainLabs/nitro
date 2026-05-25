use arb_storage::StorageError;

/// Errors raised by the L1 blockhashes ring buffer.
#[derive(Clone, thiserror::Error, Debug)]
pub enum BlockhashesError {
    /// Underlying storage failure.
    #[error(transparent)]
    Storage(#[from] StorageError),
}
