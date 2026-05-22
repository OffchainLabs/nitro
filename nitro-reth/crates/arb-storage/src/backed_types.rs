use alloy_primitives::{Address, B256, U256};
use arb_storage_errors::StorageError;
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

fn read_slot<D: Database>(
    state: *mut revm::database::State<D>,
    slot: U256,
) -> Result<U256, StorageError> {
    // SAFETY: callers of `StorageBacked*::new` must guarantee that the
    // referenced `State<D>` outlives the descriptor and that no other
    // mutable reference to it is live during this call.
    let state = unsafe { &mut *state };
    read_arbos_storage(state, slot)
}

fn write_slot<D: Database>(
    state: *mut revm::database::State<D>,
    slot: U256,
    value: U256,
) -> Result<(), StorageError> {
    // SAFETY: see `read_slot`.
    let state = unsafe { &mut *state };
    write_arbos_storage(state, slot, value)
}

fn decode_address(slot: U256, value: U256) -> Result<Address, StorageError> {
    let bytes = value.to_be_bytes::<32>();
    let addr_bytes: [u8; 20] =
        bytes[12..32]
            .try_into()
            .map_err(|_| StorageError::InvalidLayout {
                slot,
                reason: "address slot must be 20 bytes right-aligned",
            })?;
    Ok(Address::from(addr_bytes))
}

/// Storage-backed 64-bit unsigned integer.
pub struct StorageBackedUint64<D> {
    pub state: *mut revm::database::State<D>,
    pub slot: U256,
}

impl<D: Database> StorageBackedUint64<D> {
    pub fn new(state: *mut revm::database::State<D>, base_key: B256, offset: u64) -> Self {
        Self {
            state,
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get(&self) -> Result<u64, StorageError> {
        let value = read_slot(self.state, self.slot)?;
        Ok(value.try_into().unwrap_or(0))
    }

    pub fn set(&self, value: u64) -> Result<(), StorageError> {
        write_slot(self.state, self.slot, U256::from(value))
    }
}

impl<D> Clone for StorageBackedUint64<D> {
    fn clone(&self) -> Self {
        Self {
            state: self.state,
            slot: self.slot,
        }
    }
}

unsafe impl<D: Send> Send for StorageBackedUint64<D> {}
unsafe impl<D: Sync> Sync for StorageBackedUint64<D> {}

/// Storage-backed 256-bit unsigned integer.
pub struct StorageBackedBigUint<D> {
    state: *mut revm::database::State<D>,
    slot: U256,
}

impl<D: Database> StorageBackedBigUint<D> {
    pub fn new(state: *mut revm::database::State<D>, base_key: B256, offset: u64) -> Self {
        Self {
            state,
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get(&self) -> Result<U256, StorageError> {
        read_slot(self.state, self.slot)
    }

    pub fn set(&self, value: U256) -> Result<(), StorageError> {
        write_slot(self.state, self.slot, value)
    }
}

impl<D> Clone for StorageBackedBigUint<D> {
    fn clone(&self) -> Self {
        Self {
            state: self.state,
            slot: self.slot,
        }
    }
}

unsafe impl<D: Send> Send for StorageBackedBigUint<D> {}
unsafe impl<D: Sync> Sync for StorageBackedBigUint<D> {}

/// Storage-backed Ethereum address (20 bytes, right-aligned in 32-byte slot).
pub struct StorageBackedAddress<D> {
    state: *mut revm::database::State<D>,
    slot: U256,
}

impl<D: Database> StorageBackedAddress<D> {
    pub fn new(state: *mut revm::database::State<D>, base_key: B256, offset: u64) -> Self {
        Self {
            state,
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get(&self) -> Result<Address, StorageError> {
        let value = read_slot(self.state, self.slot)?;
        decode_address(self.slot, value)
    }

    pub fn set(&self, value: Address) -> Result<(), StorageError> {
        let mut value_bytes = [0u8; 32];
        value_bytes[12..32].copy_from_slice(value.as_slice());
        write_slot(self.state, self.slot, U256::from_be_bytes(value_bytes))
    }
}

impl<D> Clone for StorageBackedAddress<D> {
    fn clone(&self) -> Self {
        Self {
            state: self.state,
            slot: self.slot,
        }
    }
}

unsafe impl<D: Send> Send for StorageBackedAddress<D> {}
unsafe impl<D: Sync> Sync for StorageBackedAddress<D> {}

/// Storage-backed signed 64-bit integer, bit-reinterpreting `i64` as `u64`.
pub struct StorageBackedInt64<D> {
    state: *mut revm::database::State<D>,
    slot: U256,
}

impl<D: Database> StorageBackedInt64<D> {
    pub fn new(state: *mut revm::database::State<D>, base_key: B256, offset: u64) -> Self {
        Self {
            state,
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get(&self) -> Result<i64, StorageError> {
        let value = read_slot(self.state, self.slot)?;
        let value_u64: u64 = value.try_into().unwrap_or(0);
        Ok(value_u64 as i64)
    }

    pub fn set(&self, value: i64) -> Result<(), StorageError> {
        write_slot(self.state, self.slot, U256::from(value as u64))
    }
}

impl<D> Clone for StorageBackedInt64<D> {
    fn clone(&self) -> Self {
        Self {
            state: self.state,
            slot: self.slot,
        }
    }
}

unsafe impl<D: Send> Send for StorageBackedInt64<D> {}
unsafe impl<D: Sync> Sync for StorageBackedInt64<D> {}

/// Storage-backed signed 256-bit integer using two's complement.
pub struct StorageBackedBigInt<D> {
    pub state: *mut revm::database::State<D>,
    pub slot: U256,
}

impl<D: Database> StorageBackedBigInt<D> {
    pub fn new(state: *mut revm::database::State<D>, base_key: B256, offset: u64) -> Self {
        Self {
            state,
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get_raw(&self) -> Result<U256, StorageError> {
        read_slot(self.state, self.slot)
    }

    pub fn is_negative(&self) -> Result<bool, StorageError> {
        Ok(self.get_raw()?.bit(255))
    }

    /// Returns `(magnitude, is_negative)` decoded from two's complement.
    pub fn get_signed(&self) -> Result<(U256, bool), StorageError> {
        let raw = self.get_raw()?;
        if raw.bit(255) {
            let magnitude = (!raw).wrapping_add(U256::from(1));
            Ok((magnitude, true))
        } else {
            Ok((raw, false))
        }
    }

    pub fn set(&self, value: U256) -> Result<(), StorageError> {
        write_slot(self.state, self.slot, value)
    }

    pub fn set_negative(&self, magnitude: U256) -> Result<(), StorageError> {
        let neg_value = (!magnitude).wrapping_add(U256::from(1));
        self.set(neg_value)
    }
}

impl<D> Clone for StorageBackedBigInt<D> {
    fn clone(&self) -> Self {
        Self {
            state: self.state,
            slot: self.slot,
        }
    }
}

unsafe impl<D: Send> Send for StorageBackedBigInt<D> {}
unsafe impl<D: Sync> Sync for StorageBackedBigInt<D> {}

/// Sentinel value for nil addresses: `1 << 255`.
fn nil_address_representation() -> U256 {
    U256::from(1u64) << 255
}

/// Storage-backed optional address, using `1 << 255` to represent `None`.
pub struct StorageBackedAddressOrNil<D> {
    state: *mut revm::database::State<D>,
    slot: U256,
}

impl<D: Database> StorageBackedAddressOrNil<D> {
    pub fn new(state: *mut revm::database::State<D>, base_key: B256, offset: u64) -> Self {
        Self {
            state,
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get(&self) -> Result<Option<Address>, StorageError> {
        let value = read_slot(self.state, self.slot)?;
        if value == nil_address_representation() {
            return Ok(None);
        }
        decode_address(self.slot, value).map(Some)
    }

    pub fn set(&self, value: Option<Address>) -> Result<(), StorageError> {
        let value_u256 = match value {
            None => nil_address_representation(),
            Some(addr) => {
                let mut bytes = [0u8; 32];
                bytes[12..32].copy_from_slice(addr.as_slice());
                U256::from_be_bytes(bytes)
            }
        };
        write_slot(self.state, self.slot, value_u256)
    }
}

impl<D> Clone for StorageBackedAddressOrNil<D> {
    fn clone(&self) -> Self {
        Self {
            state: self.state,
            slot: self.slot,
        }
    }
}

unsafe impl<D: Send> Send for StorageBackedAddressOrNil<D> {}
unsafe impl<D: Sync> Sync for StorageBackedAddressOrNil<D> {}
