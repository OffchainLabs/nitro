use alloy_primitives::{B256, U256};
use revm::Database;

use crate::{
    slot::storage_key_map,
    state_ops::{read_arbos_storage, write_arbos_storage},
};

fn compute_slot(base_key: B256, offset: u64) -> U256 {
    if base_key == B256::ZERO {
        storage_key_map(&[], offset)
    } else {
        storage_key_map(base_key.as_slice(), offset)
    }
}

fn read_slot<D: Database>(state: *mut revm::database::State<D>, slot: U256) -> Result<U256, ()> {
    unsafe {
        let state = &mut *state;
        Ok(read_arbos_storage(state, slot))
    }
}

fn write_slot<D: Database>(
    state: *mut revm::database::State<D>,
    slot: U256,
    value: U256,
) -> Result<(), ()> {
    unsafe {
        let state = &mut *state;
        write_arbos_storage(state, slot, value);
        Ok(())
    }
}

/// Basis points stored as signed i64. 10000 bips = 100%.
pub struct StorageBackedBips<D> {
    state: *mut revm::database::State<D>,
    slot: U256,
}

impl<D: Database> StorageBackedBips<D> {
    pub fn new(state: *mut revm::database::State<D>, base_key: B256, offset: u64) -> Self {
        Self {
            state,
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get(&self) -> Result<i64, ()> {
        let value = read_slot(self.state, self.slot)?;
        let value_u64: u64 = value.try_into().unwrap_or(0);
        Ok(value_u64 as i64)
    }

    pub fn set(&self, value: i64) -> Result<(), ()> {
        write_slot(self.state, self.slot, U256::from(value as u64))
    }
}

impl<D> Clone for StorageBackedBips<D> {
    fn clone(&self) -> Self {
        Self {
            state: self.state,
            slot: self.slot,
        }
    }
}

unsafe impl<D: Send> Send for StorageBackedBips<D> {}
unsafe impl<D: Sync> Sync for StorageBackedBips<D> {}

/// Unsigned basis points stored as u64. 10000 ubips = 100%.
pub struct StorageBackedUBips<D> {
    state: *mut revm::database::State<D>,
    slot: U256,
}

impl<D: Database> StorageBackedUBips<D> {
    pub fn new(state: *mut revm::database::State<D>, base_key: B256, offset: u64) -> Self {
        Self {
            state,
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get(&self) -> Result<u64, ()> {
        let value = read_slot(self.state, self.slot)?;
        Ok(value.try_into().unwrap_or(0))
    }

    pub fn set(&self, value: u64) -> Result<(), ()> {
        write_slot(self.state, self.slot, U256::from(value))
    }
}

impl<D> Clone for StorageBackedUBips<D> {
    fn clone(&self) -> Self {
        Self {
            state: self.state,
            slot: self.slot,
        }
    }
}

unsafe impl<D: Send> Send for StorageBackedUBips<D> {}
unsafe impl<D: Sync> Sync for StorageBackedUBips<D> {}

/// Storage-backed 16-bit unsigned integer.
pub struct StorageBackedUint16<D> {
    state: *mut revm::database::State<D>,
    slot: U256,
}

impl<D: Database> StorageBackedUint16<D> {
    pub fn new(state: *mut revm::database::State<D>, base_key: B256, offset: u64) -> Self {
        Self {
            state,
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get(&self) -> Result<u16, ()> {
        let value = read_slot(self.state, self.slot)?;
        Ok(value.try_into().unwrap_or(0))
    }

    pub fn set(&self, value: u16) -> Result<(), ()> {
        write_slot(self.state, self.slot, U256::from(value))
    }
}

impl<D> Clone for StorageBackedUint16<D> {
    fn clone(&self) -> Self {
        Self {
            state: self.state,
            slot: self.slot,
        }
    }
}

unsafe impl<D: Send> Send for StorageBackedUint16<D> {}
unsafe impl<D: Sync> Sync for StorageBackedUint16<D> {}

/// Storage-backed 24-bit unsigned integer.
pub struct StorageBackedUint24<D> {
    state: *mut revm::database::State<D>,
    slot: U256,
}

impl<D: Database> StorageBackedUint24<D> {
    pub fn new(state: *mut revm::database::State<D>, base_key: B256, offset: u64) -> Self {
        Self {
            state,
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get(&self) -> Result<u32, ()> {
        let value = read_slot(self.state, self.slot)?;
        let raw: u32 = value.try_into().unwrap_or(0);
        Ok(raw & 0xFF_FFFF)
    }

    pub fn set(&self, value: u32) -> Result<(), ()> {
        write_slot(self.state, self.slot, U256::from(value & 0xFF_FFFF))
    }
}

impl<D> Clone for StorageBackedUint24<D> {
    fn clone(&self) -> Self {
        Self {
            state: self.state,
            slot: self.slot,
        }
    }
}

unsafe impl<D: Send> Send for StorageBackedUint24<D> {}
unsafe impl<D: Sync> Sync for StorageBackedUint24<D> {}

/// Storage-backed 32-bit unsigned integer.
pub struct StorageBackedUint32<D> {
    state: *mut revm::database::State<D>,
    slot: U256,
}

impl<D: Database> StorageBackedUint32<D> {
    pub fn new(state: *mut revm::database::State<D>, base_key: B256, offset: u64) -> Self {
        Self {
            state,
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get(&self) -> Result<u32, ()> {
        let value = read_slot(self.state, self.slot)?;
        Ok(value.try_into().unwrap_or(0))
    }

    pub fn set(&self, value: u32) -> Result<(), ()> {
        write_slot(self.state, self.slot, U256::from(value))
    }

    pub fn clear(&self) -> Result<(), ()> {
        self.set(0)
    }
}

impl<D> Clone for StorageBackedUint32<D> {
    fn clone(&self) -> Self {
        Self {
            state: self.state,
            slot: self.slot,
        }
    }
}

unsafe impl<D: Send> Send for StorageBackedUint32<D> {}
unsafe impl<D: Sync> Sync for StorageBackedUint32<D> {}
