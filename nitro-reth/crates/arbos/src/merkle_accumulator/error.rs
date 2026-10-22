use arb_storage::StorageError;

/// Errors raised by the L2-to-L1 send-merkle accumulator.
#[derive(Clone, thiserror::Error, Debug)]
pub enum MerkleAccumulatorError {
    /// Underlying storage failure.
    #[error(transparent)]
    Storage(#[from] StorageError),
}
