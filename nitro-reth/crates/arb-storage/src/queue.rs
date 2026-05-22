use alloy_primitives::B256;
use arb_storage_errors::StorageError;
use revm::Database;

use crate::{backed_types::StorageBackedUint64, backend::StorageBackend, storage::Storage};

/// FIFO queue backed by ArbOS storage.
///
/// Layout: offset 0 = next put position, offset 1 = next get position;
/// data lives at offsets 2+.
pub struct Queue<D> {
    pub storage: Storage<D>,
    next_put: StorageBackedUint64,
    next_get: StorageBackedUint64,
}

pub fn initialize_queue<D: Database>(storage: &Storage<D>) -> Result<(), StorageError> {
    storage.set_uint64_by_uint64(0, 2)?;
    storage.set_uint64_by_uint64(1, 2)?;
    Ok(())
}

pub fn open_queue<D: Database>(storage: Storage<D>) -> Queue<D> {
    let base_key = storage.base_key();
    Queue {
        next_put: StorageBackedUint64::new(base_key, 0),
        next_get: StorageBackedUint64::new(base_key, 1),
        storage,
    }
}

impl<D: Database> Queue<D> {
    pub fn is_empty<B: StorageBackend>(&self, backend: &mut B) -> Result<bool, StorageError> {
        let put = self.next_put.get(backend)?;
        let get = self.next_get.get(backend)?;
        Ok(put == get)
    }

    pub fn size<B: StorageBackend>(&self, backend: &mut B) -> Result<u64, StorageError> {
        let put = self.next_put.get(backend)?;
        let get = self.next_get.get(backend)?;
        Ok(put.saturating_sub(get))
    }

    pub fn peek<B: StorageBackend>(&self, backend: &mut B) -> Result<Option<B256>, StorageError> {
        if self.is_empty(backend)? {
            return Ok(None);
        }
        let get = self.next_get.get(backend)?;
        let val = self.storage.get_by_uint64(get)?;
        Ok(Some(val))
    }

    pub fn get<B: StorageBackend>(&self, backend: &mut B) -> Result<Option<B256>, StorageError> {
        if self.is_empty(backend)? {
            return Ok(None);
        }
        let get = self.next_get.get(backend)?;
        let val = self.storage.get_by_uint64(get)?;
        self.storage.set_by_uint64(get, B256::ZERO)?;
        self.next_get.set(backend, get + 1)?;
        Ok(Some(val))
    }

    pub fn put<B: StorageBackend>(&self, backend: &mut B, value: B256) -> Result<(), StorageError> {
        let put = self.next_put.get(backend)?;
        self.storage.set_by_uint64(put, value)?;
        self.next_put.set(backend, put + 1)?;
        Ok(())
    }

    /// Removes the last element from the back (most recently put).
    pub fn shift<B: StorageBackend>(&self, backend: &mut B) -> Result<Option<B256>, StorageError> {
        if self.is_empty(backend)? {
            return Ok(None);
        }
        let put = self.next_put.get(backend)?;
        let idx = put - 1;
        let val = self.storage.get_by_uint64(idx)?;
        self.storage.set_by_uint64(idx, B256::ZERO)?;
        self.next_put.set(backend, idx)?;
        Ok(Some(val))
    }

    pub fn for_each<F, E, B>(&self, backend: &mut B, mut f: F) -> Result<(), E>
    where
        F: FnMut(B256) -> Result<(), E>,
        E: From<StorageError>,
        B: StorageBackend,
    {
        let get = self.next_get.get(backend)?;
        let put = self.next_put.get(backend)?;
        for i in get..put {
            let val = self.storage.get_by_uint64(i)?;
            f(val)?;
        }
        Ok(())
    }
}
