use revm::Database;

use crate::backed_types::StorageBackedUint64;
use crate::storage::Storage;

const LENGTH_OFFSET: u64 = 0;

/// A vector of sub-storages backed by ArbOS storage.
///
/// Layout: offset 0 = length, sub-storages at indices 0..length.
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
    pub fn length(&self) -> Result<u64, ()> {
        self.length.get()
    }

    /// Returns the sub-storage at the given index.
    pub fn at(&self, index: u64) -> Storage<D> {
        self.storage
            .open_sub_storage(&index.to_be_bytes())
    }

    /// Appends a new sub-storage and returns it.
    pub fn push(&self) -> Result<Storage<D>, ()> {
        let len = self.length.get()?;
        self.length.set(len + 1)?;
        Ok(self.at(len))
    }

    /// Removes the last sub-storage and returns its index.
    pub fn pop(&self) -> Result<Option<u64>, ()> {
        let len = self.length.get()?;
        if len == 0 {
            return Ok(None);
        }
        let new_len = len - 1;
        self.length.set(new_len)?;
        Ok(Some(new_len))
    }
}
