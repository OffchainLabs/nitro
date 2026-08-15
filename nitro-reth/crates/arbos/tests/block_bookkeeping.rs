//! Pure tests for the block-production bookkeeping types.

use arb_chainspec::arbos_version::{ARBOS_VERSION_50, ARBOS_VERSION_FIX_REDEEM_GAS};
use arbos::block_bookkeeping::{ComputeBudget, TX_GAS};

// =====================================================================
// ComputeBudget
// =====================================================================

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
fn compute_gas_rejection_pre_arbos_50() {
    let mut budget = ComputeBudget::new(100_000);
    // Consume the first-tx bypass.
    budget.charge(ARBOS_VERSION_50 - 1, 50_000, 0, &[], true);

    // Oversized tx: compute gas (1M) exceeds what's left.
    assert!(budget.rejects_compute_gas(ARBOS_VERSION_50 - 1, 1_000_000, 0));
    // Same tx passes on ArbOS >= 50 (per-tx clamping handles it instead).
    assert!(!budget.rejects_compute_gas(ARBOS_VERSION_50, 1_000_000, 0));
    // Fitting tx passes.
    assert!(!budget.rejects_compute_gas(ARBOS_VERSION_50 - 1, 10_000, 0));
}

#[test]
fn first_user_tx_bypasses_compute_gas_rejection() {
    let budget = ComputeBudget::new(TX_GAS);
    // Would exceed the budget, but no user tx was processed yet.
    assert!(!budget.rejects_compute_gas(ARBOS_VERSION_50 - 1, 1_000_000, 0));
}

#[test]
fn compute_gas_rejection_subtracts_poster_gas_and_floors_at_tx_gas() {
    let mut budget = ComputeBudget::new(1_000_000);
    budget.charge(ARBOS_VERSION_50 - 1, 50_000, 0, &[], true);

    // Poster gas eats most of the limit: compute = max(1M - 990k, TX_GAS) = TX_GAS.
    assert!(!budget.rejects_compute_gas(ARBOS_VERSION_50 - 1, 1_000_000, 990_000));
    // Even with poster gas above the limit, the TX_GAS floor applies.
    assert!(!budget.rejects_compute_gas(ARBOS_VERSION_50 - 1, 10_000, 20_000));
}

#[test]
fn charge_deducts_compute_portion_and_counts_user_txs() {
    let mut budget = ComputeBudget::new(1_000_000);
    let charged = budget.charge(ARBOS_VERSION_50, 100_000, 30_000, &[], true);
    assert_eq!(charged, 70_000);
    assert_eq!(budget.gas_left(), 930_000);
    assert_eq!(budget.user_txs_processed(), 1);

    // Non-user txs are charged but not counted.
    budget.charge(ARBOS_VERSION_50, 100_000, 30_000, &[], false);
    assert_eq!(budget.user_txs_processed(), 1);
}

#[test]
fn charge_floors_at_tx_gas() {
    let mut budget = ComputeBudget::new(1_000_000);
    // Compute portion below TX_GAS floors at TX_GAS.
    assert_eq!(
        budget.charge(ARBOS_VERSION_50, 25_000, 10_000, &[], true),
        TX_GAS
    );
    // Gas used below data gas also charges TX_GAS.
    assert_eq!(
        budget.charge(ARBOS_VERSION_50, 10_000, 50_000, &[], true),
        TX_GAS
    );
}

#[test]
fn charge_subtracts_scheduled_retry_gas_from_fix_redeem_gas_onward() {
    // From FixRedeemGas: gas reserved by scheduled retries is deducted, since it is charged when
    // the retry itself executes.
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

    // Retry gas above gas used saturates the adjustment at zero; the TX_GAS floor still applies.
    let mut budget = ComputeBudget::new(1_000_000);
    let charged = budget.charge(ARBOS_VERSION_FIX_REDEEM_GAS, 100_000, 0, &[150_000], true);
    assert_eq!(charged, TX_GAS);
}

#[test]
fn charge_saturates_at_zero() {
    let mut budget = ComputeBudget::new(30_000);
    budget.charge(ARBOS_VERSION_50, 100_000, 0, &[], true);
    assert_eq!(budget.gas_left(), 0);
    assert!(budget.exhausted_for_user_tx());
}

#[test]
fn failed_tx_charges_tx_gas_and_counts_user_txs() {
    // Ported from the deleted block_processor tests.
    let mut budget = ComputeBudget::new(100_000);
    budget.charge_failed_tx(true);
    assert_eq!(budget.gas_left(), 100_000 - TX_GAS);
    assert_eq!(budget.user_txs_processed(), 1);

    budget.charge_failed_tx(false);
    assert_eq!(budget.gas_left(), 100_000 - 2 * TX_GAS);
    assert_eq!(budget.user_txs_processed(), 1);
}
