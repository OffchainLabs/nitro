use alloy_primitives::B256;
use revm::Database;

use arb_storage::Storage;

/// Retryable tickets state for L1→L2 message retries.
pub struct RetryableState<D> {
    pub backing_storage: Storage<D>,
}

impl<D: Database> RetryableState<D> {
    pub fn initialize(_sto: &Storage<D>) {
        // TODO: implement full initialization
    }

    pub fn open(sto: Storage<D>) -> Self {
        Self {
            backing_storage: sto,
        }
    }

    pub fn open_from_raw(state: *mut revm::database::State<D>, base_key: B256) -> Self {
        Self {
            backing_storage: Storage::new(state, base_key),
        }
    }
}
