mod gas_constraint;
mod model;
mod multi_gas_constraint;
mod multi_gas_fees;

pub use gas_constraint::{GasConstraint, open_gas_constraint};
pub use model::*;
pub use multi_gas_constraint::{MultiGasConstraint, open_multi_gas_constraint};
pub use multi_gas_fees::MultiGasFees;

use alloy_primitives::U256;
use revm::Database;

use arb_primitives::multigas::NUM_RESOURCE_KIND;
use arb_storage::{
    open_sub_storage_vector, Storage, StorageBackedBigUint, StorageBackedUint64, SubStorageVector,
};

// Storage offsets for L2 pricing state.
const SPEED_LIMIT_PER_SECOND_OFFSET: u64 = 0;
const PER_BLOCK_GAS_LIMIT_OFFSET: u64 = 1;
const BASE_FEE_WEI_OFFSET: u64 = 2;
const MIN_BASE_FEE_WEI_OFFSET: u64 = 3;
const GAS_BACKLOG_OFFSET: u64 = 4;
const PRICING_INERTIA_OFFSET: u64 = 5;
const BACKLOG_TOLERANCE_OFFSET: u64 = 6;
const PER_TX_GAS_LIMIT_OFFSET: u64 = 7;

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
pub struct L2PricingState<D> {
    pub backing_storage: Storage<D>,
    pub arbos_version: u64,
    speed_limit_per_second: StorageBackedUint64<D>,
    per_block_gas_limit: StorageBackedUint64<D>,
    base_fee_wei: StorageBackedBigUint<D>,
    min_base_fee_wei: StorageBackedBigUint<D>,
    gas_backlog: StorageBackedUint64<D>,
    pricing_inertia: StorageBackedUint64<D>,
    backlog_tolerance: StorageBackedUint64<D>,
    per_tx_gas_limit: StorageBackedUint64<D>,
    gas_constraints: SubStorageVector<D>,
    multi_gas_constraints: SubStorageVector<D>,
    multi_gas_base_fees: Storage<D>,
}

pub fn initialize_l2_pricing_state<D: Database>(sto: &Storage<D>) {
    let state = sto.state_ptr();
    let base_key = sto.base_key();

    let _ = StorageBackedUint64::new(state, base_key, SPEED_LIMIT_PER_SECOND_OFFSET)
        .set(INITIAL_SPEED_LIMIT_PER_SECOND_V0);
    let _ = StorageBackedUint64::new(state, base_key, PER_BLOCK_GAS_LIMIT_OFFSET)
        .set(INITIAL_PER_BLOCK_GAS_LIMIT_V0);
    let _ = StorageBackedUint64::new(state, base_key, BASE_FEE_WEI_OFFSET)
        .set(INITIAL_BASE_FEE_WEI);
    let _ = StorageBackedBigUint::new(state, base_key, MIN_BASE_FEE_WEI_OFFSET)
        .set(U256::from(INITIAL_MINIMUM_BASE_FEE_WEI));
    let _ = StorageBackedUint64::new(state, base_key, GAS_BACKLOG_OFFSET).set(0);
    let _ = StorageBackedUint64::new(state, base_key, PRICING_INERTIA_OFFSET)
        .set(INITIAL_PRICING_INERTIA);
    let _ = StorageBackedUint64::new(state, base_key, BACKLOG_TOLERANCE_OFFSET)
        .set(INITIAL_BACKLOG_TOLERANCE);
}

pub fn open_l2_pricing_state<D: Database>(sto: Storage<D>, arbos_version: u64) -> L2PricingState<D> {
    let state = sto.state_ptr();
    let base_key = sto.base_key();

    let gc_sto = sto.open_sub_storage(GAS_CONSTRAINTS_KEY);
    let mgc_sto = sto.open_sub_storage(MULTI_GAS_CONSTRAINTS_KEY);
    let mgf_sto = sto.open_sub_storage(MULTI_GAS_BASE_FEES_KEY);

    L2PricingState {
        arbos_version,
        speed_limit_per_second: StorageBackedUint64::new(
            state,
            base_key,
            SPEED_LIMIT_PER_SECOND_OFFSET,
        ),
        per_block_gas_limit: StorageBackedUint64::new(
            state,
            base_key,
            PER_BLOCK_GAS_LIMIT_OFFSET,
        ),
        base_fee_wei: StorageBackedBigUint::new(state, base_key, BASE_FEE_WEI_OFFSET),
        min_base_fee_wei: StorageBackedBigUint::new(state, base_key, MIN_BASE_FEE_WEI_OFFSET),
        gas_backlog: StorageBackedUint64::new(state, base_key, GAS_BACKLOG_OFFSET),
        pricing_inertia: StorageBackedUint64::new(state, base_key, PRICING_INERTIA_OFFSET),
        backlog_tolerance: StorageBackedUint64::new(state, base_key, BACKLOG_TOLERANCE_OFFSET),
        per_tx_gas_limit: StorageBackedUint64::new(state, base_key, PER_TX_GAS_LIMIT_OFFSET),
        gas_constraints: open_sub_storage_vector(gc_sto),
        multi_gas_constraints: open_sub_storage_vector(mgc_sto),
        multi_gas_base_fees: mgf_sto,
        backing_storage: sto,
    }
}

impl<D: Database> L2PricingState<D> {
    pub fn open(sto: Storage<D>, arbos_version: u64) -> Self {
        open_l2_pricing_state(sto, arbos_version)
    }

    pub fn initialize(sto: &Storage<D>) {
        initialize_l2_pricing_state(sto);
    }

    // --- Getters/Setters ---

    pub fn base_fee_wei(&self) -> Result<U256, ()> {
        self.base_fee_wei.get()
    }

    pub fn set_base_fee_wei(&self, val: U256) -> Result<(), ()> {
        self.base_fee_wei.set(val)
    }

    pub fn min_base_fee_wei(&self) -> Result<U256, ()> {
        self.min_base_fee_wei.get()
    }

    pub fn set_min_base_fee_wei(&self, val: U256) -> Result<(), ()> {
        self.min_base_fee_wei.set(val)
    }

    pub fn speed_limit_per_second(&self) -> Result<u64, ()> {
        self.speed_limit_per_second.get()
    }

    pub fn set_speed_limit_per_second(&self, limit: u64) -> Result<(), ()> {
        self.speed_limit_per_second.set(limit)
    }

    pub fn per_block_gas_limit(&self) -> Result<u64, ()> {
        self.per_block_gas_limit.get()
    }

    pub fn set_max_per_block_gas_limit(&self, limit: u64) -> Result<(), ()> {
        self.per_block_gas_limit.set(limit)
    }

    pub fn per_tx_gas_limit(&self) -> Result<u64, ()> {
        self.per_tx_gas_limit.get()
    }

    pub fn set_max_per_tx_gas_limit(&self, limit: u64) -> Result<(), ()> {
        self.per_tx_gas_limit.set(limit)
    }

    pub fn gas_backlog(&self) -> Result<u64, ()> {
        self.gas_backlog.get()
    }

    pub fn set_gas_backlog(&self, backlog: u64) -> Result<(), ()> {
        self.gas_backlog.set(backlog)
    }

    pub fn pricing_inertia(&self) -> Result<u64, ()> {
        self.pricing_inertia.get()
    }

    pub fn set_pricing_inertia(&self, val: u64) -> Result<(), ()> {
        self.pricing_inertia.set(val)
    }

    pub fn backlog_tolerance(&self) -> Result<u64, ()> {
        self.backlog_tolerance.get()
    }

    pub fn set_backlog_tolerance(&self, val: u64) -> Result<(), ()> {
        self.backlog_tolerance.set(val)
    }

    // --- Gas Constraints ---

    pub fn gas_constraints_length(&self) -> Result<u64, ()> {
        self.gas_constraints.length()
    }

    pub fn open_gas_constraint_at(&self, index: u64) -> GasConstraint<D> {
        open_gas_constraint(self.gas_constraints.at(index))
    }

    pub fn add_gas_constraint(
        &self,
        target: u64,
        adjustment_window: u64,
        backlog: u64,
    ) -> Result<(), ()> {
        let sto = self.gas_constraints.push()?;
        let c = open_gas_constraint(sto);
        c.set_target(target)?;
        c.set_adjustment_window(adjustment_window)?;
        c.set_backlog(backlog)?;
        Ok(())
    }

    pub fn clear_gas_constraints(&self) -> Result<(), ()> {
        let len = self.gas_constraints.length()?;
        for i in 0..len {
            let c = self.open_gas_constraint_at(i);
            c.clear()?;
        }
        // Reset vector length by popping all
        for _ in 0..len {
            self.gas_constraints.pop()?;
        }
        Ok(())
    }

    // --- Multi-Gas Constraints ---

    pub fn multi_gas_constraints_length(&self) -> Result<u64, ()> {
        self.multi_gas_constraints.length()
    }

    pub fn open_multi_gas_constraint_at(&self, index: u64) -> MultiGasConstraint<D> {
        open_multi_gas_constraint(self.multi_gas_constraints.at(index))
    }

    pub fn add_multi_gas_constraint(
        &self,
        target: u64,
        adjustment_window: u32,
        backlog: u64,
        weights: &[u64; NUM_RESOURCE_KIND],
    ) -> Result<(), ()> {
        let sto = self.multi_gas_constraints.push()?;
        let c = open_multi_gas_constraint(sto);
        c.set_target(target)?;
        c.set_adjustment_window(adjustment_window)?;
        c.set_backlog(backlog)?;
        c.set_resource_weights(weights)?;
        Ok(())
    }

    pub fn clear_multi_gas_constraints(&self) -> Result<(), ()> {
        let len = self.multi_gas_constraints.length()?;
        for i in 0..len {
            let c = self.open_multi_gas_constraint_at(i);
            c.clear()?;
        }
        for _ in 0..len {
            self.multi_gas_constraints.pop()?;
        }
        Ok(())
    }

    pub fn restrict(&self, _err: ()) {
        // No-op restriction
    }
}
