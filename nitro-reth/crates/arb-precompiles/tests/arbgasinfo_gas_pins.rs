//! Per-selector gas pins for ArbGasInfo (0x6c).
//!
//! Locks the exact `PrecompileOutput::gas_used` returned for every selector
//! so any future refactor that drops a charge or changes the per-method
//! schedule fails a named test instead of going unnoticed.

mod common;

use alloy_evm::precompiles::DynPrecompile;
use arb_precompiles::create_arbgasinfo_precompile;
use common::{calldata, PrecompileTest};

const ARBOS_V30: u64 = 30;
const ARBOS_V50: u64 = 50;
const ARBOS_V60: u64 = 60;

fn arbgasinfo(ctx: std::sync::Arc<arb_context::ArbPrecompileCtx>) -> DynPrecompile {
    create_arbgasinfo_precompile(ctx)
}

fn fixture(v: u64) -> PrecompileTest {
    PrecompileTest::new().arbos_version(v).arbos_state()
}

#[test]
fn get_l1_base_fee_estimate_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getL1BaseFeeEstimate()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_l1_gas_price_estimate_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getL1GasPriceEstimate()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_minimum_gas_price_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getMinimumGasPrice()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_prices_in_wei_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getPricesInWei()", &[]));
    assert_eq!(run.gas_used(), 2418);
}

#[test]
fn get_prices_in_arb_gas_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getPricesInArbGas()", &[]));
    assert_eq!(run.gas_used(), 1609);
}

#[test]
fn get_gas_accounting_params_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getGasAccountingParams()", &[]));
    assert_eq!(run.gas_used(), 2409);
}

#[test]
fn get_current_tx_l1_gas_fees_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getCurrentTxL1GasFees()", &[]));
    assert_eq!(run.gas_used(), 803);
}

#[test]
fn get_l1_base_fee_estimate_inertia_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getL1BaseFeeEstimateInertia()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_gas_backlog_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getGasBacklog()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_pricing_inertia_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getPricingInertia()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_gas_backlog_tolerance_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getGasBacklogTolerance()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_per_batch_gas_charge_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getPerBatchGasCharge()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_amortized_cost_cap_bips_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getAmortizedCostCapBips()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_l1_fees_available_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getL1FeesAvailable()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_l1_reward_rate_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getL1RewardRate()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_l1_reward_recipient_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getL1RewardRecipient()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_l1_pricing_equilibration_units_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(
        arbgasinfo,
        &calldata("getL1PricingEquilibrationUnits()", &[]),
    );
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_last_l1_pricing_update_time_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getLastL1PricingUpdateTime()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_l1_pricing_funds_due_for_rewards_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(
        arbgasinfo,
        &calldata("getL1PricingFundsDueForRewards()", &[]),
    );
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_l1_pricing_units_since_update_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getL1PricingUnitsSinceUpdate()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_last_l1_pricing_surplus_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getLastL1PricingSurplus()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_max_block_gas_limit_v50_gas_pin() {
    let run = fixture(ARBOS_V50).call(arbgasinfo, &calldata("getMaxBlockGasLimit()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_max_tx_gas_limit_v50_gas_pin() {
    let run = fixture(ARBOS_V50).call(arbgasinfo, &calldata("getMaxTxGasLimit()", &[]));
    assert_eq!(run.gas_used(), 1603);
}

#[test]
fn get_l1_pricing_surplus_v30_gas_pin() {
    let run = fixture(ARBOS_V30).call(arbgasinfo, &calldata("getL1PricingSurplus()", &[]));
    assert_eq!(run.gas_used(), 3203);
}

#[test]
fn get_prices_in_wei_with_aggregator_v30_gas_pin() {
    use alloy_primitives::Address;
    use common::word_address;
    let run = fixture(ARBOS_V30).call(
        arbgasinfo,
        &calldata(
            "getPricesInWeiWithAggregator(address)",
            &[word_address(Address::ZERO)],
        ),
    );
    assert_eq!(run.gas_used(), 2421);
}

#[test]
fn get_prices_in_arb_gas_with_aggregator_v30_gas_pin() {
    use alloy_primitives::Address;
    use common::word_address;
    let run = fixture(ARBOS_V30).call(
        arbgasinfo,
        &calldata(
            "getPricesInArbGasWithAggregator(address)",
            &[word_address(Address::ZERO)],
        ),
    );
    assert_eq!(run.gas_used(), 1612);
}

#[test]
fn get_gas_pricing_constraints_v50_gas_pin() {
    let run = fixture(ARBOS_V50).call(arbgasinfo, &calldata("getGasPricingConstraints()", &[]));
    assert_eq!(run.gas_used(), 1606);
}

#[test]
fn get_multi_gas_pricing_constraints_v60_gas_pin() {
    let run = fixture(ARBOS_V60).call(
        arbgasinfo,
        &calldata("getMultiGasPricingConstraints()", &[]),
    );
    assert_eq!(run.gas_used(), 1606);
}

#[test]
fn get_multi_gas_base_fee_v60_gas_pin() {
    let run = fixture(ARBOS_V60).call(arbgasinfo, &calldata("getMultiGasBaseFee()", &[]));
    assert_eq!(run.gas_used(), 8833);
}
