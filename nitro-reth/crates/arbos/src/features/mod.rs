use alloy_primitives::U256;
use std::marker::PhantomData;

use arb_storage::{StorageBackedBigUint, StorageBackend};

mod error;
pub use error::FeaturesError;

const INCREASED_CALLDATA: usize = 0;

/// Feature flags backed by a storage BigUint used as a bitmask.
pub struct Features<'a, D> {
    features: StorageBackedBigUint,
    _phantom: PhantomData<&'a mut revm::database::State<D>>,
}

pub fn open_features<'a, D>(base_key: alloy_primitives::B256, offset: u64) -> Features<'a, D> {
    Features {
        features: StorageBackedBigUint::new(base_key, offset),
        _phantom: PhantomData,
    }
}

impl<D> Features<'_, D> {
    pub fn set_calldata_price_increase<B: StorageBackend>(
        &self,
        backend: &mut B,
        enabled: bool,
    ) -> Result<(), FeaturesError> {
        self.set_bit(backend, INCREASED_CALLDATA, enabled)
    }

    pub fn is_increased_calldata_price_enabled<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<bool, FeaturesError> {
        self.is_set(backend, INCREASED_CALLDATA)
    }

    fn set_bit<B: StorageBackend>(
        &self,
        backend: &mut B,
        index: usize,
        enabled: bool,
    ) -> Result<(), FeaturesError> {
        let mut val = self.features.get(backend)?;
        if enabled {
            val |= U256::from(1) << index;
        } else {
            val &= !(U256::from(1) << index);
        }
        Ok(self.features.set(backend, val)?)
    }

    fn is_set<B: StorageBackend>(
        &self,
        backend: &mut B,
        index: usize,
    ) -> Result<bool, FeaturesError> {
        let val = self.features.get(backend)?;
        Ok((val >> index) & U256::from(1) != U256::ZERO)
    }
}
