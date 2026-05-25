mod error;
mod gas_constraint;
mod model;
mod multi_gas_constraint;
mod multi_gas_fees;

pub use error::L2PricingError;
pub use gas_constraint::{open_gas_constraint, GasConstraint};
pub use model::*;
pub use multi_gas_constraint::{open_multi_gas_constraint, MultiGasConstraint};
pub use multi_gas_fees::MultiGasFees;

use alloy_primitives::U256;

use arb_primitives::multigas::NUM_RESOURCE_KIND;
use arb_storage::{
    open_sub_storage_vector, Storage, StorageBackedBigUint, StorageBackedUint64, StorageBackend,
    SubStorageVector, SystemStateBackend,
};

// Storage offsets for L2 pricing state.
pub const SPEED_LIMIT_PER_SECOND_OFFSET: u64 = 0;
pub const PER_BLOCK_GAS_LIMIT_OFFSET: u64 = 1;
pub const BASE_FEE_WEI_OFFSET: u64 = 2;
pub const MIN_BASE_FEE_WEI_OFFSET: u64 = 3;
pub const GAS_BACKLOG_OFFSET: u64 = 4;
pub const PRICING_INERTIA_OFFSET: u64 = 5;
pub const BACKLOG_TOLERANCE_OFFSET: u64 = 6;
pub const PER_TX_GAS_LIMIT_OFFSET: u64 = 7;

// Subspace keys for L2 pricing partitions.
const GAS_CONSTRAINTS_KEY: &[u8] = &[0];
const MULTI_GAS_CONSTRAINTS_KEY: &[u8] = &[1];
const MULTI_GAS_BASE_FEES_KEY: &[u8] = &[2];

// Constants.
pub const GETH_BLOCK_GAS_LIMIT: u64 = 1 << 50;
pub const GAS_CONSTRAINTS_MAX_NUM: u64 = 20;
pub const MAX_PRICING_EXPONENT_BIPS: u64 = 85_000;

// EIP-2200 storage costs.
pub const STORAGE_READ_COST: u64 = 800; // SloadGasEIP2200
pub const STORAGE_WRITE_COST: u64 = 20_000; // SstoreSetGasEIP2200

// Initial values.
pub const INITIAL_SPEED_LIMIT_PER_SECOND_V0: u64 = 1_000_000;
pub const INITIAL_SPEED_LIMIT_PER_SECOND_V6: u64 = 7_000_000;
pub const INITIAL_PER_BLOCK_GAS_LIMIT_V0: u64 = 20_000_000;
pub const INITIAL_PER_BLOCK_GAS_LIMIT_V6: u64 = 32_000_000;
pub const INITIAL_MINIMUM_BASE_FEE_WEI: u64 = 100_000_000; // 0.1 Gwei
pub const INITIAL_BASE_FEE_WEI: u64 = INITIAL_MINIMUM_BASE_FEE_WEI;
pub const INITIAL_PRICING_INERTIA: u64 = 102;
pub const INITIAL_BACKLOG_TOLERANCE: u64 = 10;
pub const INITIAL_PER_TX_GAS_LIMIT_V50: u64 = 32_000_000;

/// L2 pricing state manages gas pricing for L2 execution.
pub struct L2PricingState<'a, D> {
    pub backing_storage: Storage<'a, D>,
    pub arbos_version: u64,
    speed_limit_per_second: StorageBackedUint64,
    per_block_gas_limit: StorageBackedUint64,
    base_fee_wei: StorageBackedBigUint,
    min_base_fee_wei: StorageBackedBigUint,
    gas_backlog: StorageBackedUint64,
    pricing_inertia: StorageBackedUint64,
    backlog_tolerance: StorageBackedUint64,
    per_tx_gas_limit: StorageBackedUint64,
    gas_constraints: SubStorageVector,
    multi_gas_constraints: SubStorageVector,
    multi_gas_base_fees: Storage<'a, D>,
}

pub fn initialize_l2_pricing_state<D, B: StorageBackend>(
    sto: &Storage<'_, D>,
    backend: &mut B,
) -> Result<(), L2PricingError> {
    let base_key = sto.base_key();

    StorageBackedUint64::new(base_key, SPEED_LIMIT_PER_SECOND_OFFSET)
        .set(backend, INITIAL_SPEED_LIMIT_PER_SECOND_V0)?;
    StorageBackedUint64::new(base_key, PER_BLOCK_GAS_LIMIT_OFFSET)
        .set(backend, INITIAL_PER_BLOCK_GAS_LIMIT_V0)?;
    StorageBackedUint64::new(base_key, BASE_FEE_WEI_OFFSET).set(backend, INITIAL_BASE_FEE_WEI)?;
    StorageBackedBigUint::new(base_key, MIN_BASE_FEE_WEI_OFFSET)
        .set(backend, U256::from(INITIAL_MINIMUM_BASE_FEE_WEI))?;
    StorageBackedUint64::new(base_key, GAS_BACKLOG_OFFSET).set(backend, 0)?;
    StorageBackedUint64::new(base_key, PRICING_INERTIA_OFFSET)
        .set(backend, INITIAL_PRICING_INERTIA)?;
    StorageBackedUint64::new(base_key, BACKLOG_TOLERANCE_OFFSET)
        .set(backend, INITIAL_BACKLOG_TOLERANCE)?;
    Ok(())
}

pub fn open_l2_pricing_state<D>(sto: Storage<'_, D>, arbos_version: u64) -> L2PricingState<'_, D> {
    let base_key = sto.base_key();

    let gc_sto = sto.open_sub_storage(GAS_CONSTRAINTS_KEY);
    let mgc_sto = sto.open_sub_storage(MULTI_GAS_CONSTRAINTS_KEY);
    let mgf_sto = sto.open_sub_storage(MULTI_GAS_BASE_FEES_KEY);

    L2PricingState {
        arbos_version,
        speed_limit_per_second: StorageBackedUint64::new(base_key, SPEED_LIMIT_PER_SECOND_OFFSET),
        per_block_gas_limit: StorageBackedUint64::new(base_key, PER_BLOCK_GAS_LIMIT_OFFSET),
        base_fee_wei: StorageBackedBigUint::new(base_key, BASE_FEE_WEI_OFFSET),
        min_base_fee_wei: StorageBackedBigUint::new(base_key, MIN_BASE_FEE_WEI_OFFSET),
        gas_backlog: StorageBackedUint64::new(base_key, GAS_BACKLOG_OFFSET),
        pricing_inertia: StorageBackedUint64::new(base_key, PRICING_INERTIA_OFFSET),
        backlog_tolerance: StorageBackedUint64::new(base_key, BACKLOG_TOLERANCE_OFFSET),
        per_tx_gas_limit: StorageBackedUint64::new(base_key, PER_TX_GAS_LIMIT_OFFSET),
        gas_constraints: open_sub_storage_vector(gc_sto),
        multi_gas_constraints: open_sub_storage_vector(mgc_sto),
        multi_gas_base_fees: mgf_sto,
        backing_storage: sto,
    }
}

impl<'a, D> L2PricingState<'a, D> {
    pub fn open(sto: Storage<'a, D>, arbos_version: u64) -> Self {
        open_l2_pricing_state(sto, arbos_version)
    }

    pub fn initialize<B: StorageBackend>(
        sto: &Storage<'_, D>,
        backend: &mut B,
    ) -> Result<(), L2PricingError> {
        initialize_l2_pricing_state(sto, backend)
    }

    // --- Getters/Setters ---

    pub fn base_fee_wei<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<U256, L2PricingError> {
        Ok(self.base_fee_wei.get(backend)?)
    }

    pub fn set_base_fee_wei<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: U256,
    ) -> Result<(), L2PricingError> {
        Ok(self.base_fee_wei.set(backend, val)?)
    }

    pub fn min_base_fee_wei<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<U256, L2PricingError> {
        Ok(self.min_base_fee_wei.get(backend)?)
    }

    pub fn set_min_base_fee_wei<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: U256,
    ) -> Result<(), L2PricingError> {
        Ok(self.min_base_fee_wei.set(backend, val)?)
    }

    pub fn speed_limit_per_second<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, L2PricingError> {
        Ok(self.speed_limit_per_second.get(backend)?)
    }

    pub fn set_speed_limit_per_second<B: StorageBackend>(
        &self,
        backend: &mut B,
        limit: u64,
    ) -> Result<(), L2PricingError> {
        Ok(self.speed_limit_per_second.set(backend, limit)?)
    }

    pub fn per_block_gas_limit<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, L2PricingError> {
        Ok(self.per_block_gas_limit.get(backend)?)
    }

    pub fn set_max_per_block_gas_limit<B: StorageBackend>(
        &self,
        backend: &mut B,
        limit: u64,
    ) -> Result<(), L2PricingError> {
        Ok(self.per_block_gas_limit.set(backend, limit)?)
    }

    pub fn per_tx_gas_limit<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, L2PricingError> {
        Ok(self.per_tx_gas_limit.get(backend)?)
    }

    pub fn set_max_per_tx_gas_limit<B: StorageBackend>(
        &self,
        backend: &mut B,
        limit: u64,
    ) -> Result<(), L2PricingError> {
        Ok(self.per_tx_gas_limit.set(backend, limit)?)
    }

    pub fn gas_backlog<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, L2PricingError> {
        Ok(self.gas_backlog.get(backend)?)
    }

    pub fn set_gas_backlog<B: StorageBackend>(
        &self,
        backend: &mut B,
        backlog: u64,
    ) -> Result<(), L2PricingError> {
        Ok(self.gas_backlog.set(backend, backlog)?)
    }

    pub fn pricing_inertia<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, L2PricingError> {
        Ok(self.pricing_inertia.get(backend)?)
    }

    pub fn set_pricing_inertia<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: u64,
    ) -> Result<(), L2PricingError> {
        Ok(self.pricing_inertia.set(backend, val)?)
    }

    pub fn backlog_tolerance<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, L2PricingError> {
        Ok(self.backlog_tolerance.get(backend)?)
    }

    pub fn set_backlog_tolerance<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: u64,
    ) -> Result<(), L2PricingError> {
        Ok(self.backlog_tolerance.set(backend, val)?)
    }

    // --- Gas Constraints ---

    pub fn gas_constraints_length<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, L2PricingError> {
        Ok(self.gas_constraints.length(backend)?)
    }

    pub fn open_gas_constraint_at(&self, index: u64) -> GasConstraint {
        open_gas_constraint(self.gas_constraints.at(index))
    }

    pub fn add_gas_constraint<B: StorageBackend>(
        &self,
        backend: &mut B,
        target: u64,
        adjustment_window: u64,
        backlog: u64,
    ) -> Result<(), L2PricingError> {
        let key = self.gas_constraints.push(backend)?;
        let c = open_gas_constraint(key);
        c.set_target(backend, target)?;
        c.set_adjustment_window(backend, adjustment_window)?;
        c.set_backlog(backend, backlog)?;
        Ok(())
    }

    pub fn clear_gas_constraints<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<(), L2PricingError> {
        let len = self.gas_constraints.length(backend)?;
        for i in 0..len {
            let c = self.open_gas_constraint_at(i);
            c.clear(backend)?;
        }
        for _ in 0..len {
            self.gas_constraints.pop(backend)?;
        }
        Ok(())
    }

    // --- Multi-Gas Constraints ---

    pub fn multi_gas_constraints_length<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, L2PricingError> {
        Ok(self.multi_gas_constraints.length(backend)?)
    }

    pub fn open_multi_gas_constraint_at(&self, index: u64) -> MultiGasConstraint {
        open_multi_gas_constraint(self.multi_gas_constraints.at(index))
    }

    pub fn add_multi_gas_constraint<B: StorageBackend>(
        &self,
        backend: &mut B,
        target: u64,
        adjustment_window: u32,
        backlog: u64,
        weights: &[u64; NUM_RESOURCE_KIND],
    ) -> Result<(), L2PricingError> {
        let key = self.multi_gas_constraints.push(backend)?;
        let c = open_multi_gas_constraint(key);
        c.set_target(backend, target)?;
        c.set_adjustment_window(backend, adjustment_window)?;
        c.set_backlog(backend, backlog)?;
        c.set_resource_weights(backend, weights)?;
        Ok(())
    }

    pub fn clear_multi_gas_constraints<B: StorageBackend>(
        &self,
        backend: &mut B,
    ) -> Result<(), L2PricingError> {
        let len = self.multi_gas_constraints.length(backend)?;
        for i in 0..len {
            let c = self.open_multi_gas_constraint_at(i);
            c.clear(backend)?;
        }
        for _ in 0..len {
            self.multi_gas_constraints.pop(backend)?;
        }
        Ok(())
    }

    pub fn restrict(&self, _err: ()) {
        // No-op restriction
    }

    /// Per-resource-kind base fee accessor for the current/next block.
    pub fn multi_gas_fees(&self) -> MultiGasFees<'a, D> {
        multi_gas_fees::open_multi_gas_fees(self.multi_gas_base_fees.clone())
    }
}
