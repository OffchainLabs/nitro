mod batch_poster;
mod error;

pub use batch_poster::*;
pub use error::L1PricingError;

use alloy_primitives::{Address, U256};

use arb_storage::{
    Storage, StorageBackedAddress, StorageBackedBigInt, StorageBackedBigUint, StorageBackedInt64,
    StorageBackedUint64, StorageBackend, SystemStateBackend,
};

use crate::util::BalanceError;

// Storage offsets for L1 pricing state.
pub const PAY_REWARDS_TO_OFFSET: u64 = 0;
pub const EQUILIBRATION_UNITS_OFFSET: u64 = 1;
pub const INERTIA_OFFSET: u64 = 2;
pub const PER_UNIT_REWARD_OFFSET: u64 = 3;
pub const LAST_UPDATE_TIME_OFFSET: u64 = 4;
pub const FUNDS_DUE_FOR_REWARDS_OFFSET: u64 = 5;
pub const UNITS_SINCE_OFFSET: u64 = 6;
pub const PRICE_PER_UNIT_OFFSET: u64 = 7;
pub const LAST_SURPLUS_OFFSET: u64 = 8;
pub const PER_BATCH_GAS_COST_OFFSET: u64 = 9;
pub const AMORTIZED_COST_CAP_BIPS_OFFSET: u64 = 10;
pub const L1_FEES_AVAILABLE_OFFSET: u64 = 11;
pub const GAS_FLOOR_PER_TOKEN_OFFSET: u64 = 12;

// Well-known addresses.
pub const BATCH_POSTER_ADDRESS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65, 0x71, 0x75, 0x65,
    0x6e, 0x63, 0x65, 0x72,
]);
pub const BATCH_POSTER_PAY_TO_ADDRESS: Address = BATCH_POSTER_ADDRESS;

pub const L1_PRICER_FUNDS_POOL_ADDRESS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0xf6,
]);

// Initial values.
pub const INITIAL_INERTIA: u64 = 10;
pub const INITIAL_PER_UNIT_REWARD: u64 = 10;
pub const INITIAL_EQUILIBRATION_UNITS_V0: u64 = 60 * 16 * 100_000;
pub const INITIAL_EQUILIBRATION_UNITS_V6: u64 = 16 * 10_000_000;
pub const INITIAL_PER_BATCH_GAS_COST_V6: i64 = 100_000;
pub const INITIAL_PER_BATCH_GAS_COST_V12: i64 = 210_000;

// EIP-2028 gas cost per non-zero byte of calldata.
pub const TX_DATA_NON_ZERO_GAS_EIP2028: u64 = 16;

// Estimation padding constants.
pub const ESTIMATION_PADDING_UNITS: u64 = TX_DATA_NON_ZERO_GAS_EIP2028 * 16;
pub const ESTIMATION_PADDING_BASIS_POINTS: u64 = 100;
const ONE_IN_BIPS: u64 = 10000;

/// L1 pricing state manages the cost model for L1 data posting.
pub struct L1PricingState<'a, D> {
    pub backing_storage: Storage<'a, D>,
    pay_rewards_to: StorageBackedAddress,
    equilibration_units: StorageBackedBigUint,
    inertia: StorageBackedUint64,
    per_unit_reward: StorageBackedUint64,
    last_update_time: StorageBackedUint64,
    funds_due_for_rewards: StorageBackedBigInt,
    units_since_update: StorageBackedUint64,
    price_per_unit: StorageBackedBigUint,
    last_surplus: StorageBackedBigInt,
    per_batch_gas_cost: StorageBackedInt64,
    amortized_cost_cap_bips: StorageBackedUint64,
    l1_fees_available: StorageBackedBigUint,
    gas_floor_per_token: StorageBackedUint64,
    pub arbos_version: u64,
}

pub fn initialize_l1_pricing_state<D: revm::Database, B: StorageBackend>(
    sto: &Storage<'_, D>,
    backend: &mut B,
    rewards_recipient: Address,
    initial_l1_base_fee: U256,
) -> Result<(), L1PricingError> {
    let base_key = sto.base_key();

    StorageBackedAddress::new(base_key, PAY_REWARDS_TO_OFFSET).set(backend, rewards_recipient)?;
    StorageBackedBigUint::new(base_key, EQUILIBRATION_UNITS_OFFSET)
        .set(backend, U256::from(INITIAL_EQUILIBRATION_UNITS_V0))?;
    StorageBackedUint64::new(base_key, INERTIA_OFFSET).set(backend, INITIAL_INERTIA)?;
    StorageBackedUint64::new(base_key, PER_UNIT_REWARD_OFFSET)
        .set(backend, INITIAL_PER_UNIT_REWARD)?;
    StorageBackedUint64::new(base_key, LAST_UPDATE_TIME_OFFSET).set(backend, 0)?;
    StorageBackedBigInt::new(base_key, FUNDS_DUE_FOR_REWARDS_OFFSET).set(backend, U256::ZERO)?;
    StorageBackedUint64::new(base_key, UNITS_SINCE_OFFSET).set(backend, 0)?;
    StorageBackedBigUint::new(base_key, PRICE_PER_UNIT_OFFSET).set(backend, initial_l1_base_fee)?;

    initialize_batch_posters_table(sto, backend, BATCH_POSTER_ADDRESS)?;
    Ok(())
}

pub fn open_l1_pricing_state<D>(sto: Storage<'_, D>, arbos_version: u64) -> L1PricingState<'_, D> {
    let base_key = sto.base_key();

    L1PricingState {
        pay_rewards_to: StorageBackedAddress::new(base_key, PAY_REWARDS_TO_OFFSET),
        equilibration_units: StorageBackedBigUint::new(base_key, EQUILIBRATION_UNITS_OFFSET),
        inertia: StorageBackedUint64::new(base_key, INERTIA_OFFSET),
        per_unit_reward: StorageBackedUint64::new(base_key, PER_UNIT_REWARD_OFFSET),
        last_update_time: StorageBackedUint64::new(base_key, LAST_UPDATE_TIME_OFFSET),
        funds_due_for_rewards: StorageBackedBigInt::new(base_key, FUNDS_DUE_FOR_REWARDS_OFFSET),
        units_since_update: StorageBackedUint64::new(base_key, UNITS_SINCE_OFFSET),
        price_per_unit: StorageBackedBigUint::new(base_key, PRICE_PER_UNIT_OFFSET),
        last_surplus: StorageBackedBigInt::new(base_key, LAST_SURPLUS_OFFSET),
        per_batch_gas_cost: StorageBackedInt64::new(base_key, PER_BATCH_GAS_COST_OFFSET),
        amortized_cost_cap_bips: StorageBackedUint64::new(base_key, AMORTIZED_COST_CAP_BIPS_OFFSET),
        l1_fees_available: StorageBackedBigUint::new(base_key, L1_FEES_AVAILABLE_OFFSET),
        gas_floor_per_token: StorageBackedUint64::new(base_key, GAS_FLOOR_PER_TOKEN_OFFSET),
        backing_storage: sto,
        arbos_version,
    }
}

impl<'a, D> L1PricingState<'a, D> {
    pub fn open(sto: Storage<'a, D>, arbos_version: u64) -> Self {
        open_l1_pricing_state(sto, arbos_version)
    }

    pub fn batch_poster_table(&self) -> BatchPostersTable<'a, D> {
        BatchPostersTable::open(&self.backing_storage)
    }

    // --- Getters/Setters ---

    pub fn pay_rewards_to<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<Address, L1PricingError> {
        Ok(self.pay_rewards_to.get(backend)?)
    }

    pub fn set_pay_rewards_to<B: StorageBackend>(
        &self,
        backend: &mut B,
        addr: Address,
    ) -> Result<(), L1PricingError> {
        Ok(self.pay_rewards_to.set(backend, addr)?)
    }

    pub fn equilibration_units<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<U256, L1PricingError> {
        Ok(self.equilibration_units.get(backend)?)
    }

    pub fn set_equilibration_units<B: StorageBackend>(
        &self,
        backend: &mut B,
        units: U256,
    ) -> Result<(), L1PricingError> {
        Ok(self.equilibration_units.set(backend, units)?)
    }

    pub fn inertia<B: SystemStateBackend>(&self, backend: &mut B) -> Result<u64, L1PricingError> {
        Ok(self.inertia.get(backend)?)
    }

    pub fn set_inertia<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: u64,
    ) -> Result<(), L1PricingError> {
        Ok(self.inertia.set(backend, val)?)
    }

    pub fn per_unit_reward<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, L1PricingError> {
        Ok(self.per_unit_reward.get(backend)?)
    }

    pub fn set_per_unit_reward<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: u64,
    ) -> Result<(), L1PricingError> {
        Ok(self.per_unit_reward.set(backend, val)?)
    }

    pub fn last_update_time<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, L1PricingError> {
        Ok(self.last_update_time.get(backend)?)
    }

    pub fn set_last_update_time<B: StorageBackend>(
        &self,
        backend: &mut B,
        time: u64,
    ) -> Result<(), L1PricingError> {
        Ok(self.last_update_time.set(backend, time)?)
    }

    pub fn funds_due_for_rewards<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<U256, L1PricingError> {
        Ok(self.funds_due_for_rewards.get_raw(backend)?)
    }

    pub fn set_funds_due_for_rewards<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: U256,
    ) -> Result<(), L1PricingError> {
        Ok(self.funds_due_for_rewards.set(backend, val)?)
    }

    pub fn units_since_update<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, L1PricingError> {
        Ok(self.units_since_update.get(backend)?)
    }

    pub fn set_units_since_update<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: u64,
    ) -> Result<(), L1PricingError> {
        Ok(self.units_since_update.set(backend, val)?)
    }

    pub fn add_to_units_since_update<B: StorageBackend>(
        &self,
        backend: &mut B,
        units: u64,
    ) -> Result<(), L1PricingError> {
        let current = self.units_since_update.get(backend).unwrap_or(0);
        Ok(self
            .units_since_update
            .set(backend, current.saturating_add(units))?)
    }

    pub fn subtract_from_units_since_update<B: StorageBackend>(
        &self,
        backend: &mut B,
        units: u64,
    ) -> Result<(), L1PricingError> {
        let current = self.units_since_update.get(backend).unwrap_or(0);
        Ok(self
            .units_since_update
            .set(backend, current.saturating_sub(units))?)
    }

    pub fn price_per_unit<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<U256, L1PricingError> {
        Ok(self.price_per_unit.get(backend)?)
    }

    pub fn set_price_per_unit<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: U256,
    ) -> Result<(), L1PricingError> {
        Ok(self.price_per_unit.set(backend, val)?)
    }

    pub fn last_surplus<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<(U256, bool), L1PricingError> {
        Ok(self.last_surplus.get_signed(backend)?)
    }

    pub fn set_last_surplus<B: StorageBackend>(
        &self,
        backend: &mut B,
        magnitude: U256,
        negative: bool,
    ) -> Result<(), L1PricingError> {
        if self.arbos_version < 7 {
            return Ok(());
        }
        if negative {
            Ok(self.last_surplus.set_negative(backend, magnitude)?)
        } else {
            Ok(self.last_surplus.set(backend, magnitude)?)
        }
    }

    pub fn per_batch_gas_cost<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<i64, L1PricingError> {
        Ok(self.per_batch_gas_cost.get(backend)?)
    }

    pub fn set_per_batch_gas_cost<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: i64,
    ) -> Result<(), L1PricingError> {
        Ok(self.per_batch_gas_cost.set(backend, val)?)
    }

    pub fn amortized_cost_cap_bips<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, L1PricingError> {
        Ok(self.amortized_cost_cap_bips.get(backend)?)
    }

    pub fn set_amortized_cost_cap_bips<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: u64,
    ) -> Result<(), L1PricingError> {
        Ok(self.amortized_cost_cap_bips.set(backend, val)?)
    }

    pub fn l1_fees_available<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<U256, L1PricingError> {
        Ok(self.l1_fees_available.get(backend)?)
    }

    pub fn set_l1_fees_available<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: U256,
    ) -> Result<(), L1PricingError> {
        Ok(self.l1_fees_available.set(backend, val)?)
    }

    pub fn add_to_l1_fees_available<B: StorageBackend>(
        &self,
        backend: &mut B,
        amount: U256,
    ) -> Result<(), L1PricingError> {
        let current = self.l1_fees_available.get(backend).unwrap_or(U256::ZERO);
        Ok(self
            .l1_fees_available
            .set(backend, current.saturating_add(amount))?)
    }

    pub fn transfer_from_l1_fees_available<B: StorageBackend>(
        &self,
        backend: &mut B,
        amount: U256,
    ) -> Result<U256, L1PricingError> {
        let available = self.l1_fees_available.get(backend).unwrap_or(U256::ZERO);
        let transfer = amount.min(available);
        self.l1_fees_available
            .set(backend, available.saturating_sub(transfer))?;
        Ok(transfer)
    }

    pub fn parent_gas_floor_per_token<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<u64, L1PricingError> {
        if self.arbos_version < arb_chainspec::arbos_version::ARBOS_VERSION_50 {
            return Ok(0);
        }
        Ok(self.gas_floor_per_token.get(backend)?)
    }

    pub fn set_parent_gas_floor_per_token<B: StorageBackend>(
        &self,
        backend: &mut B,
        val: u64,
    ) -> Result<(), L1PricingError> {
        if self.arbos_version < arb_chainspec::arbos_version::ARBOS_VERSION_50 {
            return Err(L1PricingError::ParentGasFloorUnsupportedVersion);
        }
        Ok(self.gas_floor_per_token.set(backend, val)?)
    }

    // --- Pricing logic ---

    pub fn get_l1_pricing_surplus<B: SystemStateBackend>(
        &self,
        backend: &mut B,
    ) -> Result<(U256, bool), L1PricingError> {
        let l1_fees_available = self.l1_fees_available.get(backend).unwrap_or(U256::ZERO);
        let bpt = self.batch_poster_table();
        let total_funds_due = bpt.total_funds_due(backend).unwrap_or(U256::ZERO);
        let funds_due_for_rewards = self.funds_due_for_rewards(backend).unwrap_or(U256::ZERO);

        let need = total_funds_due.saturating_add(funds_due_for_rewards);
        if l1_fees_available >= need {
            Ok((l1_fees_available.saturating_sub(need), false))
        } else {
            Ok((need.saturating_sub(l1_fees_available), true))
        }
    }

    pub fn poster_data_cost<B: SystemStateBackend>(
        &self,
        backend: &mut B,
        calldata_units: u64,
    ) -> Result<U256, L1PricingError> {
        let price = self.price_per_unit(backend)?;
        let batch_cost = self.per_batch_gas_cost(backend)?;

        let calldata_cost = price.saturating_mul(U256::from(calldata_units));
        if batch_cost >= 0 {
            Ok(calldata_cost.saturating_add(U256::from(batch_cost as u64)))
        } else {
            Ok(calldata_cost.saturating_sub(U256::from((-batch_cost) as u64)))
        }
    }

    /// Compute poster cost and units for a transaction on-chain.
    pub fn compute_poster_cost<B: SystemStateBackend>(
        &self,
        backend: &mut B,
        poster: Address,
        tx_bytes: &[u8],
        brotli_compression_level: u64,
    ) -> Result<(U256, u64), L1PricingError> {
        if poster != BATCH_POSTER_ADDRESS {
            return Ok((U256::ZERO, 0));
        }
        let units = self.get_poster_units_without_cache(tx_bytes, brotli_compression_level);
        let price = self.price_per_unit(backend)?;
        Ok((price.saturating_mul(U256::from(units)), units))
    }

    /// Compute poster data cost for gas estimation (with padding).
    pub fn poster_data_cost_for_estimation<B: SystemStateBackend>(
        &self,
        backend: &mut B,
        tx_bytes: &[u8],
        brotli_compression_level: u64,
    ) -> Result<(U256, u64), L1PricingError> {
        let raw_units = self.get_poster_units_without_cache(tx_bytes, brotli_compression_level);
        let padded = (raw_units.saturating_add(ESTIMATION_PADDING_UNITS))
            .saturating_mul(ONE_IN_BIPS + ESTIMATION_PADDING_BASIS_POINTS)
            / ONE_IN_BIPS;
        let price = self.price_per_unit(backend)?;
        Ok((price.saturating_mul(U256::from(padded)), padded))
    }

    /// Compute the L1 calldata units for a transaction.
    pub fn get_poster_units_without_cache(
        &self,
        tx_bytes: &[u8],
        brotli_compression_level: u64,
    ) -> u64 {
        let l1_bytes = byte_count_after_brotli_level(tx_bytes, brotli_compression_level);
        TX_DATA_NON_ZERO_GAS_EIP2028.saturating_mul(l1_bytes)
    }

    fn _preversion10_update(
        &self,
        _update_time: u64,
        _current_time: u64,
        _wei_spent: U256,
        _l1_basefee: U256,
    ) -> Result<(), L1PricingError> {
        Ok(())
    }

    fn _preversion2_update(
        &self,
        _update_time: u64,
        _current_time: u64,
        _wei_spent: U256,
        _l1_basefee: U256,
    ) -> Result<(), L1PricingError> {
        Ok(())
    }
}

impl<D: revm::Database> L1PricingState<'_, D> {
    pub fn initialize<B: StorageBackend>(
        sto: &Storage<'_, D>,
        backend: &mut B,
        rewards_recipient: Address,
        initial_l1_base_fee: U256,
    ) -> Result<(), L1PricingError> {
        initialize_l1_pricing_state(sto, backend, rewards_recipient, initial_l1_base_fee)
    }

    pub fn get_poster_info<B: StorageBackend>(
        &self,
        backend: &mut B,
        poster: Address,
    ) -> Result<(U256, Address), L1PricingError> {
        let bpt = self.batch_poster_table();
        let state = bpt.open_poster(backend, poster, false)?;
        let due = state.funds_due(backend)?;
        let pay_to = state.pay_to(backend)?;
        Ok((due, pay_to))
    }

    /// Update pricing based on a batch poster spending report.
    pub fn update_for_batch_poster_spending<F, B>(
        &self,
        backend: &mut B,
        update_time: u64,
        current_time: u64,
        batch_poster: Address,
        wei_spent: U256,
        l1_basefee: U256,
        mut transfer_fn: F,
    ) -> Result<(), L1PricingError>
    where
        F: FnMut(Address, Address, U256) -> Result<(), BalanceError>,
        B: StorageBackend,
    {
        if self.arbos_version < 10 {
            return self._preversion10_update(update_time, current_time, wei_spent, l1_basefee);
        }

        let bpt = self.batch_poster_table();
        let poster_state = bpt.open_poster(backend, batch_poster, true)?;

        let funds_due_for_rewards = self.funds_due_for_rewards(backend).unwrap_or(U256::ZERO);
        let l1_fees_available = self.l1_fees_available.get(backend).unwrap_or(U256::ZERO);

        let mut last_update_time = self.last_update_time(backend).unwrap_or(0);
        if last_update_time == 0 && update_time > 0 {
            last_update_time = update_time.saturating_sub(1);
        }

        if update_time > current_time || update_time < last_update_time {
            return Err(L1PricingError::InvalidUpdateTime);
        }

        let alloc_num = update_time.saturating_sub(last_update_time);
        let alloc_denom = current_time.saturating_sub(last_update_time);
        let (alloc_num, alloc_denom) = if alloc_denom == 0 {
            (1u64, 1u64)
        } else {
            (alloc_num, alloc_denom)
        };

        let units_since = self.units_since_update(backend).unwrap_or(0);
        let units_allocated = units_since
            .saturating_mul(alloc_num)
            .checked_div(alloc_denom)
            .unwrap_or(0);
        self.set_units_since_update(backend, units_since.saturating_sub(units_allocated))?;

        let mut wei_spent = wei_spent;
        if self.arbos_version >= 3 {
            let cap_bips = self.amortized_cost_cap_bips(backend).unwrap_or(0);
            if cap_bips != 0 {
                let cap = l1_basefee
                    .saturating_mul(U256::from(units_allocated))
                    .saturating_mul(U256::from(cap_bips))
                    .checked_div(U256::from(10000u64))
                    .unwrap_or(U256::MAX);
                if cap < wei_spent {
                    wei_spent = cap;
                }
            }
        }

        let due = poster_state.funds_due(backend).unwrap_or(U256::ZERO);
        let _ = poster_state.set_funds_due(
            backend,
            due.saturating_add(wei_spent),
            &bpt.total_funds_due,
        );

        let per_unit_reward = self.per_unit_reward(backend).unwrap_or(0);
        let reward_amount = U256::from(units_allocated).saturating_mul(U256::from(per_unit_reward));
        self.set_funds_due_for_rewards(
            backend,
            funds_due_for_rewards.saturating_add(reward_amount),
        )?;

        let mut l1_fees = l1_fees_available;
        let mut payment_for_rewards = reward_amount;
        if l1_fees < payment_for_rewards {
            payment_for_rewards = l1_fees;
        }
        let fdr_after = self
            .funds_due_for_rewards(backend)
            .unwrap_or(U256::ZERO)
            .saturating_sub(payment_for_rewards);
        self.set_funds_due_for_rewards(backend, fdr_after)?;

        let pay_rewards_to = self.pay_rewards_to(backend).unwrap_or(Address::ZERO);
        if payment_for_rewards > U256::ZERO {
            // payment_for_rewards was clamped to l1_fees just above, which mirrors
            // the L1 pricer pool balance. A typed shortfall here would indicate
            // pool/state drift and must not block the rest of the bookkeeping.
            let _ = transfer_fn(
                L1_PRICER_FUNDS_POOL_ADDRESS,
                pay_rewards_to,
                payment_for_rewards,
            );
            l1_fees = l1_fees.saturating_sub(payment_for_rewards);
            self.set_l1_fees_available(backend, l1_fees)?;
        }

        let balance_due = poster_state.funds_due(backend).unwrap_or(U256::ZERO);
        let mut transfer_amount = balance_due;
        if l1_fees < transfer_amount {
            transfer_amount = l1_fees;
        }
        if transfer_amount > U256::ZERO {
            let addr_to_pay = poster_state.pay_to(backend).unwrap_or(batch_poster);
            // transfer_amount is capped to the remaining pool balance above; a
            // shortfall here would be a pool/state inconsistency rather than a
            // user-driven error, so do not surface it as Err.
            let _ = transfer_fn(L1_PRICER_FUNDS_POOL_ADDRESS, addr_to_pay, transfer_amount);
            l1_fees = l1_fees.saturating_sub(transfer_amount);
            self.set_l1_fees_available(backend, l1_fees)?;
            let _ = poster_state.set_funds_due(
                backend,
                balance_due.saturating_sub(transfer_amount),
                &bpt.total_funds_due,
            );
        }

        self.set_last_update_time(backend, update_time)?;

        if units_allocated > 0 {
            let total_funds_due = bpt.total_funds_due(backend).unwrap_or(U256::ZERO);
            let fdr = self.funds_due_for_rewards(backend).unwrap_or(U256::ZERO);

            let need_funds = total_funds_due.saturating_add(fdr);
            let (surplus_mag, surplus_positive) = if l1_fees >= need_funds {
                (l1_fees.saturating_sub(need_funds), true)
            } else {
                (need_funds.saturating_sub(l1_fees), false)
            };

            let inertia = self.inertia(backend).unwrap_or(INITIAL_INERTIA);
            let equil_units = self
                .equilibration_units(backend)
                .unwrap_or(U256::from(INITIAL_EQUILIBRATION_UNITS_V6));
            let inertia_units = equil_units
                .checked_div(U256::from(inertia))
                .unwrap_or(U256::ZERO);
            let price = self.price_per_unit(backend).unwrap_or(U256::ZERO);

            let alloc_plus_inert = inertia_units.saturating_add(U256::from(units_allocated));
            let (old_surplus_mag, old_surplus_neg) = self
                .last_surplus
                .get_signed(backend)
                .unwrap_or((U256::ZERO, false));

            let units_u256 = U256::from(units_allocated);

            let (desired_mag, desired_pos) =
                signed_div(surplus_mag, !surplus_positive, equil_units);

            let (diff_mag, diff_pos) = signed_sub(
                surplus_mag,
                surplus_positive,
                old_surplus_mag,
                !old_surplus_neg,
            );
            let (actual_mag, actual_pos) = signed_div(diff_mag, diff_pos, units_u256);

            let (change_mag, change_pos) =
                signed_sub(desired_mag, desired_pos, actual_mag, actual_pos);

            let change_times_units = change_mag.saturating_mul(units_u256);
            let (price_change, price_change_pos) =
                signed_div(change_times_units, change_pos, alloc_plus_inert);

            let new_price = if price_change_pos {
                price.saturating_add(price_change)
            } else {
                price.saturating_sub(price_change)
            };

            self.set_last_surplus(backend, surplus_mag, !surplus_positive)?;
            self.set_price_per_unit(backend, new_price)?;
        }

        Ok(())
    }
}

/// Euclidean division (remainder is always non-negative).
///
/// For a negative dividend with a positive divisor, this rounds toward negative
/// infinity rather than toward zero: -7 / 2 = -4 (not -3), -3 / 10 = -1 (not 0).
fn signed_div(mag: U256, positive: bool, divisor: U256) -> (U256, bool) {
    if divisor.is_zero() {
        return (U256::ZERO, true);
    }

    if positive {
        // Positive / positive: truncation and Euclidean are the same.
        return (mag / divisor, true);
    }

    // Negative dividend: Euclidean division (matching Go's big.Int.Div).
    // Go's big.Int.Div rounds toward negative infinity with non-negative remainder.
    // -7 / 2 = -4 (since -7 = 2*(-4) + 1, remainder 1 >= 0)
    let quotient = mag / divisor;
    let remainder = mag % divisor;
    if remainder.is_zero() {
        if quotient.is_zero() {
            (U256::ZERO, true) // -0 = +0
        } else {
            (quotient, false)
        }
    } else {
        // Non-zero remainder: round toward negative infinity.
        (quotient + U256::from(1), false)
    }
}

/// Signed subtraction: (a_mag, a_pos) - (b_mag, b_pos)
fn signed_sub(a_mag: U256, a_pos: bool, b_mag: U256, b_pos: bool) -> (U256, bool) {
    // a - b = a + (-b)
    let (neg_b_mag, neg_b_pos) = (b_mag, !b_pos);
    signed_add(a_mag, a_pos, neg_b_mag, neg_b_pos)
}

/// Signed addition: (a_mag, a_pos) + (b_mag, b_pos)
fn signed_add(a_mag: U256, a_pos: bool, b_mag: U256, b_pos: bool) -> (U256, bool) {
    if a_pos == b_pos {
        (a_mag.saturating_add(b_mag), a_pos)
    } else if a_mag >= b_mag {
        (a_mag.saturating_sub(b_mag), a_pos)
    } else {
        (b_mag.saturating_sub(a_mag), b_pos)
    }
}

/// Compute poster cost and calldata units from pre-loaded pricing parameters.
///
/// This is the standalone version used by the block executor which has already
/// extracted L1 pricing state values into the execution context.
pub fn compute_poster_cost_standalone(
    tx_bytes: &[u8],
    poster: Address,
    price_per_unit: U256,
    brotli_compression_level: u64,
) -> (U256, u64) {
    if poster != BATCH_POSTER_ADDRESS {
        return (U256::ZERO, 0);
    }
    let units = poster_units_from_bytes(tx_bytes, brotli_compression_level);
    (price_per_unit.saturating_mul(U256::from(units)), units)
}

/// Compute calldata units from tx bytes using brotli compression.
pub fn poster_units_from_bytes(tx_bytes: &[u8], brotli_compression_level: u64) -> u64 {
    let l1_bytes = byte_count_after_brotli_level(tx_bytes, brotli_compression_level);
    TX_DATA_NON_ZERO_GAS_EIP2028.saturating_mul(l1_bytes)
}

/// Brotli window size matching the reference C implementation.
const BROTLI_DEFAULT_WINDOW_SIZE: i32 = 22;

/// Computes the brotli-compressed size at a given compression level.
pub fn byte_count_after_brotli_level(data: &[u8], level: u64) -> u64 {
    use std::{ffi::c_int, os::raw::c_void, ptr};

    type BrotliBool = c_int;
    const BROTLI_PARAM_QUALITY: u32 = 1;
    const BROTLI_PARAM_LGWIN: u32 = 2;
    const BROTLI_OPERATION_FINISH: u32 = 2;

    extern "C" {
        fn BrotliEncoderCreateInstance(
            alloc: Option<extern "C" fn(*mut c_void, usize) -> *mut c_void>,
            free: Option<extern "C" fn(*mut c_void, *mut c_void)>,
            opaque: *mut c_void,
        ) -> *mut c_void;
        fn BrotliEncoderSetParameter(state: *mut c_void, param: u32, value: u32) -> BrotliBool;
        fn BrotliEncoderCompressStream(
            state: *mut c_void,
            op: u32,
            available_in: *mut usize,
            next_in: *mut *const u8,
            available_out: *mut usize,
            next_out: *mut *mut u8,
            total_out: *mut usize,
        ) -> BrotliBool;
        fn BrotliEncoderIsFinished(state: *const c_void) -> BrotliBool;
        fn BrotliEncoderDestroyInstance(state: *mut c_void);
        fn BrotliEncoderMaxCompressedSize(input_size: usize) -> usize;
    }

    // SAFETY: FFI into libbrotlienc. The encoder state is created,
    // configured, fed, then unconditionally destroyed in this block;
    // input and output buffers are stack/heap allocations whose lifetime
    // exceeds the encoder. Null state is checked before any use.
    unsafe {
        let state = BrotliEncoderCreateInstance(None, None, ptr::null_mut());
        if state.is_null() {
            return data.len() as u64;
        }

        BrotliEncoderSetParameter(state, BROTLI_PARAM_QUALITY, level.min(11) as u32);
        BrotliEncoderSetParameter(state, BROTLI_PARAM_LGWIN, BROTLI_DEFAULT_WINDOW_SIZE as u32);

        let max_size = BrotliEncoderMaxCompressedSize(data.len());
        let max_size = max_size.max(data.len() + (data.len() >> 10) * 8 + 64);
        let mut output = vec![0u8; max_size];

        let mut in_len = data.len();
        let mut in_ptr = data.as_ptr();
        let mut out_left = output.len();
        let mut out_ptr = output.as_mut_ptr();
        let mut out_len = 0usize;

        let ok = BrotliEncoderCompressStream(
            state,
            BROTLI_OPERATION_FINISH,
            &mut in_len,
            &mut in_ptr,
            &mut out_left,
            &mut out_ptr,
            &mut out_len,
        );
        let finished = BrotliEncoderIsFinished(state);
        BrotliEncoderDestroyInstance(state);

        if ok != 0 && finished != 0 {
            out_len as u64
        } else {
            data.len() as u64
        }
    }
}
