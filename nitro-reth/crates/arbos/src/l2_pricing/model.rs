use alloy_primitives::U256;
use arb_primitives::multigas::{MultiGas, ResourceKind, NUM_RESOURCE_KIND};
use revm::Database;

use arb_chainspec::arbos_version as version;

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
    /// Determine which gas model to use based on ArbOS version and stored constraints.
    pub fn gas_model_to_use(&self) -> Result<GasModel, ()> {
        if self.arbos_version >= version::ARBOS_VERSION_60 {
            let mgc_len = self.multi_gas_constraints_length()?;
            if mgc_len > 0 {
                return Ok(GasModel::MultiGasConstraints);
            }
        }
        if self.arbos_version >= version::ARBOS_VERSION_50 {
            let gc_len = self.gas_constraints_length()?;
            if gc_len > 0 {
                return Ok(GasModel::SingleGasConstraints);
            }
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
        arbos_version: u64,
    ) -> Result<(), ()> {
        let _ = arbos_version; // version gating handled by gas_model_to_use via self.arbos_version
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
        // Drain backlogs and compute total exponent (sum across all constraints).
        let mut total_exponent: u64 = 0;
        let len = self.gas_constraints_length()?;

        for i in 0..len {
            let c = self.open_gas_constraint_at(i);
            let target = c.target()?;

            // Drain backlog.
            let backlog = c.backlog()?;
            let drain = (time_passed as u128).saturating_mul(target as u128);
            let new_backlog = backlog.saturating_sub(drain.min(u64::MAX as u128) as u64);
            c.set_backlog(new_backlog)?;

            if new_backlog == 0 {
                continue;
            }

            let window = c.adjustment_window()?;
            if target == 0 || window == 0 {
                continue;
            }

            let exponent = (new_backlog as u128 * 10000) / (window as u128 * target as u128);
            let exponent = exponent.min(u64::MAX as u128) as u64;
            total_exponent = total_exponent.saturating_add(exponent);
        }

        let base_fee = self.calc_base_fee_from_exponent(total_exponent)?;
        self.set_base_fee_wei(base_fee)
    }

    fn update_pricing_model_multi_constraints(&self, time_passed: u64) -> Result<(), ()> {
        self.update_multi_gas_constraints_backlogs(time_passed)?;

        let exponent_per_kind = self.calc_multi_gas_constraints_exponents()?;

        // Compute base fee per resource kind, store as next-block fee,
        // and track the maximum for the overall base fee.
        let mut max_base_fee = self.min_base_fee_wei()?;
        let fees = &self.multi_gas_base_fees;

        for (i, &exp) in exponent_per_kind.iter().enumerate() {
            let base_fee = self.calc_base_fee_from_exponent(exp)?;
            if let Some(kind) = ResourceKind::from_u8(i as u8) {
                let mgf = super::multi_gas_fees::open_multi_gas_fees(fees.clone());
                mgf.set_next_block_fee(kind, base_fee)?;
            }
            if base_fee > max_base_fee {
                max_base_fee = base_fee;
            }
        }

        self.set_base_fee_wei(max_base_fee)
    }

    fn update_legacy_backlog(&self, time_passed: u64) -> Result<(), ()> {
        let speed_limit = self.speed_limit_per_second()?;
        let gas_to_drain = (time_passed as u128).saturating_mul(speed_limit as u128);
        let backlog = self.gas_backlog()?;
        let new_backlog = backlog.saturating_sub(gas_to_drain.min(u64::MAX as u128) as u64);
        self.set_gas_backlog(new_backlog)
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

    /// Calculate exponent (in basis points) per resource kind across all constraints.
    ///
    /// Aggregates weighted backlog contributions from each constraint into
    /// a per-resource-kind exponent array.
    pub fn calc_multi_gas_constraints_exponents(
        &self,
    ) -> Result<[u64; NUM_RESOURCE_KIND], ()> {
        let len = self.multi_gas_constraints_length()?;
        let mut exponent_per_kind = [0u64; NUM_RESOURCE_KIND];

        for i in 0..len {
            let c = self.open_multi_gas_constraint_at(i);
            let target = c.target()?;
            let backlog = c.backlog()?;

            if backlog == 0 {
                continue;
            }

            let window = c.adjustment_window()?;
            let max_weight = c.max_weight()?;

            if target == 0 || window == 0 || max_weight == 0 {
                continue;
            }

            let divisor = (window as u128)
                .saturating_mul(target as u128)
                .saturating_mul(max_weight as u128);

            if divisor == 0 {
                continue;
            }

            for kind in ResourceKind::ALL {
                let weight = c.resource_weight(kind)?;
                if weight == 0 {
                    continue;
                }

                let dividend = (backlog as u128)
                    .saturating_mul(weight as u128)
                    .saturating_mul(10000);

                let exp = (dividend / divisor).min(MAX_PRICING_EXPONENT_BIPS as u128) as u64;
                exponent_per_kind[kind as usize] =
                    exponent_per_kind[kind as usize].saturating_add(exp);
            }
        }

        Ok(exponent_per_kind)
    }

    /// Calculate base fee from an exponent in basis points.
    /// base_fee = min_base_fee * exp(exponent_bips / 10000)
    pub fn calc_base_fee_from_exponent(&self, exponent_bips: u64) -> Result<U256, ()> {
        let min_base_fee = self.min_base_fee_wei()?;
        if exponent_bips == 0 {
            return Ok(min_base_fee);
        }

        let exp_result = approx_exp_basis_points(exponent_bips);
        let base_fee = (min_base_fee * U256::from(exp_result)) / U256::from(10000u64);

        if base_fee < min_base_fee {
            Ok(min_base_fee)
        } else {
            Ok(base_fee)
        }
    }

    /// Get multi-gas current-block base fee per resource kind.
    ///
    /// L1Calldata kind is always forced to the global base fee,
    /// and any zero fee is replaced with the global base fee.
    pub fn get_multi_gas_base_fee_per_resource(
        &self,
    ) -> Result<[U256; NUM_RESOURCE_KIND], ()> {
        let base_fee = self.base_fee_wei()?;
        let mgf = super::multi_gas_fees::open_multi_gas_fees(
            self.multi_gas_base_fees.clone(),
        );
        let mut fees = [U256::ZERO; NUM_RESOURCE_KIND];
        for kind in ResourceKind::ALL {
            // L1Calldata always uses the global base fee.
            if kind == ResourceKind::L1Calldata {
                fees[kind as usize] = base_fee;
                continue;
            }
            let fee = mgf.get_current_block_fee(kind)?;
            fees[kind as usize] = if fee.is_zero() { base_fee } else { fee };
        }
        Ok(fees)
    }

    /// Rotate next-block multi-gas fees into current-block fees.
    ///
    /// Called at block start before executing transactions.
    pub fn commit_multi_gas_fees(&self) -> Result<(), ()> {
        if self.gas_model_to_use()? != GasModel::MultiGasConstraints {
            return Ok(());
        }
        let mgf = super::multi_gas_fees::open_multi_gas_fees(
            self.multi_gas_base_fees.clone(),
        );
        mgf.commit_next_to_current()
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
    ///
    /// Iterates existing single-gas constraints, reads their target/window/backlog,
    /// and creates corresponding multi-gas constraints with equal weights across
    /// all resource dimensions.
    pub fn set_multi_gas_constraints_from_single_gas_constraints(&self) -> Result<(), ()> {
        self.clear_multi_gas_constraints()?;

        let length = self.gas_constraints_length()?;

        for i in 0..length {
            let c = self.open_gas_constraint_at(i);

            let target = c.target()?;
            let window = c.adjustment_window()?;
            let backlog = c.backlog()?;

            // Equal weights for all resource kinds.
            let mut weights = [1u64; NUM_RESOURCE_KIND];
            weights[0] = 1; // Unknown/default dimension also gets weight 1

            let adjustment_window = if window > u32::MAX as u64 {
                u32::MAX as u64
            } else {
                window
            };

            self.add_multi_gas_constraint(target, adjustment_window, &weights)?;

            // Transfer existing backlog to the new multi-gas constraint.
            let new_len = self.multi_gas_constraints_length()?;
            if new_len > 0 {
                let mc = self.open_multi_gas_constraint_at(new_len - 1);
                mc.set_backlog(backlog)?;
            }
        }
        Ok(())
    }

    /// Compute total cost for a multi-gas usage, for refund calculations.
    ///
    /// Returns `sum(gas_used[kind] * base_fee[kind])` across all resource kinds.
    pub fn multi_dimensional_price_for_refund(
        &self,
        gas_used: MultiGas,
    ) -> Result<U256, ()> {
        let fees = self.get_multi_gas_base_fee_per_resource()?;
        let mut total = U256::ZERO;
        for kind in ResourceKind::ALL {
            let amount = gas_used.get(kind);
            if amount == 0 {
                continue;
            }
            total = total.saturating_add(
                U256::from(amount).saturating_mul(fees[kind as usize]),
            );
        }
        Ok(total)
    }
}

/// Approximate e^(x/10000) * 10000 using Horner's method (degree 4).
///
/// Matches Go's `ApproxExpBasisPoints(value, 4)` exactly.
fn approx_exp_basis_points(bips: u64) -> u64 {
    const ACCURACY: u64 = 4;
    const B: u64 = 10_000; // OneInBips

    if bips == 0 {
        return B;
    }

    // Horner's method: b*(1 + x/b*(1 + x/(2b)*(1 + x/(3b))))
    let mut res = B.saturating_add(bips / ACCURACY);
    let mut i = ACCURACY - 1;
    while i > 0 {
        res = B.saturating_add(res.saturating_mul(bips) / (i * B));
        i -= 1;
    }

    res
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
