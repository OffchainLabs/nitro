use alloy_primitives::{B256, U256};
use arb_storage_errors::StorageError;

use crate::{
    backed_types::StorageBackedUint64, backend::StorageBackend, slot::storage_key_map,
    state_ops::ARBOS_STATE_ADDRESS, storage::Storage,
};

/// FIFO queue backed by ArbOS storage.
///
/// Layout: offset 0 = next put position, offset 1 = next get position;
/// data lives at offsets 2+.
#[derive(Clone, Copy, Debug)]
pub struct Queue {
    pub base_key: B256,
    next_put: StorageBackedUint64,
    next_get: StorageBackedUint64,
}

fn compute_slot(base_key: B256, offset: u64) -> U256 {
    if base_key == B256::ZERO {
        storage_key_map(&[], offset)
    } else {
        storage_key_map(base_key.as_slice(), offset)
    }
}

pub fn initialize_queue<D: revm::Database>(storage: &Storage<D>) -> Result<(), StorageError> {
    storage.set_uint64_by_uint64(0, 2)?;
    storage.set_uint64_by_uint64(1, 2)?;
    Ok(())
}

pub fn open_queue<D>(storage: Storage<D>) -> Queue {
    open_queue_at(storage.base_key())
}

pub fn open_queue_at(base_key: B256) -> Queue {
    Queue {
        base_key,
        next_put: StorageBackedUint64::new(base_key, 0),
        next_get: StorageBackedUint64::new(base_key, 1),
    }
}

impl Queue {
    fn load_slot<B: StorageBackend>(
        &self,
        backend: &mut B,
        offset: u64,
    ) -> Result<B256, StorageError> {
        let slot = compute_slot(self.base_key, offset);
        let value = backend
            .sload(ARBOS_STATE_ADDRESS, slot)
            .map_err(Into::into)?;
        Ok(B256::from(value.to_be_bytes::<32>()))
    }

    fn store_slot<B: StorageBackend>(
        &self,
        backend: &mut B,
        offset: u64,
        value: B256,
    ) -> Result<(), StorageError> {
        let slot = compute_slot(self.base_key, offset);
        backend
            .sstore(ARBOS_STATE_ADDRESS, slot, U256::from_be_bytes(value.0))
            .map_err(Into::into)
    }

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
        let val = self.load_slot(backend, get)?;
        Ok(Some(val))
    }

    pub fn get<B: StorageBackend>(&self, backend: &mut B) -> Result<Option<B256>, StorageError> {
        if self.is_empty(backend)? {
            return Ok(None);
        }
        let get = self.next_get.get(backend)?;
        let val = self.load_slot(backend, get)?;
        self.store_slot(backend, get, B256::ZERO)?;
        self.next_get.set(backend, get + 1)?;
        Ok(Some(val))
    }

    pub fn put<B: StorageBackend>(&self, backend: &mut B, value: B256) -> Result<(), StorageError> {
        let put = self.next_put.get(backend)?;
        self.store_slot(backend, put, value)?;
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
        let val = self.load_slot(backend, idx)?;
        self.store_slot(backend, idx, B256::ZERO)?;
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
            let val = self.load_slot(backend, i)?;
            f(val)?;
        }
        Ok(())
    }
}
