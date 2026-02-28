use alloy_primitives::U256;
use arb_primitives::multigas::{MultiGas, ResourceKind, NUM_RESOURCE_KIND};
use revm::Database;

use arb_chainspec::arbos_version as version;

use super::L2PricingState;

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
// StorageReadCost (SloadGasEIP2200 = 800) + StorageWriteCost (SstoreSetGasEIP2200 = 20000)
pub const MULTI_CONSTRAINT_STATIC_BACKLOG_UPDATE_COST: u64 = 20_800;

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

    /// Shrink the gas backlog for the active pricing model.
    pub fn shrink_backlog(
        &self,
        used_gas: u64,
        used_multi_gas: MultiGas,
    ) -> Result<(), ()> {
        self.update_backlog(BacklogOperation::Shrink, used_gas, used_multi_gas)
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
        let speed_limit = self.speed_limit_per_second()?;
        let drain = time_passed.saturating_mul(speed_limit);
        self.update_legacy_backlog_op(BacklogOperation::Shrink, drain)?;

        let inertia = self.pricing_inertia()?;
        let tolerance = self.backlog_tolerance()?;
        let backlog = self.gas_backlog()?;
        let min_base_fee = self.min_base_fee_wei()?;

        // Go uses plain `tolerance * speedLimit` (wrapping on overflow).
        // We match the wrapping behavior for consensus parity.
        let tolerance_limit = tolerance.wrapping_mul(speed_limit);
        let base_fee = if backlog > tolerance_limit {
            // Divisor: SaturatingUMul(inertia, speedLimit). If zero, Go panics.
            // Guard against division by zero (speed_limit/inertia are validated nonzero by ArbOwner).
            let divisor = saturating_cast_to_i64(inertia.saturating_mul(speed_limit));
            if divisor == 0 {
                return self.set_base_fee_wei(min_base_fee);
            }
            // Go: SaturatingCast[int64](backlog - tolerance*speedLimit)
            let excess = saturating_cast_to_i64(backlog.wrapping_sub(tolerance_limit));
            // Go: NaturalToBips(excess) / SaturatingCastToBips(SaturatingUMul(inertia, speedLimit))
            let exponent_bips = natural_to_bips(excess) / divisor;
            // Go: BigMulByBips(minBaseFee, ApproxExpBasisPoints(exponentBips, 4))
            self.calc_base_fee_from_exponent(exponent_bips.max(0) as u64)?
        } else {
            min_base_fee
        };

        self.set_base_fee_wei(base_fee)
    }

    fn update_pricing_model_single_constraints(&self, time_passed: u64) -> Result<(), ()> {
        // Drain backlogs and compute total exponent (sum across all constraints).
        // Uses signed Bips (int64) arithmetic matching Go.
        let mut total_exponent: i64 = 0;
        let len = self.gas_constraints_length()?;

        for i in 0..len {
            let c = self.open_gas_constraint_at(i);
            let target = c.target()?;

            // Pay off backlog: gas = SaturatingUMul(timePassed, target)
            let backlog = c.backlog()?;
            let gas = time_passed.saturating_mul(target);
            let new_backlog = backlog.saturating_sub(gas);
            c.set_backlog(new_backlog)?;

            // Calculate exponent with the formula backlog/divisor
            if new_backlog > 0 {
                let window = c.adjustment_window()?;
                // divisor = SaturatingCastToBips(SaturatingUMul(inertia, target))
                let divisor = saturating_cast_to_i64(window.saturating_mul(target));
                if divisor != 0 {
                    // NaturalToBips(SaturatingCast[int64](backlog))
                    let exponent = natural_to_bips(saturating_cast_to_i64(new_backlog)) / divisor;
                    total_exponent = total_exponent.saturating_add(exponent);
                }
            }
        }

        let base_fee = self.calc_base_fee_from_exponent(total_exponent.max(0) as u64)?;
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

    fn update_multi_gas_constraints_backlogs(&self, time_passed: u64) -> Result<(), ()> {
        let len = self.multi_gas_constraints_length()?;
        for i in 0..len {
            let c = self.open_multi_gas_constraint_at(i);
            let target = c.target()?;
            let backlog = c.backlog()?;
            let gas = time_passed.saturating_mul(target);
            let new_backlog = backlog.saturating_sub(gas);
            c.set_backlog(new_backlog)?;
        }
        Ok(())
    }

    /// Calculate exponent (in basis points) per resource kind across all constraints.
    ///
    /// Aggregates weighted backlog contributions from each constraint into
    /// a per-resource-kind exponent array.
    ///
    /// Uses signed saturation arithmetic matching Go's Bips (int64) computation:
    /// dividend = NaturalToBips(SaturatingCast[int64](SaturatingUMul(backlog, weight)))
    /// divisor  = SaturatingCastToBips(SaturatingUMul(window, SaturatingUMul(target, maxWeight)))
    /// exp      = dividend / divisor  (signed int64 division)
    pub fn calc_multi_gas_constraints_exponents(
        &self,
    ) -> Result<[u64; NUM_RESOURCE_KIND], ()> {
        let len = self.multi_gas_constraints_length()?;
        let mut exponent_per_kind = [0i64; NUM_RESOURCE_KIND];

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

            // divisor = SaturatingCastToBips(SaturatingUMul(window, SaturatingUMul(target, maxWeight)))
            let divisor_u64 = (window as u64)
                .saturating_mul(target.saturating_mul(max_weight));
            let divisor = saturating_cast_to_i64(divisor_u64);
            if divisor == 0 {
                continue;
            }

            for kind in ResourceKind::ALL {
                let weight = c.resource_weight(kind)?;
                if weight == 0 {
                    continue;
                }

                // dividend = NaturalToBips(SaturatingCast[int64](SaturatingUMul(backlog, weight)))
                let product = backlog.saturating_mul(weight);
                let cast = saturating_cast_to_i64(product);
                let dividend = natural_to_bips(cast);

                let exp = dividend / divisor;
                exponent_per_kind[kind as usize] =
                    exponent_per_kind[kind as usize].saturating_add(exp);
            }
        }

        // Convert back to u64 for the caller (exponents are always non-negative).
        let mut result = [0u64; NUM_RESOURCE_KIND];
        for i in 0..NUM_RESOURCE_KIND {
            result[i] = exponent_per_kind[i].max(0) as u64;
        }
        Ok(result)
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
    ///
    /// Version-gated cost accounting matching the Go implementation:
    /// - v60+: static cost (StorageReadCost + StorageWriteCost)
    /// - v51+: overhead for single-gas constraint traversal
    /// - v50+: base overhead for GasModelToUse() read
    /// - legacy: read + write for backlog
    pub fn backlog_update_cost(&self) -> Result<u64, ()> {
        use super::{STORAGE_READ_COST, STORAGE_WRITE_COST};

        // v60+: charge a flat static price regardless of gas model
        if self.arbos_version >= version::ARBOS_VERSION_60 {
            return Ok(MULTI_CONSTRAINT_STATIC_BACKLOG_UPDATE_COST);
        }

        let mut result = 0u64;

        // v50+: overhead for reading gas constraints length in GasModelToUse()
        if self.arbos_version >= version::ARBOS_VERSION_50 {
            result += STORAGE_READ_COST;
        }

        // v51+ (multi-constraint fix): per-constraint read+write costs
        if self.arbos_version >= version::ARBOS_VERSION_MULTI_CONSTRAINT_FIX {
            let constraints_length = self.gas_constraints_length()?;
            if constraints_length > 0 {
                // Read length to traverse
                result += STORAGE_READ_COST;
                // Read + write backlog for each constraint
                result += constraints_length * (STORAGE_READ_COST + STORAGE_WRITE_COST);
                return Ok(result);
            }
            // No return here -- fallthrough to legacy costs
        }

        // Legacy pricer: single read + write
        result += STORAGE_READ_COST + STORAGE_WRITE_COST;

        Ok(result)
    }

    /// Set gas constraints from legacy parameters (for upgrades).
    pub fn set_gas_constraints_from_legacy(&self) -> Result<(), ()> {
        self.clear_gas_constraints()?;
        let target = self.speed_limit_per_second()?;
        let adjustment_window = self.pricing_inertia()?;
        let old_backlog = self.gas_backlog()?;
        let backlog_tolerance = self.backlog_tolerance()?;

        let backlog = old_backlog.saturating_sub(
            backlog_tolerance.saturating_mul(target),
        );
        self.add_gas_constraint(target, adjustment_window, backlog)
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
            let weights = [1u64; NUM_RESOURCE_KIND];

            // Cap adjustment_window to u32::MAX (matching Go's uint32 type).
            let adjustment_window: u32 = if window > u32::MAX as u64 {
                u32::MAX
            } else {
                window as u32
            };

            self.add_multi_gas_constraint(target, adjustment_window, backlog, &weights)?;
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

/// Saturating cast from u64 to i64, capping at i64::MAX.
/// Matches Go's `SaturatingCast[int64](value uint64)`.
fn saturating_cast_to_i64(value: u64) -> i64 {
    if value > i64::MAX as u64 {
        i64::MAX
    } else {
        value as i64
    }
}

/// Convert a natural number to basis points (multiply by 10000), saturating.
/// Matches Go's `NaturalToBips(natural int64) Bips`.
fn natural_to_bips(natural: i64) -> i64 {
    natural.saturating_mul(10000)
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
