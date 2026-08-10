use alloy_primitives::{Address, B256, U256};
use arb_storage_errors::StorageError;

use crate::{
    backend::{StorageBackend, SystemStateBackend},
    slot::storage_key_map,
    state_ops::ARBOS_STATE_ADDRESS,
};

/// Variable-length byte storage.
///
/// Layout: offset 0 holds the byte length; offsets 1..N hold 32-byte chunks
/// with the trailing partial chunk right-aligned in its slot.
#[derive(Clone, Copy, Debug)]
pub struct StorageBackedBytes {
    pub base_key: B256,
    pub account: Address,
}

impl StorageBackedBytes {
    pub fn new(base_key: B256) -> Self {
        Self {
            base_key,
            account: ARBOS_STATE_ADDRESS,
        }
    }

    pub fn new_with_account(base_key: B256, account: Address) -> Self {
        Self { base_key, account }
    }

    fn slot(&self, offset: u64) -> U256 {
        let key: &[u8] = if self.base_key == B256::ZERO {
            &[]
        } else {
            self.base_key.as_slice()
        };
        storage_key_map(key, offset)
    }

    fn load_u64<B: SystemStateBackend>(
        &self,
        backend: &mut B,
        offset: u64,
    ) -> Result<u64, StorageError> {
        let value = backend
            .sload_system(self.account, self.slot(offset))
            .map_err(Into::into)?;
        Ok(value.try_into().unwrap_or(0))
    }

    fn store_u64<B: StorageBackend>(
        &self,
        backend: &mut B,
        offset: u64,
        value: u64,
    ) -> Result<(), StorageError> {
        backend
            .sstore(self.account, self.slot(offset), U256::from(value))
            .map_err(Into::into)
    }

    fn load_word<B: SystemStateBackend>(
        &self,
        backend: &mut B,
        offset: u64,
    ) -> Result<[u8; 32], StorageError> {
        let value = backend
            .sload_system(self.account, self.slot(offset))
            .map_err(Into::into)?;
        Ok(value.to_be_bytes::<32>())
    }

    fn store_word<B: StorageBackend>(
        &self,
        backend: &mut B,
        offset: u64,
        word: [u8; 32],
    ) -> Result<(), StorageError> {
        backend
            .sstore(self.account, self.slot(offset), U256::from_be_bytes(word))
            .map_err(Into::into)
    }

    pub fn get<B: SystemStateBackend>(&self, backend: &mut B) -> Result<Vec<u8>, StorageError> {
        let mut bytes_left = self.load_u64(backend, 0)? as usize;
        if bytes_left == 0 {
            return Ok(Vec::new());
        }
        let mut ret = Vec::with_capacity(bytes_left);
        let mut offset = 1u64;
        while bytes_left >= 32 {
            let word = self.load_word(backend, offset)?;
            ret.extend_from_slice(&word);
            bytes_left -= 32;
            offset += 1;
        }
        if bytes_left > 0 {
            let word = self.load_word(backend, offset)?;
            ret.extend_from_slice(&word[32 - bytes_left..]);
        }
        Ok(ret)
    }

    pub fn set<B: StorageBackend>(&self, backend: &mut B, b: &[u8]) -> Result<(), StorageError> {
        self.clear(backend)?;
        self.store_u64(backend, 0, b.len() as u64)?;
        let mut remaining = b;
        let mut offset = 1u64;
        while remaining.len() >= 32 {
            let mut word = [0u8; 32];
            word.copy_from_slice(&remaining[..32]);
            self.store_word(backend, offset, word)?;
            remaining = &remaining[32..];
            offset += 1;
        }
        if !remaining.is_empty() {
            let mut word = [0u8; 32];
            word[32 - remaining.len()..].copy_from_slice(remaining);
            self.store_word(backend, offset, word)?;
        }
        Ok(())
    }

    pub fn clear<B: StorageBackend>(&self, backend: &mut B) -> Result<(), StorageError> {
        let bytes_left = self.load_u64(backend, 0)?;
        let mut offset = 1u64;
        let mut remaining = bytes_left;
        while remaining > 0 {
            self.store_word(backend, offset, [0u8; 32])?;
            offset += 1;
            remaining = remaining.saturating_sub(32);
        }
        self.store_u64(backend, 0, 0)
    }

    pub fn size<B: SystemStateBackend>(&self, backend: &mut B) -> Result<u64, StorageError> {
        self.load_u64(backend, 0)
    }
}
