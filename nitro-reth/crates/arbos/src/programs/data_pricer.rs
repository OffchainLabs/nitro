use alloy_primitives::U256;
use revm::Database;

use arb_storage::{Storage, StorageBackedUint32, StorageBackedUint64};

const DEMAND_OFFSET: u64 = 0;
const BYTES_PER_SECOND_OFFSET: u64 = 1;
const LAST_UPDATE_TIME_OFFSET: u64 = 2;
const MIN_PRICE_OFFSET: u64 = 3;
const INERTIA_OFFSET: u64 = 4;

/// The day it all began (Arbitrum genesis timestamp).
pub const ARBITRUM_START_TIME: u64 = 1421388000;

const INITIAL_DEMAND: u32 = 0;
/// 1TB total footprint per year, refilled each second.
pub const INITIAL_HOURLY_BYTES: u64 = 1 * (1u64 << 40) / (365 * 24);
const INITIAL_BYTES_PER_SECOND: u32 = (INITIAL_HOURLY_BYTES / (60 * 60)) as u32;
const INITIAL_LAST_UPDATE_TIME: u64 = ARBITRUM_START_TIME;
const INITIAL_MIN_PRICE: u32 = 82928201; // 5Mb = $1
const INITIAL_INERTIA: u32 = 21360419; // expensive at 1Tb

/// One in basis points (10000).
const ONE_IN_BIPS: u64 = 10000;

/// Stylus data pricing model using exponential demand curve.
pub struct DataPricer<D> {
    pub demand: StorageBackedUint32<D>,
    pub bytes_per_second: StorageBackedUint32<D>,
    pub last_update_time: StorageBackedUint64<D>,
    pub min_price: StorageBackedUint32<D>,
    pub inertia: StorageBackedUint32<D>,
}

pub fn init_data_pricer<D: Database>(sto: &Storage<D>) {
    let state = sto.state_ptr();
    let base_key = sto.base_key();
    let _ = StorageBackedUint32::new(state, base_key, DEMAND_OFFSET).set(INITIAL_DEMAND);
    let _ = StorageBackedUint32::new(state, base_key, BYTES_PER_SECOND_OFFSET).set(INITIAL_BYTES_PER_SECOND);
    let _ = StorageBackedUint64::new(state, base_key, LAST_UPDATE_TIME_OFFSET).set(INITIAL_LAST_UPDATE_TIME);
    let _ = StorageBackedUint32::new(state, base_key, MIN_PRICE_OFFSET).set(INITIAL_MIN_PRICE);
    let _ = StorageBackedUint32::new(state, base_key, INERTIA_OFFSET).set(INITIAL_INERTIA);
}

pub fn open_data_pricer<D: Database>(sto: &Storage<D>) -> DataPricer<D> {
    let state = sto.state_ptr();
    let base_key = sto.base_key();
    DataPricer {
        demand: StorageBackedUint32::new(state, base_key, DEMAND_OFFSET),
        bytes_per_second: StorageBackedUint32::new(state, base_key, BYTES_PER_SECOND_OFFSET),
        last_update_time: StorageBackedUint64::new(state, base_key, LAST_UPDATE_TIME_OFFSET),
        min_price: StorageBackedUint32::new(state, base_key, MIN_PRICE_OFFSET),
        inertia: StorageBackedUint32::new(state, base_key, INERTIA_OFFSET),
    }
}

impl<D: Database> DataPricer<D> {
    /// Update the pricing model with new data usage and return cost in wei.
    pub fn update_model(&self, temp_bytes: u32, time: u64) -> Result<U256, ()> {
        let demand = self.demand.get().unwrap_or(0);
        let bytes_per_second = self.bytes_per_second.get().unwrap_or(0);
        let last_update_time = self.last_update_time.get().unwrap_or(0);
        let min_price = self.min_price.get().unwrap_or(0);
        let inertia = self.inertia.get()?;

        if inertia == 0 {
            return Ok(U256::ZERO);
        }

        let passed = (time.saturating_sub(last_update_time)) as u32;
        let credit = bytes_per_second.saturating_mul(passed);
        let demand = demand.saturating_sub(credit).saturating_add(temp_bytes);

        self.demand.set(demand)?;
        self.last_update_time.set(time)?;

        let exponent = ONE_IN_BIPS * (demand as u64) / (inertia as u64);
        let multiplier = approx_exp_basis_points(exponent, 12);
        let cost_per_byte = saturating_mul_by_bips(min_price as u64, multiplier);
        let cost_in_wei = cost_per_byte.saturating_mul(temp_bytes as u64);
        Ok(U256::from(cost_in_wei))
    }
}

/// Approximate e^(x/10000) * 10000 using a Taylor series with `terms` terms.
fn approx_exp_basis_points(x: u64, terms: u32) -> u64 {
    if x == 0 {
        return ONE_IN_BIPS;
    }

    let mut result = ONE_IN_BIPS;
    let mut term = ONE_IN_BIPS;

    for k in 1..=terms {
        term = term * x / (ONE_IN_BIPS * k as u64);
        result = result.saturating_add(term);
        if term == 0 {
            break;
        }
    }
    result
}

/// Multiply a u64 by a bips value, saturating on overflow.
fn saturating_mul_by_bips(value: u64, bips: u64) -> u64 {
    (value as u128 * bips as u128 / ONE_IN_BIPS as u128).min(u64::MAX as u128) as u64
}
