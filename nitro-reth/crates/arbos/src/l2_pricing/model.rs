use alloy_primitives::U256;
use arb_primitives::multigas::MultiGas;
use revm::Database;

use super::{L2PricingState, MAX_PRICING_EXPONENT_BIPS};

/// Which gas pricing model to use.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum GasModel {
    Unknown,
    Legacy,
    SingleGasConstraints,
    MultiGasConstraints,
}

/// Whether a backlog update grows or shrinks the backlog.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum BacklogOperation {
    Shrink,
    Grow,
}

// Initial constants for pricing model.
pub const MULTI_CONSTRAINT_STATIC_BACKLOG_UPDATE_COST: u64 = 500;

impl<D: Database> L2PricingState<D> {
    /// Determine which gas model to use based on stored constraint counts.
    pub fn gas_model_to_use(&self) -> Result<GasModel, ()> {
        let mgc_len = self.multi_gas_constraints_length()?;
        if mgc_len > 0 {
            return Ok(GasModel::MultiGasConstraints);
        }
        let gc_len = self.gas_constraints_length()?;
        if gc_len > 0 {
            return Ok(GasModel::SingleGasConstraints);
        }
        Ok(GasModel::Legacy)
    }

    /// Grow the gas backlog for the active pricing model.
    pub fn grow_backlog(
        &self,
        used_gas: u64,
        used_multi_gas: MultiGas,
    ) -> Result<(), ()> {
        self.update_backlog(BacklogOperation::Grow, used_gas, used_multi_gas)
    }

    /// Dispatch backlog update to the active pricing model.
    fn update_backlog(
        &self,
        op: BacklogOperation,
        used_gas: u64,
        used_multi_gas: MultiGas,
    ) -> Result<(), ()> {
        match self.gas_model_to_use()? {
            GasModel::Legacy | GasModel::Unknown => {
                self.update_legacy_backlog_op(op, used_gas)
            }
            GasModel::SingleGasConstraints => {
                self.update_single_gas_constraints_backlogs_op(op, used_gas)
            }
            GasModel::MultiGasConstraints => {
                self.update_multi_gas_constraints_backlogs_op(op, used_multi_gas)
            }
        }
    }

    fn update_legacy_backlog_op(&self, op: BacklogOperation, gas: u64) -> Result<(), ()> {
        let backlog = self.gas_backlog()?;
        self.set_gas_backlog(apply_gas_delta_op(op, backlog, gas))
    }

    fn update_single_gas_constraints_backlogs_op(
        &self,
        op: BacklogOperation,
        gas: u64,
    ) -> Result<(), ()> {
        let len = self.gas_constraints_length()?;
        for i in 0..len {
            let c = self.open_gas_constraint_at(i);
            let backlog = c.backlog()?;
            c.set_backlog(apply_gas_delta_op(op, backlog, gas))?;
        }
        Ok(())
    }

    fn update_multi_gas_constraints_backlogs_op(
        &self,
        op: BacklogOperation,
        multi_gas: MultiGas,
    ) -> Result<(), ()> {
        let len = self.multi_gas_constraints_length()?;
        for i in 0..len {
            let c = self.open_multi_gas_constraint_at(i);
            match op {
                BacklogOperation::Grow => c.grow_backlog(multi_gas)?,
                BacklogOperation::Shrink => c.shrink_backlog(multi_gas)?,
            }
        }
        Ok(())
    }

    /// Update the pricing model for a new block.
    pub fn update_pricing_model(
        &self,
        time_passed: u64,
        _arbos_version: u64,
    ) -> Result<(), ()> {
        match self.gas_model_to_use()? {
            GasModel::Legacy | GasModel::Unknown => {
                self.update_pricing_model_legacy(time_passed)
            }
            GasModel::SingleGasConstraints => {
                self.update_pricing_model_single_constraints(time_passed)
            }
            GasModel::MultiGasConstraints => {
                self.update_pricing_model_multi_constraints(time_passed)
            }
        }
    }

    fn update_pricing_model_legacy(&self, time_passed: u64) -> Result<(), ()> {
        self.update_legacy_backlog(time_passed)?;

        let inertia = self.pricing_inertia()?;
        let tolerance = self.backlog_tolerance()?;
        let speed_limit = self.speed_limit_per_second()?;
        let backlog = self.gas_backlog()?;
        let min_base_fee = self.min_base_fee_wei()?;

        if speed_limit == 0 || inertia == 0 {
            return Ok(());
        }

        let tolerance_limit = tolerance.saturating_mul(speed_limit);
        let base_fee = if backlog > tolerance_limit {
            let excess = backlog.saturating_sub(tolerance_limit);
            let exponent_bips =
                (excess as u128 * 10000) / (inertia as u128 * speed_limit as u128);
            self.calc_base_fee_from_exponent(exponent_bips.min(u64::MAX as u128) as u64)?
        } else {
            min_base_fee
        };

        self.set_base_fee_wei(base_fee)
    }

    fn update_pricing_model_single_constraints(&self, time_passed: u64) -> Result<(), ()> {
        self.update_single_gas_constraints_backlogs(time_passed)?;

        let mut max_exponent: u64 = 0;
        let len = self.gas_constraints_length()?;

        for i in 0..len {
            let c = self.open_gas_constraint_at(i);
            let target = c.target()?;
            let window = c.adjustment_window()?;
            let backlog = c.backlog()?;

            if target == 0 || window == 0 {
                continue;
            }

            let exponent = (backlog as u128 * 10000) / (window as u128 * target as u128);
            let exponent = exponent.min(u64::MAX as u128) as u64;
            if exponent > max_exponent {
                max_exponent = exponent;
            }
        }

        let base_fee = self.calc_base_fee_from_exponent(max_exponent)?;
        self.set_base_fee_wei(base_fee)
    }

    fn update_pricing_model_multi_constraints(&self, time_passed: u64) -> Result<(), ()> {
        self.update_multi_gas_constraints_backlogs(time_passed)?;

        let exponents = self.calc_multi_gas_constraints_exponents()?;
        let mut max_exponent: u64 = 0;
        for &e in &exponents {
            if e > max_exponent {
                max_exponent = e;
            }
        }

        let base_fee = self.calc_base_fee_from_exponent(max_exponent)?;
        self.set_base_fee_wei(base_fee)?;

        self.commit_multi_gas_fees(&exponents)
    }

    fn update_legacy_backlog(&self, time_passed: u64) -> Result<(), ()> {
        let speed_limit = self.speed_limit_per_second()?;
        let gas_to_drain = (time_passed as u128).saturating_mul(speed_limit as u128);
        let backlog = self.gas_backlog()?;
        let new_backlog = backlog.saturating_sub(gas_to_drain.min(u64::MAX as u128) as u64);
        self.set_gas_backlog(new_backlog)
    }

    fn update_single_gas_constraints_backlogs(&self, time_passed: u64) -> Result<(), ()> {
        let len = self.gas_constraints_length()?;
        for i in 0..len {
            let c = self.open_gas_constraint_at(i);
            let target = c.target()?;
            let drain = (time_passed as u128).saturating_mul(target as u128);
            let backlog = c.backlog()?;
            let new_backlog = backlog.saturating_sub(drain.min(u64::MAX as u128) as u64);
            c.set_backlog(new_backlog)?;
        }
        Ok(())
    }

    fn update_multi_gas_constraints_backlogs(&self, time_passed: u64) -> Result<(), ()> {
        let len = self.multi_gas_constraints_length()?;
        for i in 0..len {
            let c = self.open_multi_gas_constraint_at(i);
            let target = c.target()?;
            let drain = (time_passed as u128).saturating_mul(target as u128);
            let backlog = c.backlog()?;
            let new_backlog = backlog.saturating_sub(drain.min(u64::MAX as u128) as u64);
            c.set_backlog(new_backlog)?;
        }
        Ok(())
    }

    /// Calculate exponent (in basis points) for each multi-gas constraint.
    pub fn calc_multi_gas_constraints_exponents(&self) -> Result<Vec<u64>, ()> {
        let len = self.multi_gas_constraints_length()?;
        let mut exponents = Vec::with_capacity(len as usize);
        for i in 0..len {
            let c = self.open_multi_gas_constraint_at(i);
            let target = c.target()?;
            let window = c.adjustment_window()?;
            let backlog = c.backlog()?;

            let exponent = if target == 0 || window == 0 {
                0
            } else {
                let e = (backlog as u128 * 10000) / (window as u128 * target as u128);
                e.min(MAX_PRICING_EXPONENT_BIPS as u128) as u64
            };
            exponents.push(exponent);
        }
        Ok(exponents)
    }

    /// Calculate base fee from an exponent in basis points.
    /// base_fee = min_base_fee * exp(exponent_bips / 10000)
    pub fn calc_base_fee_from_exponent(&self, exponent_bips: u64) -> Result<U256, ()> {
        let min_base_fee = self.min_base_fee_wei()?;
        if exponent_bips == 0 {
            return Ok(min_base_fee);
        }

        let exp_result = approx_exp_basis_points(exponent_bips as u128);
        let base_fee = (min_base_fee * U256::from(exp_result)) / U256::from(10000u64);

        if base_fee < min_base_fee {
            Ok(min_base_fee)
        } else {
            Ok(base_fee)
        }
    }

    /// Get multi-gas base fee per resource kind.
    pub fn get_multi_gas_base_fee_per_resource(
        &self,
        exponents: &[u64],
    ) -> Result<Vec<U256>, ()> {
        let min_base_fee = self.min_base_fee_wei()?;
        let mut fees = Vec::with_capacity(exponents.len());
        for &e in exponents {
            if e == 0 {
                fees.push(min_base_fee);
            } else {
                let exp_result = approx_exp_basis_points(e as u128);
                let fee = (min_base_fee * U256::from(exp_result)) / U256::from(10000u64);
                fees.push(if fee < min_base_fee { min_base_fee } else { fee });
            }
        }
        Ok(fees)
    }

    /// Commit multi-gas fees for the next block.
    pub fn commit_multi_gas_fees(&self, _exponents: &[u64]) -> Result<(), ()> {
        // TODO: implement full multi-gas fee commitment logic
        Ok(())
    }

    /// Calculate the cost for a backlog update operation.
    pub fn backlog_update_cost(&self) -> Result<u64, ()> {
        match self.gas_model_to_use()? {
            GasModel::Legacy | GasModel::Unknown => Ok(0),
            GasModel::SingleGasConstraints => {
                let len = self.gas_constraints_length()?;
                Ok(len * 100)
            }
            GasModel::MultiGasConstraints => {
                let len = self.multi_gas_constraints_length()?;
                Ok(MULTI_CONSTRAINT_STATIC_BACKLOG_UPDATE_COST + len * 100)
            }
        }
    }

    /// Set gas constraints from legacy parameters (for upgrades).
    pub fn set_gas_constraints_from_legacy(&self) -> Result<(), ()> {
        self.clear_gas_constraints()?;
        let speed_limit = self.speed_limit_per_second()?;
        let inertia = self.pricing_inertia()?;
        let tolerance = self.backlog_tolerance()?;
        let backlog = self.gas_backlog()?;

        if speed_limit > 0 && inertia > 0 {
            let target = speed_limit;
            let window = inertia;
            self.add_gas_constraint(target, window)?;

            // Transfer existing backlog
            let c = self.open_gas_constraint_at(0);
            let tolerance_offset = tolerance.saturating_mul(speed_limit);
            let effective_backlog = backlog.saturating_sub(tolerance_offset);
            c.set_backlog(effective_backlog)?;
        }
        Ok(())
    }

    /// Convert single-gas constraints to multi-gas constraints (for upgrades).
    pub fn set_multi_gas_constraints_from_single_gas_constraints(&self) -> Result<(), ()> {
        // TODO: implement when multi-gas constraint upgrade logic is needed
        Ok(())
    }

    /// Multi-dimensional price for refund calculation.
    pub fn multi_dimensional_price_for_refund(&self) -> Result<U256, ()> {
        self.base_fee_wei()
    }
}

/// Approximate e^(x/10000) * 10000 using a Taylor series.
fn approx_exp_basis_points(bips: u128) -> u128 {
    if bips >= MAX_PRICING_EXPONENT_BIPS as u128 {
        return u128::MAX;
    }

    let mut result = 10000u128;
    let mut term = 10000u128;

    for i in 1..=20 {
        term = term.saturating_mul(bips) / (i * 10000);
        result = result.saturating_add(term);
        if term < 1 {
            break;
        }
    }

    result
}

/// Apply a gas delta to a backlog value (signed).
pub fn apply_gas_delta(backlog: u64, delta: i64) -> u64 {
    if delta > 0 {
        backlog.saturating_add(delta as u64)
    } else {
        backlog.saturating_sub((-delta) as u64)
    }
}

/// Apply a gas delta with a backlog operation.
fn apply_gas_delta_op(op: BacklogOperation, backlog: u64, delta: u64) -> u64 {
    match op {
        BacklogOperation::Grow => backlog.saturating_add(delta),
        BacklogOperation::Shrink => backlog.saturating_sub(delta),
    }
}
