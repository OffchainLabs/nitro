use alloy_primitives::U256;
use revm::Database;

use arb_storage::Storage;

/// L2 pricing state manages gas pricing for L2 execution.
pub struct L2PricingState<D> {
    pub backing_storage: Storage<D>,
}

impl<D: Database> L2PricingState<D> {
    pub fn initialize(_sto: &Storage<D>, _initial_l2_base_fee: U256) {
        // TODO: implement full initialization
    }

    pub fn open(sto: Storage<D>) -> Self {
        Self {
            backing_storage: sto,
        }
    }
}
