use alloy_primitives::{Address, B256, U256};
use arb_storage_errors::StorageError;
use revm::Database;

use crate::{
    slot::{derive_sub_key, storage_key_map, storage_key_map_b256},
    state_ops::{read_storage_at, write_storage_at, ARBOS_STATE_ADDRESS},
};

/// Hierarchical storage abstraction over EVM account state.
///
/// State lives in a specific account. Storage uses keccak256-based
/// subspace derivation to create a hierarchical key space.
///
/// Safety invariant: the `state` pointer must outlive every `Storage`
/// holding it, and dereferences across handles pointing at the same
/// `State<D>` must be sequential.
pub struct Storage<D> {
    pub(crate) state: *mut revm::database::State<D>,
    pub base_key: B256,
    pub account: Address,
}

impl<D: Database> Storage<D> {
    /// Creates a new Storage backed by the ArbOS state account.
    pub fn new(state: *mut revm::database::State<D>, base_key: B256) -> Self {
        Self {
            state,
            base_key,
            account: ARBOS_STATE_ADDRESS,
        }
    }

    /// Creates a new Storage backed by a specific account.
    pub fn new_with_account(
        state: *mut revm::database::State<D>,
        base_key: B256,
        account: Address,
    ) -> Self {
        Self {
            state,
            base_key,
            account,
        }
    }

    /// Opens a child subspace by hashing the parent key with the child ID.
    pub fn open_sub_storage(&self, sub_key: &[u8]) -> Storage<D> {
        let new_key = derive_sub_key(self.base_key, sub_key);
        Storage::new_with_account(self.state, new_key, self.account)
    }

    /// Opens a child subspace using a pre-derived key, avoiding a keccak hash.
    pub fn open_sub_storage_with_key(&self, key: B256) -> Storage<D> {
        Storage::new_with_account(self.state, key, self.account)
    }

    /// Reads a 32-byte value by uint64 offset.
    pub fn get_by_uint64(&self, offset: u64) -> Result<B256, StorageError> {
        let slot = self.compute_slot(offset);
        // SAFETY: see struct-level invariant.
        let state = unsafe { &mut *self.state };
        read_storage_at(state, self.account, slot).map(B256::from)
    }

    /// Writes a 32-byte value by uint64 offset.
    pub fn set_by_uint64(&self, offset: u64, value: B256) -> Result<(), StorageError> {
        let slot = self.compute_slot(offset);
        let value_u256 = U256::from_be_bytes(value.0);
        // SAFETY: see struct-level invariant.
        let state = unsafe { &mut *self.state };
        write_storage_at(state, self.account, slot, value_u256)
    }

    /// Reads a `u64` by uint64 offset, truncating values that exceed `u64::MAX`.
    pub fn get_uint64_by_uint64(&self, offset: u64) -> Result<u64, StorageError> {
        let slot = self.compute_slot(offset);
        // SAFETY: see struct-level invariant.
        let state = unsafe { &mut *self.state };
        let value = read_storage_at(state, self.account, slot)?;
        Ok(value.try_into().unwrap_or(0))
    }

    /// Writes a `u64` by uint64 offset.
    pub fn set_uint64_by_uint64(&self, offset: u64, value: u64) -> Result<(), StorageError> {
        let slot = self.compute_slot(offset);
        // SAFETY: see struct-level invariant.
        let state = unsafe { &mut *self.state };
        write_storage_at(state, self.account, slot, U256::from(value))
    }

    /// Reads a 32-byte value by B256 key using mapAddress algorithm.
    pub fn get(&self, key: B256) -> Result<B256, StorageError> {
        let slot = self.compute_slot_for_key(key);
        // SAFETY: see struct-level invariant.
        let state = unsafe { &mut *self.state };
        read_storage_at(state, self.account, slot).map(B256::from)
    }

    /// Writes a 32-byte value by B256 key using mapAddress algorithm.
    pub fn set(&self, key: B256, value: B256) -> Result<(), StorageError> {
        let slot = self.compute_slot_for_key(key);
        let value_u256 = U256::from_be_bytes(value.0);
        // SAFETY: see struct-level invariant.
        let state = unsafe { &mut *self.state };
        write_storage_at(state, self.account, slot, value_u256)
    }

    fn storage_key(&self) -> &[u8] {
        if self.base_key == B256::ZERO {
            &[]
        } else {
            self.base_key.as_slice()
        }
    }

    fn compute_slot(&self, offset: u64) -> U256 {
        storage_key_map(self.storage_key(), offset)
    }

    fn compute_slot_for_key(&self, key: B256) -> U256 {
        storage_key_map_b256(self.storage_key(), &key.0)
    }

    /// Creates a StorageSlot handle for a specific offset.
    pub fn new_slot(&self, offset: u64) -> U256 {
        self.compute_slot(offset)
    }

    /// Returns the raw `*mut State<D>`. See the struct-level safety invariant.
    pub fn state_ptr(&self) -> *mut revm::database::State<D> {
        self.state
    }

    /// Returns the base key for this storage subspace.
    pub fn base_key(&self) -> B256 {
        self.base_key
    }
}

impl<D> Clone for Storage<D> {
    fn clone(&self) -> Self {
        Self {
            state: self.state,
            base_key: self.base_key,
            account: self.account,
        }
    }
}

// SAFETY: see struct-level invariant.
unsafe impl<D: Send> Send for Storage<D> {}
unsafe impl<D: Sync> Sync for Storage<D> {}
