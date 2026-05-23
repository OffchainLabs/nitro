use alloy_primitives::{Address, B256, U256};
use arb_storage_errors::StorageError;

use crate::{backend::StorageBackend, slot::storage_key_map, state_ops::ARBOS_STATE_ADDRESS};

fn compute_slot(base_key: B256, offset: u64) -> U256 {
    if base_key == B256::ZERO {
        storage_key_map(&[], offset)
    } else {
        storage_key_map(base_key.as_slice(), offset)
    }
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
///
/// Holds only the storage slot; the caller passes a [`StorageBackend`] at
/// access time.
#[derive(Clone, Copy, Debug)]
pub struct StorageBackedUint64 {
    pub slot: U256,
}

impl StorageBackedUint64 {
    pub fn new(base_key: B256, offset: u64) -> Self {
        Self {
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get<B: StorageBackend>(&self, backend: &mut B) -> Result<u64, StorageError> {
        let value = backend
            .sload(ARBOS_STATE_ADDRESS, self.slot)
            .map_err(Into::into)?;
        Ok(value.try_into().unwrap_or(0))
    }

    pub fn set<B: StorageBackend>(&self, backend: &mut B, value: u64) -> Result<(), StorageError> {
        backend
            .sstore(ARBOS_STATE_ADDRESS, self.slot, U256::from(value))
            .map_err(Into::into)
    }
}
/// Storage-backed 256-bit unsigned integer.
///
/// Holds only the storage slot; the caller passes a [`StorageBackend`] at
/// access time.
#[derive(Clone, Copy, Debug)]
pub struct StorageBackedBigUint {
    pub slot: U256,
}

impl StorageBackedBigUint {
    pub fn new(base_key: B256, offset: u64) -> Self {
        Self {
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get<B: StorageBackend>(&self, backend: &mut B) -> Result<U256, StorageError> {
        backend
            .sload(ARBOS_STATE_ADDRESS, self.slot)
            .map_err(Into::into)
    }

    pub fn set<B: StorageBackend>(&self, backend: &mut B, value: U256) -> Result<(), StorageError> {
        backend
            .sstore(ARBOS_STATE_ADDRESS, self.slot, value)
            .map_err(Into::into)
    }
}

/// Storage-backed Ethereum address (20 bytes, right-aligned in 32-byte slot).
#[derive(Clone, Copy, Debug)]
pub struct StorageBackedAddress {
    pub slot: U256,
}

impl StorageBackedAddress {
    pub fn new(base_key: B256, offset: u64) -> Self {
        Self {
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get<B: StorageBackend>(&self, backend: &mut B) -> Result<Address, StorageError> {
        let value = backend
            .sload(ARBOS_STATE_ADDRESS, self.slot)
            .map_err(Into::into)?;
        decode_address(self.slot, value)
    }

    pub fn set<B: StorageBackend>(
        &self,
        backend: &mut B,
        value: Address,
    ) -> Result<(), StorageError> {
        let mut value_bytes = [0u8; 32];
        value_bytes[12..32].copy_from_slice(value.as_slice());
        backend
            .sstore(
                ARBOS_STATE_ADDRESS,
                self.slot,
                U256::from_be_bytes(value_bytes),
            )
            .map_err(Into::into)
    }
}

/// Storage-backed signed 64-bit integer, bit-reinterpreting `i64` as `u64`.
#[derive(Clone, Copy, Debug)]
pub struct StorageBackedInt64 {
    pub slot: U256,
}

impl StorageBackedInt64 {
    pub fn new(base_key: B256, offset: u64) -> Self {
        Self {
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get<B: StorageBackend>(&self, backend: &mut B) -> Result<i64, StorageError> {
        let value = backend
            .sload(ARBOS_STATE_ADDRESS, self.slot)
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

/// Storage-backed signed 256-bit integer using two's complement.
#[derive(Clone, Copy, Debug)]
pub struct StorageBackedBigInt {
    pub slot: U256,
}

impl StorageBackedBigInt {
    pub fn new(base_key: B256, offset: u64) -> Self {
        Self {
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get_raw<B: StorageBackend>(&self, backend: &mut B) -> Result<U256, StorageError> {
        backend
            .sload(ARBOS_STATE_ADDRESS, self.slot)
            .map_err(Into::into)
    }

    pub fn is_negative<B: StorageBackend>(&self, backend: &mut B) -> Result<bool, StorageError> {
        Ok(self.get_raw(backend)?.bit(255))
    }

    /// Returns `(magnitude, is_negative)` decoded from two's complement.
    pub fn get_signed<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<(U256, bool), StorageError> {
        let raw = self.get_raw(backend)?;
        if raw.bit(255) {
            let magnitude = (!raw).wrapping_add(U256::from(1));
            Ok((magnitude, true))
        } else {
            Ok((raw, false))
        }
    }

    pub fn set<B: StorageBackend>(&self, backend: &mut B, value: U256) -> Result<(), StorageError> {
        backend
            .sstore(ARBOS_STATE_ADDRESS, self.slot, value)
            .map_err(Into::into)
    }

    pub fn set_negative<B: StorageBackend>(
        &self,
        backend: &mut B,
        magnitude: U256,
    ) -> Result<(), StorageError> {
        let neg_value = (!magnitude).wrapping_add(U256::from(1));
        self.set(backend, neg_value)
    }
}

/// Sentinel value for nil addresses: `1 << 255`.
fn nil_address_representation() -> U256 {
    U256::from(1u64) << 255
}

/// Storage-backed optional address, using `1 << 255` to represent `None`.
#[derive(Clone, Copy, Debug)]
pub struct StorageBackedAddressOrNil {
    pub slot: U256,
}

impl StorageBackedAddressOrNil {
    pub fn new(base_key: B256, offset: u64) -> Self {
        Self {
            slot: compute_slot(base_key, offset),
        }
    }

    pub fn get<B: StorageBackend>(&self, backend: &mut B) -> Result<Option<Address>, StorageError> {
        let value = backend
            .sload(ARBOS_STATE_ADDRESS, self.slot)
            .map_err(Into::into)?;
        if value == nil_address_representation() {
            return Ok(None);
        }
        decode_address(self.slot, value).map(Some)
    }

    pub fn set<B: StorageBackend>(
        &self,
        backend: &mut B,
        value: Option<Address>,
    ) -> Result<(), StorageError> {
        let value_u256 = match value {
            None => nil_address_representation(),
            Some(addr) => {
                let mut bytes = [0u8; 32];
                bytes[12..32].copy_from_slice(addr.as_slice());
                U256::from_be_bytes(bytes)
            }
        };
        backend
            .sstore(ARBOS_STATE_ADDRESS, self.slot, value_u256)
            .map_err(Into::into)
    }
}
