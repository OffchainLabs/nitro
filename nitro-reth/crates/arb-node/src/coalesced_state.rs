//! State provider wrapper that pre-coalesces the in-memory block storage
//! into a single hashmap keyed by `(address, slot)`, so each SLOAD is one
//! lookup instead of a linear walk over every unflushed in-memory block.

use alloy_primitives::{Address, BlockNumber, Bytes, StorageKey, StorageValue, B256, U256};
use reth_chain_state::BlockState;
use reth_primitives_traits::{Account, Bytecode, NodePrimitives};
use reth_storage_api::{
    errors::provider::ProviderResult, AccountReader, BlockHashReader, BytecodeReader,
    HashedPostStateProvider, StateProofProvider, StateProvider, StateProviderBox,
    StateRootProvider, StorageRootProvider,
};
use reth_trie::{
    updates::TrieUpdates, AccountProof, HashedPostState, HashedStorage, MultiProof,
    MultiProofTargets, StorageMultiProof, TrieInput,
};
use revm_database::BundleState;
use rustc_hash::{FxHashMap, FxHashSet};

/// Coalesced view of the in-memory block storage overlay.
///
/// Every `(address, slot)` explicit write from any unflushed block is
/// collapsed to its newest value. Accounts whose newest state has
/// "storage known" semantics (newly created or destroyed, matching
/// `BundleAccount::storage_slot`) are tracked separately so slot reads
/// that miss the explicit map still return `Some(U256::ZERO)` for them.
pub struct CoalescedOverlay {
    /// Explicit `(address, slot) -> value` from the newest block that
    /// touched each pair, after wipes have been applied.
    slots: FxHashMap<(Address, B256), U256>,
    /// Addresses whose newest state is "storage known" with no subsequent
    /// explicit write. Reads for slots not in `slots` return zero.
    wiped: FxHashSet<Address>,
}

impl CoalescedOverlay {
    /// Builds the coalesced overlay by iterating the in-memory chain from
    /// oldest to newest. A wipe (status == storage_known) for an account
    /// drops all older explicit entries for that address; later explicit
    /// writes reinstate specific slots.
    pub fn from_chain<N: NodePrimitives>(head: &BlockState<N>) -> Self {
        let mut slots: FxHashMap<(Address, B256), U256> = FxHashMap::default();
        let mut wiped: FxHashSet<Address> = FxHashSet::default();

        // `BlockState::chain` yields newest-first. Reverse to apply writes
        // in chronological order so newer values overwrite older ones.
        // A wipe is sticky once set: only an explicit newer write for a
        // specific `(addr, slot)` reinstates that slot; a non-wipe touch
        // on the same account does not "unwipe" the other slots.
        let blocks: Vec<&BlockState<N>> = head.chain().collect();
        for block_state in blocks.into_iter().rev() {
            let bundle = &block_state.block_ref().execution_output.state;
            for (addr, account) in bundle.state.iter() {
                if account.status.is_storage_known() {
                    wiped.insert(*addr);
                    slots.retain(|(a, _), _| a != addr);
                }
                for (slot_u256, slot_entry) in account.storage.iter() {
                    let key = B256::from(*slot_u256);
                    slots.insert((*addr, key), slot_entry.present_value);
                }
            }
        }

        Self { slots, wiped }
    }

    pub fn is_empty(&self) -> bool {
        self.slots.is_empty() && self.wiped.is_empty()
    }

    fn lookup(&self, address: Address, key: B256) -> Option<Option<U256>> {
        if let Some(&val) = self.slots.get(&(address, key)) {
            return Some(Some(val));
        }
        if self.wiped.contains(&address) {
            return Some(Some(U256::ZERO));
        }
        None
    }
}

/// Wraps an existing `StateProvider` (typically a `MemoryOverlayStateProvider`)
/// with a precomputed storage overlay so SLOADs served from the overlay
/// cost a single hashmap probe instead of scanning every in-memory block.
pub struct CoalescedStateProvider {
    inner: StateProviderBox,
    overlay: CoalescedOverlay,
}

impl CoalescedStateProvider {
    pub fn new(inner: StateProviderBox, overlay: CoalescedOverlay) -> Self {
        Self { inner, overlay }
    }

    pub fn boxed(self) -> StateProviderBox {
        Box::new(self)
    }
}

impl BlockHashReader for CoalescedStateProvider {
    fn block_hash(&self, number: BlockNumber) -> ProviderResult<Option<B256>> {
        self.inner.block_hash(number)
    }

    fn canonical_hashes_range(
        &self,
        start: BlockNumber,
        end: BlockNumber,
    ) -> ProviderResult<Vec<B256>> {
        self.inner.canonical_hashes_range(start, end)
    }
}

impl AccountReader for CoalescedStateProvider {
    fn basic_account(&self, address: &Address) -> ProviderResult<Option<Account>> {
        self.inner.basic_account(address)
    }
}

impl BytecodeReader for CoalescedStateProvider {
    fn bytecode_by_hash(&self, code_hash: &B256) -> ProviderResult<Option<Bytecode>> {
        self.inner.bytecode_by_hash(code_hash)
    }
}

impl StateRootProvider for CoalescedStateProvider {
    fn state_root(&self, state: HashedPostState) -> ProviderResult<B256> {
        self.inner.state_root(state)
    }

    fn state_root_from_nodes(&self, input: TrieInput) -> ProviderResult<B256> {
        self.inner.state_root_from_nodes(input)
    }

    fn state_root_with_updates(
        &self,
        state: HashedPostState,
    ) -> ProviderResult<(B256, TrieUpdates)> {
        self.inner.state_root_with_updates(state)
    }

    fn state_root_from_nodes_with_updates(
        &self,
        input: TrieInput,
    ) -> ProviderResult<(B256, TrieUpdates)> {
        self.inner.state_root_from_nodes_with_updates(input)
    }
}

impl StorageRootProvider for CoalescedStateProvider {
    fn storage_root(&self, address: Address, storage: HashedStorage) -> ProviderResult<B256> {
        self.inner.storage_root(address, storage)
    }

    fn storage_proof(
        &self,
        address: Address,
        slot: B256,
        storage: HashedStorage,
    ) -> ProviderResult<reth_trie::StorageProof> {
        self.inner.storage_proof(address, slot, storage)
    }

    fn storage_multiproof(
        &self,
        address: Address,
        slots: &[B256],
        storage: HashedStorage,
    ) -> ProviderResult<StorageMultiProof> {
        self.inner.storage_multiproof(address, slots, storage)
    }
}

impl StateProofProvider for CoalescedStateProvider {
    fn proof(
        &self,
        input: TrieInput,
        address: Address,
        slots: &[B256],
    ) -> ProviderResult<AccountProof> {
        self.inner.proof(input, address, slots)
    }

    fn multiproof(
        &self,
        input: TrieInput,
        targets: MultiProofTargets,
    ) -> ProviderResult<MultiProof> {
        self.inner.multiproof(input, targets)
    }

    fn witness(&self, input: TrieInput, target: HashedPostState) -> ProviderResult<Vec<Bytes>> {
        self.inner.witness(input, target)
    }
}

impl HashedPostStateProvider for CoalescedStateProvider {
    fn hashed_post_state(&self, bundle_state: &BundleState) -> HashedPostState {
        self.inner.hashed_post_state(bundle_state)
    }
}

impl StateProvider for CoalescedStateProvider {
    fn storage(
        &self,
        address: Address,
        storage_key: StorageKey,
    ) -> ProviderResult<Option<StorageValue>> {
        if let Some(val) = self.overlay.lookup(address, storage_key) {
            return Ok(val);
        }
        self.inner.storage(address, storage_key)
    }
}
