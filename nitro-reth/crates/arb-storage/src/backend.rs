use alloy_primitives::{Address, U256};
use arb_storage_errors::{DatabaseError, StorageError};
use revm::Database;

use crate::{
    state_ops::{read_storage_at, write_storage_at},
    storage::Storage,
};

/// Abstraction over the two backing stores `arb-storage` accessor types are
/// driven from: the block executor's `&mut State<D>` and the precompile
/// handler's `&mut EvmInternals<'_>`.
///
/// The trait sits beneath the typed accessor layer (`StorageBackedX`) so the
/// same descriptors serve both call paths without having to fork the
/// accessor API.
pub trait StorageBackend: SystemStateBackend {
    /// Reads the value at `(account, slot)`. Reads through `StorageBackend`
    /// follow the host's normal storage path (journaled when invoked on
    /// `EvmInternals`); for non-journaled reads see [`SystemStateBackend`].
    fn sload(
        &mut self,
        account: Address,
        slot: U256,
    ) -> Result<U256, <Self as SystemStateBackend>::Error>;

    /// Writes `value` to `(account, slot)`.
    fn sstore(
        &mut self,
        account: Address,
        slot: U256,
        value: U256,
    ) -> Result<(), <Self as SystemStateBackend>::Error>;
}

/// Non-journaled read access to system state.
///
/// Reads bypass the EVM journal: no access-list entry, no cold/warm gas
/// tracking, no account-touch propagation. Use for ArbOS state and other
/// system reads with no consensus relationship to user-visible EVM storage.
///
/// Writes are NOT in this trait. System-state mutations that happen inside
/// a user-callable precompile must remain journaled so they revert with
/// the outer tx on failure (matching geth's StateDB semantics).
/// Writes stay on [`StorageBackend`].
pub trait SystemStateBackend {
    /// Concrete failure type produced by the backend. Convertible into
    /// [`StorageError`] so callers can stay uniform.
    type Error: Into<StorageError>;

    /// Reads the value at `(account, slot)` without journaling.
    fn sload_system(&mut self, account: Address, slot: U256) -> Result<U256, Self::Error>;
}

impl<D: Database> StorageBackend for revm::database::State<D> {
    fn sload(&mut self, account: Address, slot: U256) -> Result<U256, StorageError> {
        read_storage_at(self, account, slot)
    }

    fn sstore(&mut self, account: Address, slot: U256, value: U256) -> Result<(), StorageError> {
        write_storage_at(self, account, slot, value)
    }
}

impl<D: Database> SystemStateBackend for revm::database::State<D> {
    type Error = StorageError;

    fn sload_system(&mut self, account: Address, slot: U256) -> Result<U256, Self::Error> {
        read_storage_at(self, account, slot)
    }
}

impl StorageBackend for alloy_evm::EvmInternals<'_> {
    fn sload(&mut self, account: Address, slot: U256) -> Result<U256, StorageError> {
        alloy_evm::EvmInternals::sload(self, account, slot)
            .map(|state_load| state_load.data)
            .map_err(|e| StorageError::Database(DatabaseError::custom(e)))
    }

    fn sstore(&mut self, account: Address, slot: U256, value: U256) -> Result<(), StorageError> {
        alloy_evm::EvmInternals::sstore(self, account, slot, value)
            .map(|_| ())
            .map_err(|e| StorageError::Database(DatabaseError::custom(e)))
    }
}

impl SystemStateBackend for alloy_evm::EvmInternals<'_> {
    type Error = StorageError;

    fn sload_system(&mut self, account: Address, slot: U256) -> Result<U256, Self::Error> {
        // Reads route through the journal so that in-flight writes within the
        // current tx are observed, matching geth-StateDB semantics. The
        // journal's access-list bookkeeping is unavoidable on this path; the
        // perf win comes from the per-block ArbosState cache reusing the
        // descriptor across calls instead of reconstructing it.
        alloy_evm::EvmInternals::sload(self, account, slot)
            .map(|state_load| state_load.data)
            .map_err(|e| StorageError::Database(DatabaseError::custom(e)))
    }
}

impl<D: Database> StorageBackend for Storage<'_, D> {
    fn sload(&mut self, account: Address, slot: U256) -> Result<U256, StorageError> {
        // SAFETY: see `Storage` struct-level invariant.
        let state = unsafe { self.state_mut() };
        read_storage_at(state, account, slot)
    }

    fn sstore(&mut self, account: Address, slot: U256, value: U256) -> Result<(), StorageError> {
        // SAFETY: see `Storage` struct-level invariant.
        let state = unsafe { self.state_mut() };
        write_storage_at(state, account, slot, value)
    }
}

impl<D: Database> SystemStateBackend for Storage<'_, D> {
    type Error = StorageError;

    fn sload_system(&mut self, account: Address, slot: U256) -> Result<U256, Self::Error> {
        // SAFETY: see `Storage` struct-level invariant.
        let state = unsafe { self.state_mut() };
        read_storage_at(state, account, slot)
    }
}
