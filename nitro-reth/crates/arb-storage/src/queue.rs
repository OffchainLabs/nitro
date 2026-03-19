use alloy_primitives::B256;
use revm::Database;

use crate::{backed_types::StorageBackedUint64, storage::Storage};

/// A FIFO queue backed by ArbOS storage.
///
/// Layout: offset 0 = next put position, offset 1 = next get position.
/// Data stored at offsets 2+.
pub struct Queue<D> {
    pub storage: Storage<D>,
    next_put: StorageBackedUint64<D>,
    next_get: StorageBackedUint64<D>,
}

/// Initializes a queue by setting both offsets to 2 (data starts at offset 2).
pub fn initialize_queue<D: Database>(storage: &Storage<D>) -> Result<(), ()> {
    storage.set_uint64_by_uint64(0, 2)?;
    storage.set_uint64_by_uint64(1, 2)?;
    Ok(())
}

/// Opens an existing queue from storage.
pub fn open_queue<D: Database>(storage: Storage<D>) -> Queue<D> {
    let state = storage.state_ptr();
    let base_key = storage.base_key();
    Queue {
        next_put: StorageBackedUint64::new(state, base_key, 0),
        next_get: StorageBackedUint64::new(state, base_key, 1),
        storage,
    }
}

impl<D: Database> Queue<D> {
    pub fn is_empty(&self) -> Result<bool, ()> {
        let put = self.next_put.get()?;
        let get = self.next_get.get()?;
        Ok(put == get)
    }

    pub fn size(&self) -> Result<u64, ()> {
        let put = self.next_put.get()?;
        let get = self.next_get.get()?;
        Ok(put.saturating_sub(get))
    }

    pub fn peek(&self) -> Result<Option<B256>, ()> {
        if self.is_empty()? {
            return Ok(None);
        }
        let get = self.next_get.get()?;
        let val = self.storage.get_by_uint64(get)?;
        Ok(Some(val))
    }

    pub fn get(&self) -> Result<Option<B256>, ()> {
        if self.is_empty()? {
            return Ok(None);
        }
        let get = self.next_get.get()?;
        let val = self.storage.get_by_uint64(get)?;
        self.storage.set_by_uint64(get, B256::ZERO)?;
        self.next_get.set(get + 1)?;
        Ok(Some(val))
    }

    pub fn put(&self, value: B256) -> Result<(), ()> {
        let put = self.next_put.get()?;
        self.storage.set_by_uint64(put, value)?;
        self.next_put.set(put + 1)?;
        Ok(())
    }

    /// Removes the last element from the back (most recently put).
    pub fn shift(&self) -> Result<Option<B256>, ()> {
        if self.is_empty()? {
            return Ok(None);
        }
        let put = self.next_put.get()?;
        let idx = put - 1;
        let val = self.storage.get_by_uint64(idx)?;
        self.storage.set_by_uint64(idx, B256::ZERO)?;
        self.next_put.set(idx)?;
        Ok(Some(val))
    }

    /// Iterates over all elements in order.
    pub fn for_each<F>(&self, mut f: F) -> Result<(), ()>
    where
        F: FnMut(B256) -> Result<(), ()>,
    {
        let get = self.next_get.get()?;
        let put = self.next_put.get()?;
        for i in get..put {
            let val = self.storage.get_by_uint64(i)?;
            f(val)?;
        }
        Ok(())
    }
}
