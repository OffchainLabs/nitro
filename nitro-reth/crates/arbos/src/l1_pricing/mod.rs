use alloy_primitives::{Address, U256};
use revm::Database;

use arb_storage::Storage;

/// L1 pricing state manages the cost model for L1 data posting.
pub struct L1PricingState<D> {
    pub backing_storage: Storage<D>,
    pub arbos_version: u64,
}

impl<D: Database> L1PricingState<D> {
    pub fn initialize(
        _sto: &Storage<D>,
        _rewards_recipient: Address,
        _initial_l1_base_fee: U256,
    ) {
        // TODO: implement full initialization
    }

    pub fn open(sto: Storage<D>, arbos_version: u64) -> Self {
        Self {
            backing_storage: sto,
            arbos_version,
        }
    }
}
