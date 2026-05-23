use std::marker::PhantomData;

use revm::Database;

use arb_storage::{Storage, StorageBackedUint64, StorageBackend};

use super::L2PricingError;

const TARGET_OFFSET: u64 = 0;
const ADJUSTMENT_WINDOW_OFFSET: u64 = 1;
const BACKLOG_OFFSET: u64 = 2;

/// A single-dimensional gas constraint with target, adjustment window, and backlog.
pub struct GasConstraint<D> {
    target: StorageBackedUint64,
    adjustment_window: StorageBackedUint64,
    backlog: StorageBackedUint64,
    _phantom: PhantomData<D>,
}

pub fn open_gas_constraint<D: Database>(sto: Storage<D>) -> GasConstraint<D> {
    let base_key = sto.base_key();
    GasConstraint {
        target: StorageBackedUint64::new(base_key, TARGET_OFFSET),
        adjustment_window: StorageBackedUint64::new(base_key, ADJUSTMENT_WINDOW_OFFSET),
        backlog: StorageBackedUint64::new(base_key, BACKLOG_OFFSET),
        _phantom: PhantomData,
    }
}

impl<D: Database> GasConstraint<D> {
    pub fn target<B: StorageBackend>(&self, backend: &mut B) -> Result<u64, L2PricingError> {
        Ok(self.target.get(backend)?)
    }

    pub fn set_target<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: u64,
    ) -> Result<(), L2PricingError> {
        Ok(self.target.set(backend, val)?)
    }

    pub fn adjustment_window<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, L2PricingError> {
        Ok(self.adjustment_window.get(backend)?)
    }

    pub fn set_adjustment_window<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: u64,
    ) -> Result<(), L2PricingError> {
        Ok(self.adjustment_window.set(backend, val)?)
    }

    pub fn backlog<B: StorageBackend>(&self, backend: &mut B) -> Result<u64, L2PricingError> {
        Ok(self.backlog.get(backend)?)
    }

    pub fn set_backlog<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: u64,
    ) -> Result<(), L2PricingError> {
        Ok(self.backlog.set(backend, val)?)
    }

    pub fn clear<B: StorageBackend>(&self, backend: &mut B) -> Result<(), L2PricingError> {
        self.target.set(backend, 0)?;
        self.adjustment_window.set(backend, 0)?;
        Ok(self.backlog.set(backend, 0)?)
    }
}
