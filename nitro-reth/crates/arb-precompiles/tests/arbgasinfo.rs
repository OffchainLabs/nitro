mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, U256};
use arb_precompiles::{
    create_arbgasinfo_precompile,
    storage_slot::{
        subspace_slot, ARBOS_STATE_ADDRESS, L1_PRICING_SUBSPACE, L2_PRICING_SUBSPACE,
    },
};
use common::{calldata, decode_address, decode_u256, decode_word, PrecompileTest};

fn arbgasinfo() -> DynPrecompile {
    create_arbgasinfo_precompile()
}

const L2_SPEED_LIMIT: u64 = 0;
const L2_PER_BLOCK_GAS_LIMIT: u64 = 1;
const L2_MIN_BASE_FEE: u64 = 3;
const L2_GAS_BACKLOG: u64 = 4;
const L2_PRICING_INERTIA: u64 = 5;
const L2_BACKLOG_TOLERANCE: u64 = 6;
const L2_PER_TX_GAS_LIMIT: u64 = 7;

const L1_PAY_REWARDS_TO: u64 = 0;
const L1_EQUILIBRATION_UNITS: u64 = 1;
const L1_INERTIA: u64 = 2;
const L1_PER_UNIT_REWARD: u64 = 3;
const L1_LAST_UPDATE_TIME: u64 = 4;
const L1_FUNDS_DUE_FOR_REWARDS: u64 = 5;
const L1_UNITS_SINCE: u64 = 6;
const L1_PRICE_PER_UNIT: u64 = 7;
const L1_LAST_SURPLUS: u64 = 8;
const L1_PER_BATCH_GAS_COST: u64 = 9;
const L1_AMORTIZED_COST_CAP_BIPS: u64 = 10;
const L1_FEES_AVAILABLE: u64 = 11;

fn fixture(arbos_version: u64) -> PrecompileTest {
    PrecompileTest::new()
        .arbos_version(arbos_version)
        .arbos_state()
}

fn put_l1(test: PrecompileTest, offset: u64, value: U256) -> PrecompileTest {
    test.storage(ARBOS_STATE_ADDRESS, subspace_slot(L1_PRICING_SUBSPACE, offset), value)
}

fn put_l2(test: PrecompileTest, offset: u64, value: U256) -> PrecompileTest {
    test.storage(ARBOS_STATE_ADDRESS, subspace_slot(L2_PRICING_SUBSPACE, offset), value)
}

#[test]
fn get_l1_basefee_estimate_returns_l1_price_per_unit() {
    let val = U256::from(123_456_789_u64);
    let run = put_l1(fixture(30), L1_PRICE_PER_UNIT, val)
        .call(&arbgasinfo(), &calldata("getL1BaseFeeEstimate()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_l1_gas_price_estimate_aliases_basefee() {
    let val = U256::from(987_654_321_u64);
    let run = put_l1(fixture(30), L1_PRICE_PER_UNIT, val)
        .call(&arbgasinfo(), &calldata("getL1GasPriceEstimate()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_minimum_gas_price_returns_l2_min_base_fee() {
    let val = U256::from(100_000_000_u64);
    let run = put_l2(fixture(30), L2_MIN_BASE_FEE, val)
        .call(&arbgasinfo(), &calldata("getMinimumGasPrice()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_gas_accounting_params_returns_speed_block_block() {
    let speed = U256::from(1_000_000_u64);
    let block_limit = U256::from(32_000_000_u64);
    let run = put_l2(put_l2(fixture(30), L2_SPEED_LIMIT, speed), L2_PER_BLOCK_GAS_LIMIT, block_limit)
        .call(&arbgasinfo(), &calldata("getGasAccountingParams()", &[]));
    let out = run.output();
    assert_eq!(decode_word(out, 0), common::word_u256(speed));
    assert_eq!(decode_word(out, 1), common::word_u256(block_limit));
    assert_eq!(decode_word(out, 2), common::word_u256(block_limit));
}

#[test]
fn get_gas_backlog_returns_l2_field() {
    let val = U256::from(7_777_u64);
    let run = put_l2(fixture(30), L2_GAS_BACKLOG, val)
        .call(&arbgasinfo(), &calldata("getGasBacklog()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_pricing_inertia_returns_l2_field() {
    let val = U256::from(102_u64);
    let run = put_l2(fixture(30), L2_PRICING_INERTIA, val)
        .call(&arbgasinfo(), &calldata("getPricingInertia()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_gas_backlog_tolerance_returns_l2_field() {
    let val = U256::from(11_u64);
    let run = put_l2(fixture(30), L2_BACKLOG_TOLERANCE, val)
        .call(&arbgasinfo(), &calldata("getGasBacklogTolerance()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_l1_basefee_estimate_inertia_returns_l1_field() {
    let val = U256::from(10_u64);
    let run = put_l1(fixture(30), L1_INERTIA, val)
        .call(&arbgasinfo(), &calldata("getL1BaseFeeEstimateInertia()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_per_batch_gas_charge_returns_l1_field() {
    let val = U256::from(210_000_u64);
    let run = put_l1(fixture(30), L1_PER_BATCH_GAS_COST, val)
        .call(&arbgasinfo(), &calldata("getPerBatchGasCharge()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_amortized_cost_cap_bips_returns_l1_field() {
    let val = U256::from(2_000_u64);
    let run = put_l1(fixture(30), L1_AMORTIZED_COST_CAP_BIPS, val)
        .call(&arbgasinfo(), &calldata("getAmortizedCostCapBips()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_l1_fees_available_gated_to_v10() {
    let val = U256::from(42_u64);
    let run = put_l1(fixture(9).gas(50_000), L1_FEES_AVAILABLE, val)
        .call(&arbgasinfo(), &calldata("getL1FeesAvailable()", &[]));
    let out = run.assert_ok();
    assert!(out.reverted, "below ArbosVersion_10 must revert");
    assert_eq!(out.gas_used, 50_000);
}

#[test]
fn get_l1_fees_available_returns_field_at_v10() {
    let val = U256::from(42_u64);
    let run = put_l1(fixture(10), L1_FEES_AVAILABLE, val)
        .call(&arbgasinfo(), &calldata("getL1FeesAvailable()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_l1_reward_rate_gated_to_v11() {
    let run = put_l1(fixture(10).gas(50_000), L1_PER_UNIT_REWARD, U256::from(7))
        .call(&arbgasinfo(), &calldata("getL1RewardRate()", &[]));
    assert!(run.assert_ok().reverted);
}

#[test]
fn get_l1_reward_rate_returns_field_at_v11() {
    let val = U256::from(7);
    let run = put_l1(fixture(11), L1_PER_UNIT_REWARD, val)
        .call(&arbgasinfo(), &calldata("getL1RewardRate()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_l1_reward_recipient_returns_address_at_v11() {
    let recipient: Address = address!("00000000000000000000000000000000000000ee");
    let val = U256::from_be_slice(recipient.as_slice());
    let run = put_l1(fixture(11), L1_PAY_REWARDS_TO, val)
        .call(&arbgasinfo(), &calldata("getL1RewardRecipient()", &[]));
    assert_eq!(decode_address(run.output()), recipient);
}

#[test]
fn get_l1_pricing_equilibration_units_gated_to_v20() {
    let run = put_l1(fixture(19).gas(50_000), L1_EQUILIBRATION_UNITS, U256::from(1_000_000))
        .call(
            &arbgasinfo(),
            &calldata("getL1PricingEquilibrationUnits()", &[]),
        );
    assert!(run.assert_ok().reverted);
}

#[test]
fn get_l1_pricing_equilibration_units_returns_field_at_v20() {
    let val = U256::from(1_000_000_u64);
    let run = put_l1(fixture(20), L1_EQUILIBRATION_UNITS, val)
        .call(
            &arbgasinfo(),
            &calldata("getL1PricingEquilibrationUnits()", &[]),
        );
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_last_l1_pricing_update_time_at_v20() {
    let val = U256::from(1_700_000_000_u64);
    let run = put_l1(fixture(20), L1_LAST_UPDATE_TIME, val)
        .call(&arbgasinfo(), &calldata("getLastL1PricingUpdateTime()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_l1_pricing_funds_due_for_rewards_at_v20() {
    let val = U256::from(123_u64);
    let run = put_l1(fixture(20), L1_FUNDS_DUE_FOR_REWARDS, val)
        .call(
            &arbgasinfo(),
            &calldata("getL1PricingFundsDueForRewards()", &[]),
        );
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_l1_pricing_units_since_update_at_v20() {
    let val = U256::from(456_u64);
    let run = put_l1(fixture(20), L1_UNITS_SINCE, val)
        .call(&arbgasinfo(), &calldata("getL1PricingUnitsSinceUpdate()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_last_l1_pricing_surplus_at_v20() {
    let val = U256::from(789_u64);
    let run = put_l1(fixture(20), L1_LAST_SURPLUS, val)
        .call(&arbgasinfo(), &calldata("getLastL1PricingSurplus()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_max_block_gas_limit_gated_to_v50() {
    let run = put_l2(fixture(49).gas(50_000), L2_PER_BLOCK_GAS_LIMIT, U256::from(32_000_000))
        .call(&arbgasinfo(), &calldata("getMaxBlockGasLimit()", &[]));
    assert!(run.assert_ok().reverted);
}

#[test]
fn get_max_block_gas_limit_returns_field_at_v50() {
    let val = U256::from(32_000_000_u64);
    let run = put_l2(fixture(50), L2_PER_BLOCK_GAS_LIMIT, val)
        .call(&arbgasinfo(), &calldata("getMaxBlockGasLimit()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_max_tx_gas_limit_returns_field_at_v50() {
    let val = U256::from(7_000_000_u64);
    let run = put_l2(fixture(50), L2_PER_TX_GAS_LIMIT, val)
        .call(&arbgasinfo(), &calldata("getMaxTxGasLimit()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_multi_gas_pricing_constraints_gated_to_v60() {
    let run = fixture(59).gas(50_000)
        .call(&arbgasinfo(), &calldata("getMultiGasPricingConstraints()", &[]));
    assert!(run.assert_ok().reverted);
}

#[test]
fn get_multi_gas_base_fee_gated_to_v60() {
    let run = fixture(59).gas(50_000)
        .call(&arbgasinfo(), &calldata("getMultiGasBaseFee()", &[]));
    assert!(run.assert_ok().reverted);
}
