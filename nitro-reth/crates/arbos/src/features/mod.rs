use alloy_primitives::U256;
use revm::Database;

use arb_storage::StorageBackedBigUint;

const INCREASED_CALLDATA: usize = 0;

/// Feature flags backed by a storage BigUint used as a bitmask.
pub struct Features<D> {
    features: StorageBackedBigUint<D>,
}

pub fn open_features<D: Database>(
    state: *mut revm::database::State<D>,
    base_key: alloy_primitives::B256,
    offset: u64,
) -> Features<D> {
    Features {
        features: StorageBackedBigUint::new(state, base_key, offset),
    }
}

impl<D: Database> Features<D> {
    pub fn set_calldata_price_increase(&self, enabled: bool) -> Result<(), ()> {
        self.set_bit(INCREASED_CALLDATA, enabled)
    }

    pub fn is_increased_calldata_price_enabled(&self) -> Result<bool, ()> {
        self.is_set(INCREASED_CALLDATA)
    }

    fn set_bit(&self, index: usize, enabled: bool) -> Result<(), ()> {
        let mut val = self.features.get()?;
        if enabled {
            val |= U256::from(1) << index;
        } else {
            val &= !(U256::from(1) << index);
        }
        self.features.set(val)
    }

    fn is_set(&self, index: usize) -> Result<bool, ()> {
        let val = self.features.get()?;
        Ok((val >> index) & U256::from(1) != U256::ZERO)
    }
}
