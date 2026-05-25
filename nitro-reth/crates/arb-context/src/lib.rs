//! Per-block and per-tx context threaded into Arbitrum precompile handlers.

use alloy_primitives::{Address, B256, U256};
use arb_storage::{Detached, SystemStateBackend};
use arbos::{
    arbos_state::{arbos_from_input_system, ArbosState, ArbosStateError},
    burn::SystemBurner,
};
use parking_lot::Mutex;
use std::{
    collections::HashMap,
    sync::{
        atomic::{AtomicU64, AtomicUsize},
        Arc, OnceLock,
    },
};

/// LRU cache of recently invoked Stylus program codehashes.
#[derive(Debug, Default)]
pub struct RecentWasms {
    entries: Vec<B256>,
    capacity: usize,
}

impl RecentWasms {
    pub fn new(capacity: usize) -> Self {
        Self {
            entries: Vec::new(),
            capacity,
        }
    }

    pub fn reset(&mut self, capacity: usize) {
        self.entries.clear();
        self.capacity = capacity;
    }

    /// Insert a hash. Returns `true` if it was already present.
    pub fn insert(&mut self, hash: B256) -> bool {
        let was_present = if let Some(pos) = self.entries.iter().position(|h| *h == hash) {
            self.entries.remove(pos);
            true
        } else {
            false
        };
        self.entries.push(hash);
        if self.capacity > 0 && self.entries.len() > self.capacity {
            self.entries.remove(0);
        }
        was_present
    }
}

/// Per-block parameters populated once at block start.
#[derive(Default)]
pub struct BlockCtx {
    pub arbos_version: u64,
    pub block_timestamp: u64,
    /// L1 block number observed by the EVM `NUMBER` opcode and precompiles
    /// that surface the recorded L1 height.
    pub l1_block_number_for_evm: u64,
    pub l2_block_number: u64,
    pub allow_debug_precompiles: bool,
    /// Live counter mutated by the executor between transactions in the same block.
    pub current_gas_backlog: AtomicU64,
    /// L2 block number -> L1 block number recorded at that L2 height.
    pub l1_block_cache: Mutex<HashMap<u64, u64>>,
    /// L2 block number -> L2 block hash, populated for `arbBlockHash` lookups.
    pub l2_blockhash_cache: Mutex<HashMap<u64, B256>>,
    pub recent_wasms: Mutex<RecentWasms>,
    /// Per-block descriptor cache for the detached [`ArbosState`].
    arbos_state: OnceLock<Result<ArbosState<'static, Detached, SystemBurner>, ArbosStateError>>,
}

impl std::fmt::Debug for BlockCtx {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.debug_struct("BlockCtx")
            .field("arbos_version", &self.arbos_version)
            .field("block_timestamp", &self.block_timestamp)
            .field("l1_block_number_for_evm", &self.l1_block_number_for_evm)
            .field("l2_block_number", &self.l2_block_number)
            .field("allow_debug_precompiles", &self.allow_debug_precompiles)
            .field("current_gas_backlog", &self.current_gas_backlog)
            .field("l1_block_cache", &self.l1_block_cache)
            .field("l2_blockhash_cache", &self.l2_blockhash_cache)
            .field("recent_wasms", &self.recent_wasms)
            .field("arbos_state_cached", &self.arbos_state.get().is_some())
            .finish()
    }
}

impl BlockCtx {
    pub fn new(
        arbos_version: u64,
        block_timestamp: u64,
        l1_block_number_for_evm: u64,
        l2_block_number: u64,
        allow_debug_precompiles: bool,
    ) -> Self {
        Self {
            arbos_version,
            block_timestamp,
            l1_block_number_for_evm,
            l2_block_number,
            allow_debug_precompiles,
            current_gas_backlog: AtomicU64::new(0),
            l1_block_cache: Mutex::new(HashMap::new()),
            l2_blockhash_cache: Mutex::new(HashMap::new()),
            recent_wasms: Mutex::new(RecentWasms::default()),
            arbos_state: OnceLock::new(),
        }
    }

    /// Borrow the cached [`ArbosState`], constructing it on first call via
    /// [`arbos_from_input_system`]. Returns a clone of the cached error if
    /// the first call failed; the cache is not cleared on error since the
    /// version slot does not change within a single block.
    pub fn arbos_state<S: SystemStateBackend>(
        &self,
        backend: &mut S,
    ) -> Result<&ArbosState<'static, Detached, SystemBurner>, ArbosStateError> {
        self.arbos_state
            .get_or_init(|| arbos_from_input_system(backend, SystemBurner::new(None, false)))
            .as_ref()
            .map_err(Clone::clone)
    }

    pub fn current_gas_backlog(&self) -> u64 {
        self.current_gas_backlog
            .load(std::sync::atomic::Ordering::Relaxed)
    }

    pub fn set_current_gas_backlog(&self, value: u64) {
        self.current_gas_backlog
            .store(value, std::sync::atomic::Ordering::Relaxed);
    }

    /// Insert an L1 block number into the cache, retaining a rolling window
    /// of recent L2 heights.
    pub fn cache_l1_block_number(&self, l2_block: u64, l1_block: u64) {
        let mut map = self.l1_block_cache.lock();
        map.insert(l2_block, l1_block);
        if l2_block > 100 {
            map.retain(|&k, _| k >= l2_block - 100);
        }
    }

    pub fn cached_l1_block_number(&self, l2_block: u64) -> Option<u64> {
        self.l1_block_cache.lock().get(&l2_block).copied()
    }

    pub fn cache_l2_block_hash(&self, l2_block: u64, hash: B256) {
        self.l2_blockhash_cache.lock().insert(l2_block, hash);
    }

    pub fn cached_l2_block_hash(&self, l2_block: u64) -> Option<B256> {
        self.l2_blockhash_cache.lock().get(&l2_block).copied()
    }

    pub fn reset_recent_wasms(&self, capacity: usize) {
        self.recent_wasms.lock().reset(capacity);
    }

    pub fn insert_recent_wasm(&self, hash: B256) -> bool {
        self.recent_wasms.lock().insert(hash)
    }
}

/// Per-tx scratch written by the executor between transactions.
#[derive(Debug, Default, Clone)]
pub struct TxCtx {
    pub sender: Address,
    pub effective_gas_price: u128,
    pub poster_fee: u128,
    pub poster_balance_correction: u128,
    pub retryable_id: B256,
    pub redeemer: Address,
    pub tx_is_aliased: bool,
    pub stylus_activation_addr: Option<Address>,
    pub stylus_keepalive_hash: Option<B256>,
    pub stylus_activation_data_fee: U256,
    pub stylus_call_value: U256,
    pub stylus_program_counts: HashMap<Address, u32>,
    pub stylus_pages_open: u16,
    pub stylus_pages_ever: u16,
}

impl TxCtx {
    /// Redeemer address packed into the low 20 bytes of a 32-byte word.
    pub fn redeemer_word(&self) -> U256 {
        U256::from_be_bytes(B256::left_padding_from(self.redeemer.as_slice()).0)
    }
}

/// Handle threaded into precompile handlers. Cheap to clone (all `Arc`).
#[derive(Debug, Default, Clone)]
pub struct ArbPrecompileCtx {
    pub block: Arc<BlockCtx>,
    pub tx: Arc<Mutex<TxCtx>>,
    /// EVM call depth at the most recent precompile dispatch. Mirrors the
    /// journal depth surfaced by revm to the precompile provider.
    pub evm_depth: Arc<AtomicUsize>,
    /// Caller addresses by depth, pushed at each frame boundary so that
    /// precompile handlers can resolve the on-chain caller at arbitrary
    /// depth (alloy-evm's `EvmInternals` does not surface this).
    pub caller_stack: Arc<Mutex<Vec<Address>>>,
}

impl ArbPrecompileCtx {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn with_block(block: Arc<BlockCtx>) -> Self {
        Self {
            block,
            tx: Arc::new(Mutex::new(TxCtx::default())),
            evm_depth: Arc::new(AtomicUsize::new(0)),
            caller_stack: Arc::new(Mutex::new(Vec::new())),
        }
    }

    /// Snapshot the current per-tx scratch.
    pub fn tx_snapshot(&self) -> TxCtx {
        self.tx.lock().clone()
    }

    /// Reset per-tx scratch between transactions.
    pub fn reset_tx(&self) {
        *self.tx.lock() = TxCtx::default();
    }

    pub fn set_sender(&self, sender: Address) {
        self.tx.lock().sender = sender;
    }

    pub fn set_effective_gas_price(&self, price: u128) {
        self.tx.lock().effective_gas_price = price;
    }

    pub fn set_poster_fee(&self, fee: u128) {
        self.tx.lock().poster_fee = fee;
    }

    pub fn set_poster_balance_correction(&self, correction: u128) {
        self.tx.lock().poster_balance_correction = correction;
    }

    pub fn set_retryable_id(&self, id: B256) {
        self.tx.lock().retryable_id = id;
    }

    pub fn set_redeemer(&self, redeemer: Address) {
        self.tx.lock().redeemer = redeemer;
    }

    pub fn set_tx_is_aliased(&self, aliased: bool) {
        self.tx.lock().tx_is_aliased = aliased;
    }

    pub fn tx_is_aliased(&self) -> bool {
        self.tx.lock().tx_is_aliased
    }

    pub fn set_evm_depth(&self, depth: usize) {
        self.evm_depth
            .store(depth, std::sync::atomic::Ordering::Relaxed);
    }

    pub fn evm_depth(&self) -> usize {
        self.evm_depth.load(std::sync::atomic::Ordering::Relaxed)
    }

    pub fn push_caller(&self, caller: Address) {
        self.caller_stack.lock().push(caller);
    }

    pub fn pop_caller(&self) {
        self.caller_stack.lock().pop();
    }

    pub fn reset_caller_stack(&self) {
        self.caller_stack.lock().clear();
    }

    /// Return the caller at the given depth (1-indexed). Mirrors the
    /// frame-depth conventions used by ArbSys.
    pub fn caller_at_depth(&self, depth: usize) -> Option<Address> {
        if depth == 0 {
            return None;
        }
        self.caller_stack.lock().get(depth - 1).copied()
    }

    pub fn set_stylus_activation_addr(&self, addr: Option<Address>) {
        self.tx.lock().stylus_activation_addr = addr;
    }

    pub fn take_stylus_activation_addr(&self) -> Option<Address> {
        self.tx.lock().stylus_activation_addr.take()
    }

    pub fn set_stylus_keepalive_hash(&self, hash: Option<B256>) {
        self.tx.lock().stylus_keepalive_hash = hash;
    }

    pub fn take_stylus_keepalive_hash(&self) -> Option<B256> {
        self.tx.lock().stylus_keepalive_hash.take()
    }

    pub fn set_stylus_activation_data_fee(&self, fee: U256) {
        self.tx.lock().stylus_activation_data_fee = fee;
    }

    pub fn take_stylus_activation_data_fee(&self) -> U256 {
        std::mem::replace(&mut self.tx.lock().stylus_activation_data_fee, U256::ZERO)
    }

    pub fn set_stylus_call_value(&self, value: U256) {
        self.tx.lock().stylus_call_value = value;
    }

    pub fn stylus_call_value(&self) -> U256 {
        self.tx.lock().stylus_call_value
    }

    /// Increment the reentrancy counter for `addr` and return `true` if this
    /// is a reentrant entry (counter was already > 0).
    pub fn push_stylus_program(&self, addr: Address) -> bool {
        let mut tx = self.tx.lock();
        let count = tx.stylus_program_counts.entry(addr).or_insert(0);
        *count += 1;
        *count > 1
    }

    pub fn stylus_program_count(&self, addr: Address) -> u32 {
        self.tx
            .lock()
            .stylus_program_counts
            .get(&addr)
            .copied()
            .unwrap_or(0)
    }

    pub fn pop_stylus_program(&self, addr: Address) {
        let mut tx = self.tx.lock();
        if let Some(count) = tx.stylus_program_counts.get_mut(&addr) {
            *count = count.saturating_sub(1);
            if *count == 0 {
                tx.stylus_program_counts.remove(&addr);
            }
        }
    }
}
