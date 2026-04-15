mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, U256};
use arb_precompiles::{
    create_arbgasinfo_precompile,
    storage_slot::{subspace_slot, ARBOS_STATE_ADDRESS, L1_PRICING_SUBSPACE, L2_PRICING_SUBSPACE},
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
    test.storage(
        ARBOS_STATE_ADDRESS,
        subspace_slot(L1_PRICING_SUBSPACE, offset),
        value,
    )
}

fn put_l2(test: PrecompileTest, offset: u64, value: U256) -> PrecompileTest {
    test.storage(
        ARBOS_STATE_ADDRESS,
        subspace_slot(L2_PRICING_SUBSPACE, offset),
        value,
    )
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
    let run = put_l2(
        put_l2(fixture(30), L2_SPEED_LIMIT, speed),
        L2_PER_BLOCK_GAS_LIMIT,
        block_limit,
    )
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
    let run = put_l1(fixture(30), L1_INERTIA, val).call(
        &arbgasinfo(),
        &calldata("getL1BaseFeeEstimateInertia()", &[]),
    );
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
    let run = put_l1(
        fixture(19).gas(50_000),
        L1_EQUILIBRATION_UNITS,
        U256::from(1_000_000),
    )
    .call(
        &arbgasinfo(),
        &calldata("getL1PricingEquilibrationUnits()", &[]),
    );
    assert!(run.assert_ok().reverted);
}

#[test]
fn get_l1_pricing_equilibration_units_returns_field_at_v20() {
    let val = U256::from(1_000_000_u64);
    let run = put_l1(fixture(20), L1_EQUILIBRATION_UNITS, val).call(
        &arbgasinfo(),
        &calldata("getL1PricingEquilibrationUnits()", &[]),
    );
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_last_l1_pricing_update_time_at_v20() {
    let val = U256::from(1_700_000_000_u64);
    let run = put_l1(fixture(20), L1_LAST_UPDATE_TIME, val).call(
        &arbgasinfo(),
        &calldata("getLastL1PricingUpdateTime()", &[]),
    );
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_l1_pricing_funds_due_for_rewards_at_v20() {
    let val = U256::from(123_u64);
    let run = put_l1(fixture(20), L1_FUNDS_DUE_FOR_REWARDS, val).call(
        &arbgasinfo(),
        &calldata("getL1PricingFundsDueForRewards()", &[]),
    );
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn get_l1_pricing_units_since_update_at_v20() {
    let val = U256::from(456_u64);
    let run = put_l1(fixture(20), L1_UNITS_SINCE, val).call(
        &arbgasinfo(),
        &calldata("getL1PricingUnitsSinceUpdate()", &[]),
    );
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
    let run = put_l2(
        fixture(49).gas(50_000),
        L2_PER_BLOCK_GAS_LIMIT,
        U256::from(32_000_000),
    )
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
    let run = fixture(59).gas(50_000).call(
        &arbgasinfo(),
        &calldata("getMultiGasPricingConstraints()", &[]),
    );
    assert!(run.assert_ok().reverted);
}

#[test]
fn get_multi_gas_base_fee_gated_to_v60() {
    let run = fixture(59)
        .gas(50_000)
        .call(&arbgasinfo(), &calldata("getMultiGasBaseFee()", &[]));
    assert!(run.assert_ok().reverted);
}

#[test]
fn get_prices_in_wei_uses_block_basefee_not_storage() {
    // Regression for the wrong-source bug: handle_prices_in_wei used to read
    // L2_BASE_FEE from storage instead of evm.Context.BaseFee. Set them to
    // different values to lock the behavior in.
    let l1_price = U256::from(50_000_000_u64);
    let stored_l2_base = U256::from(999_999_999_u64); // would-be wrong source
    let block_basefee = 100_000_000_u64; // the correct source per Nitro
    let l2_min = U256::from(50_000_000_u64);

    let test = put_l1(fixture(30), L1_PRICE_PER_UNIT, l1_price);
    let test = put_l2(test, 2 /* L2_BASE_FEE */, stored_l2_base);
    let test = put_l2(test, L2_MIN_BASE_FEE, l2_min);
    let run = test
        .block_basefee(block_basefee)
        .call(&arbgasinfo(), &calldata("getPricesInWei()", &[]));
    let out = run.output();
    // perArbGasTotal (slot 5) should be the block base fee, not the stored value.
    assert_eq!(decode_word(out, 5), common::word_u64(block_basefee));
    // perArbGasBase = min(block_basefee, l2_min)
    let expected_base = std::cmp::min(U256::from(block_basefee), l2_min);
    assert_eq!(decode_word(out, 3), common::word_u256(expected_base));
}

#[test]
fn get_prices_in_arbgas_uses_block_basefee_not_storage() {
    // Same regression for prices-in-arbgas: Nitro divides wei costs by
    // evm.Context.BaseFee, not by the stored L2_BASE_FEE field.
    let l1_price = U256::from(40_000_000_u64);
    let stored_l2_base = U256::from(1u64); // 1 wei would yield huge wrong values
    let block_basefee = 200_000_000_u64;

    let test = put_l1(fixture(30), L1_PRICE_PER_UNIT, l1_price);
    let test = put_l2(test, 2 /* L2_BASE_FEE */, stored_l2_base);
    let run = test
        .block_basefee(block_basefee)
        .call(&arbgasinfo(), &calldata("getPricesInArbGas()", &[]));
    let out = run.output();
    // gas_for_l1_calldata = (l1_price * 16) / block_basefee
    let expected_calldata = (l1_price * U256::from(16u64)) / U256::from(block_basefee);
    assert_eq!(decode_word(out, 1), common::word_u256(expected_calldata));
}

// ── Ported from Nitro ──────────────────────────────────────────────────

/// Nitro-parity value pin for `getPricesInWei()`. Derived directly from
/// Nitro's `GetPricesInWeiWithAggregator` implementation in
/// `precompiles/ArbGasInfo.go`:
///
///   weiForL1Calldata = l1_price * TxDataNonZeroGas(16)
///   perL2Tx          = weiForL1Calldata * AssumedSimpleTxSize(140)
///   perArbGasBase    = min(l2_gas_price, l2_min_base_fee)
///   perArbGasCong    = l2_gas_price - perArbGasBase
///   perArbGasTotal   = l2_gas_price
///   weiForL2Storage  = l2_gas_price * StorageWriteCost(20_000)
///
/// Run with Nitro's default `DefaultInitialL1BaseFee` (50 GWei) and a
/// block basefee of 1005 so the arithmetic matches `getPricesInArbGas`'s
/// pinned test and proves both functions use the same formula family.
#[test]
fn nitro_parity_get_prices_in_wei() {
    const DEFAULT_INITIAL_L1_BASE_FEE: u64 = 50_000_000_000;
    const STORAGE_WRITE_COST: u64 = 20_000;
    const TX_DATA_NON_ZERO_GAS: u64 = 16;
    const ASSUMED_SIMPLE_TX_SIZE: u64 = 140;
    let basefee: u64 = 1005;
    let l2_min: u64 = 500; // < basefee so perArbGasCong > 0

    let test = put_l1(
        fixture(30),
        L1_PRICE_PER_UNIT,
        U256::from(DEFAULT_INITIAL_L1_BASE_FEE),
    );
    let test = put_l2(test, L2_MIN_BASE_FEE, U256::from(l2_min));
    let run = test
        .block_basefee(basefee)
        .call(&arbgasinfo(), &calldata("getPricesInWei()", &[]));
    let out = run.output();

    let wei_for_l1_calldata = DEFAULT_INITIAL_L1_BASE_FEE * TX_DATA_NON_ZERO_GAS;
    let per_l2_tx = wei_for_l1_calldata * ASSUMED_SIMPLE_TX_SIZE;
    let wei_for_l2_storage = basefee * STORAGE_WRITE_COST;
    let per_arbgas_base = std::cmp::min(basefee, l2_min);
    let per_arbgas_cong = basefee - per_arbgas_base;
    let per_arbgas_total = basefee;

    assert_eq!(decode_word(out, 0), common::word_u64(per_l2_tx));
    assert_eq!(decode_word(out, 1), common::word_u64(wei_for_l1_calldata));
    assert_eq!(decode_word(out, 2), common::word_u64(wei_for_l2_storage));
    assert_eq!(decode_word(out, 3), common::word_u64(per_arbgas_base));
    assert_eq!(decode_word(out, 4), common::word_u64(per_arbgas_cong));
    assert_eq!(decode_word(out, 5), common::word_u64(per_arbgas_total));
}

/// Port of `TestGetPricesInArbGas` in
/// nitro/precompiles/ArbGasInfo_test.go. Nitro's default L1BaseFee is
/// `DefaultInitialL1BaseFee = 50 GWei = 50,000,000,000 wei`
/// (arbos/arbostypes/incomingmessage.go:317). With block.basefee = 1005
/// and AssumedSimpleTxSize = 140, the Nitro test pins the exact three
/// return values of `getPricesInArbGas()`. Reproducing them here
/// byte-for-byte proves our integer arithmetic matches Go's big.Int
/// truncation semantics across multiply-then-divide, which is the
/// subtle part of the formula.
#[test]
fn nitro_parity_get_prices_in_arbgas() {
    const DEFAULT_INITIAL_L1_BASE_FEE: u64 = 50_000_000_000;
    const STORAGE_WRITE_COST: u64 = 20_000;

    let test = put_l1(
        fixture(30),
        L1_PRICE_PER_UNIT,
        U256::from(DEFAULT_INITIAL_L1_BASE_FEE),
    );
    let run = test
        .block_basefee(1005)
        .call(&arbgasinfo(), &calldata("getPricesInArbGas()", &[]));
    let out = run.output();

    // gasPerL2Tx   = (l1_price * 16 * 140) / basefee = 111_442_786_069
    // gasForL1Cd   = (l1_price * 16)       / basefee =     796_019_900
    // storageArbGas = StorageWriteCost                =          20_000
    assert_eq!(decode_word(out, 0), common::word_u64(111_442_786_069));
    assert_eq!(decode_word(out, 1), common::word_u64(796_019_900));
    assert_eq!(decode_word(out, 2), common::word_u64(STORAGE_WRITE_COST));
}

// ── Protocol gas-cost pins ─────────────────────────────────────────────
//
// These tests lock in the *exact* gas cost returned by the precompile so
// that any future refactor of the value source cannot silently drop an
// SloadGas charge. The expected numbers are derived from Nitro's burn
// model: OpenArbosState + one SloadGas per storage field read by the
// method body, plus CopyGas (3) per return word and per input arg word.
//
// Regression for commit b627a908 → 984e13c: the switch from storage
// L2_BASE_FEE to evm.Context.BaseFee correctly removed the storage read,
// but the gas cost must still cover OpenArbosState + L1_PRICE_PER_UNIT +
// L2_MIN_BASE_FEE = 3 * 800 = 2400, plus 6 return words * 3 = 18 → 2418.
// The bug silently reduced this to 2 * 800 + 18 = 1618, causing an
// 800-gas undercharge per call and a state-root mismatch at block
// 55,705,814 tx 1 (ERC-4337 handleOps).

const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;

#[test]
fn get_prices_in_wei_charges_three_sloads_and_six_copy_words() {
    let test = put_l1(fixture(30), L1_PRICE_PER_UNIT, U256::from(1u64));
    let test = put_l2(test, L2_MIN_BASE_FEE, U256::from(1u64));
    let run = test
        .block_basefee(100_000_000)
        .call(&arbgasinfo(), &calldata("getPricesInWei()", &[]));
    // OpenArbosState(1) + PricePerUnit(1) + MinBaseFeeWei(1) = 3 sloads;
    // return tuple is 6 * uint256 = 6 words of copy gas.
    assert_eq!(run.gas_used(), 3 * SLOAD_GAS + 6 * COPY_GAS);
}

#[test]
fn get_prices_in_arbgas_charges_two_sloads_and_three_copy_words() {
    let test = put_l1(fixture(30), L1_PRICE_PER_UNIT, U256::from(1u64));
    let run = test
        .block_basefee(100_000_000)
        .call(&arbgasinfo(), &calldata("getPricesInArbGas()", &[]));
    // OpenArbosState(1) + PricePerUnit(1) = 2 sloads; return tuple is
    // 3 * uint256 = 3 words.
    assert_eq!(run.gas_used(), 2 * SLOAD_GAS + 3 * COPY_GAS);
}

#[test]
fn get_l1_basefee_estimate_charges_two_sloads_and_one_copy_word() {
    let test = put_l1(fixture(30), L1_PRICE_PER_UNIT, U256::from(42u64));
    let run = test.call(&arbgasinfo(), &calldata("getL1BaseFeeEstimate()", &[]));
    // OpenArbosState(1) + PricePerUnit(1) = 2 sloads; 1 return word.
    assert_eq!(run.gas_used(), 2 * SLOAD_GAS + COPY_GAS);
}

#[test]
fn get_minimum_gas_price_charges_two_sloads_and_one_copy_word() {
    let test = put_l2(fixture(30), L2_MIN_BASE_FEE, U256::from(42u64));
    let run = test.call(&arbgasinfo(), &calldata("getMinimumGasPrice()", &[]));
    assert_eq!(run.gas_used(), 2 * SLOAD_GAS + COPY_GAS);
}

// ── L1 pricing surplus ─────────────────────────────────────────────────

const BATCH_POSTER_TABLE_KEY: &[u8] = &[0];
const TOTAL_FUNDS_DUE_OFFSET: u64 = 0;

const L1_PRICER_FUNDS_POOL: Address = address!("a4b05fffffffffffffffffffffffffffffffffff");

fn batch_poster_total_funds_due_slot() -> U256 {
    use arb_precompiles::storage_slot::derive_subspace_key;
    let l1_key = derive_subspace_key(
        arb_precompiles::storage_slot::ROOT_STORAGE_KEY,
        L1_PRICING_SUBSPACE,
    );
    let bpt_key = derive_subspace_key(l1_key.as_slice(), BATCH_POSTER_TABLE_KEY);
    arb_precompiles::storage_slot::map_slot(bpt_key.as_slice(), TOTAL_FUNDS_DUE_OFFSET)
}

#[test]
fn get_l1_pricing_surplus_pre_v10_uses_pool_balance() {
    // Pre-v10: surplus = poolBalance - (totalFundsDue + fundsDueForRewards).
    let pool_balance = U256::from(1_000_000_u64);
    let total_due = U256::from(300_000_u64);
    let funds_due_rewards = U256::from(200_000_u64);
    let test = fixture(9)
        .balance(L1_PRICER_FUNDS_POOL, pool_balance)
        .storage(
            ARBOS_STATE_ADDRESS,
            batch_poster_total_funds_due_slot(),
            total_due,
        );
    let test = put_l1(test, L1_FUNDS_DUE_FOR_REWARDS, funds_due_rewards);
    let run = test.call(&arbgasinfo(), &calldata("getL1PricingSurplus()", &[]));
    let want = pool_balance - total_due - funds_due_rewards;
    assert_eq!(decode_u256(run.output()), want);
}

#[test]
fn get_l1_pricing_surplus_v10_plus_uses_stored_field() {
    // v10+: surplus = L1FeesAvailable - (totalFundsDue + fundsDueForRewards).
    let stored_available = U256::from(2_000_000_u64);
    let total_due = U256::from(500_000_u64);
    let funds_due_rewards = U256::from(100_000_u64);
    let test = fixture(10).storage(
        ARBOS_STATE_ADDRESS,
        batch_poster_total_funds_due_slot(),
        total_due,
    );
    let test = put_l1(test, L1_FUNDS_DUE_FOR_REWARDS, funds_due_rewards);
    let test = put_l1(test, L1_FEES_AVAILABLE, stored_available);
    let run = test.call(&arbgasinfo(), &calldata("getL1PricingSurplus()", &[]));
    let want = stored_available - total_due - funds_due_rewards;
    assert_eq!(decode_u256(run.output()), want);
}

#[test]
fn get_l1_pricing_surplus_returns_negative_two_complement_when_deficit() {
    // L1FeesAvailable smaller than need → surplus is negative; encoded as
    // two's complement in U256.
    let stored_available = U256::from(100_u64);
    let total_due = U256::from(500_u64);
    let funds_due_rewards = U256::from(50_u64);
    let deficit = total_due + funds_due_rewards - stored_available;
    let test = fixture(10).storage(
        ARBOS_STATE_ADDRESS,
        batch_poster_total_funds_due_slot(),
        total_due,
    );
    let test = put_l1(test, L1_FUNDS_DUE_FOR_REWARDS, funds_due_rewards);
    let test = put_l1(test, L1_FEES_AVAILABLE, stored_available);
    let run = test.call(&arbgasinfo(), &calldata("getL1PricingSurplus()", &[]));
    // Expected: -deficit in 256-bit two's complement.
    let want = U256::ZERO.wrapping_sub(deficit);
    assert_eq!(decode_u256(run.output()), want);
}

#[test]
fn get_gas_accounting_params_layout_is_three_words() {
    let speed = U256::from(7_000_000_u64);
    let block_lim = U256::from(32_000_000_u64);
    let test = put_l2(fixture(30), L2_SPEED_LIMIT, speed);
    let test = put_l2(test, L2_PER_BLOCK_GAS_LIMIT, block_lim);
    let run = test.call(&arbgasinfo(), &calldata("getGasAccountingParams()", &[]));
    let out = run.output();
    assert_eq!(out.len(), 96);
    assert_eq!(decode_word(out, 0), common::word_u256(speed));
    assert_eq!(decode_word(out, 1), common::word_u256(block_lim));
    assert_eq!(decode_word(out, 2), common::word_u256(block_lim));
}
