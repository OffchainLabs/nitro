use alloy_primitives::B256;
use arb_storage_errors::StorageError;

use crate::{
    backed_types::StorageBackedUint64,
    backend::{StorageBackend, SystemStateBackend},
    slot::derive_sub_key,
    storage::Storage,
};

const LENGTH_OFFSET: u64 = 0;

/// Vector of sub-storages backed by ArbOS storage.
///
/// Layout: offset 0 = length; sub-storages live at indices `0..length`.
#[derive(Clone, Copy, Debug)]
pub struct SubStorageVector {
    pub base_key: B256,
    length: StorageBackedUint64,
}

pub fn open_sub_storage_vector<D>(storage: Storage<'_, D>) -> SubStorageVector {
    open_sub_storage_vector_at(storage.base_key())
}

pub(crate) fn open_sub_storage_vector_at(base_key: B256) -> SubStorageVector {
    SubStorageVector {
        base_key,
        length: StorageBackedUint64::new(base_key, LENGTH_OFFSET),
    }
}

impl SubStorageVector {
    pub fn length<B: SystemStateBackend>(&self, backend: &mut B) -> Result<u64, StorageError> {
        self.length.get(backend)
    }

    /// Returns the base key for the sub-storage at `index`.
    pub fn at(&self, index: u64) -> B256 {
        derive_sub_key(self.base_key, &index.to_be_bytes())
    }

    pub fn push<B: StorageBackend>(&self, backend: &mut B) -> Result<B256, StorageError> {
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
