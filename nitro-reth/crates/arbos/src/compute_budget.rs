//! Per-block compute budget, mirroring the compute-gas accounting of Go's `blockBuildState`
//! (`arbos/block_processor.go`).

use arb_chainspec::arbos_version::{ARBOS_VERSION_50, ARBOS_VERSION_FIX_REDEEM_GAS};

/// Standard Ethereum transaction gas; the compute-charging floor.
pub const TX_GAS: u64 = 21_000;

/// Per-block compute budget for block gas rate limiting. Caps the compute portion of gas: total
/// minus the L1 poster component. Distinct from the header gas limit.
#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ComputeBudget {
    gas_left: u64,
    user_txs_processed: u64,
}

impl ComputeBudget {
    pub fn new(per_block_gas_limit: u64) -> Self {
        Self {
            gas_left: per_block_gas_limit,
            user_txs_processed: 0,
        }
    }

    pub fn gas_left(&self) -> u64 {
        self.gas_left
    }

    /// User transactions processed so far.
    pub fn user_txs_processed(&self) -> u64 {
        self.user_txs_processed
    }

    /// Whether the block is out of compute gas for user transactions.
    pub fn exhausted_for_user_tx(&self) -> bool {
        self.gas_left < TX_GAS
    }

    /// For ArbOS < 50: rejects a user tx whose compute gas (gas limit minus poster gas, floored at
    /// [`TX_GAS`]) exceeds the remaining budget — except the block's first user tx.
    /// ArbOS >= 50 clamps per-tx gas in the gas charging hook instead.
    pub fn rejects_compute_gas(
        &self,
        arbos_version: u64,
        tx_gas_limit: u64,
        poster_gas: u64,
        is_user_tx: bool,
    ) -> bool {
        if arbos_version >= ARBOS_VERSION_50 || !is_user_tx || self.user_txs_processed == 0 {
            return false;
        }
        let compute_gas = tx_gas_limit.saturating_sub(poster_gas).max(TX_GAS);
        compute_gas > self.gas_left
    }

    /// Charges a committed transaction and counts it; returns the compute gas charged.
    /// `header_arbos_version` is the post-execution header version — it differs from the pre-tx
    /// state version (used by [`Self::rejects_compute_gas`]) in the block applying an ArbOS
    /// upgrade.
    pub fn charge(
        &mut self,
        header_arbos_version: u64,
        gas_used: u64,
        poster_gas: u64,
        scheduled_retry_gas: &[u64],
        is_user_tx: bool,
    ) -> u64 {
        let adjusted =
            adjust_for_scheduled_retries(header_arbos_version, gas_used, scheduled_retry_gas);
        let compute_used = compute_used(adjusted, poster_gas);

        self.gas_left = self.gas_left.saturating_sub(compute_used);
        if is_user_tx {
            self.user_txs_processed += 1;
        }

        compute_used
    }

    /// Charges a failed transaction ([`TX_GAS`]) and counts it.
    pub fn charge_failed_tx(&mut self, is_user_tx: bool) {
        self.gas_left = self.gas_left.saturating_sub(TX_GAS);
        if is_user_tx {
            self.user_txs_processed += 1;
        }
    }
}

/// From ArbOS >= FixRedeemGas, gas reserved by scheduled retry txs is subtracted from a tx's gas
/// used — it is charged when the retry itself executes.
fn adjust_for_scheduled_retries(
    header_arbos_version: u64,
    gas_used: u64,
    scheduled_retry_gas: &[u64],
) -> u64 {
    if header_arbos_version < ARBOS_VERSION_FIX_REDEEM_GAS {
        return gas_used;
    }
    scheduled_retry_gas
        .iter()
        .fold(gas_used, |gas, retry| gas.saturating_sub(*retry))
}

/// The compute portion of a tx's gas: `gas_used - poster_gas`, floored at [`TX_GAS`].
fn compute_used(gas_used: u64, poster_gas: u64) -> u64 {
    gas_used.saturating_sub(poster_gas).max(TX_GAS)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn fresh_budget_has_full_gas_and_no_user_txs() {
        let budget = ComputeBudget::new(1_000_000);
        assert_eq!(budget.gas_left(), 1_000_000);
        assert_eq!(budget.user_txs_processed(), 0);
        assert!(!budget.exhausted_for_user_tx());
    }

    #[test]
    fn exhaustion_threshold_is_tx_gas() {
        assert!(!ComputeBudget::new(TX_GAS).exhausted_for_user_tx());
        assert!(ComputeBudget::new(TX_GAS - 1).exhausted_for_user_tx());
    }

    #[test]
    fn first_user_tx_bypasses_compute_gas_rejection() {
        let budget = ComputeBudget::new(1_000);
        for v in [ARBOS_VERSION_50 - 1, ARBOS_VERSION_50, ARBOS_VERSION_50 + 1] {
            assert!(!budget.rejects_compute_gas(v, 1_000_000, 0, true));
        }
    }

    #[test]
    fn user_tx_can_bypass_compute_gas_rejection_based_on_arbos_version() {
        let mut budget = ComputeBudget::new(100_000);
        // Consume the first-tx (to avoid unconditional bypass). Version doesn't matter.
        budget.charge(0, 50_000, 0, &[], true);

        // Arbos >= 50
        assert!(!budget.rejects_compute_gas(ARBOS_VERSION_50, 1_000_000, 0, true));
        assert!(!budget.rejects_compute_gas(ARBOS_VERSION_50 + 1, 1_000_000, 0, true));

        // Arbos < 50
        // Oversized tx: compute gas (1M) exceeds what's left.
        assert!(budget.rejects_compute_gas(ARBOS_VERSION_50 - 1, 1_000_000, 0, true));
        // Fitting tx passes.
        assert!(!budget.rejects_compute_gas(ARBOS_VERSION_50 - 1, 10_000, 0, true));
    }

    #[test]
    fn non_user_tx_is_never_rejected() {
        let mut budget = ComputeBudget::new(100_000);
        budget.charge(0, 50_000, 0, &[], true);

        // Same oversized tx: rejected as a user tx, admitted as a non-user tx.
        assert!(budget.rejects_compute_gas(ARBOS_VERSION_50 - 1, 1_000_000, 0, true));
        assert!(!budget.rejects_compute_gas(ARBOS_VERSION_50 - 1, 1_000_000, 0, false));
    }

    #[test]
    fn compute_gas_rejection_subtracts_poster_gas_and_floors_at_tx_gas_pre_arbos_50() {
        let mut budget = ComputeBudget::new(1_000_000);
        budget.charge(0, 50_000, 0, &[], true);

        // Poster gas eats most of the limit: compute = max(1M - 990k, TX_GAS) = TX_GAS.
        assert!(!budget.rejects_compute_gas(ARBOS_VERSION_50 - 1, 1_000_000, 990_000, true));
        // Even with poster gas above the limit, the TX_GAS floor applies.
        assert!(!budget.rejects_compute_gas(ARBOS_VERSION_50 - 1, 10_000, 20_000, true));
    }

    #[test]
    fn charge_deducts_compute_portion_and_counts_user_txs() {
        let mut budget = ComputeBudget::new(1_000_000);
        let charged = budget.charge(0, 100_000, 30_000, &[], true);
        assert_eq!(charged, 70_000);
        assert_eq!(budget.gas_left(), 930_000);
        assert_eq!(budget.user_txs_processed(), 1);

        // Non-user txs are charged but not counted.
        budget.charge(0, 100_000, 30_000, &[], false);
        assert_eq!(budget.gas_left(), 860_000);
        assert_eq!(budget.user_txs_processed(), 1);
    }

    #[test]
    fn charge_floors_at_tx_gas() {
        let mut budget = ComputeBudget::new(1_000_000);

        for (gas_used, poster_gas, user_tx) in [
            (25_000, 10_000, true),
            (25_000, 10_000, false),
            (25_000, 30_000, true),
        ] {
            assert_eq!(budget.charge(0, gas_used, poster_gas, &[], user_tx), TX_GAS);
        }
    }

    #[test]
    fn charge_subtracts_scheduled_retry_gas_from_fix_redeem_gas_onward() {
        // From FixRedeemGas: gas reserved by scheduled retries is deducted, since it is charged
        // when the retry itself executes.
        let mut budget = ComputeBudget::new(1_000_000);
        let charged = budget.charge(
            ARBOS_VERSION_FIX_REDEEM_GAS,
            500_000,
            0,
            &[150_000, 50_000],
            true,
        );
        assert_eq!(charged, 300_000);

        // Before FixRedeemGas the reserved gas is charged twice (historical behavior, preserved).
        let mut budget = ComputeBudget::new(1_000_000);
        let charged = budget.charge(
            ARBOS_VERSION_FIX_REDEEM_GAS - 1,
            500_000,
            0,
            &[150_000, 50_000],
            true,
        );
        assert_eq!(charged, 500_000);

        // Retry gas above gas used saturates the adjustment at zero; the TX_GAS floor still
        // applies.
        let mut budget = ComputeBudget::new(1_000_000);
        let charged = budget.charge(ARBOS_VERSION_FIX_REDEEM_GAS, 100_000, 0, &[150_000], true);
        assert_eq!(charged, TX_GAS);
    }

    #[test]
    fn charge_saturates_at_zero() {
        let mut budget = ComputeBudget::new(30_000);
        budget.charge(0, 100_000, 0, &[], true);
        assert_eq!(budget.gas_left(), 0);
        assert!(budget.exhausted_for_user_tx());
    }

    #[test]
    fn failed_tx_charges_tx_gas_and_counts_user_txs() {
        let mut budget = ComputeBudget::new(100_000);
        budget.charge_failed_tx(true);
        assert_eq!(budget.gas_left(), 100_000 - TX_GAS);
        assert_eq!(budget.user_txs_processed(), 1);

        budget.charge_failed_tx(false);
        assert_eq!(budget.gas_left(), 100_000 - 2 * TX_GAS);
        assert_eq!(budget.user_txs_processed(), 1);
    }
}
