use arb_storage_errors::StorageError;
use revm::Database;

use crate::{backed_types::StorageBackedUint64, storage::Storage};

const LENGTH_OFFSET: u64 = 0;

/// Vector of sub-storages backed by ArbOS storage.
///
/// Layout: offset 0 = length; sub-storages live at indices `0..length`.
pub struct SubStorageVector<D> {
    storage: Storage<D>,
    length: StorageBackedUint64<D>,
}

pub fn open_sub_storage_vector<D: Database>(storage: Storage<D>) -> SubStorageVector<D> {
    let state = storage.state_ptr();
    let base_key = storage.base_key();
    SubStorageVector {
        length: StorageBackedUint64::new(state, base_key, LENGTH_OFFSET),
        storage,
    }
}

impl<D: Database> SubStorageVector<D> {
    pub fn length(&self) -> Result<u64, StorageError> {
        self.length.get()
    }

    pub fn at(&self, index: u64) -> Storage<D> {
        self.storage.open_sub_storage(&index.to_be_bytes())
    }

    pub fn push(&self) -> Result<Storage<D>, StorageError> {
        let len = self.length.get()?;
        self.length.set(len + 1)?;
        Ok(self.at(len))
    }

    pub fn pop(&self) -> Result<Option<u64>, StorageError> {
        let len = self.length.get()?;
        if len == 0 {
            return Ok(None);
        }
        let new_len = len - 1;
        self.length.set(new_len)?;
        Ok(Some(new_len))
    }
}
