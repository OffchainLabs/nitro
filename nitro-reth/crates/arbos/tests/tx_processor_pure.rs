use alloy_primitives::{address, Address, U256};
use arbos::tx_processor::{
    compute_poster_gas, compute_retryable_gas_split, compute_submit_retryable_fees, get_poster_gas,
    take_funds, SubmitRetryableParams,
};

const ONE_GWEI: u64 = 1_000_000_000;
const ARBOS_V30: u64 = 30;
const ARBOS_V11: u64 = 11;
const ARBOS_V10: u64 = 10;

#[test]
fn take_funds_drains_pool_partially() {
    let mut pool = U256::from(100u64);
    let taken = take_funds(&mut pool, U256::from(30u64));
    assert_eq!(taken, U256::from(30u64));
    assert_eq!(pool, U256::from(70u64));
}

#[test]
fn take_funds_caps_at_pool_size() {
    let mut pool = U256::from(50u64);
    let taken = take_funds(&mut pool, U256::from(80u64));
    assert_eq!(taken, U256::from(50u64));
    assert_eq!(pool, U256::ZERO);
}

#[test]
fn take_funds_zero_pool_returns_zero() {
    let mut pool = U256::ZERO;
    let taken = take_funds(&mut pool, U256::from(10u64));
    assert_eq!(taken, U256::ZERO);
    assert_eq!(pool, U256::ZERO);
}

#[test]
fn compute_poster_gas_base_fee_zero_returns_zero() {
    let g = compute_poster_gas(
        U256::from(1_000_000u64),
        U256::ZERO,
        false,
        U256::from(1u64),
    );
    assert_eq!(g, 0);
}

#[test]
fn compute_poster_gas_basic_division() {
    let cost = U256::from(10_000u64) * U256::from(ONE_GWEI);
    let base_fee = U256::from(ONE_GWEI);
    let g = compute_poster_gas(cost, base_fee, false, U256::ZERO);
    assert_eq!(g, 10_000);
}

#[test]
fn compute_poster_gas_estimation_pads_cost_and_drops_base_fee() {
    let cost = U256::from(10_000u64) * U256::from(ONE_GWEI);
    let base_fee = U256::from(ONE_GWEI);
    let plain = compute_poster_gas(cost, base_fee, false, U256::ZERO);
    let estimated = compute_poster_gas(cost, base_fee, true, U256::ZERO);
    assert!(estimated > plain);
}

#[test]
fn compute_poster_gas_estimation_floor_uses_min_gas_price() {
    let cost = U256::from(10_000u64) * U256::from(ONE_GWEI);
    let tiny_base_fee = U256::from(1u64);
    let min_gas_price = U256::from(ONE_GWEI);
    let g = compute_poster_gas(cost, tiny_base_fee, true, min_gas_price);
    let expected = cost * U256::from(11000u64) / U256::from(10000u64) / min_gas_price;
    assert_eq!(U256::from(g), expected);
}

#[test]
fn get_poster_gas_returns_zero_when_either_fee_zero() {
    let data = b"hello";
    assert_eq!(
        get_poster_gas(data, U256::ZERO, U256::from(1u64), ARBOS_V30),
        (0, 0)
    );
    assert_eq!(
        get_poster_gas(data, U256::from(1u64), U256::ZERO, ARBOS_V30),
        (0, 0)
    );
}

#[test]
fn get_poster_gas_counts_zero_and_nonzero_bytes() {
    let data = vec![0u8, 1u8, 0u8, 2u8, 3u8];
    let (_, units) = get_poster_gas(&data, U256::from(ONE_GWEI), U256::from(ONE_GWEI), ARBOS_V30);
    assert_eq!(units, 3 * 16 + 2 * 4);
}

#[test]
fn get_poster_gas_scales_with_l1_base_fee() {
    let data = b"abcdef";
    let l2 = U256::from(ONE_GWEI);
    let g_low = get_poster_gas(data, U256::from(1u64), l2, ARBOS_V30).0;
    let g_high = get_poster_gas(data, U256::from(ONE_GWEI), l2, ARBOS_V30).0;
    assert!(g_high > g_low);
}

#[test]
fn retryable_gas_split_no_infra_below_v11() {
    let infra: Address = address!("AAAA000000000000000000000000000000000000");
    let (i, n) = compute_retryable_gas_split(
        100_000,
        U256::from(ONE_GWEI),
        infra,
        U256::from(ONE_GWEI / 2),
        ARBOS_V10,
    );
    assert_eq!(i, U256::ZERO);
    assert_eq!(n, U256::from(ONE_GWEI) * U256::from(100_000u64));
}

#[test]
fn retryable_gas_split_no_infra_when_account_zero() {
    let (i, n) = compute_retryable_gas_split(
        100_000,
        U256::from(ONE_GWEI),
        Address::ZERO,
        U256::from(ONE_GWEI / 2),
        ARBOS_V11,
    );
    assert_eq!(i, U256::ZERO);
    assert_eq!(n, U256::from(ONE_GWEI) * U256::from(100_000u64));
}

#[test]
fn retryable_gas_split_takes_min_of_min_and_effective_for_infra() {
    let infra: Address = address!("AAAA000000000000000000000000000000000000");
    let effective = U256::from(ONE_GWEI);
    let min = U256::from(ONE_GWEI / 4);
    let (i, n) = compute_retryable_gas_split(100_000, effective, infra, min, ARBOS_V11);
    assert_eq!(i, min * U256::from(100_000u64));
    assert_eq!(n, (effective - min) * U256::from(100_000u64));
}

#[test]
fn submit_retryable_fees_returns_escrow_address_and_timeout() {
    let ticket = alloy_primitives::B256::repeat_byte(0x42);
    let params = SubmitRetryableParams {
        ticket_id: ticket,
        from: address!("0000000000000000000000000000000000000A11"),
        fee_refund_addr: address!("0000000000000000000000000000000000000B0B"),
        deposit_value: U256::from(10u64) * U256::from(ONE_GWEI),
        retry_value: U256::ZERO,
        gas_fee_cap: U256::from(ONE_GWEI),
        gas: 100_000,
        max_submission_fee: U256::from(10u64) * U256::from(ONE_GWEI),
        retry_data_len: 0,
        l1_base_fee: U256::from(ONE_GWEI),
        effective_base_fee: U256::from(ONE_GWEI),
        current_time: 1_000_000,
        balance_after_mint: U256::from(10u64) * U256::from(ONE_GWEI),
        infra_fee_account: Address::ZERO,
        min_base_fee: U256::from(ONE_GWEI / 2),
        arbos_version: ARBOS_V30,
    };
    let fees = compute_submit_retryable_fees(&params);
    assert_eq!(
        fees.escrow,
        arbos::retryables::retryable_escrow_address(ticket)
    );
    assert!(fees.timeout > params.current_time);
}
