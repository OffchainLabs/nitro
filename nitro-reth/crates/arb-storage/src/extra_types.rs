use alloy_primitives::{B256, U256};
use arb_storage_errors::StorageError;

use crate::{
    backend::{StorageBackend, SystemStateBackend},
    slot::storage_key_map,
    state_ops::ARBOS_STATE_ADDRESS,
};

fn compute_slot(base_key: B256, offset: u64) -> U256 {
    if base_key == B256::ZERO {
        storage_key_map(&[], offset)
    } else {
        storage_key_map(base_key.as_slice(), offset)
    }
}

/// Basis points stored as signed `i64`. 10000 bips = 100%.
#[derive(Clone, Copy, Debug)]
pub struct StorageBackedBips {
    pub slot: U256,
}

impl StorageBackedBips {
    pub fn new(base_key: B256, offset: u64) -> Self {
        Self {
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get<B: SystemStateBackend>(&self, backend: &mut B) -> Result<i64, StorageError> {
        let value = backend
            .sload_system(ARBOS_STATE_ADDRESS, self.slot)
            .map_err(Into::into)?;
        let value_u64: u64 = value.try_into().unwrap_or(0);
        Ok(value_u64 as i64)
    }

    pub fn set<B: StorageBackend>(&self, backend: &mut B, value: i64) -> Result<(), StorageError> {
        backend
            .sstore(ARBOS_STATE_ADDRESS, self.slot, U256::from(value as u64))
            .map_err(Into::into)
    }
}

/// Unsigned basis points stored as `u64`. 10000 ubips = 100%.
#[derive(Clone, Copy, Debug)]
pub struct StorageBackedUBips {
    pub slot: U256,
}

impl StorageBackedUBips {
    pub fn new(base_key: B256, offset: u64) -> Self {
        Self {
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get<B: SystemStateBackend>(&self, backend: &mut B) -> Result<u64, StorageError> {
        let value = backend
            .sload_system(ARBOS_STATE_ADDRESS, self.slot)
            .map_err(Into::into)?;
        Ok(value.try_into().unwrap_or(0))
    }

    pub fn set<B: StorageBackend>(&self, backend: &mut B, value: u64) -> Result<(), StorageError> {
        backend
            .sstore(ARBOS_STATE_ADDRESS, self.slot, U256::from(value))
            .map_err(Into::into)
    }
}

/// Storage-backed 16-bit unsigned integer.
#[derive(Clone, Copy, Debug)]
pub struct StorageBackedUint16 {
    pub slot: U256,
}

impl StorageBackedUint16 {
    pub fn new(base_key: B256, offset: u64) -> Self {
        Self {
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get<B: SystemStateBackend>(&self, backend: &mut B) -> Result<u16, StorageError> {
        let value = backend
            .sload_system(ARBOS_STATE_ADDRESS, self.slot)
            .map_err(Into::into)?;
        Ok(value.try_into().unwrap_or(0))
    }

    pub fn set<B: StorageBackend>(&self, backend: &mut B, value: u16) -> Result<(), StorageError> {
        backend
            .sstore(ARBOS_STATE_ADDRESS, self.slot, U256::from(value))
            .map_err(Into::into)
    }
}

/// Storage-backed 24-bit unsigned integer.
#[derive(Clone, Copy, Debug)]
pub struct StorageBackedUint24 {
    pub slot: U256,
}

impl StorageBackedUint24 {
    pub fn new(base_key: B256, offset: u64) -> Self {
        Self {
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get<B: SystemStateBackend>(&self, backend: &mut B) -> Result<u32, StorageError> {
        let value = backend
            .sload_system(ARBOS_STATE_ADDRESS, self.slot)
            .map_err(Into::into)?;
        let raw: u32 = value.try_into().unwrap_or(0);
        Ok(raw & 0xFF_FFFF)
    }

    pub fn set<B: StorageBackend>(&self, backend: &mut B, value: u32) -> Result<(), StorageError> {
        backend
            .sstore(
                ARBOS_STATE_ADDRESS,
                self.slot,
                U256::from(value & 0xFF_FFFF),
            )
            .map_err(Into::into)
    }
}

/// Storage-backed 32-bit unsigned integer.
#[derive(Clone, Copy, Debug)]
pub struct StorageBackedUint32 {
    pub slot: U256,
}

impl StorageBackedUint32 {
    pub fn new(base_key: B256, offset: u64) -> Self {
        Self {
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get<B: SystemStateBackend>(&self, backend: &mut B) -> Result<u32, StorageError> {
        let value = backend
            .sload_system(ARBOS_STATE_ADDRESS, self.slot)
            .map_err(Into::into)?;
        Ok(value.try_into().unwrap_or(0))
    }

    pub fn set<B: StorageBackend>(&self, backend: &mut B, value: u32) -> Result<(), StorageError> {
        backend
            .sstore(ARBOS_STATE_ADDRESS, self.slot, U256::from(value))
            .map_err(Into::into)
    }

    pub fn clear<B: StorageBackend>(&self, backend: &mut B) -> Result<(), StorageError> {
        self.set(backend, 0)
    }
}
