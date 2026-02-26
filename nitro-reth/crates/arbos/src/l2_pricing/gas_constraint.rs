use revm::Database;

use arb_storage::{Storage, StorageBackedUint64};

const TARGET_OFFSET: u64 = 0;
const ADJUSTMENT_WINDOW_OFFSET: u64 = 1;
const BACKLOG_OFFSET: u64 = 2;

/// A single-dimensional gas constraint with target, adjustment window, and backlog.
pub struct GasConstraint<D> {
    target: StorageBackedUint64<D>,
    adjustment_window: StorageBackedUint64<D>,
    backlog: StorageBackedUint64<D>,
}

pub fn open_gas_constraint<D: Database>(sto: Storage<D>) -> GasConstraint<D> {
    let state = sto.state_ptr();
    let base_key = sto.base_key();
    GasConstraint {
        target: StorageBackedUint64::new(state, base_key, TARGET_OFFSET),
        adjustment_window: StorageBackedUint64::new(state, base_key, ADJUSTMENT_WINDOW_OFFSET),
        backlog: StorageBackedUint64::new(state, base_key, BACKLOG_OFFSET),
    }
}

impl<D: Database> GasConstraint<D> {
    pub fn target(&self) -> Result<u64, ()> {
        self.target.get()
    }

    pub fn set_target(&self, val: u64) -> Result<(), ()> {
        self.target.set(val)
    }

    pub fn adjustment_window(&self) -> Result<u64, ()> {
        self.adjustment_window.get()
    }

    pub fn set_adjustment_window(&self, val: u64) -> Result<(), ()> {
        self.adjustment_window.set(val)
    }

    pub fn backlog(&self) -> Result<u64, ()> {
        self.backlog.get()
    }

    pub fn set_backlog(&self, val: u64) -> Result<(), ()> {
        self.backlog.set(val)
    }

    pub fn clear(&self) -> Result<(), ()> {
        self.target.set(0)?;
        self.adjustment_window.set(0)?;
        self.backlog.set(0)
    }
}
