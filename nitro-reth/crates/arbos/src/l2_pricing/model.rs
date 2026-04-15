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
    pub fn grow_backlog(&self, used_gas: u64, used_multi_gas: MultiGas) -> Result<(), ()> {
        self.update_backlog(BacklogOperation::Grow, used_gas, used_multi_gas)
    }

    /// Shrink the gas backlog for the active pricing model.
    pub fn shrink_backlog(&self, used_gas: u64, used_multi_gas: MultiGas) -> Result<(), ()> {
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
            GasModel::Legacy | GasModel::Unknown => self.update_legacy_backlog_op(op, used_gas),
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
        let new_backlog = apply_gas_delta_op(op, backlog, gas);
        self.set_gas_backlog(new_backlog)
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
    pub fn update_pricing_model(&self, time_passed: u64, arbos_version: u64) -> Result<(), ()> {
        let _ = arbos_version; // version gating handled by gas_model_to_use via self.arbos_version
        match self.gas_model_to_use()? {
            GasModel::Legacy | GasModel::Unknown => self.update_pricing_model_legacy(time_passed),
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

        // Plain `tolerance * speedLimit` (wrapping on overflow).
        let tolerance_limit = tolerance.wrapping_mul(speed_limit);
        let base_fee = if backlog > tolerance_limit {
            // Divisor: SaturatingUMul(inertia, speedLimit).
            // Guard against division by zero (speed_limit/inertia are validated nonzero by
            // ArbOwner).
            let divisor = saturating_cast_to_i64(inertia.saturating_mul(speed_limit));
            if divisor == 0 {
                return self.set_base_fee_wei(min_base_fee);
            }
            // SaturatingCast]int64\](backlog - tolerance*speedLimit)
            let excess = saturating_cast_to_i64(backlog.wrapping_sub(tolerance_limit));
            // NaturalToBips(excess) / SaturatingCastToBips(SaturatingUMul(inertia, speedLimit))
            let exponent_bips = natural_to_bips(excess) / divisor;
            // BigMulByBips(minBaseFee, ApproxExpBasisPoints(exponentBips, 4))
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
                    // NaturalToBips(SaturatingCast]int64\](backlog))
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
    /// Uses signed saturation arithmetic with Bips (int64) computation:
    /// dividend = NaturalToBips(SaturatingCast]int64\](SaturatingUMul(backlog, weight)))
    /// divisor  = SaturatingCastToBips(SaturatingUMul(window, SaturatingUMul(target, maxWeight)))
    /// exp      = dividend / divisor  (signed int64 division)
    pub fn calc_multi_gas_constraints_exponents(&self) -> Result<[u64; NUM_RESOURCE_KIND], ()> {
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

            // divisor = SaturatingCastToBips(SaturatingUMul(window, SaturatingUMul(target,
            // maxWeight)))
            let divisor_u64 = (window as u64).saturating_mul(target.saturating_mul(max_weight));
            let divisor = saturating_cast_to_i64(divisor_u64);
            if divisor == 0 {
                continue;
            }

            for kind in ResourceKind::ALL {
                let weight = c.resource_weight(kind)?;
                if weight == 0 {
                    continue;
                }

                // dividend = NaturalToBips(SaturatingCast]int64\](SaturatingUMul(backlog,
                // weight)))
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
    pub fn get_multi_gas_base_fee_per_resource(&self) -> Result<[U256; NUM_RESOURCE_KIND], ()> {
        let base_fee = self.base_fee_wei()?;
        let mgf = super::multi_gas_fees::open_multi_gas_fees(self.multi_gas_base_fees.clone());
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
        let mgf = super::multi_gas_fees::open_multi_gas_fees(self.multi_gas_base_fees.clone());
        mgf.commit_next_to_current()
    }

    /// Calculate the cost for a backlog update operation.
    ///
    /// Version-gated cost accounting:
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

        let backlog = old_backlog.saturating_sub(backlog_tolerance.saturating_mul(target));
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

            // Cap adjustment_window to u32::MAX.
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
    pub fn multi_dimensional_price_for_refund(&self, gas_used: MultiGas) -> Result<U256, ()> {
        let fees = self.get_multi_gas_base_fee_per_resource()?;
        let mut total = U256::ZERO;
        for kind in ResourceKind::ALL {
            let amount = gas_used.get(kind);
            if amount == 0 {
                continue;
            }
            total = total.saturating_add(U256::from(amount).saturating_mul(fees[kind as usize]));
        }
        Ok(total)
    }
}

/// Approximate e^(x/10000) * 10000 using Horner's method (degree 4).
///
/// Matches `ApproxExpBasisPoints(value, 4)` exactly.
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
fn saturating_cast_to_i64(value: u64) -> i64 {
    if value > i64::MAX as u64 {
        i64::MAX
    } else {
        value as i64
    }
}

/// Convert a natural number to basis points (multiply by 10000), saturating.
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

#[cfg(test)]
mod tests {
    use alloy_primitives::{address, keccak256, Address, B256, U256};
    use arb_primitives::multigas::MultiGas;
    use arb_storage::Storage;
    use revm::{database::StateBuilder, Database};

    const ARBOS_STATE_ADDRESS: Address = address!("A4B05FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF");

    #[derive(Default)]
    struct EmptyDb;

    impl Database for EmptyDb {
        type Error = std::convert::Infallible;
        fn basic(
            &mut self,
            _address: Address,
        ) -> Result<Option<revm::state::AccountInfo>, Self::Error> {
            Ok(None)
        }
        fn code_by_hash(&mut self, _code_hash: B256) -> Result<revm::state::Bytecode, Self::Error> {
            Ok(revm::state::Bytecode::default())
        }
        fn storage(&mut self, _address: Address, _index: U256) -> Result<U256, Self::Error> {
            Ok(U256::ZERO)
        }
        fn block_hash(&mut self, _number: u64) -> Result<B256, Self::Error> {
            Ok(B256::ZERO)
        }
    }

    /// Create ArbOS account in the cache if it doesn't exist.
    fn ensure_cache_account(state: &mut revm::database::State<EmptyDb>, addr: Address) {
        use revm::database::{states::account_status::AccountStatus, PlainAccount};

        let _ = state.load_cache_account(addr);
        if let Some(cached) = state.cache.accounts.get_mut(&addr) {
            if cached.account.is_none() {
                cached.account = Some(PlainAccount {
                    info: revm::state::AccountInfo {
                        balance: U256::ZERO,
                        nonce: 0,
                        code_hash: keccak256([]),
                        code: None,
                        account_id: None,
                    },
                    storage: Default::default(),
                });
                cached.status = AccountStatus::InMemoryChange;
            }
        }
    }

    #[test]
    fn test_grow_backlog_through_l2_pricing_state() {
        let mut state = StateBuilder::new()
            .with_database(EmptyDb)
            .with_bundle_update()
            .build();

        // Ensure ArbOS account exists with nonce=1
        ensure_cache_account(&mut state, ARBOS_STATE_ADDRESS);
        arb_storage::set_account_nonce(&mut state, ARBOS_STATE_ADDRESS, 1);

        let state_ptr: *mut revm::database::State<EmptyDb> = &mut state;

        // Create L2 pricing storage (subspace [1] off root)
        let backing = Storage::new(state_ptr, B256::ZERO);
        let l2_sto = backing.open_sub_storage(&[1]);

        // Initialize L2 pricing state
        super::super::initialize_l2_pricing_state(&l2_sto);

        // Verify gasBacklog starts at 0
        let l2_pricing = super::super::open_l2_pricing_state(
            backing.open_sub_storage(&[1]),
            10, // ArbOS v10
        );
        let initial_backlog = l2_pricing.gas_backlog().unwrap();
        assert_eq!(initial_backlog, 0, "Initial gasBacklog should be 0");

        // Grow backlog by 100000 gas
        let result = l2_pricing.grow_backlog(100_000, MultiGas::default());
        assert!(result.is_ok(), "grow_backlog should succeed");

        // Verify gasBacklog is now 100000
        let after_grow = l2_pricing.gas_backlog().unwrap();
        assert_eq!(
            after_grow, 100_000,
            "gasBacklog should be 100000 after grow"
        );

        // Grow again by 50000
        let result = l2_pricing.grow_backlog(50_000, MultiGas::default());
        assert!(result.is_ok(), "second grow_backlog should succeed");

        let after_second_grow = l2_pricing.gas_backlog().unwrap();
        assert_eq!(
            after_second_grow, 150_000,
            "gasBacklog should be 150000 after second grow"
        );

        // Shrink by 30000
        let result = l2_pricing.shrink_backlog(30_000, MultiGas::default());
        assert!(result.is_ok(), "shrink_backlog should succeed");

        let after_shrink = l2_pricing.gas_backlog().unwrap();
        assert_eq!(
            after_shrink, 120_000,
            "gasBacklog should be 120000 after shrink"
        );

        // Verify bundle contains the gasBacklog change
        use revm::database::states::bundle_state::BundleRetention;
        state.merge_transitions(BundleRetention::Reverts);
        let bundle = state.take_bundle();

        let acct = bundle
            .state
            .get(&ARBOS_STATE_ADDRESS)
            .expect("ArbOS account should be in bundle");

        // The gasBacklog slot should be in the bundle storage
        // Compute the expected slot
        let l2_base = keccak256([1u8]); // open_sub_storage([1]) from root
        let gas_backlog_offset: u64 = 4;
        let slot = arb_storage::storage_key_map(l2_base.as_slice(), gas_backlog_offset);

        let bundle_slot = acct
            .storage
            .get(&slot)
            .expect("gasBacklog slot should be in bundle");
        assert_eq!(
            bundle_slot.present_value,
            U256::from(120_000u64),
            "Bundle should contain final gasBacklog value"
        );
    }

    /// A 3-tx block pattern that previously lost a `grow_backlog` write from
    /// the bundle: StartBlock with drain=0, then a SubmitRetryable, then an
    /// auto-redeem RetryTx. Starting backlog 552_756, after `grow_backlog(357_751)`
    /// must equal 910_507 in the bundle.
    #[test]
    fn grow_backlog_survives_submit_retryable_then_retry_tx_flow() {
        use alloy_primitives::map::HashMap;
        use revm::{database::states::bundle_state::BundleRetention, DatabaseCommit};

        // --- Compute the real ArbOS slot addresses ---
        let l2_base = keccak256([1u8]); // L2 pricing subspace key
        let gas_backlog_slot = arb_storage::storage_key_map(l2_base.as_slice(), 4);
        let speed_limit_slot = arb_storage::storage_key_map(l2_base.as_slice(), 0);
        let per_block_gas_limit_slot = arb_storage::storage_key_map(l2_base.as_slice(), 1);
        let base_fee_slot = arb_storage::storage_key_map(l2_base.as_slice(), 2);
        let min_base_fee_slot = arb_storage::storage_key_map(l2_base.as_slice(), 3);
        let pricing_inertia_slot = arb_storage::storage_key_map(l2_base.as_slice(), 5);
        let backlog_tolerance_slot = arb_storage::storage_key_map(l2_base.as_slice(), 6);

        // ArbOS version slot (offset 0 from root key = B256::ZERO)
        let version_slot = arb_storage::storage_key_map(&[], 0);

        // --- PreloadedDb: returns realistic pre-block values for ArbOS storage ---
        struct PreloadedDb {
            slots: HashMap<(Address, U256), U256>,
        }

        impl PreloadedDb {
            fn new() -> Self {
                Self {
                    slots: HashMap::default(),
                }
            }
            fn set(&mut self, addr: Address, slot: U256, val: U256) {
                self.slots.insert((addr, slot), val);
            }
        }

        impl Database for PreloadedDb {
            type Error = std::convert::Infallible;
            fn basic(
                &mut self,
                addr: Address,
            ) -> Result<Option<revm::state::AccountInfo>, Self::Error> {
                if addr == ARBOS_STATE_ADDRESS {
                    Ok(Some(revm::state::AccountInfo {
                        nonce: 1,
                        balance: U256::ZERO,
                        code_hash: keccak256([]),
                        code: None,
                        account_id: None,
                    }))
                } else {
                    Ok(None)
                }
            }
            fn code_by_hash(&mut self, _: B256) -> Result<revm::state::Bytecode, Self::Error> {
                Ok(revm::state::Bytecode::default())
            }
            fn storage(&mut self, addr: Address, index: U256) -> Result<U256, Self::Error> {
                Ok(self
                    .slots
                    .get(&(addr, index))
                    .copied()
                    .unwrap_or(U256::ZERO))
            }
            fn block_hash(&mut self, _: u64) -> Result<B256, Self::Error> {
                Ok(B256::ZERO)
            }
        }

        let arbos = ARBOS_STATE_ADDRESS;

        // Pre-block state for the scenario under test.
        let mut db = PreloadedDb::new();
        db.set(arbos, gas_backlog_slot, U256::from(552_756u64));
        db.set(arbos, speed_limit_slot, U256::from(7_000_000u64));
        db.set(arbos, per_block_gas_limit_slot, U256::from(32_000_000u64));
        db.set(arbos, base_fee_slot, U256::from(100_000_000u64));
        db.set(arbos, min_base_fee_slot, U256::from(100_000_000u64));
        db.set(arbos, pricing_inertia_slot, U256::from(102u64));
        db.set(arbos, backlog_tolerance_slot, U256::from(10u64));
        db.set(arbos, version_slot, U256::from(20u64)); // ArbOS v20

        let mut state = StateBuilder::new()
            .with_database(db)
            .with_bundle_update()
            .build();

        let state_ptr: *mut revm::database::State<PreloadedDb> = &mut state;

        // ================================================================
        // TX0: StartBlock internal transaction
        // ================================================================
        // update_pricing_model(time_passed=0) → drain=0 → no-op write
        {
            let backing = Storage::new(state_ptr, B256::ZERO);
            let l2_sto = backing.open_sub_storage(&[1]);
            let l2_pricing = super::super::open_l2_pricing_state(l2_sto, 20);

            // This should read gasBacklog=552756, drain 0, try to write 552756 → no-op
            let result = l2_pricing.update_pricing_model(0, 20);
            assert!(result.is_ok(), "update_pricing_model should succeed");

            // Verify gasBacklog is still readable as 552756
            let backlog = l2_pricing.gas_backlog().unwrap();
            assert_eq!(
                backlog, 552_756,
                "gasBacklog should be 552756 after no-op drain"
            );
        }

        // Commit empty EVM state for StartBlock (internal tx has no EVM changes)
        let empty_changes: HashMap<Address, revm::state::Account> = Default::default();
        state.commit(empty_changes);

        // ================================================================
        // TX1: SubmitRetryable — writes many ArbOS storage slots
        // ================================================================
        // Simulate retryable creation writing ~10 storage slots to ArbOS
        {
            // These are approximate retryable storage slots (different subspace)
            let retryable_base = keccak256([2u8]); // retryable subspace
            for i in 0u64..10 {
                let slot = arb_storage::storage_key_map(retryable_base.as_slice(), i);
                arb_storage::write_storage_at(
                    unsafe { &mut *state_ptr },
                    arbos,
                    slot,
                    U256::from(1000 + i),
                );
            }

            // Write scratch slots (poster_fee, retryable_id, redeemer)
            let scratch_slot_1 = arb_storage::storage_key_map(&[], 5); // approximate
            let scratch_slot_2 = arb_storage::storage_key_map(&[], 6);
            let scratch_slot_3 = arb_storage::storage_key_map(&[], 7);
            arb_storage::write_storage_at(
                unsafe { &mut *state_ptr },
                arbos,
                scratch_slot_1,
                U256::from(42),
            );
            arb_storage::write_storage_at(
                unsafe { &mut *state_ptr },
                arbos,
                scratch_slot_2,
                U256::from(43),
            );
            arb_storage::write_storage_at(
                unsafe { &mut *state_ptr },
                arbos,
                scratch_slot_3,
                U256::from(44),
            );
        }

        // Commit empty EVM state for SubmitRetryable (endTxNow=true, no EVM execution)
        let empty_changes2: HashMap<Address, revm::state::Account> = Default::default();
        state.commit(empty_changes2);

        // Clear scratch slots (as done in commit_transaction)
        {
            let scratch_slot_1 = arb_storage::storage_key_map(&[], 5);
            let scratch_slot_2 = arb_storage::storage_key_map(&[], 6);
            let scratch_slot_3 = arb_storage::storage_key_map(&[], 7);
            arb_storage::write_arbos_storage(
                unsafe { &mut *state_ptr },
                scratch_slot_1,
                U256::ZERO,
            );
            arb_storage::write_arbos_storage(
                unsafe { &mut *state_ptr },
                scratch_slot_2,
                U256::ZERO,
            );
            arb_storage::write_arbos_storage(
                unsafe { &mut *state_ptr },
                scratch_slot_3,
                U256::ZERO,
            );
        }

        // ================================================================
        // TX2: RetryTx — complex EVM commit, then grow_backlog
        // ================================================================

        // Write scratch slots for RetryTx
        {
            let scratch_slot_1 = arb_storage::storage_key_map(&[], 5);
            let scratch_slot_2 = arb_storage::storage_key_map(&[], 6);
            let scratch_slot_3 = arb_storage::storage_key_map(&[], 7);
            arb_storage::write_storage_at(
                unsafe { &mut *state_ptr },
                arbos,
                scratch_slot_1,
                U256::from(99),
            );
            arb_storage::write_storage_at(
                unsafe { &mut *state_ptr },
                arbos,
                scratch_slot_2,
                U256::from(100),
            );
            arb_storage::write_storage_at(
                unsafe { &mut *state_ptr },
                arbos,
                scratch_slot_3,
                U256::from(101),
            );
        }

        // Simulate an EVM transaction that touches many accounts and emits
        // many logs — the regression needed ~11 logs across 7+ contracts to
        // reproduce.
        {
            let mut evm_changes: HashMap<Address, revm::state::Account> = Default::default();

            // Sender account
            let sender = address!("fd86e9a33fd52e4085fb94d24b759448a621cd36");
            let _ = state.load_cache_account(sender);
            let mut sender_acct = revm::state::Account::default();
            sender_acct.info.balance = U256::from(1_000_000_000u64);
            sender_acct.info.nonce = 1;
            sender_acct.mark_touch();
            evm_changes.insert(sender, sender_acct);

            // Target contract + 6 sub-contracts (simulating 7 accounts from 11 logs)
            let contracts = [
                address!("4453d0eaf066a61c9b81ddc18bb5a2bf2fc52224"),
                address!("7c7db13e5d385bcc797422d3c767856d15d24c5c"),
                address!("0057892cb8bb5f1ce1b3c6f5ade899732249713f"),
                address!("35aa95ac4747d928e2cd42fe4461f6d9d1826346"),
                address!("e1e3b1cbacc870cb6e5f4bdf246feb6eb5cd351b"),
                address!("7348fdf6f3e090c635b23d970945093455214f3b"),
                address!("d50e4a971bc8ed55af6aebc0a2178456069e87b5"),
            ];

            for (i, &contract) in contracts.iter().enumerate() {
                let _ = state.load_cache_account(contract);
                let mut acct = revm::state::Account::default();
                acct.info.nonce = 1;
                acct.info.code_hash = keccak256(format!("code_{}", i).as_bytes());
                acct.mark_touch();
                // Add some storage changes to simulate real contract execution
                for j in 0u64..3 {
                    let slot = U256::from(j);
                    let mut evm_slot =
                        revm::state::EvmStorageSlot::new(U256::from(i as u64 * 100 + j), 0);
                    evm_slot.present_value = U256::from(i as u64 * 100 + j + 1);
                    acct.storage.insert(slot, evm_slot);
                }
                evm_changes.insert(contract, acct);
            }

            state.commit(evm_changes);
        }

        // Clear scratch slots
        {
            let scratch_slot_1 = arb_storage::storage_key_map(&[], 5);
            let scratch_slot_2 = arb_storage::storage_key_map(&[], 6);
            let scratch_slot_3 = arb_storage::storage_key_map(&[], 7);
            arb_storage::write_arbos_storage(
                unsafe { &mut *state_ptr },
                scratch_slot_1,
                U256::ZERO,
            );
            arb_storage::write_arbos_storage(
                unsafe { &mut *state_ptr },
                scratch_slot_2,
                U256::ZERO,
            );
            arb_storage::write_arbos_storage(
                unsafe { &mut *state_ptr },
                scratch_slot_3,
                U256::ZERO,
            );
        }

        // Delete retryable: clears the retryable storage slots
        {
            let retryable_base = keccak256([2u8]);
            for i in 0u64..10 {
                let slot = arb_storage::storage_key_map(retryable_base.as_slice(), i);
                arb_storage::write_storage_at(unsafe { &mut *state_ptr }, arbos, slot, U256::ZERO);
            }
        }

        // === THE CRITICAL OPERATION: grow_backlog ===
        {
            let backing = Storage::new(state_ptr, B256::ZERO);
            let l2_sto = backing.open_sub_storage(&[1]);
            let l2_pricing = super::super::open_l2_pricing_state(l2_sto, 20);

            let backlog_before = l2_pricing.gas_backlog().unwrap();
            assert_eq!(
                backlog_before, 552_756,
                "gasBacklog should still be 552756 before grow"
            );

            let result = l2_pricing.grow_backlog(357_751, MultiGas::default());
            assert!(result.is_ok(), "grow_backlog should succeed");

            let backlog_after = l2_pricing.gas_backlog().unwrap();
            assert_eq!(
                backlog_after, 910_507,
                "gasBacklog should be 910507 after grow"
            );
        }

        // ================================================================
        // Post-block: merge transitions and verify bundle
        // ================================================================
        state.merge_transitions(BundleRetention::Reverts);
        let mut bundle = state.take_bundle();

        // --- Check 1: Is gasBacklog in the bundle BEFORE filtering? ---
        let pre_filter_backlog = bundle
            .state
            .get(&arbos)
            .and_then(|a| a.storage.get(&gas_backlog_slot))
            .map(|s| s.present_value);
        assert_eq!(
            pre_filter_backlog,
            Some(U256::from(910_507u64)),
            "gasBacklog should be in bundle before filter with value 910507"
        );

        // --- Simulate filter_unchanged_storage (inline, since it's private in producer.rs) ---
        for (_addr, account) in bundle.state.iter_mut() {
            account
                .storage
                .retain(|_key, slot| slot.present_value != slot.previous_or_original_value);
        }

        // --- Check 2: Is gasBacklog in the bundle AFTER filtering? ---
        let post_filter_backlog = bundle
            .state
            .get(&arbos)
            .and_then(|a| a.storage.get(&gas_backlog_slot))
            .map(|s| s.present_value);
        assert_eq!(
            post_filter_backlog,
            Some(U256::from(910_507u64)),
            "gasBacklog should survive filter_unchanged_storage with value 910507"
        );

        // --- Check 3: Verify the original value is correct ---
        let original = bundle
            .state
            .get(&arbos)
            .and_then(|a| a.storage.get(&gas_backlog_slot))
            .map(|s| s.previous_or_original_value);
        assert_eq!(
            original,
            Some(U256::from(552_756u64)),
            "gasBacklog original should be the pre-block DB value 552756"
        );

        // --- Check 4: Simulate augment_bundle_from_cache (simplified) ---
        // In production, augment_bundle_from_cache runs BEFORE filter.
        // But let's verify the cache has the right value too.
        let cache_backlog = state
            .cache
            .accounts
            .get(&arbos)
            .and_then(|ca| ca.account.as_ref())
            .and_then(|a| a.storage.get(&gas_backlog_slot).copied());
        assert_eq!(
            cache_backlog,
            Some(U256::from(910_507u64)),
            "gasBacklog should be in cache with value 910507"
        );
    }

    /// Same flow as above but the EVM commit touches the ArbOS account
    /// directly, simulating a precompile SLOAD during the RetryTx.
    #[test]
    fn grow_backlog_with_arbos_in_evm_commit() {
        use alloy_primitives::map::HashMap;
        use revm::{database::states::bundle_state::BundleRetention, DatabaseCommit};

        let l2_base = keccak256([1u8]);
        let gas_backlog_slot = arb_storage::storage_key_map(l2_base.as_slice(), 4);
        let speed_limit_slot = arb_storage::storage_key_map(l2_base.as_slice(), 0);
        let base_fee_slot = arb_storage::storage_key_map(l2_base.as_slice(), 2);
        let min_base_fee_slot = arb_storage::storage_key_map(l2_base.as_slice(), 3);
        let pricing_inertia_slot = arb_storage::storage_key_map(l2_base.as_slice(), 5);
        let backlog_tolerance_slot = arb_storage::storage_key_map(l2_base.as_slice(), 6);
        let version_slot = arb_storage::storage_key_map(&[], 0);
        // Scratch slots
        let scratch_1 = arb_storage::storage_key_map(&[], 5);
        let scratch_2 = arb_storage::storage_key_map(&[], 6);

        struct PreloadedDb {
            slots: HashMap<(Address, U256), U256>,
        }
        impl PreloadedDb {
            fn new() -> Self {
                Self {
                    slots: HashMap::default(),
                }
            }
            fn set(&mut self, addr: Address, slot: U256, val: U256) {
                self.slots.insert((addr, slot), val);
            }
        }
        impl Database for PreloadedDb {
            type Error = std::convert::Infallible;
            fn basic(
                &mut self,
                addr: Address,
            ) -> Result<Option<revm::state::AccountInfo>, Self::Error> {
                if addr == ARBOS_STATE_ADDRESS {
                    Ok(Some(revm::state::AccountInfo {
                        nonce: 1,
                        balance: U256::ZERO,
                        code_hash: keccak256([]),
                        code: None,
                        account_id: None,
                    }))
                } else {
                    Ok(None)
                }
            }
            fn code_by_hash(&mut self, _: B256) -> Result<revm::state::Bytecode, Self::Error> {
                Ok(revm::state::Bytecode::default())
            }
            fn storage(&mut self, addr: Address, index: U256) -> Result<U256, Self::Error> {
                Ok(self
                    .slots
                    .get(&(addr, index))
                    .copied()
                    .unwrap_or(U256::ZERO))
            }
            fn block_hash(&mut self, _: u64) -> Result<B256, Self::Error> {
                Ok(B256::ZERO)
            }
        }

        let arbos = ARBOS_STATE_ADDRESS;
        let mut db = PreloadedDb::new();
        db.set(arbos, gas_backlog_slot, U256::from(552_756u64));
        db.set(arbos, speed_limit_slot, U256::from(7_000_000u64));
        db.set(arbos, base_fee_slot, U256::from(100_000_000u64));
        db.set(arbos, min_base_fee_slot, U256::from(100_000_000u64));
        db.set(arbos, pricing_inertia_slot, U256::from(102u64));
        db.set(arbos, backlog_tolerance_slot, U256::from(10u64));
        db.set(arbos, version_slot, U256::from(20u64));

        let mut state = StateBuilder::new()
            .with_database(db)
            .with_bundle_update()
            .build();
        let state_ptr: *mut revm::database::State<PreloadedDb> = &mut state;

        // TX0: StartBlock (no-op drain)
        {
            let backing = Storage::new(state_ptr, B256::ZERO);
            let l2_sto = backing.open_sub_storage(&[1]);
            let l2_pricing = super::super::open_l2_pricing_state(l2_sto, 20);
            let _ = l2_pricing.update_pricing_model(0, 20);
        }
        state.commit(HashMap::default());

        // TX1: SubmitRetryable — write scratch slots + retryable storage
        {
            arb_storage::write_storage_at(
                unsafe { &mut *state_ptr },
                arbos,
                scratch_1,
                U256::from(42),
            );
            arb_storage::write_storage_at(
                unsafe { &mut *state_ptr },
                arbos,
                scratch_2,
                U256::from(43),
            );
            let retryable_base = keccak256([2u8]);
            for i in 0u64..5 {
                let slot = arb_storage::storage_key_map(retryable_base.as_slice(), i);
                arb_storage::write_storage_at(
                    unsafe { &mut *state_ptr },
                    arbos,
                    slot,
                    U256::from(1000 + i),
                );
            }
        }
        state.commit(HashMap::default());
        // Clear scratch
        arb_storage::write_arbos_storage(unsafe { &mut *state_ptr }, scratch_1, U256::ZERO);
        arb_storage::write_arbos_storage(unsafe { &mut *state_ptr }, scratch_2, U256::ZERO);

        // TX2: RetryTx — write scratch, then EVM commit WITH ArbOS account
        arb_storage::write_storage_at(unsafe { &mut *state_ptr }, arbos, scratch_1, U256::from(99));
        arb_storage::write_storage_at(
            unsafe { &mut *state_ptr },
            arbos,
            scratch_2,
            U256::from(100),
        );

        // EVM commit that INCLUDES the ArbOS account (the critical difference!)
        {
            let mut evm_changes: HashMap<Address, revm::state::Account> = Default::default();

            // Sender
            let sender = address!("fd86e9a33fd52e4085fb94d24b759448a621cd36");
            let _ = state.load_cache_account(sender);
            let mut sender_acct = revm::state::Account::default();
            sender_acct.info.balance = U256::from(1_000_000_000u64);
            sender_acct.info.nonce = 1;
            sender_acct.mark_touch();
            evm_changes.insert(sender, sender_acct);

            // ArbOS account IN the EVM commit — simulates a precompile/SLOAD
            // that caused the EVM to track the ArbOS account
            let _ = state.load_cache_account(arbos);
            let mut arbos_acct = revm::state::Account {
                info: revm::state::AccountInfo {
                    nonce: 1,
                    balance: U256::ZERO,
                    code_hash: keccak256([]),
                    code: None,
                    account_id: None,
                },
                ..Default::default()
            };
            // The EVM "read" the scratch slot — it appears in the EVM's storage
            // with is_changed=false (just loaded, not modified)
            arbos_acct.storage.insert(
                scratch_1,
                revm::state::EvmStorageSlot::new(U256::from(99), 0),
            );
            arbos_acct.mark_touch();
            evm_changes.insert(arbos, arbos_acct);

            state.commit(evm_changes);
        }

        // Check: is gasBacklog still readable?
        let backlog_check =
            arb_storage::read_storage_at(unsafe { &mut *state_ptr }, arbos, gas_backlog_slot);

        // Clear scratch
        arb_storage::write_arbos_storage(unsafe { &mut *state_ptr }, scratch_1, U256::ZERO);
        arb_storage::write_arbos_storage(unsafe { &mut *state_ptr }, scratch_2, U256::ZERO);

        // Delete retryable
        {
            let retryable_base = keccak256([2u8]);
            for i in 0u64..5 {
                let slot = arb_storage::storage_key_map(retryable_base.as_slice(), i);
                arb_storage::write_storage_at(unsafe { &mut *state_ptr }, arbos, slot, U256::ZERO);
            }
        }

        // THE CRITICAL OPERATION: grow_backlog
        {
            let backing = Storage::new(state_ptr, B256::ZERO);
            let l2_sto = backing.open_sub_storage(&[1]);
            let l2_pricing = super::super::open_l2_pricing_state(l2_sto, 20);

            let backlog_before = l2_pricing.gas_backlog().unwrap();
            assert_eq!(
                backlog_before, 552_756,
                "gasBacklog should be 552756 before grow"
            );

            let _ = l2_pricing.grow_backlog(357_751, MultiGas::default());

            let backlog_after = l2_pricing.gas_backlog().unwrap();
            assert_eq!(backlog_after, 910_507, "gasBacklog should be 910507");
        }

        // Verify bundle
        state.merge_transitions(BundleRetention::Reverts);
        let mut bundle = state.take_bundle();

        let pre_filter = bundle
            .state
            .get(&arbos)
            .and_then(|a| a.storage.get(&gas_backlog_slot))
            .map(|s| (s.present_value, s.previous_or_original_value));
        assert_eq!(
            pre_filter.map(|p| p.0),
            Some(U256::from(910_507u64)),
            "gasBacklog should be 910507 in bundle before filter"
        );

        // filter_unchanged_storage
        for (_addr, account) in bundle.state.iter_mut() {
            account
                .storage
                .retain(|_key, slot| slot.present_value != slot.previous_or_original_value);
        }

        let post_filter = bundle
            .state
            .get(&arbos)
            .and_then(|a| a.storage.get(&gas_backlog_slot))
            .map(|s| s.present_value);
        assert_eq!(
            post_filter,
            Some(U256::from(910_507u64)),
            "gasBacklog should survive filter when ArbOS is in EVM commit"
        );
    }

    /// When `transition_state` is None (already consumed by a prior
    /// `merge_transitions`), later `write_storage_at` calls must still end up
    /// in the bundle rather than being silently dropped.
    #[test]
    fn grow_backlog_with_transition_state_consumed() {
        use alloy_primitives::map::HashMap;
        use revm::{
            database::states::{bundle_state::BundleRetention, plain_account::StorageSlot},
            DatabaseCommit,
        };

        let l2_base = keccak256([1u8]);
        let gas_backlog_slot = arb_storage::storage_key_map(l2_base.as_slice(), 4);
        let speed_limit_slot = arb_storage::storage_key_map(l2_base.as_slice(), 0);
        let base_fee_slot = arb_storage::storage_key_map(l2_base.as_slice(), 2);
        let min_base_fee_slot = arb_storage::storage_key_map(l2_base.as_slice(), 3);
        let pricing_inertia_slot = arb_storage::storage_key_map(l2_base.as_slice(), 5);
        let backlog_tolerance_slot = arb_storage::storage_key_map(l2_base.as_slice(), 6);
        let version_slot = arb_storage::storage_key_map(&[], 0);

        struct PreloadedDb(HashMap<(Address, U256), U256>);
        impl PreloadedDb {
            fn new() -> Self {
                Self(HashMap::default())
            }
            fn set(&mut self, a: Address, s: U256, v: U256) {
                self.0.insert((a, s), v);
            }
        }
        impl Database for PreloadedDb {
            type Error = std::convert::Infallible;
            fn basic(
                &mut self,
                addr: Address,
            ) -> Result<Option<revm::state::AccountInfo>, Self::Error> {
                if addr == ARBOS_STATE_ADDRESS {
                    Ok(Some(revm::state::AccountInfo {
                        nonce: 1,
                        balance: U256::ZERO,
                        code_hash: keccak256([]),
                        code: None,
                        account_id: None,
                    }))
                } else {
                    Ok(None)
                }
            }
            fn code_by_hash(&mut self, _: B256) -> Result<revm::state::Bytecode, Self::Error> {
                Ok(revm::state::Bytecode::default())
            }
            fn storage(&mut self, a: Address, i: U256) -> Result<U256, Self::Error> {
                Ok(self.0.get(&(a, i)).copied().unwrap_or(U256::ZERO))
            }
            fn block_hash(&mut self, _: u64) -> Result<B256, Self::Error> {
                Ok(B256::ZERO)
            }
        }

        let arbos = ARBOS_STATE_ADDRESS;
        let mut db = PreloadedDb::new();
        db.set(arbos, gas_backlog_slot, U256::from(552_756u64));
        db.set(arbos, speed_limit_slot, U256::from(7_000_000u64));
        db.set(arbos, base_fee_slot, U256::from(100_000_000u64));
        db.set(arbos, min_base_fee_slot, U256::from(100_000_000u64));
        db.set(arbos, pricing_inertia_slot, U256::from(102u64));
        db.set(arbos, backlog_tolerance_slot, U256::from(10u64));
        db.set(arbos, version_slot, U256::from(20u64));

        let mut state = StateBuilder::new()
            .with_database(db)
            .with_bundle_update()
            .build();
        let state_ptr: *mut revm::database::State<PreloadedDb> = &mut state;

        // TX0: StartBlock (no-op drain)
        {
            let backing = Storage::new(state_ptr, B256::ZERO);
            let l2_sto = backing.open_sub_storage(&[1]);
            let l2_pricing = super::super::open_l2_pricing_state(l2_sto, 20);
            let _ = l2_pricing.update_pricing_model(0, 20);
        }
        state.commit(HashMap::default());

        // === SIMULATE BUG: merge_transitions called mid-block ===
        // This consumes transition_state, setting it to None.
        // All subsequent write_storage_at calls will have their
        // transitions SILENTLY DROPPED.
        state.merge_transitions(BundleRetention::Reverts);
        let _mid_bundle = state.take_bundle();

        // Check: is transition_state None?
        let ts_is_none = state.transition_state.is_none();

        // TX2: grow_backlog — the write goes to cache but transition is dropped
        {
            let backing = Storage::new(state_ptr, B256::ZERO);
            let l2_sto = backing.open_sub_storage(&[1]);
            let l2_pricing = super::super::open_l2_pricing_state(l2_sto, 20);

            let backlog_before = l2_pricing.gas_backlog().unwrap();

            let _ = l2_pricing.grow_backlog(357_751, MultiGas::default());

            let backlog_after = l2_pricing.gas_backlog().unwrap();
        }

        // End of block: merge_transitions again
        state.merge_transitions(BundleRetention::Reverts);
        let mut bundle = state.take_bundle();

        // Check: is gasBacklog in the bundle?
        let in_bundle = bundle
            .state
            .get(&arbos)
            .and_then(|a| a.storage.get(&gas_backlog_slot))
            .map(|s| s.present_value);

        // If transition_state was None, the gasBacklog transition was dropped.
        // The bundle from the 2nd merge would NOT have the gasBacklog.
        // Now simulate augment_bundle_from_cache which should rescue it from cache.
        {
            let cache_val = state
                .cache
                .accounts
                .get(&arbos)
                .and_then(|ca| ca.account.as_ref())
                .and_then(|a| a.storage.get(&gas_backlog_slot).copied());

            // Inline augment_bundle_from_cache for ArbOS account
            if let Some(bundle_acct) = bundle.state.get_mut(&arbos) {
                if let Some(cached_acc) = state.cache.accounts.get(&arbos) {
                    if let Some(ref plain) = cached_acc.account {
                        for (key, value) in &plain.storage {
                            if let Some(slot) = bundle_acct.storage.get_mut(key) {
                                slot.present_value = *value;
                            } else {
                                let original =
                                    state.database.storage(arbos, *key).unwrap_or(U256::ZERO);
                                if *value != original {
                                    bundle_acct.storage.insert(
                                        *key,
                                        StorageSlot {
                                            previous_or_original_value: original,
                                            present_value: *value,
                                        },
                                    );
                                }
                            }
                        }
                    }
                }
            } else {
                // ArbOS not in bundle — add it from cache
                if let Some(cached_acc) = state.cache.accounts.get(&arbos) {
                    if let Some(ref plain) = cached_acc.account {
                        let mut storage_changes: HashMap<U256, StorageSlot> = HashMap::default();
                        for (key, value) in &plain.storage {
                            let original =
                                state.database.storage(arbos, *key).unwrap_or(U256::ZERO);
                            if *value != original {
                                storage_changes.insert(
                                    *key,
                                    StorageSlot {
                                        previous_or_original_value: original,
                                        present_value: *value,
                                    },
                                );
                            }
                        }
                        if !storage_changes.is_empty() {
                            bundle.state.insert(
                                arbos,
                                revm::database::BundleAccount {
                                    info: Some(plain.info.clone()),
                                    original_info: None,
                                    storage: storage_changes,
                                    status: revm::database::AccountStatus::Changed,
                                },
                            );
                        }
                    }
                }
            }
        }

        let after_augment = bundle
            .state
            .get(&arbos)
            .and_then(|a| a.storage.get(&gas_backlog_slot))
            .map(|s| s.present_value);

        // filter_unchanged_storage
        for (_addr, account) in bundle.state.iter_mut() {
            account
                .storage
                .retain(|_key, slot| slot.present_value != slot.previous_or_original_value);
        }

        let after_filter = bundle
            .state
            .get(&arbos)
            .and_then(|a| a.storage.get(&gas_backlog_slot))
            .map(|s| s.present_value);

        assert_eq!(
            after_filter,
            Some(U256::from(910_507u64)),
            "gasBacklog MUST survive even when transition_state was consumed mid-block"
        );
    }

    /// Simulates the full production flow step-by-step to find why
    /// gas_backlog writes are lost. Tests the interaction between:
    /// - ArbOS storage writes (via Storage/write_storage_at)
    /// - EVM state commits (state.commit)
    /// - Bundle construction (merge_transitions + take_bundle)
    /// - Post-bundle augmentation (augment_bundle_from_cache logic)
    /// - Storage filtering (filter_unchanged_storage logic)
    #[test]
    fn test_grow_backlog_survives_evm_commit_and_augment() {
        use revm::{
            database::states::{bundle_state::BundleRetention, plain_account::StorageSlot},
            DatabaseCommit,
        };

        // Compute the actual gasBacklog slot for assertions
        let l2_base = keccak256([1u8]); // open_sub_storage([1]) from root
        let gas_backlog_offset: u64 = 4;
        let gas_backlog_slot = arb_storage::storage_key_map(l2_base.as_slice(), gas_backlog_offset);

        // ===== VARIANT A: EVM commit with EMPTY HashMap (no ArbOS account touched) =====
        {
            let mut state = StateBuilder::new()
                .with_database(EmptyDb)
                .with_bundle_update()
                .build();

            // Step 1: Ensure ArbOS account exists with nonce=1
            ensure_cache_account(&mut state, ARBOS_STATE_ADDRESS);
            arb_storage::set_account_nonce(&mut state, ARBOS_STATE_ADDRESS, 1);

            let state_ptr: *mut revm::database::State<EmptyDb> = &mut state;

            // Step 2: Initialize L2 pricing state
            let backing = Storage::new(state_ptr, B256::ZERO);
            let l2_sto = backing.open_sub_storage(&[1]);
            super::super::initialize_l2_pricing_state(&l2_sto);

            // Step 3: Set gas_backlog to 552756 (simulate pre-existing backlog)
            let l2_pricing =
                super::super::open_l2_pricing_state(backing.open_sub_storage(&[1]), 10);
            l2_pricing.set_gas_backlog(552756).unwrap();
            let pre_start = l2_pricing.gas_backlog().unwrap();
            assert_eq!(pre_start, 552756, "Pre-existing backlog should be 552756");

            // Step 4: Simulate StartBlock: update_pricing_model(time_passed=0)
            l2_pricing.update_pricing_model(0, 10).unwrap();
            let after_start = l2_pricing.gas_backlog().unwrap();
            assert_eq!(
                after_start, 552756,
                "time_passed=0 should not change backlog"
            );

            // Step 5: EVM commit with empty HashMap
            let empty_state: alloy_primitives::map::HashMap<Address, revm::state::Account> =
                Default::default();
            state.commit(empty_state);

            // Check cache after commit
            let cache_val = state
                .cache
                .accounts
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|ca| ca.account.as_ref())
                .and_then(|a| a.storage.get(&gas_backlog_slot).copied());

            // Step 6: grow_backlog(357751)
            let l2_pricing2 =
                super::super::open_l2_pricing_state(backing.open_sub_storage(&[1]), 10);
            l2_pricing2
                .grow_backlog(357751, MultiGas::default())
                .unwrap();
            let after_grow = l2_pricing2.gas_backlog().unwrap();
            assert_eq!(after_grow, 552756 + 357751, "backlog should be sum");

            // Check cache after grow
            let cache_val2 = state
                .cache
                .accounts
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|ca| ca.account.as_ref())
                .and_then(|a| a.storage.get(&gas_backlog_slot).copied());

            // Step 7: merge_transitions + take_bundle
            state.merge_transitions(BundleRetention::Reverts);
            let mut bundle = state.take_bundle();

            // Check bundle before augment
            let bundle_has_slot = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| s.present_value);

            // Step 8: Simulate augment_bundle_from_cache (inline replication)
            // In production, this is called on the same state after take_bundle
            for (addr, cache_acct) in &state.cache.accounts {
                let current_info = cache_acct.account.as_ref().map(|a| a.info.clone());
                let current_storage = cache_acct
                    .account
                    .as_ref()
                    .map(|a| &a.storage)
                    .cloned()
                    .unwrap_or_default();

                if let Some(bundle_acct) = bundle.state.get_mut(addr) {
                    bundle_acct.info = current_info;
                    for (key, value) in &current_storage {
                        if let Some(slot) = bundle_acct.storage.get_mut(key) {
                            slot.present_value = *value;
                        } else {
                            // Slot from cache not in bundle: compare with DB original (0 for
                            // EmptyDb)
                            let original_value = U256::ZERO;
                            if *value != original_value {
                                bundle_acct.storage.insert(
                                    *key,
                                    StorageSlot {
                                        previous_or_original_value: original_value,
                                        present_value: *value,
                                    },
                                );
                            }
                        }
                    }
                } else {
                    // Account not in bundle — add if changed
                    let storage_changes: alloy_primitives::map::HashMap<U256, StorageSlot> =
                        current_storage
                            .iter()
                            .filter_map(|(key, value)| {
                                let original_value = U256::ZERO;
                                if original_value != *value {
                                    Some((
                                        *key,
                                        StorageSlot {
                                            previous_or_original_value: original_value,
                                            present_value: *value,
                                        },
                                    ))
                                } else {
                                    None
                                }
                            })
                            .collect();

                    let info_changed = current_info.is_some(); // was None in DB
                    if info_changed || !storage_changes.is_empty() {
                        bundle.state.insert(
                            *addr,
                            revm::database::BundleAccount {
                                info: current_info,
                                original_info: None,
                                storage: storage_changes,
                                status: revm::database::AccountStatus::InMemoryChange,
                            },
                        );
                    }
                }
            }

            let bundle_after_augment = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| (s.present_value, s.previous_or_original_value));

            // Step 9: filter_unchanged_storage
            for (_addr, account) in bundle.state.iter_mut() {
                account
                    .storage
                    .retain(|_key, slot| slot.present_value != slot.previous_or_original_value);
            }

            let final_slot = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| s.present_value);
            assert!(
                final_slot.is_some(),
                "VARIANT A FAILED: gas_backlog slot MISSING from bundle after empty EVM commit"
            );
            assert_eq!(
                final_slot.unwrap(),
                U256::from(552756u64 + 357751u64),
                "VARIANT A: gas_backlog should be 910507"
            );
        }

        // ===== VARIANT B: EVM commit WITH ArbOS account touched (simulates precompile read) =====
        {
            let mut state = StateBuilder::new()
                .with_database(EmptyDb)
                .with_bundle_update()
                .build();

            ensure_cache_account(&mut state, ARBOS_STATE_ADDRESS);
            arb_storage::set_account_nonce(&mut state, ARBOS_STATE_ADDRESS, 1);

            let state_ptr: *mut revm::database::State<EmptyDb> = &mut state;

            let backing = Storage::new(state_ptr, B256::ZERO);
            let l2_sto = backing.open_sub_storage(&[1]);
            super::super::initialize_l2_pricing_state(&l2_sto);

            let l2_pricing =
                super::super::open_l2_pricing_state(backing.open_sub_storage(&[1]), 10);
            l2_pricing.set_gas_backlog(552756).unwrap();
            l2_pricing.update_pricing_model(0, 10).unwrap();
            let before_commit = l2_pricing.gas_backlog().unwrap();

            // Count cache slots before commit
            let cache_slots_before = state
                .cache
                .accounts
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|ca| ca.account.as_ref())
                .map(|a| a.storage.len())
                .unwrap_or(0);

            // EVM commit with ArbOS account TOUCHED but no storage changes
            // This simulates what happens when EVM executes a precompile that
            // reads ArbOS state — the account appears in the EVM output with
            // is_touched=true but storage unchanged.
            let _ = state.load_cache_account(ARBOS_STATE_ADDRESS);
            let mut arbos_evm_account = revm::state::Account {
                info: revm::state::AccountInfo {
                    balance: U256::ZERO,
                    nonce: 1,
                    code_hash: keccak256([]),
                    code: None,
                    account_id: None,
                },
                ..Default::default()
            };
            arbos_evm_account.mark_touch();
            // No storage entries — EVM read slots but didn't write them
            let mut evm_changes: alloy_primitives::map::HashMap<Address, revm::state::Account> =
                Default::default();
            evm_changes.insert(ARBOS_STATE_ADDRESS, arbos_evm_account);
            state.commit(evm_changes);

            // Check cache after commit — this is the critical check!
            let cache_slots_after = state
                .cache
                .accounts
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|ca| ca.account.as_ref())
                .map(|a| a.storage.len())
                .unwrap_or(0);

            let cache_val = state
                .cache
                .accounts
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|ca| ca.account.as_ref())
                .and_then(|a| a.storage.get(&gas_backlog_slot).copied());

            if cache_slots_after < cache_slots_before {}

            // Now grow_backlog AFTER the EVM commit
            let l2_pricing2 =
                super::super::open_l2_pricing_state(backing.open_sub_storage(&[1]), 10);
            let read_before_grow = l2_pricing2.gas_backlog().unwrap();

            l2_pricing2
                .grow_backlog(357751, MultiGas::default())
                .unwrap();
            let after_grow = l2_pricing2.gas_backlog().unwrap();

            // Check cache after grow
            let cache_val2 = state
                .cache
                .accounts
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|ca| ca.account.as_ref())
                .and_then(|a| a.storage.get(&gas_backlog_slot).copied());

            // merge + take_bundle
            state.merge_transitions(BundleRetention::Reverts);
            let mut bundle = state.take_bundle();

            let bundle_pre = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| (s.present_value, s.previous_or_original_value));

            // augment_bundle_from_cache (inline)
            for (addr, cache_acct) in &state.cache.accounts {
                let current_info = cache_acct.account.as_ref().map(|a| a.info.clone());
                let current_storage = cache_acct
                    .account
                    .as_ref()
                    .map(|a| &a.storage)
                    .cloned()
                    .unwrap_or_default();

                if let Some(bundle_acct) = bundle.state.get_mut(addr) {
                    bundle_acct.info = current_info;
                    for (key, value) in &current_storage {
                        if let Some(slot) = bundle_acct.storage.get_mut(key) {
                            slot.present_value = *value;
                        } else {
                            let original_value = U256::ZERO;
                            if *value != original_value {
                                bundle_acct.storage.insert(
                                    *key,
                                    StorageSlot {
                                        previous_or_original_value: original_value,
                                        present_value: *value,
                                    },
                                );
                            }
                        }
                    }
                }
            }

            let bundle_post = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| (s.present_value, s.previous_or_original_value));

            // filter_unchanged_storage
            for (_addr, account) in bundle.state.iter_mut() {
                account
                    .storage
                    .retain(|_key, slot| slot.present_value != slot.previous_or_original_value);
            }

            let final_slot = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| s.present_value);
            assert!(
                final_slot.is_some(),
                "VARIANT B FAILED: gas_backlog slot MISSING from bundle after EVM commit with ArbOS touched"
            );
            assert_eq!(
                final_slot.unwrap(),
                U256::from(552756u64 + 357751u64),
                "VARIANT B: gas_backlog should be 910507"
            );
        }

        // ===== VARIANT C: EVM commit WITH ArbOS account AND storage slot that was read =====
        // This simulates the most realistic case: EVM reads gasBacklog slot during
        // execution (e.g., GetPricesInWei precompile reads ArbOS state), and the
        // slot appears in EVM output with is_changed()=false but present in Account.storage
        {
            let mut state = StateBuilder::new()
                .with_database(EmptyDb)
                .with_bundle_update()
                .build();

            ensure_cache_account(&mut state, ARBOS_STATE_ADDRESS);
            arb_storage::set_account_nonce(&mut state, ARBOS_STATE_ADDRESS, 1);

            let state_ptr: *mut revm::database::State<EmptyDb> = &mut state;

            let backing = Storage::new(state_ptr, B256::ZERO);
            let l2_sto = backing.open_sub_storage(&[1]);
            super::super::initialize_l2_pricing_state(&l2_sto);

            let l2_pricing =
                super::super::open_l2_pricing_state(backing.open_sub_storage(&[1]), 10);
            l2_pricing.set_gas_backlog(552756).unwrap();
            l2_pricing.update_pricing_model(0, 10).unwrap();

            let cache_slots_before = state
                .cache
                .accounts
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|ca| ca.account.as_ref())
                .map(|a| a.storage.len())
                .unwrap_or(0);

            // EVM commit with ArbOS account touched AND a storage slot that was
            // read but not written (EvmStorageSlot with original_value == present_value).
            let _ = state.load_cache_account(ARBOS_STATE_ADDRESS);
            let mut arbos_evm_account = revm::state::Account {
                info: revm::state::AccountInfo {
                    balance: U256::ZERO,
                    nonce: 1,
                    code_hash: keccak256([]),
                    code: None,
                    account_id: None,
                },
                ..Default::default()
            };
            arbos_evm_account.mark_touch();

            // Add gas_backlog slot as READ-ONLY (original == present, is_changed()=false)
            // This is what happens when the EVM loads a storage slot via SLOAD
            arbos_evm_account.storage.insert(
                gas_backlog_slot,
                revm::state::EvmStorageSlot::new(U256::from(552756u64), 0),
                // new() sets original_value = present_value, so is_changed() = false
            );

            let mut evm_changes: alloy_primitives::map::HashMap<Address, revm::state::Account> =
                Default::default();
            evm_changes.insert(ARBOS_STATE_ADDRESS, arbos_evm_account);
            state.commit(evm_changes);

            let cache_slots_after = state
                .cache
                .accounts
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|ca| ca.account.as_ref())
                .map(|a| a.storage.len())
                .unwrap_or(0);

            let cache_val = state
                .cache
                .accounts
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|ca| ca.account.as_ref())
                .and_then(|a| a.storage.get(&gas_backlog_slot).copied());

            if cache_slots_after < cache_slots_before {}

            // grow_backlog after commit
            let l2_pricing2 =
                super::super::open_l2_pricing_state(backing.open_sub_storage(&[1]), 10);
            let read_before_grow = l2_pricing2.gas_backlog().unwrap();

            l2_pricing2
                .grow_backlog(357751, MultiGas::default())
                .unwrap();
            let after_grow = l2_pricing2.gas_backlog().unwrap();

            // merge + take_bundle
            state.merge_transitions(BundleRetention::Reverts);
            let mut bundle = state.take_bundle();

            let bundle_pre = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| (s.present_value, s.previous_or_original_value));

            // augment (inline)
            for (addr, cache_acct) in &state.cache.accounts {
                let current_info = cache_acct.account.as_ref().map(|a| a.info.clone());
                let current_storage = cache_acct
                    .account
                    .as_ref()
                    .map(|a| &a.storage)
                    .cloned()
                    .unwrap_or_default();

                if let Some(bundle_acct) = bundle.state.get_mut(addr) {
                    bundle_acct.info = current_info;
                    for (key, value) in &current_storage {
                        if let Some(slot) = bundle_acct.storage.get_mut(key) {
                            slot.present_value = *value;
                        } else {
                            let original_value = U256::ZERO;
                            if *value != original_value {
                                bundle_acct.storage.insert(
                                    *key,
                                    StorageSlot {
                                        previous_or_original_value: original_value,
                                        present_value: *value,
                                    },
                                );
                            }
                        }
                    }
                }
            }

            let bundle_post = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| (s.present_value, s.previous_or_original_value));

            // filter_unchanged_storage
            for (_addr, account) in bundle.state.iter_mut() {
                account
                    .storage
                    .retain(|_key, slot| slot.present_value != slot.previous_or_original_value);
            }

            let final_slot = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| s.present_value);
            assert!(
                final_slot.is_some(),
                "VARIANT C FAILED: gas_backlog slot MISSING from bundle after EVM commit with ArbOS storage read"
            );
            assert_eq!(
                final_slot.unwrap(),
                U256::from(552756u64 + 357751u64),
                "VARIANT C: gas_backlog should be 910507"
            );
        }

        // ===== VARIANT D: Two EVM commits (StartBlock + user tx) then grow_backlog =====
        // Most realistic production sequence
        {
            let mut state = StateBuilder::new()
                .with_database(EmptyDb)
                .with_bundle_update()
                .build();

            ensure_cache_account(&mut state, ARBOS_STATE_ADDRESS);
            arb_storage::set_account_nonce(&mut state, ARBOS_STATE_ADDRESS, 1);

            let state_ptr: *mut revm::database::State<EmptyDb> = &mut state;

            let backing = Storage::new(state_ptr, B256::ZERO);
            let l2_sto = backing.open_sub_storage(&[1]);
            super::super::initialize_l2_pricing_state(&l2_sto);

            // Set initial backlog
            let l2_pricing =
                super::super::open_l2_pricing_state(backing.open_sub_storage(&[1]), 10);
            l2_pricing.set_gas_backlog(552756).unwrap();

            // Simulate StartBlock: update_pricing_model writes base_fee
            l2_pricing.update_pricing_model(0, 10).unwrap();

            // First EVM commit (StartBlock internal tx - empty output)
            state.commit(Default::default());

            // Second EVM commit (user tx - touches sender + receiver, NOT ArbOS)
            let sender = address!("1111111111111111111111111111111111111111");
            let receiver = address!("2222222222222222222222222222222222222222");
            let _ = state.load_cache_account(sender);
            let _ = state.load_cache_account(receiver);

            let mut user_changes: alloy_primitives::map::HashMap<Address, revm::state::Account> =
                Default::default();
            let mut sender_acct = revm::state::Account::default();
            sender_acct.info.balance = U256::from(999_000u64);
            sender_acct.info.nonce = 1;
            sender_acct.mark_touch();
            user_changes.insert(sender, sender_acct);

            let mut receiver_acct = revm::state::Account::default();
            receiver_acct.info.balance = U256::from(1_000u64);
            receiver_acct.mark_touch();
            user_changes.insert(receiver, receiver_acct);

            state.commit(user_changes);

            // Check cache
            let cache_val_after_user = state
                .cache
                .accounts
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|ca| ca.account.as_ref())
                .and_then(|a| a.storage.get(&gas_backlog_slot).copied());

            // Post-commit: grow_backlog (this is what happens in production after
            // commit_transaction)
            let l2_pricing2 =
                super::super::open_l2_pricing_state(backing.open_sub_storage(&[1]), 10);
            let read_val = l2_pricing2.gas_backlog().unwrap();
            l2_pricing2
                .grow_backlog(357751, MultiGas::default())
                .unwrap();
            let after_grow = l2_pricing2.gas_backlog().unwrap();
            assert_eq!(after_grow, 552756 + 357751, "backlog should be sum");

            // merge + take_bundle
            state.merge_transitions(BundleRetention::Reverts);
            let mut bundle = state.take_bundle();

            let bundle_pre = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| (s.present_value, s.previous_or_original_value));

            // augment (inline)
            for (addr, cache_acct) in &state.cache.accounts {
                let current_info = cache_acct.account.as_ref().map(|a| a.info.clone());
                let current_storage = cache_acct
                    .account
                    .as_ref()
                    .map(|a| &a.storage)
                    .cloned()
                    .unwrap_or_default();

                if let Some(bundle_acct) = bundle.state.get_mut(addr) {
                    bundle_acct.info = current_info;
                    for (key, value) in &current_storage {
                        if let Some(slot) = bundle_acct.storage.get_mut(key) {
                            slot.present_value = *value;
                        } else {
                            let original_value = U256::ZERO;
                            if *value != original_value {
                                bundle_acct.storage.insert(
                                    *key,
                                    StorageSlot {
                                        previous_or_original_value: original_value,
                                        present_value: *value,
                                    },
                                );
                            }
                        }
                    }
                } else {
                    let storage_changes: alloy_primitives::map::HashMap<U256, StorageSlot> =
                        current_storage
                            .iter()
                            .filter_map(|(key, value)| {
                                let original_value = U256::ZERO;
                                if original_value != *value {
                                    Some((
                                        *key,
                                        StorageSlot {
                                            previous_or_original_value: original_value,
                                            present_value: *value,
                                        },
                                    ))
                                } else {
                                    None
                                }
                            })
                            .collect();
                    let info_changed = current_info.is_some();
                    if info_changed || !storage_changes.is_empty() {
                        bundle.state.insert(
                            *addr,
                            revm::database::BundleAccount {
                                info: current_info,
                                original_info: None,
                                storage: storage_changes,
                                status: revm::database::AccountStatus::InMemoryChange,
                            },
                        );
                    }
                }
            }

            let bundle_post = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| (s.present_value, s.previous_or_original_value));

            // filter
            for (_addr, account) in bundle.state.iter_mut() {
                account
                    .storage
                    .retain(|_key, slot| slot.present_value != slot.previous_or_original_value);
            }

            let final_slot = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| s.present_value);
            assert!(
                final_slot.is_some(),
                "VARIANT D FAILED: gas_backlog slot MISSING from bundle"
            );
            assert_eq!(
                final_slot.unwrap(),
                U256::from(552756u64 + 357751u64),
                "VARIANT D: gas_backlog should be 910507"
            );
        }

        // ===== VARIANT E: Database with PRE-EXISTING gas_backlog (production scenario) =====
        // In production, the state provider has the previous block's gas_backlog.
        // write_storage_at reads original_value from DB. If the DB already has the
        // value, the transition's original_value matches, and filter_unchanged_storage
        // may remove it.
        {
            // Create a DB that returns the pre-existing backlog value
            struct PrePopulatedDb {
                gas_backlog_slot: U256,
                pre_existing_backlog: U256,
            }

            impl Database for PrePopulatedDb {
                type Error = std::convert::Infallible;
                fn basic(
                    &mut self,
                    _address: Address,
                ) -> Result<Option<revm::state::AccountInfo>, Self::Error> {
                    // ArbOS account exists with nonce=1
                    Ok(Some(revm::state::AccountInfo {
                        balance: U256::ZERO,
                        nonce: 1,
                        code_hash: keccak256([]),
                        code: None,
                        account_id: None,
                    }))
                }
                fn code_by_hash(
                    &mut self,
                    _code_hash: B256,
                ) -> Result<revm::state::Bytecode, Self::Error> {
                    Ok(revm::state::Bytecode::default())
                }
                fn storage(&mut self, _address: Address, index: U256) -> Result<U256, Self::Error> {
                    // Return pre-existing backlog for the gas_backlog slot
                    if index == self.gas_backlog_slot {
                        Ok(self.pre_existing_backlog)
                    } else {
                        Ok(U256::ZERO)
                    }
                }
                fn block_hash(&mut self, _number: u64) -> Result<B256, Self::Error> {
                    Ok(B256::ZERO)
                }
            }

            let pre_existing_backlog = U256::from(552756u64);
            let mut state = StateBuilder::new()
                .with_database(PrePopulatedDb {
                    gas_backlog_slot,
                    pre_existing_backlog,
                })
                .with_bundle_update()
                .build();

            // Load ArbOS account from DB (nonce=1 already in DB)
            let _ = state.load_cache_account(ARBOS_STATE_ADDRESS);

            let state_ptr: *mut revm::database::State<PrePopulatedDb> = &mut state;

            // Open L2 pricing state — gas_backlog already in DB as 552756
            let backing = Storage::new(state_ptr, B256::ZERO);
            let l2_pricing =
                super::super::open_l2_pricing_state(backing.open_sub_storage(&[1]), 10);

            // Read current backlog — should come from DB
            let current = l2_pricing.gas_backlog().unwrap();
            assert_eq!(current, 552756, "Should read from DB");

            // Simulate StartBlock: update_pricing_model(time_passed=0)
            l2_pricing.update_pricing_model(0, 10).unwrap();
            let after_start = l2_pricing.gas_backlog().unwrap();

            // EVM commit: empty (StartBlock internal tx)
            {
                use revm::DatabaseCommit;
                let empty: alloy_primitives::map::HashMap<Address, revm::state::Account> =
                    Default::default();
                state.commit(empty);
            }

            // EVM commit: user tx touching only sender/receiver (NOT ArbOS)
            {
                use revm::DatabaseCommit;
                let sender = address!("1111111111111111111111111111111111111111");
                let _ = state.load_cache_account(sender);
                let mut user_changes: alloy_primitives::map::HashMap<
                    Address,
                    revm::state::Account,
                > = Default::default();
                let mut sender_acct = revm::state::Account::default();
                sender_acct.info.balance = U256::from(999_000u64);
                sender_acct.info.nonce = 1;
                sender_acct.mark_touch();
                user_changes.insert(sender, sender_acct);
                state.commit(user_changes);
            }

            // Post-commit: grow_backlog
            let l2_pricing2 =
                super::super::open_l2_pricing_state(backing.open_sub_storage(&[1]), 10);
            let read_before = l2_pricing2.gas_backlog().unwrap();

            l2_pricing2
                .grow_backlog(357751, MultiGas::default())
                .unwrap();
            let after_grow = l2_pricing2.gas_backlog().unwrap();

            // Check cache
            let cache_val = state
                .cache
                .accounts
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|ca| ca.account.as_ref())
                .and_then(|a| a.storage.get(&gas_backlog_slot).copied());

            // merge + take_bundle
            state.merge_transitions(BundleRetention::Reverts);
            let mut bundle = state.take_bundle();

            let bundle_pre = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| (s.present_value, s.previous_or_original_value));

            // augment (inline) — for PrePopulatedDb, original values come from DB
            for (addr, cache_acct) in &state.cache.accounts {
                let current_info = cache_acct.account.as_ref().map(|a| a.info.clone());
                let current_storage = cache_acct
                    .account
                    .as_ref()
                    .map(|a| &a.storage)
                    .cloned()
                    .unwrap_or_default();

                if let Some(bundle_acct) = bundle.state.get_mut(addr) {
                    bundle_acct.info = current_info;
                    for (key, value) in &current_storage {
                        if let Some(slot) = bundle_acct.storage.get_mut(key) {
                            slot.present_value = *value;
                        } else {
                            // Use pre_existing_backlog for DB lookup simulation
                            let original_value =
                                if *addr == ARBOS_STATE_ADDRESS && *key == gas_backlog_slot {
                                    pre_existing_backlog
                                } else {
                                    U256::ZERO
                                };
                            if *value != original_value {
                                bundle_acct.storage.insert(
                                    *key,
                                    StorageSlot {
                                        previous_or_original_value: original_value,
                                        present_value: *value,
                                    },
                                );
                            }
                        }
                    }
                } else {
                    // Account not in bundle
                    let storage_changes: alloy_primitives::map::HashMap<U256, StorageSlot> =
                        current_storage
                            .iter()
                            .filter_map(|(key, value)| {
                                let original_value =
                                    if *addr == ARBOS_STATE_ADDRESS && *key == gas_backlog_slot {
                                        pre_existing_backlog
                                    } else {
                                        U256::ZERO
                                    };
                                if original_value != *value {
                                    Some((
                                        *key,
                                        StorageSlot {
                                            previous_or_original_value: original_value,
                                            present_value: *value,
                                        },
                                    ))
                                } else {
                                    None
                                }
                            })
                            .collect();
                    let info_changed = false; // account existed in DB
                    if info_changed || !storage_changes.is_empty() {
                        bundle.state.insert(
                            *addr,
                            revm::database::BundleAccount {
                                info: current_info,
                                original_info: None,
                                storage: storage_changes,
                                status: revm::database::AccountStatus::Changed,
                            },
                        );
                    }
                }
            }

            let bundle_post = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| (s.present_value, s.previous_or_original_value));

            // filter_unchanged_storage
            for (_addr, account) in bundle.state.iter_mut() {
                account
                    .storage
                    .retain(|_key, slot| slot.present_value != slot.previous_or_original_value);
            }

            let final_slot = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| s.present_value);
            assert!(
                final_slot.is_some(),
                "VARIANT E FAILED: gas_backlog slot MISSING from bundle (pre-populated DB)"
            );
            assert_eq!(
                final_slot.unwrap(),
                U256::from(552756u64 + 357751u64),
                "VARIANT E: gas_backlog should be 910507"
            );
        }

        // ===== VARIANT F: Pre-existing DB + StartBlock DRAIN (time_passed > 0) =====
        // The most realistic production scenario: gas_backlog exists in DB,
        // StartBlock drains some, then user tx grows it back.
        // The drain writes the same slot, and the grow writes it again.
        // If drain writes backlog=0 and grow writes backlog=357751,
        // but the original_value from DB was 552756, the filter should keep it.
        // BUT: what if drain writes backlog=552756 (no change from DB) and the
        // transition records original_value=552756? Then grow writes 910507 with
        // original_value=552756. This should still work. Let's verify.
        {
            struct PrePopulatedDb2 {
                gas_backlog_slot: U256,
                pre_existing_backlog: U256,
                speed_limit_slot: U256,
                speed_limit_value: U256,
            }

            impl Database for PrePopulatedDb2 {
                type Error = std::convert::Infallible;
                fn basic(
                    &mut self,
                    _address: Address,
                ) -> Result<Option<revm::state::AccountInfo>, Self::Error> {
                    Ok(Some(revm::state::AccountInfo {
                        balance: U256::ZERO,
                        nonce: 1,
                        code_hash: keccak256([]),
                        code: None,
                        account_id: None,
                    }))
                }
                fn code_by_hash(
                    &mut self,
                    _code_hash: B256,
                ) -> Result<revm::state::Bytecode, Self::Error> {
                    Ok(revm::state::Bytecode::default())
                }
                fn storage(&mut self, _address: Address, index: U256) -> Result<U256, Self::Error> {
                    if index == self.gas_backlog_slot {
                        Ok(self.pre_existing_backlog)
                    } else if index == self.speed_limit_slot {
                        Ok(self.speed_limit_value)
                    } else {
                        Ok(U256::ZERO)
                    }
                }
                fn block_hash(&mut self, _number: u64) -> Result<B256, Self::Error> {
                    Ok(B256::ZERO)
                }
            }

            // Compute speed_limit slot
            let speed_limit_slot = arb_storage::storage_key_map(l2_base.as_slice(), 0); // offset 0
            let pre_existing_backlog = U256::from(552756u64);

            let mut state = StateBuilder::new()
                .with_database(PrePopulatedDb2 {
                    gas_backlog_slot,
                    pre_existing_backlog,
                    speed_limit_slot,
                    speed_limit_value: U256::from(7_000_000u64), // 7M gas/sec
                })
                .with_bundle_update()
                .build();

            let _ = state.load_cache_account(ARBOS_STATE_ADDRESS);
            let state_ptr: *mut revm::database::State<PrePopulatedDb2> = &mut state;

            let backing = Storage::new(state_ptr, B256::ZERO);
            let l2_pricing =
                super::super::open_l2_pricing_state(backing.open_sub_storage(&[1]), 10);

            let initial = l2_pricing.gas_backlog().unwrap();

            // StartBlock with time_passed=1 → drain = 1 * 7_000_000 = 7M
            // 552756 - 7M = 0 (saturating sub)
            l2_pricing.update_pricing_model(1, 10).unwrap();
            let after_drain = l2_pricing.gas_backlog().unwrap();

            // EVM commit
            {
                use revm::DatabaseCommit;
                state.commit(Default::default());
            }

            // grow_backlog
            let l2_pricing2 =
                super::super::open_l2_pricing_state(backing.open_sub_storage(&[1]), 10);
            l2_pricing2
                .grow_backlog(357751, MultiGas::default())
                .unwrap();
            let after_grow = l2_pricing2.gas_backlog().unwrap();

            // merge + take_bundle
            state.merge_transitions(BundleRetention::Reverts);
            let mut bundle = state.take_bundle();

            let bundle_pre = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| (s.present_value, s.previous_or_original_value));

            // Note: In this variant we skip augment since all writes go through
            // write_storage_at which creates transitions. The bundle should
            // already have the slot.

            // filter
            for (_addr, account) in bundle.state.iter_mut() {
                account
                    .storage
                    .retain(|_key, slot| slot.present_value != slot.previous_or_original_value);
            }

            let final_slot = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| (s.present_value, s.previous_or_original_value));

            // Drain brought it to 0, grow added 357751 → final = 357751
            // original from DB = 552756
            // present=357751, original=552756 → different → should survive filter
            assert!(
                final_slot.is_some(),
                "VARIANT F FAILED: gas_backlog slot MISSING from bundle (drain+grow)"
            );
            assert_eq!(
                final_slot.unwrap().0,
                U256::from(357751u64),
                "VARIANT F: gas_backlog should be 357751"
            );
        }

        // ===== VARIANT G: Pre-existing DB + drain to 0 + NO grow =====
        // Edge case: if backlog drains to 0 and no user tx grows it,
        // the write is 0 and original from DB is 552756.
        // write_storage_at should NOT skip this (0 != 552756).
        // But wait — what if update_pricing_model drains to 0, and
        // write_storage_at's no-op check sees value=0 and prev_value=0?
        // This would happen if the cache already has backlog=0 from a previous
        // write... Let's check.
        {
            struct PrePopDb3 {
                gas_backlog_slot: U256,
                speed_limit_slot: U256,
            }

            impl Database for PrePopDb3 {
                type Error = std::convert::Infallible;
                fn basic(
                    &mut self,
                    _address: Address,
                ) -> Result<Option<revm::state::AccountInfo>, Self::Error> {
                    Ok(Some(revm::state::AccountInfo {
                        balance: U256::ZERO,
                        nonce: 1,
                        code_hash: keccak256([]),
                        code: None,
                        account_id: None,
                    }))
                }
                fn code_by_hash(
                    &mut self,
                    _code_hash: B256,
                ) -> Result<revm::state::Bytecode, Self::Error> {
                    Ok(revm::state::Bytecode::default())
                }
                fn storage(&mut self, _address: Address, index: U256) -> Result<U256, Self::Error> {
                    if index == self.gas_backlog_slot {
                        Ok(U256::from(552756u64))
                    } else if index == self.speed_limit_slot {
                        Ok(U256::from(7_000_000u64))
                    } else {
                        Ok(U256::ZERO)
                    }
                }
                fn block_hash(&mut self, _number: u64) -> Result<B256, Self::Error> {
                    Ok(B256::ZERO)
                }
            }

            let speed_limit_slot = arb_storage::storage_key_map(l2_base.as_slice(), 0);

            let mut state = StateBuilder::new()
                .with_database(PrePopDb3 {
                    gas_backlog_slot,
                    speed_limit_slot,
                })
                .with_bundle_update()
                .build();

            let _ = state.load_cache_account(ARBOS_STATE_ADDRESS);
            let state_ptr: *mut revm::database::State<PrePopDb3> = &mut state;

            let backing = Storage::new(state_ptr, B256::ZERO);
            let l2_pricing =
                super::super::open_l2_pricing_state(backing.open_sub_storage(&[1]), 10);

            let initial = l2_pricing.gas_backlog().unwrap();

            // Drain with time_passed=1 → 552756 - 7M = 0
            l2_pricing.update_pricing_model(1, 10).unwrap();
            let after_drain = l2_pricing.gas_backlog().unwrap();
            assert_eq!(after_drain, 0);

            // merge + take_bundle
            state.merge_transitions(BundleRetention::Reverts);
            let mut bundle = state.take_bundle();

            let pre_filter = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| (s.present_value, s.previous_or_original_value));

            // filter
            for (_addr, account) in bundle.state.iter_mut() {
                account
                    .storage
                    .retain(|_key, slot| slot.present_value != slot.previous_or_original_value);
            }

            let final_slot = bundle
                .state
                .get(&ARBOS_STATE_ADDRESS)
                .and_then(|a| a.storage.get(&gas_backlog_slot))
                .map(|s| (s.present_value, s.previous_or_original_value));

            // present=0, original=552756 → different → should survive
            assert!(
                final_slot.is_some(),
                "VARIANT G FAILED: drain-to-0 write was lost!"
            );
        }
    }
}
