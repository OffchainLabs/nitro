use alloy_primitives::{Address, B256, U256};
use arb_storage_errors::StorageError;
use revm::Database;
use std::{marker::PhantomData, ptr::NonNull};

use crate::{
    slot::{derive_sub_key, storage_key_map, storage_key_map_b256},
    state_ops::{read_storage_at, write_storage_at, ARBOS_STATE_ADDRESS},
};

/// Phantom backend used by read paths that drive ArbOS accessors through a
/// [`StorageBackend`] (such as the precompile handlers reading via
/// `EvmInternals`). Direct state-pointer I/O on a `Storage<Detached>` is
/// unreachable by construction — the type does not implement [`Database`].
pub enum Detached {}

/// Hierarchical storage abstraction over EVM account state.
///
/// Subspaces are derived from `base_key` using keccak-based mixing; direct
/// state I/O (when available) targets `account`. The struct is parameterised
/// over the executor's `Database` so the same shape can be inhabited by an
/// executor `&'a mut State<D>` or by [`Detached`] for read paths that operate
/// purely through a [`StorageBackend`].
///
/// The lifetime `'a` ties the handle to the originating `&mut State<D>` borrow
/// so a `Storage` value cannot outlive the state it references. The state is
/// held as a [`NonNull`] internally (with a [`PhantomData`] for the lifetime)
/// so multiple disjoint subspaces over the same state can coexist; direct I/O
/// goes through one `unsafe` deref controlled by the same SAFETY invariant
/// described below.
///
/// # Safety
///
/// All direct I/O methods on `Storage<'a, D>` materialise `&mut *state` for the
/// duration of a single SLOAD/SSTORE. The executor runs each block on a single
/// thread and the EVM call graph is sequential, so no two such borrows overlap
/// at runtime. The lifetime `'a` ensures the underlying `State<D>` cannot be
/// dropped while `Storage` handles into it are alive.
pub struct Storage<'a, D> {
    state: NonNull<revm::database::State<D>>,
    base_key: B256,
    account: Address,
    _marker: PhantomData<&'a mut revm::database::State<D>>,
}

impl<'a, D> Storage<'a, D> {
    /// Creates a new Storage backed by the ArbOS state account.
    pub fn new(state: &'a mut revm::database::State<D>, base_key: B256) -> Self {
        Self {
            // SAFETY: `state` is a live `&mut` so `NonNull::new_unchecked` is sound.
            state: unsafe { NonNull::new_unchecked(state as *mut _) },
            base_key,
            account: ARBOS_STATE_ADDRESS,
            _marker: PhantomData,
        }
    }

    /// Creates a new Storage backed by a specific account.
    pub fn new_with_account(
        state: &'a mut revm::database::State<D>,
        base_key: B256,
        account: Address,
    ) -> Self {
        Self {
            // SAFETY: `state` is a live `&mut` so `NonNull::new_unchecked` is sound.
            state: unsafe { NonNull::new_unchecked(state as *mut _) },
            base_key,
            account,
            _marker: PhantomData,
        }
    }

    /// Opens a child subspace by hashing the parent key with the child ID.
    pub fn open_sub_storage(&self, sub_key: &[u8]) -> Storage<'a, D> {
        let new_key = derive_sub_key(self.base_key, sub_key);
        Storage {
            state: self.state,
            base_key: new_key,
            account: self.account,
            _marker: PhantomData,
        }
    }

    /// Opens a child subspace using a pre-derived key, avoiding a keccak hash.
    pub fn open_sub_storage_with_key(&self, key: B256) -> Storage<'a, D> {
        Storage {
            state: self.state,
            base_key: key,
            account: self.account,
            _marker: PhantomData,
        }
    }

    /// Opens a sibling handle bound to the same backing state but targeting a
    /// different account and `base_key`. Used when constructing accessors for
    /// well-known non-ArbOS addresses (e.g. the filtered-transactions store).
    pub fn open_account(&self, account: Address, base_key: B256) -> Storage<'a, D> {
        Storage {
            state: self.state,
            base_key,
            account,
            _marker: PhantomData,
        }
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

    /// Computes the EVM slot for a `B256`-keyed entry under this subspace.
    pub fn slot_for_key(&self, key: B256) -> U256 {
        self.compute_slot_for_key(key)
    }

    /// Returns the base key for this storage subspace.
    pub fn base_key(&self) -> B256 {
        self.base_key
    }

    /// Returns the account address this storage subspace is bound to.
    pub fn account(&self) -> Address {
        self.account
    }
}

impl Storage<'static, Detached> {
    /// Builds a `Storage` handle that has no executor state pointer.
    ///
    /// All reads and writes must be routed through a [`StorageBackend`];
    /// direct I/O methods on `Storage` are gated on `D: Database` and are
    /// therefore inaccessible here.
    pub fn detached(account: Address, base_key: B256) -> Self {
        Self {
            // SAFETY: `Detached` storage never has its state dereferenced —
            // there is no `Database` impl that would expose the direct I/O
            // surface. The pointer value here is intentionally dangling and
            // unused.
            state: NonNull::dangling(),
            base_key,
            account,
            _marker: PhantomData,
        }
    }
}

impl<'a, D: Database> Storage<'a, D> {
    /// Reads a 32-byte value by uint64 offset.
    pub fn get_by_uint64(&self, offset: u64) -> Result<B256, StorageError> {
        let slot = self.compute_slot(offset);
        // SAFETY: see struct-level invariant.
        let state = unsafe { &mut *self.state.as_ptr() };
        read_storage_at(state, self.account, slot).map(B256::from)
    }

    /// Writes a 32-byte value by uint64 offset.
    pub fn set_by_uint64(&self, offset: u64, value: B256) -> Result<(), StorageError> {
        let slot = self.compute_slot(offset);
        let value_u256 = U256::from_be_bytes(value.0);
        // SAFETY: see struct-level invariant.
        let state = unsafe { &mut *self.state.as_ptr() };
        write_storage_at(state, self.account, slot, value_u256)
    }

    /// Reads a `u64` by uint64 offset, truncating values that exceed `u64::MAX`.
    pub fn get_uint64_by_uint64(&self, offset: u64) -> Result<u64, StorageError> {
        let slot = self.compute_slot(offset);
        // SAFETY: see struct-level invariant.
        let state = unsafe { &mut *self.state.as_ptr() };
        let value = read_storage_at(state, self.account, slot)?;
        Ok(value.try_into().unwrap_or(0))
    }

    /// Writes a `u64` by uint64 offset.
    pub fn set_uint64_by_uint64(&self, offset: u64, value: u64) -> Result<(), StorageError> {
        let slot = self.compute_slot(offset);
        // SAFETY: see struct-level invariant.
        let state = unsafe { &mut *self.state.as_ptr() };
        write_storage_at(state, self.account, slot, U256::from(value))
    }

    /// Reads a 32-byte value by B256 key using mapAddress algorithm.
    pub fn get(&self, key: B256) -> Result<B256, StorageError> {
        let slot = self.compute_slot_for_key(key);
        // SAFETY: see struct-level invariant.
        let state = unsafe { &mut *self.state.as_ptr() };
        read_storage_at(state, self.account, slot).map(B256::from)
    }

    /// Writes a 32-byte value by B256 key using mapAddress algorithm.
    pub fn set(&self, key: B256, value: B256) -> Result<(), StorageError> {
        let slot = self.compute_slot_for_key(key);
        let value_u256 = U256::from_be_bytes(value.0);
        // SAFETY: see struct-level invariant.
        let state = unsafe { &mut *self.state.as_ptr() };
        write_storage_at(state, self.account, slot, value_u256)
    }
}

impl<'a, D> Storage<'a, D> {
    /// Re-borrows the underlying state for direct `State<D>`-level operations
    /// (e.g. account-level reads such as `get_account_balance` and writes such
    /// as `set_account_nonce`/`set_account_code`).
    ///
    /// The returned reference inherits the lifetime parameter `'a` of the
    /// `Storage`, so it is decoupled from any temporary borrow on `&self`.
    /// This shape lets callers thread the same backing state through methods
    /// that take `&mut self` without artificial borrow conflicts.
    ///
    /// # Safety
    ///
    /// arbreth's executor runs each block on a single thread and the EVM call
    /// graph is sequential, so no two such borrows are live at the same time
    /// at runtime. Callers must not nest two `state_mut()` returns from
    /// overlapping `Storage` handles.
    pub unsafe fn state_mut(&self) -> &'a mut revm::database::State<D> {
        // SAFETY: see method-level invariant. `'a` ensures lifetime safety.
        unsafe { &mut *self.state.as_ptr() }
    }
}

impl<D> Clone for Storage<'_, D> {
    fn clone(&self) -> Self {
        Self {
            state: self.state,
            base_key: self.base_key,
            account: self.account,
            _marker: PhantomData,
        }
    }
}

// SAFETY: `Storage` over a `D: Send` state is safe to send between threads
// when no concurrent access occurs. The arbreth executor runs each block on a
// single thread; reentrant Stylus calls execute synchronously on the same
// thread. The lifetime `'a` prevents the handle from outliving the source
// state borrow.
unsafe impl<D: Send> Send for Storage<'_, D> {}
unsafe impl<D: Sync> Sync for Storage<'_, D> {}
