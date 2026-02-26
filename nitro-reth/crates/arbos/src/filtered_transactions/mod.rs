use revm::Database;

use arb_storage::Storage;

/// Filtered transactions state for transaction-level filtering.
pub struct FilteredTransactionsState<D> {
    pub backing_storage: Storage<D>,
}

impl<D: Database> FilteredTransactionsState<D> {
    pub fn open(sto: Storage<D>) -> Self {
        Self {
            backing_storage: sto,
        }
    }
}
