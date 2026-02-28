use revm::Database;

use arb_primitives::multigas::{MultiGas, ResourceKind, NUM_RESOURCE_KIND};
use arb_storage::{Storage, StorageBackedUint32, StorageBackedUint64};

const TARGET_OFFSET: u64 = 0;
const ADJUSTMENT_WINDOW_OFFSET: u64 = 1;
const BACKLOG_OFFSET: u64 = 2;
const MAX_WEIGHT_OFFSET: u64 = 3;
const WEIGHTED_RESOURCES_BASE_OFFSET: u64 = 4;

/// A multi-dimensional gas constraint with per-resource-kind weights.
pub struct MultiGasConstraint<D> {
    storage: Storage<D>,
    target: StorageBackedUint64<D>,
    adjustment_window: StorageBackedUint32<D>,
    backlog: StorageBackedUint64<D>,
    max_weight: StorageBackedUint64<D>,
}

pub fn open_multi_gas_constraint<D: Database>(sto: Storage<D>) -> MultiGasConstraint<D> {
    let state = sto.state_ptr();
    let base_key = sto.base_key();
    MultiGasConstraint {
        target: StorageBackedUint64::new(state, base_key, TARGET_OFFSET),
        adjustment_window: StorageBackedUint32::new(state, base_key, ADJUSTMENT_WINDOW_OFFSET),
        backlog: StorageBackedUint64::new(state, base_key, BACKLOG_OFFSET),
        max_weight: StorageBackedUint64::new(state, base_key, MAX_WEIGHT_OFFSET),
        storage: sto,
    }
}

impl<D: Database> MultiGasConstraint<D> {
    pub fn target(&self) -> Result<u64, ()> {
        self.target.get()
    }

    pub fn set_target(&self, val: u64) -> Result<(), ()> {
        self.target.set(val)
    }

    pub fn adjustment_window(&self) -> Result<u32, ()> {
        self.adjustment_window.get()
    }

    pub fn set_adjustment_window(&self, val: u32) -> Result<(), ()> {
        self.adjustment_window.set(val)
    }

    pub fn backlog(&self) -> Result<u64, ()> {
        self.backlog.get()
    }

    pub fn set_backlog(&self, val: u64) -> Result<(), ()> {
        self.backlog.set(val)
    }

    pub fn max_weight(&self) -> Result<u64, ()> {
        self.max_weight.get()
    }

    pub fn resource_weight(&self, kind: ResourceKind) -> Result<u64, ()> {
        self.storage
            .get_uint64_by_uint64(WEIGHTED_RESOURCES_BASE_OFFSET + kind as u64)
    }

    pub fn set_resource_weights(&self, weights: &[u64; NUM_RESOURCE_KIND]) -> Result<(), ()> {
        let mut max = 0u64;
        for (i, &w) in weights.iter().enumerate() {
            self.storage
                .set_uint64_by_uint64(WEIGHTED_RESOURCES_BASE_OFFSET + i as u64, w)?;
            if w > max {
                max = w;
            }
        }
        self.max_weight.set(max)
    }

    /// Returns pairs of (ResourceKind, weight) for all resources with non-zero weight.
    pub fn resources_with_weights(&self) -> Result<Vec<(ResourceKind, u64)>, ()> {
        let mut result = Vec::new();
        for kind in ResourceKind::ALL {
            let w = self.resource_weight(kind)?;
            if w > 0 {
                result.push((kind, w));
            }
        }
        Ok(result)
    }

    /// Compute the weighted total of used resources.
    pub fn used_resources(&self, gas: MultiGas) -> Result<u64, ()> {
        let max_w = self.max_weight.get()?;
        if max_w == 0 {
            return Ok(0);
        }
        let mut total = 0u128;
        for kind in ResourceKind::ALL {
            let w = self.resource_weight(kind)?;
            if w > 0 {
                let amount = gas.get(kind) as u128;
                total += amount * w as u128 / max_w as u128;
            }
        }
        Ok(total.min(u64::MAX as u128) as u64)
    }

    /// Grow the backlog by the weighted resource usage.
    pub fn grow_backlog(&self, gas: MultiGas) -> Result<(), ()> {
        self.update_backlog(super::model::BacklogOperation::Grow, gas)
    }

    /// Shrink the backlog by the weighted resource usage.
    pub fn shrink_backlog(&self, gas: MultiGas) -> Result<(), ()> {
        self.update_backlog(super::model::BacklogOperation::Shrink, gas)
    }

    fn update_backlog(
        &self,
        op: super::model::BacklogOperation,
        gas: MultiGas,
    ) -> Result<(), ()> {
        let mut backlog = self.backlog.get()?;
        for kind in ResourceKind::ALL {
            let weight = self.resource_weight(kind)?;
            if weight == 0 {
                continue;
            }
            let amount = gas.get(kind);
            let weighted = amount.saturating_mul(weight);
            backlog = match op {
                super::model::BacklogOperation::Grow => backlog.saturating_add(weighted),
                super::model::BacklogOperation::Shrink => backlog.saturating_sub(weighted),
            };
        }
        self.backlog.set(backlog)
    }

    pub fn clear(&self) -> Result<(), ()> {
        self.target.set(0)?;
        self.adjustment_window.set(0)?;
        self.backlog.set(0)?;
        self.max_weight.set(0)?;
        for i in 0..NUM_RESOURCE_KIND {
            self.storage
                .set_uint64_by_uint64(WEIGHTED_RESOURCES_BASE_OFFSET + i as u64, 0)?;
        }
        Ok(())
    }
}
