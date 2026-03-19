use alloy_primitives::B256;
use revm::Database;

use arb_storage::Storage;

const PRESENT_HASH: B256 = {
    let mut bytes = [0u8; 32];
    bytes[31] = 1;
    B256::new(bytes)
};

/// Tracks transaction hashes that have been filtered (censored/blocked).
pub struct FilteredTransactionsState<D> {
    store: Storage<D>,
}

impl<D: Database> FilteredTransactionsState<D> {
    pub fn open(sto: Storage<D>) -> Self {
        Self { store: sto }
    }

    pub fn add(&self, tx_hash: B256) -> Result<(), ()> {
        self.store.set(tx_hash, PRESENT_HASH)
    }

    pub fn delete(&self, tx_hash: B256) -> Result<(), ()> {
        self.store.set(tx_hash, B256::ZERO)
    }

    pub fn is_filtered(&self, tx_hash: B256) -> Result<bool, ()> {
        let value = self.store.get(tx_hash)?;
        Ok(value == PRESENT_HASH)
    }

    /// Check if a tx is filtered without charging gas.
    pub fn is_filtered_free(&self, tx_hash: B256) -> bool {
        self.store
            .get(tx_hash)
            .map(|v| v == PRESENT_HASH)
            .unwrap_or(false)
    }

    /// Delete a tx hash without charging gas (cleanup after no-op execution).
    pub fn delete_free(&self, tx_hash: B256) {
        let _ = self.store.set(tx_hash, B256::ZERO);
    }
}
