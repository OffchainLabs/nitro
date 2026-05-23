//! Per-block and per-tx context threaded into Arbitrum precompile handlers.
//!
//! Replaces process-wide thread-locals that previously held per-tx scratch.
//! The executor constructs an [`ArbPrecompileCtx`] once per block, the
//! precompile registration captures it by clone, and writer sites update
//! [`TxCtx`] between transactions.

use alloy_primitives::{Address, B256, U256};
use parking_lot::Mutex;
use std::sync::Arc;

/// Per-block parameters populated once at block start.
#[derive(Debug, Default)]
pub struct BlockCtx {}

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
///
/// `TxCtx` is wrapped in a [`parking_lot::Mutex`] because precompile
/// handler closures must be `Send + Sync + 'static` while still reading
/// values the executor updates between transactions. Locks are uncontended
/// in the current single-threaded-per-block model.
#[derive(Debug, Default, Clone)]
pub struct ArbPrecompileCtx {
    pub block: Arc<BlockCtx>,
    pub tx: Arc<Mutex<TxCtx>>,
    pub debug: Arc<DebugFlags>,
}

impl ArbPrecompileCtx {
    pub fn new() -> Self {
        Self::default()
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
