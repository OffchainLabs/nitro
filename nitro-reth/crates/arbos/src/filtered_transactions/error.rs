use arb_storage::StorageError;

/// Errors raised by the filtered-transactions ledger.
#[derive(Clone, thiserror::Error, Debug)]
pub enum FilteredTxError {
    /// Underlying storage failure.
    #[error(transparent)]
    Storage(#[from] StorageError),
}
