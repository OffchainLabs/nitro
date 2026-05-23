//! Per-block and per-tx context threaded into Arbitrum precompile handlers.

use alloy_primitives::{Address, B256, U256};
use parking_lot::Mutex;
use std::{
    collections::HashMap,
    sync::{
        atomic::{AtomicU64, AtomicUsize},
        Arc,
    },
};

/// Per-block parameters populated once at block start.
#[derive(Debug, Default)]
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
        }
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
}

/// Per-tx scratch written by the executor between transactions.
#[derive(Debug, Default, Clone, Copy)]
pub struct TxCtx {
    pub sender: Address,
    pub effective_gas_price: u128,
    pub poster_fee: u128,
    pub poster_balance_correction: u128,
    pub retryable_id: B256,
    pub redeemer: Address,
}

impl TxCtx {
    /// Redeemer address packed into the low 20 bytes of a 32-byte word.
    pub fn redeemer_word(&self) -> U256 {
        U256::from_be_bytes(B256::left_padding_from(self.redeemer.as_slice()).0)
    }
}

/// Toggles that gate debug-only precompiles or features.
#[derive(Debug, Default)]
pub struct DebugFlags {}

/// Handle threaded into precompile handlers. Cheap to clone (all `Arc`).
#[derive(Debug, Default, Clone)]
pub struct ArbPrecompileCtx {
    pub block: Arc<BlockCtx>,
    pub tx: Arc<Mutex<TxCtx>>,
    pub debug: Arc<DebugFlags>,
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
            debug: Arc::new(DebugFlags::default()),
            evm_depth: Arc::new(AtomicUsize::new(0)),
            caller_stack: Arc::new(Mutex::new(Vec::new())),
        }
    }

    /// Snapshot the current per-tx scratch.
    pub fn tx_snapshot(&self) -> TxCtx {
        *self.tx.lock()
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
}

thread_local! {
    static ACTIVE_CTX: std::cell::RefCell<Option<Arc<ArbPrecompileCtx>>> =
        const { std::cell::RefCell::new(None) };
}

/// Install the context active on the current thread for the lifetime of a block.
pub fn install_active(ctx: Arc<ArbPrecompileCtx>) {
    ACTIVE_CTX.with(|cell| *cell.borrow_mut() = Some(ctx));
}

/// Clear the per-thread active context once block execution is complete.
pub fn clear_active() {
    ACTIVE_CTX.with(|cell| *cell.borrow_mut() = None);
}

/// Clone the active context for the current thread, if any.
pub fn active() -> Option<Arc<ArbPrecompileCtx>> {
    ACTIVE_CTX.with(|cell| cell.borrow().clone())
}

/// Borrow the active context. Returns `None` when no block is executing on this thread.
pub fn with_active<R>(f: impl FnOnce(&ArbPrecompileCtx) -> R) -> Option<R> {
    ACTIVE_CTX.with(|cell| cell.borrow().as_ref().map(|arc| f(arc.as_ref())))
}
