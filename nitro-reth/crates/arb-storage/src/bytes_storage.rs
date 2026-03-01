use alloy_primitives::B256;
use revm::Database;

use crate::storage::Storage;

/// Variable-length byte storage.
///
/// Layout: slot 0 = length, slots 1..N = data (32 bytes per slot, left-aligned for last chunk).
pub struct StorageBackedBytes<D> {
    storage: Storage<D>,
}

impl<D: Database> StorageBackedBytes<D> {
    pub fn new(storage: Storage<D>) -> Self {
        Self { storage }
    }

    pub fn get(&self) -> Result<Vec<u8>, ()> {
        let mut bytes_left = self.storage.get_uint64_by_uint64(0)? as usize;
        if bytes_left == 0 {
            return Ok(Vec::new());
        }
        let mut ret = Vec::with_capacity(bytes_left);
        let mut offset = 1u64;
        while bytes_left >= 32 {
            let next = self.storage.get_by_uint64(offset)?;
            ret.extend_from_slice(next.as_slice());
            bytes_left -= 32;
            offset += 1;
        }
        if bytes_left > 0 {
            let next = self.storage.get_by_uint64(offset)?;
            ret.extend_from_slice(&next.as_slice()[32 - bytes_left..]);
        }
        Ok(ret)
    }

    pub fn set(&self, b: &[u8]) -> Result<(), ()> {
        self.clear()?;
        self.storage.set_uint64_by_uint64(0, b.len() as u64)?;
        let mut remaining = b;
        let mut offset = 1u64;
        while remaining.len() >= 32 {
            let mut slot = [0u8; 32];
            slot.copy_from_slice(&remaining[..32]);
            self.storage.set_by_uint64(offset, B256::from(slot))?;
            remaining = &remaining[32..];
            offset += 1;
        }
        if !remaining.is_empty() {
            // Right-align remaining bytes (matching Go's common.BytesToHash).
            let mut slot = [0u8; 32];
            slot[32 - remaining.len()..].copy_from_slice(remaining);
            self.storage.set_by_uint64(offset, B256::from(slot))?;
        }
        Ok(())
    }

    pub fn clear(&self) -> Result<(), ()> {
        let bytes_left = self.storage.get_uint64_by_uint64(0)?;
        let mut offset = 1u64;
        let mut remaining = bytes_left;
        while remaining > 0 {
            self.storage.set_by_uint64(offset, B256::ZERO)?;
            offset += 1;
            remaining = remaining.saturating_sub(32);
        }
        self.storage.set_uint64_by_uint64(0, 0)?;
        Ok(())
    }

    pub fn size(&self) -> Result<u64, ()> {
        self.storage.get_uint64_by_uint64(0)
    }
}

impl<D> Clone for StorageBackedBytes<D> {
    fn clone(&self) -> Self {
        Self {
            storage: self.storage.clone(),
        }
    }
}
