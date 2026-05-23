use alloy_primitives::{B256, U256};

use arb_primitives::multigas::{MultiGas, ResourceKind, NUM_RESOURCE_KIND};
use arb_storage::{
    storage_key_map, StorageBackedUint32, StorageBackedUint64, StorageBackend, ARBOS_STATE_ADDRESS,
};

use super::L2PricingError;

const TARGET_OFFSET: u64 = 0;
const ADJUSTMENT_WINDOW_OFFSET: u64 = 1;
const BACKLOG_OFFSET: u64 = 2;
const MAX_WEIGHT_OFFSET: u64 = 3;
const WEIGHTED_RESOURCES_BASE_OFFSET: u64 = 4;

/// A multi-dimensional gas constraint with per-resource-kind weights.
#[derive(Clone, Copy, Debug)]
pub struct MultiGasConstraint {
    base_key: B256,
    target: StorageBackedUint64,
    adjustment_window: StorageBackedUint32,
    backlog: StorageBackedUint64,
    max_weight: StorageBackedUint64,
}

pub fn open_multi_gas_constraint(base_key: B256) -> MultiGasConstraint {
    MultiGasConstraint {
        base_key,
        target: StorageBackedUint64::new(base_key, TARGET_OFFSET),
        adjustment_window: StorageBackedUint32::new(base_key, ADJUSTMENT_WINDOW_OFFSET),
        backlog: StorageBackedUint64::new(base_key, BACKLOG_OFFSET),
        max_weight: StorageBackedUint64::new(base_key, MAX_WEIGHT_OFFSET),
    }
}

fn weight_slot(base_key: B256, kind_index: u64) -> U256 {
    let key: &[u8] = if base_key == B256::ZERO {
        &[]
    } else {
        base_key.as_slice()
    };
    storage_key_map(key, WEIGHTED_RESOURCES_BASE_OFFSET + kind_index)
}

impl MultiGasConstraint {
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
    ) -> Result<u32, L2PricingError> {
        Ok(self.adjustment_window.get(backend)?)
    }

    pub fn set_adjustment_window<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: u32,
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

    pub fn max_weight<B: StorageBackend>(&self, backend: &mut B) -> Result<u64, L2PricingError> {
        Ok(self.max_weight.get(backend)?)
    }

    pub fn resource_weight<B: StorageBackend>(
        &self,
        backend: &mut B,
        kind: ResourceKind,
    ) -> Result<u64, L2PricingError> {
        let slot = weight_slot(self.base_key, kind as u64);
        let value = backend
            .sload(ARBOS_STATE_ADDRESS, slot)
            .map_err(Into::into)?;
        Ok(value.try_into().unwrap_or(0))
    }

    pub fn set_resource_weights<B: StorageBackend>(
        &self,
        backend: &mut B,
        weights: &[u64; NUM_RESOURCE_KIND],
    ) -> Result<(), L2PricingError> {
        let mut max = 0u64;
        for (i, &w) in weights.iter().enumerate() {
            let slot = weight_slot(self.base_key, i as u64);
            backend
                .sstore(ARBOS_STATE_ADDRESS, slot, U256::from(w))
                .map_err(Into::into)?;
            if w > max {
                max = w;
            }
        }
        Ok(self.max_weight.set(backend, max)?)
    }

    /// Returns pairs of (ResourceKind, weight) for all resources with non-zero weight.
    pub fn resources_with_weights<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<Vec<(ResourceKind, u64)>, L2PricingError> {
        let mut result = Vec::new();
        for kind in ResourceKind::ALL {
            let w = self.resource_weight(backend, kind)?;
            if w > 0 {
                result.push((kind, w));
            }
        }
        Ok(result)
    }

    /// Compute the weighted total of used resources.
    pub fn used_resources<B: StorageBackend>(
        &self,
        backend: &mut B,
        gas: MultiGas,
    ) -> Result<u64, L2PricingError> {
        let max_w = self.max_weight.get(backend)?;
        if max_w == 0 {
            return Ok(0);
        }
        let mut total = 0u128;
        for kind in ResourceKind::ALL {
            let w = self.resource_weight(backend, kind)?;
            if w > 0 {
                let amount = gas.get(kind) as u128;
                total += amount * w as u128 / max_w as u128;
            }
        }
        Ok(total.min(u64::MAX as u128) as u64)
    }

    /// Grow the backlog by the weighted resource usage.
    pub fn grow_backlog<B: StorageBackend>(
        &self,
        backend: &mut B,
        gas: MultiGas,
    ) -> Result<(), L2PricingError> {
        self.update_backlog(backend, super::model::BacklogOperation::Grow, gas)
    }

    /// Shrink the backlog by the weighted resource usage.
    pub fn shrink_backlog<B: StorageBackend>(
        &self,
        backend: &mut B,
        gas: MultiGas,
    ) -> Result<(), L2PricingError> {
        self.update_backlog(backend, super::model::BacklogOperation::Shrink, gas)
    }

    fn update_backlog<B: StorageBackend>(
        &self,
        backend: &mut B,
        op: super::model::BacklogOperation,
        gas: MultiGas,
    ) -> Result<(), L2PricingError> {
        let mut backlog = self.backlog.get(backend)?;
        for kind in ResourceKind::ALL {
            let weight = self.resource_weight(backend, kind)?;
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
        Ok(self.backlog.set(backend, backlog)?)
    }

    pub fn clear<B: StorageBackend>(&self, backend: &mut B) -> Result<(), L2PricingError> {
        self.target.set(backend, 0)?;
        self.adjustment_window.set(backend, 0)?;
        self.backlog.set(backend, 0)?;
        self.max_weight.set(backend, 0)?;
        for i in 0..NUM_RESOURCE_KIND {
            let slot = weight_slot(self.base_key, i as u64);
            backend
                .sstore(ARBOS_STATE_ADDRESS, slot, U256::ZERO)
                .map_err(Into::into)?;
        }
        Ok(())
    }
}
