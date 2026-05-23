use alloy_primitives::{Address, U256};
use arb_storage_errors::{DatabaseError, StorageError};
use revm::Database;

use crate::state_ops::{read_storage_at, write_storage_at};

/// Abstraction over the two backing stores `arb-storage` accessor types are
/// driven from: the block executor's `&mut State<D>` and the precompile
/// handler's `&mut EvmInternals<'_>`.
///
/// The trait sits beneath the typed accessor layer (`StorageBackedX`) so the
/// same descriptors serve both call paths without having to fork the
/// accessor API.
pub trait StorageBackend {
    /// Concrete failure type produced by the backend. Convertible into
    /// [`StorageError`] so callers can stay uniform.
    type Error: Into<StorageError>;

    /// Reads the value at `(account, slot)`.
    fn sload(&mut self, account: Address, slot: U256) -> Result<U256, Self::Error>;

    /// Writes `value` to `(account, slot)`.
    fn sstore(&mut self, account: Address, slot: U256, value: U256) -> Result<(), Self::Error>;
}

impl<D: Database> StorageBackend for revm::database::State<D> {
    type Error = StorageError;

    fn sload(&mut self, account: Address, slot: U256) -> Result<U256, Self::Error> {
        read_storage_at(self, account, slot)
    }

    fn sstore(&mut self, account: Address, slot: U256, value: U256) -> Result<(), Self::Error> {
        write_storage_at(self, account, slot, value)
    }
}

impl StorageBackend for alloy_evm::EvmInternals<'_> {
    type Error = StorageError;

    fn sload(&mut self, account: Address, slot: U256) -> Result<U256, Self::Error> {
        alloy_evm::EvmInternals::sload(self, account, slot)
            .map(|state_load| state_load.data)
            .map_err(|e| StorageError::Database(DatabaseError::custom(e)))
    }

    fn sstore(&mut self, account: Address, slot: U256, value: U256) -> Result<(), Self::Error> {
        alloy_evm::EvmInternals::sstore(self, account, slot, value)
            .map(|_| ())
            .map_err(|e| StorageError::Database(DatabaseError::custom(e)))
    }
}
