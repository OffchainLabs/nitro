use alloy_primitives::{address, Address, U256};
use arb_test_utils::ArbosHarness;
use arbos::l1_pricing::{
    byte_count_after_brotli_level, compute_poster_cost_standalone, poster_units_from_bytes,
    BATCH_POSTER_ADDRESS, ESTIMATION_PADDING_BASIS_POINTS, ESTIMATION_PADDING_UNITS,
    INITIAL_EQUILIBRATION_UNITS_V6, INITIAL_INERTIA, INITIAL_PER_BATCH_GAS_COST_V12,
    INITIAL_PER_BATCH_GAS_COST_V6, INITIAL_PER_UNIT_REWARD, TX_DATA_NON_ZERO_GAS_EIP2028,
};

const ONE_GWEI: u64 = 1_000_000_000;

#[test]
fn eip2028_nonzero_gas_constant_is_16() {
    assert_eq!(TX_DATA_NON_ZERO_GAS_EIP2028, 16);
}

#[test]
fn initial_params_match_spec() {
    assert_eq!(INITIAL_INERTIA, 10);
    assert_eq!(INITIAL_PER_UNIT_REWARD, 10);
    assert_eq!(INITIAL_EQUILIBRATION_UNITS_V6, 16 * 10_000_000);
    assert_eq!(INITIAL_PER_BATCH_GAS_COST_V6, 100_000);
    assert_eq!(INITIAL_PER_BATCH_GAS_COST_V12, 210_000);
}

#[test]
fn estimation_padding_matches_spec() {
    assert_eq!(ESTIMATION_PADDING_UNITS, TX_DATA_NON_ZERO_GAS_EIP2028 * 16);
    assert_eq!(ESTIMATION_PADDING_BASIS_POINTS, 100);
}

#[test]
fn compute_poster_cost_standalone_zero_if_not_batch_poster() {
    let (cost, units) = compute_poster_cost_standalone(
        b"some tx bytes",
        address!("0101010101010101010101010101010101010101"),
        U256::from(ONE_GWEI),
        0,
    );
    assert_eq!(cost, U256::ZERO);
    assert_eq!(units, 0);
}

#[test]
fn compute_poster_cost_standalone_nonzero_for_batch_poster() {
    let (cost, units) = compute_poster_cost_standalone(
        b"hello world",
        BATCH_POSTER_ADDRESS,
        U256::from(ONE_GWEI),
        11,
    );
    assert!(units > 0);
    assert!(cost > U256::ZERO);
}

#[test]
fn poster_units_scales_with_brotli_size() {
    let small = poster_units_from_bytes(b"", 11);
    let big = poster_units_from_bytes(&[0x77u8; 1024], 11);
    assert!(big > small);
}

#[test]
fn poster_units_divisible_by_16_non_zero_gas() {
    let units = poster_units_from_bytes(&[0x55u8; 100], 11);
    assert_eq!(units % TX_DATA_NON_ZERO_GAS_EIP2028, 0);
}

#[test]
fn brotli_output_is_smaller_for_repetitive_input() {
    let input = vec![0xABu8; 2048];
    let compressed = byte_count_after_brotli_level(&input, 11);
    assert!(compressed < input.len() as u64);
}

#[test]
fn brotli_empty_input_is_bounded_small() {
    let compressed = byte_count_after_brotli_level(&[], 11);
    assert!(compressed < 16);
}

#[test]
fn brotli_level_0_larger_than_level_11_for_compressible_data() {
    let input = vec![0xABu8; 10_000];
    let c0 = byte_count_after_brotli_level(&input, 0);
    let c11 = byte_count_after_brotli_level(&input, 11);
    assert!(c11 <= c0);
}

// ======================================================================
// Stateful L1PricingState tests (fresh harness per test)
// ======================================================================

fn fresh() -> ArbosHarness {
    ArbosHarness::new().initialize()
}

#[test]
fn initial_pay_rewards_to_is_readable() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    let _ = ps.pay_rewards_to().unwrap();
}

#[test]
fn set_and_get_pay_rewards_to() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    let new = address!("AABBCCDDEEFF00112233445566778899AABBCCDD");
    ps.set_pay_rewards_to(new).unwrap();
    assert_eq!(ps.pay_rewards_to().unwrap(), new);
}

#[test]
fn set_and_get_equilibration_units() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    let initial = ps.equilibration_units().unwrap();
    assert_eq!(initial, U256::from(INITIAL_EQUILIBRATION_UNITS_V6));
    ps.set_equilibration_units(U256::from(999_999u64)).unwrap();
    assert_eq!(ps.equilibration_units().unwrap(), U256::from(999_999u64));
}

#[test]
fn set_and_get_inertia() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    assert_eq!(ps.inertia().unwrap(), INITIAL_INERTIA);
    ps.set_inertia(42).unwrap();
    assert_eq!(ps.inertia().unwrap(), 42);
}

#[test]
fn set_and_get_per_unit_reward() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    assert_eq!(ps.per_unit_reward().unwrap(), INITIAL_PER_UNIT_REWARD);
    ps.set_per_unit_reward(33).unwrap();
    assert_eq!(ps.per_unit_reward().unwrap(), 33);
}

#[test]
fn units_since_update_add_and_subtract() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    assert_eq!(ps.units_since_update().unwrap(), 0);
    ps.add_to_units_since_update(100).unwrap();
    ps.add_to_units_since_update(50).unwrap();
    assert_eq!(ps.units_since_update().unwrap(), 150);
    ps.subtract_from_units_since_update(30).unwrap();
    assert_eq!(ps.units_since_update().unwrap(), 120);
}

#[test]
fn set_price_per_unit_stores_exact_u256() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    let big = U256::from(u64::MAX) * U256::from(1_000_000u64);
    ps.set_price_per_unit(big).unwrap();
    assert_eq!(ps.price_per_unit().unwrap(), big);
}

#[test]
fn last_surplus_default_zero_positive() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    let (mag, neg) = ps.last_surplus().unwrap();
    assert_eq!(mag, U256::ZERO);
    assert!(!neg);
}

#[test]
fn set_last_surplus_positive_and_negative() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    ps.set_last_surplus(U256::from(1000u64), false).unwrap();
    assert_eq!(ps.last_surplus().unwrap(), (U256::from(1000u64), false));
    ps.set_last_surplus(U256::from(500u64), true).unwrap();
    assert_eq!(ps.last_surplus().unwrap(), (U256::from(500u64), true));
}

#[test]
fn per_batch_gas_cost_default_is_initial() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    assert_eq!(ps.per_batch_gas_cost().unwrap(), INITIAL_PER_BATCH_GAS_COST_V12);
}

#[test]
fn per_batch_gas_cost_set_accepts_negative() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    ps.set_per_batch_gas_cost(-50_000).unwrap();
    assert_eq!(ps.per_batch_gas_cost().unwrap(), -50_000);
}

#[test]
fn amortized_cost_cap_bips_default_zero_and_settable() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    assert_eq!(ps.amortized_cost_cap_bips().unwrap(), 0);
    ps.set_amortized_cost_cap_bips(10_000).unwrap();
    assert_eq!(ps.amortized_cost_cap_bips().unwrap(), 10_000);
}

#[test]
fn l1_fees_available_add_and_transfer() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    assert_eq!(ps.l1_fees_available().unwrap(), U256::ZERO);
    ps.add_to_l1_fees_available(U256::from(1_000u64)).unwrap();
    ps.add_to_l1_fees_available(U256::from(500u64)).unwrap();
    assert_eq!(ps.l1_fees_available().unwrap(), U256::from(1_500u64));
    let taken = ps.transfer_from_l1_fees_available(U256::from(200u64)).unwrap();
    assert_eq!(taken, U256::from(200u64));
    assert_eq!(ps.l1_fees_available().unwrap(), U256::from(1_300u64));
}

#[test]
fn transfer_from_l1_fees_caps_at_available() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    ps.add_to_l1_fees_available(U256::from(100u64)).unwrap();
    let taken = ps.transfer_from_l1_fees_available(U256::from(500u64)).unwrap();
    assert_eq!(taken, U256::from(100u64));
    assert_eq!(ps.l1_fees_available().unwrap(), U256::ZERO);
}

#[test]
fn parent_gas_floor_per_token_zero_for_pre_v50() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    assert_eq!(ps.parent_gas_floor_per_token().unwrap(), 0);
    assert!(ps.set_parent_gas_floor_per_token(400).is_err());
}

#[test]
fn parent_gas_floor_per_token_settable_at_v50() {
    let mut h = ArbosHarness::new().with_arbos_version(50).initialize();
    let ps = h.l1_pricing_state();
    ps.set_parent_gas_floor_per_token(400).unwrap();
    assert_eq!(ps.parent_gas_floor_per_token().unwrap(), 400);
}

#[test]
fn funds_due_for_rewards_set_get() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    assert_eq!(ps.funds_due_for_rewards().unwrap(), U256::ZERO);
    ps.set_funds_due_for_rewards(U256::from(12_345u64)).unwrap();
    assert_eq!(ps.funds_due_for_rewards().unwrap(), U256::from(12_345u64));
}

#[test]
fn surplus_default_is_zero_positive() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    let (mag, neg) = ps.get_l1_pricing_surplus().unwrap();
    assert_eq!(mag, U256::ZERO);
    assert!(!neg);
}

#[test]
fn surplus_is_negative_when_funds_due_exceeds_available() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    ps.set_funds_due_for_rewards(U256::from(1_000u64)).unwrap();
    let (mag, neg) = ps.get_l1_pricing_surplus().unwrap();
    assert!(neg, "surplus should be negative when funds_due > available");
    assert_eq!(mag, U256::from(1_000u64));
}

#[test]
fn surplus_is_positive_when_available_exceeds_funds_due() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    ps.add_to_l1_fees_available(U256::from(1_000u64)).unwrap();
    let (mag, neg) = ps.get_l1_pricing_surplus().unwrap();
    assert!(!neg);
    assert_eq!(mag, U256::from(1_000u64));
}

#[test]
fn poster_data_cost_adds_price_times_units_plus_per_batch_cost() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    ps.set_price_per_unit(U256::from(10u64)).unwrap();
    ps.set_per_batch_gas_cost(0).unwrap();
    let c = ps.poster_data_cost(100).unwrap();
    assert_eq!(c, U256::from(1_000u64));
    ps.set_per_batch_gas_cost(500).unwrap();
    assert_eq!(ps.poster_data_cost(100).unwrap(), U256::from(1_500u64));
}

#[test]
fn poster_data_cost_subtracts_negative_per_batch_cost() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    ps.set_price_per_unit(U256::from(10u64)).unwrap();
    ps.set_per_batch_gas_cost(-200).unwrap();
    assert_eq!(ps.poster_data_cost(100).unwrap(), U256::from(800u64));
}

#[test]
fn poster_data_cost_for_estimation_adds_padding() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    ps.set_price_per_unit(U256::from(10u64)).unwrap();
    let tx_bytes = vec![0x42u8; 100];
    let (raw_cost, raw_units) = ps.compute_poster_cost(BATCH_POSTER_ADDRESS, &tx_bytes, 11).unwrap();
    let (padded_cost, padded_units) =
        ps.poster_data_cost_for_estimation(&tx_bytes, 11).unwrap();
    assert!(padded_units > raw_units, "estimation pads raw units");
    assert!(padded_cost > raw_cost);
}

#[test]
fn get_poster_info_returns_funds_due_and_pay_to() {
    let mut h = fresh();
    let ps = h.l1_pricing_state();
    let (funds_due, pay_to) = ps.get_poster_info(BATCH_POSTER_ADDRESS).unwrap();
    assert_eq!(funds_due, U256::ZERO);
    assert_ne!(pay_to, Address::ZERO);
}
