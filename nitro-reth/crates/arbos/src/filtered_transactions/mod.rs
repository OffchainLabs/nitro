use alloy_primitives::B256;
use revm::Database;

use arb_storage::Storage;

mod error;
pub use error::FilteredTxError;

const PRESENT_HASH: B256 = {
    let mut bytes = [0u8; 32];
    bytes[31] = 1;
    B256::new(bytes)
};

/// Tracks transaction hashes that have been filtered (censored/blocked).
pub struct FilteredTransactionsState<D> {
    store: Storage<D>,
}

impl<D> FilteredTransactionsState<D> {
    pub fn open(sto: Storage<D>) -> Self {
        Self { store: sto }
    }
}

impl<D: Database> FilteredTransactionsState<D> {
    /// Check if a tx is filtered without charging gas.
    pub fn is_filtered_free(&self, tx_hash: B256) -> bool {
        self.store
            .get(tx_hash)
            .map(|v| v == PRESENT_HASH)
            .unwrap_or(false)
    }
}
