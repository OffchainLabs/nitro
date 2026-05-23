use alloy_primitives::U256;

use arb_primitives::multigas::{ResourceKind, NUM_RESOURCE_KIND};
use arb_storage::{Storage, StorageBackedBigUint, StorageBackend};

use super::L2PricingError;

// Storage layout: next=0, current=NUM_RESOURCE_KIND.
const NEXT_BLOCK_FEES_OFFSET: u64 = 0;
const CURRENT_BLOCK_FEES_OFFSET: u64 = NUM_RESOURCE_KIND as u64;

/// Per-resource-kind base fee tracking for multi-dimensional gas pricing.
///
/// The `next` field stores fees computed during pricing model updates.
/// The `current` field holds fees for the current block, rotated from
/// `next` at block start via `commit_next_to_current`.
pub struct MultiGasFees<'a, D> {
    storage: Storage<'a, D>,
}

pub fn open_multi_gas_fees<D>(sto: Storage<'_, D>) -> MultiGasFees<'_, D> {
    MultiGasFees { storage: sto }
}

impl<D> MultiGasFees<'_, D> {
    pub fn get_current_block_fee<B: StorageBackend>(
        &self,
        backend: &mut B,
        kind: ResourceKind,
    ) -> Result<U256, L2PricingError> {
        let sbu = StorageBackedBigUint::new(
            self.storage.base_key(),
            CURRENT_BLOCK_FEES_OFFSET + kind as u64,
        );
        Ok(sbu.get(backend)?)
    }

    pub fn get_next_block_fee<B: StorageBackend>(
        &self,
        backend: &mut B,
        kind: ResourceKind,
    ) -> Result<U256, L2PricingError> {
        let sbu = StorageBackedBigUint::new(
            self.storage.base_key(),
            NEXT_BLOCK_FEES_OFFSET + kind as u64,
        );
        Ok(sbu.get(backend)?)
    }

    pub fn set_next_block_fee<B: StorageBackend>(
        &self,
        backend: &mut B,
        kind: ResourceKind,
        fee: U256,
    ) -> Result<(), L2PricingError> {
        let sbu = StorageBackedBigUint::new(
            self.storage.base_key(),
            NEXT_BLOCK_FEES_OFFSET + kind as u64,
        );
        Ok(sbu.set(backend, fee)?)
    }

    /// Copy next-block fees to current-block fees.
    pub fn commit_next_to_current<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<(), L2PricingError> {
        for kind in ResourceKind::ALL {
            let fee = self.get_next_block_fee(backend, kind)?;
            let current = StorageBackedBigUint::new(
                self.storage.base_key(),
                CURRENT_BLOCK_FEES_OFFSET + kind as u64,
            );
            current.set(backend, fee)?;
        }
        Ok(())
    }
}
