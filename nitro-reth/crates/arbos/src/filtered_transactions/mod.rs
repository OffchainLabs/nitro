use alloy_primitives::{B256, U256};
use revm::Database;

use arb_storage::{Storage, StorageBackend, SystemStateBackend};

mod error;
pub use error::FilteredTxError;

const PRESENT_HASH: B256 = {
    let mut bytes = [0u8; 32];
    bytes[31] = 1;
    B256::new(bytes)
};

/// Tracks transaction hashes that have been filtered (censored/blocked).
pub struct FilteredTransactionsState<'a, D> {
    store: Storage<'a, D>,
}

impl<'a, D> FilteredTransactionsState<'a, D> {
    pub fn open(sto: Storage<'a, D>) -> Self {
        Self { store: sto }
    }

    pub fn set<B: StorageBackend>(
        &self,
        backend: &mut B,
        tx_hash: B256,
        present: bool,
    ) -> Result<(), FilteredTxError> {
        let value = if present {
            U256::from_be_bytes(PRESENT_HASH.0)
        } else {
            U256::ZERO
        };
        backend
            .sstore(
                self.store.account(),
                self.store.slot_for_key(tx_hash),
                value,
            )
            .map_err(Into::into)?;
        Ok(())
    }

    pub fn is_filtered<B: SystemStateBackend>(
        &self,
        backend: &mut B,
        tx_hash: B256,
    ) -> Result<bool, FilteredTxError> {
        let value = backend
            .sload_system(self.store.account(), self.store.slot_for_key(tx_hash))
            .map_err(Into::into)?;
        Ok(value == U256::from_be_bytes(PRESENT_HASH.0))
    }
}

impl<D: Database> FilteredTransactionsState<'_, D> {
    /// Check if a tx is filtered without charging gas.
    pub fn is_filtered_free(&self, tx_hash: B256) -> bool {
        self.store
            .get(tx_hash)
            .map(|v| v == PRESENT_HASH)
            .unwrap_or(false)
    }
}
