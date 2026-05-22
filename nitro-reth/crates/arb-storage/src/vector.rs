use arb_storage_errors::StorageError;
use revm::Database;

use crate::{backed_types::StorageBackedUint64, backend::StorageBackend, storage::Storage};

const LENGTH_OFFSET: u64 = 0;

/// Vector of sub-storages backed by ArbOS storage.
///
/// Layout: offset 0 = length; sub-storages live at indices `0..length`.
pub struct SubStorageVector<D> {
    storage: Storage<D>,
    length: StorageBackedUint64,
}

pub fn open_sub_storage_vector<D: Database>(storage: Storage<D>) -> SubStorageVector<D> {
    let base_key = storage.base_key();
    SubStorageVector {
        length: StorageBackedUint64::new(base_key, LENGTH_OFFSET),
        storage,
    }
}

impl<D: Database> SubStorageVector<D> {
    pub fn length<B: StorageBackend>(&self, backend: &mut B) -> Result<u64, StorageError> {
        self.length.get(backend)
    }

    pub fn at(&self, index: u64) -> Storage<D> {
        self.storage.open_sub_storage(&index.to_be_bytes())
    }

    pub fn push<B: StorageBackend>(&self, backend: &mut B) -> Result<Storage<D>, StorageError> {
        let len = self.length.get(backend)?;
        self.length.set(backend, len + 1)?;
        Ok(self.at(len))
    }

    pub fn pop<B: StorageBackend>(&self, backend: &mut B) -> Result<Option<u64>, StorageError> {
        let len = self.length.get(backend)?;
        if len == 0 {
            return Ok(None);
        }
        let new_len = len - 1;
        self.length.set(backend, new_len)?;
        Ok(Some(new_len))
    }
}
