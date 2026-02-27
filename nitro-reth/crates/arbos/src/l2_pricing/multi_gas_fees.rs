use alloy_primitives::U256;
use revm::Database;

use arb_primitives::multigas::{ResourceKind, NUM_RESOURCE_KIND};
use arb_storage::{Storage, StorageBackedBigUint};

// Go uses iota * NumResourceKind: next=0, current=NUM_RESOURCE_KIND.
const NEXT_BLOCK_FEES_OFFSET: u64 = 0;
const CURRENT_BLOCK_FEES_OFFSET: u64 = NUM_RESOURCE_KIND as u64;

/// Per-resource-kind base fee tracking for multi-dimensional gas pricing.
///
/// The `next` field stores fees computed during pricing model updates.
/// The `current` field holds fees for the current block, rotated from
/// `next` at block start via `commit_next_to_current`.
pub struct MultiGasFees<D> {
    storage: Storage<D>,
}

pub fn open_multi_gas_fees<D: Database>(sto: Storage<D>) -> MultiGasFees<D> {
    MultiGasFees { storage: sto }
}

impl<D: Database> MultiGasFees<D> {
    pub fn get_current_block_fee(&self, kind: ResourceKind) -> Result<U256, ()> {
        let sbu = StorageBackedBigUint::new(
            self.storage.state_ptr(),
            self.storage.base_key(),
            CURRENT_BLOCK_FEES_OFFSET + kind as u64,
        );
        sbu.get()
    }

    pub fn get_next_block_fee(&self, kind: ResourceKind) -> Result<U256, ()> {
        let sbu = StorageBackedBigUint::new(
            self.storage.state_ptr(),
            self.storage.base_key(),
            NEXT_BLOCK_FEES_OFFSET + kind as u64,
        );
        sbu.get()
    }

    pub fn set_next_block_fee(&self, kind: ResourceKind, fee: U256) -> Result<(), ()> {
        let sbu = StorageBackedBigUint::new(
            self.storage.state_ptr(),
            self.storage.base_key(),
            NEXT_BLOCK_FEES_OFFSET + kind as u64,
        );
        sbu.set(fee)
    }

    /// Copy next-block fees to current-block fees.
    pub fn commit_next_to_current(&self) -> Result<(), ()> {
        for kind in ResourceKind::ALL {
            let fee = self.get_next_block_fee(kind)?;
            let current = StorageBackedBigUint::new(
                self.storage.state_ptr(),
                self.storage.base_key(),
                CURRENT_BLOCK_FEES_OFFSET + kind as u64,
            );
            current.set(fee)?;
        }
        Ok(())
    }
}
