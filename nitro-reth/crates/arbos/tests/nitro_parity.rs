use alloy_primitives::U256;
use arb_test_utils::ArbosHarness;
use arbos::{
    l1_pricing::{INITIAL_EQUILIBRATION_UNITS_V0, INITIAL_EQUILIBRATION_UNITS_V6},
    retryables::retryable_submission_fee,
};

const ONE_GWEI: u64 = 1_000_000_000;

#[test]
fn submission_fee_zero_calldata() {
    let l1 = U256::from(ONE_GWEI);
    assert_eq!(retryable_submission_fee(0, l1), U256::from(1400u64) * l1);
}

#[test]
fn submission_fee_100_bytes() {
    let l1 = U256::from(ONE_GWEI);
    assert_eq!(retryable_submission_fee(100, l1), U256::from(2000u64) * l1);
}

#[test]
fn submission_fee_10kb() {
    let l1 = U256::from(ONE_GWEI);
    assert_eq!(
        retryable_submission_fee(10_000, l1),
        U256::from(61400u64) * l1
    );
}

#[test]
fn submission_fee_with_zero_base_fee_is_zero() {
    assert_eq!(retryable_submission_fee(1000, U256::ZERO), U256::ZERO);
}

#[test]
fn submission_fee_handles_1mb_calldata() {
    let l1 = U256::from(ONE_GWEI);
    let len = 1_048_576usize;
    let fee = retryable_submission_fee(len, l1);
    assert_eq!(fee, U256::from(1400u64 + 6 * len as u64) * l1);
}

#[test]
fn equilibration_units_v0_matches_nitro() {
    assert_eq!(INITIAL_EQUILIBRATION_UNITS_V0, 96_000_000);
}

#[test]
fn equilibration_units_v6_matches_nitro() {
    assert_eq!(INITIAL_EQUILIBRATION_UNITS_V6, 160_000_000);
}

#[test]
fn equilibration_units_after_v30_init_is_v6() {
    let mut h = ArbosHarness::new().with_arbos_version(30).initialize();
    let l1 = h.l1_pricing_state();
    assert_eq!(
        l1.equilibration_units().unwrap(),
        U256::from(INITIAL_EQUILIBRATION_UNITS_V6)
    );
}

#[test]
fn brotli_level_is_1_at_v30() {
    let mut h = ArbosHarness::new().with_arbos_version(30).initialize();
    assert_eq!(h.arbos_state().brotli_compression_level().unwrap(), 1);
}

#[test]
fn brotli_level_is_0_at_v10() {
    let mut h = ArbosHarness::new().with_arbos_version(10).initialize();
    assert_eq!(h.arbos_state().brotli_compression_level().unwrap(), 0);
}

#[test]
fn per_batch_gas_cost_at_v30_is_v12() {
    let mut h = ArbosHarness::new().with_arbos_version(30).initialize();
    assert_eq!(h.l1_pricing_state().per_batch_gas_cost().unwrap(), 210_000);
}

#[test]
fn per_batch_gas_cost_at_v10_is_v6() {
    let mut h = ArbosHarness::new().with_arbos_version(10).initialize();
    assert_eq!(h.l1_pricing_state().per_batch_gas_cost().unwrap(), 100_000);
}

#[test]
fn per_tx_gas_limit_at_v50_is_32m() {
    let mut h = ArbosHarness::new().with_arbos_version(50).initialize();
    assert_eq!(h.l2_pricing_state().per_tx_gas_limit().unwrap(), 32_000_000);
}

#[test]
fn per_tx_gas_limit_below_v50_is_zero() {
    let mut h = ArbosHarness::new().with_arbos_version(40).initialize();
    assert_eq!(h.l2_pricing_state().per_tx_gas_limit().unwrap(), 0);
}

#[test]
fn l1_inertia_initial_value_is_10() {
    let mut h = ArbosHarness::new().initialize();
    assert_eq!(h.l1_pricing_state().inertia().unwrap(), 10);
}

#[test]
fn l1_per_unit_reward_initial_value_is_10() {
    let mut h = ArbosHarness::new().initialize();
    assert_eq!(h.l1_pricing_state().per_unit_reward().unwrap(), 10);
}

#[test]
fn l2_pricing_inertia_is_102() {
    let mut h = ArbosHarness::new().initialize();
    assert_eq!(h.l2_pricing_state().pricing_inertia().unwrap(), 102);
}

#[test]
fn l2_backlog_tolerance_is_10() {
    let mut h = ArbosHarness::new().initialize();
    assert_eq!(h.l2_pricing_state().backlog_tolerance().unwrap(), 10);
}

#[test]
fn l2_min_base_fee_is_0_1_gwei() {
    let mut h = ArbosHarness::new().initialize();
    assert_eq!(
        h.l2_pricing_state().min_base_fee_wei().unwrap(),
        U256::from(100_000_000u64)
    );
}

#[test]
fn l2_speed_limit_v6_plus_is_7m() {
    let mut h = ArbosHarness::new().initialize();
    assert_eq!(
        h.l2_pricing_state().speed_limit_per_second().unwrap(),
        7_000_000
    );
}

#[test]
fn l2_per_block_gas_limit_v6_plus_is_32m() {
    let mut h = ArbosHarness::new().initialize();
    assert_eq!(
        h.l2_pricing_state().per_block_gas_limit().unwrap(),
        32_000_000
    );
}

/// Hand-computed against Nitro's `ApproxExpBasisPoints(10000, 4)`.
/// Loop expansion: 12500 -> 14166 -> 17083 -> 27083.
/// With min_base_fee=100M and exponent=10000:
///   new fee = 100_000_000 * 27083 / 10000 = 270_830_000
#[test]
fn approx_exp_basis_points_10000_yields_27083() {
    let mut h = ArbosHarness::new().with_arbos_version(30).initialize();
    let p = h.l2_pricing_state();
    let speed = p.speed_limit_per_second().unwrap();
    let inertia = p.pricing_inertia().unwrap();
    let tolerance = p.backlog_tolerance().unwrap();
    let backlog = tolerance * speed + inertia * speed;
    p.set_gas_backlog(backlog).unwrap();
    p.update_pricing_model(0, 30).unwrap();
    assert_eq!(p.base_fee_wei().unwrap(), U256::from(270_830_000u64));
}

#[test]
fn approx_exp_basis_points_zero_returns_one_in_bips() {
    let mut h = ArbosHarness::new().with_arbos_version(30).initialize();
    let p = h.l2_pricing_state();
    let min = p.min_base_fee_wei().unwrap();
    p.set_gas_backlog(0).unwrap();
    p.update_pricing_model(0, 30).unwrap();
    assert_eq!(p.base_fee_wei().unwrap(), min);
}

mod end_tx_retryable {
    use super::*;
    use alloy_primitives::{address, Address, B256};
    use arbos::tx_processor::{EndTxRetryableParams, TxProcessor};
    use std::cell::RefCell;

    const FROM: Address = address!("00000000000000000000000000000000000A11CE");
    const REFUND_TO: Address = address!("00000000000000000000000000000000000B0B00");
    const NETWORK: Address = address!("00000000000000000000000000000000C4A841E0");
    const INFRA: Address = address!("0000000000000000000000000000000000DA7E00");
    const COINBASE: Address = arbos::l1_pricing::BATCH_POSTER_ADDRESS;

    fn base_params(success: bool, arbos_version: u64, infra: Address) -> EndTxRetryableParams {
        EndTxRetryableParams {
            gas_left: 50_000,
            gas_used: 50_000,
            effective_base_fee: U256::from(ONE_GWEI),
            from: FROM,
            refund_to: REFUND_TO,
            max_refund: U256::from(10u64) * U256::from(ONE_GWEI) * U256::from(1_000_000u64),
            submission_fee_refund: U256::from(1_000u64) * U256::from(ONE_GWEI),
            ticket_id: B256::repeat_byte(0x42),
            value: U256::ZERO,
            success,
            network_fee_account: NETWORK,
            infra_fee_account: infra,
            min_base_fee: U256::from(ONE_GWEI / 2),
            arbos_version,
            multi_dimensional_cost: None,
            block_base_fee: U256::from(ONE_GWEI),
        }
    }

    type Burns = RefCell<Vec<(Address, U256)>>;
    type Transfers = RefCell<Vec<(Address, Address, U256)>>;

    fn run(p: &TxProcessor, params: &EndTxRetryableParams) -> (Burns, Transfers) {
        let burns: Burns = Default::default();
        let transfers: Transfers = Default::default();
        let _ = p.end_tx_retryable(
            params,
            |addr, amount| burns.borrow_mut().push((addr, amount)),
            |from, to, amount| {
                transfers.borrow_mut().push((from, to, amount));
                Ok(())
            },
        );
        (burns, transfers)
    }

    #[test]
    fn success_burns_gas_refund_from_user_first() {
        let p = TxProcessor::new(COINBASE);
        let params = base_params(true, 30, INFRA);
        let (burns, _) = run(&p, &params);
        let expected = params.effective_base_fee * U256::from(params.gas_left);
        assert_eq!(burns.borrow()[0], (FROM, expected));
    }

    /// At v11+ the gas refund splits: infra gets `min(min_base_fee, effective)*gas_left`,
    /// network gets the rest. Submission fee refund is a separate transfer from network.
    #[test]
    fn v11_with_infra_gas_refund_routes_infra_portion_to_infra() {
        let p = TxProcessor::new(COINBASE);
        let params = base_params(true, 11, INFRA);
        let (_, transfers) = run(&p, &params);
        let log = transfers.borrow();
        let expected_infra =
            params.min_base_fee.min(params.effective_base_fee) * U256::from(params.gas_left);
        let to_infra: U256 = log.iter().filter(|t| t.0 == INFRA).map(|t| t.2).sum();
        assert_eq!(to_infra, expected_infra);
    }

    #[test]
    fn v11_with_infra_gas_refund_routes_remainder_to_network() {
        let p = TxProcessor::new(COINBASE);
        let params = base_params(true, 11, INFRA);
        let (_, transfers) = run(&p, &params);
        let log = transfers.borrow();
        let infra_amt =
            params.min_base_fee.min(params.effective_base_fee) * U256::from(params.gas_left);
        let total_gas_refund = params.effective_base_fee * U256::from(params.gas_left);
        let net_portion = total_gas_refund - infra_amt;
        let from_network_to_refund_to: Vec<U256> = log
            .iter()
            .filter(|(from, to, _)| *from == NETWORK && *to == REFUND_TO)
            .map(|(_, _, amt)| *amt)
            .collect();
        assert!(from_network_to_refund_to.contains(&net_portion));
        assert!(from_network_to_refund_to.contains(&params.submission_fee_refund));
    }

    #[test]
    fn v10_no_infra_split() {
        let p = TxProcessor::new(COINBASE);
        let params = base_params(true, 10, INFRA);
        let (_, transfers) = run(&p, &params);
        let log = transfers.borrow();
        let to_infra: U256 = log.iter().filter(|t| t.0 == INFRA).map(|t| t.2).sum();
        assert_eq!(to_infra, U256::ZERO);
    }

    #[test]
    fn v10_full_gas_refund_to_network() {
        let p = TxProcessor::new(COINBASE);
        let params = base_params(true, 10, INFRA);
        let (_, transfers) = run(&p, &params);
        let log = transfers.borrow();
        let total_gas_refund = params.effective_base_fee * U256::from(params.gas_left);
        let from_network_to_refund_to: Vec<U256> = log
            .iter()
            .filter(|(from, to, _)| *from == NETWORK && *to == REFUND_TO)
            .map(|(_, _, amt)| *amt)
            .collect();
        assert!(from_network_to_refund_to.contains(&total_gas_refund));
    }

    #[test]
    fn v11_zero_infra_no_split() {
        let p = TxProcessor::new(COINBASE);
        let params = base_params(true, 11, Address::ZERO);
        let (_, transfers) = run(&p, &params);
        let log = transfers.borrow();
        let to_infra: U256 = log.iter().filter(|t| t.0 == INFRA).map(|t| t.2).sum();
        assert_eq!(to_infra, U256::ZERO);
        let total_gas_refund = params.effective_base_fee * U256::from(params.gas_left);
        let from_network_to_refund_to: Vec<U256> = log
            .iter()
            .filter(|(from, to, _)| *from == NETWORK && *to == REFUND_TO)
            .map(|(_, _, amt)| *amt)
            .collect();
        assert!(from_network_to_refund_to.contains(&total_gas_refund));
    }

    #[test]
    fn success_refunds_submission_fee_from_network() {
        let p = TxProcessor::new(COINBASE);
        let params = base_params(true, 30, INFRA);
        let (_, transfers) = run(&p, &params);
        let log = transfers.borrow();
        let from_net_to_refund: U256 = log
            .iter()
            .filter(|(from, to, _)| *from == NETWORK && *to == REFUND_TO)
            .map(|(_, _, amt)| *amt)
            .sum();
        assert!(from_net_to_refund >= params.submission_fee_refund);
    }

    #[test]
    fn failure_does_not_refund_submission_fee() {
        let p = TxProcessor::new(COINBASE);
        let params = base_params(false, 30, INFRA);
        let (_, transfers) = run(&p, &params);
        let log = transfers.borrow();
        let count: usize = log
            .iter()
            .filter(|(from, to, amt)| {
                *from == NETWORK && *to == REFUND_TO && *amt == params.submission_fee_refund
            })
            .count();
        assert_eq!(count, 0);
    }

    #[test]
    fn success_signals_delete_retryable() {
        let p = TxProcessor::new(COINBASE);
        let result = p.end_tx_retryable(&base_params(true, 30, INFRA), |_, _| {}, |_, _, _| Ok(()));
        assert!(result.should_delete_retryable);
        assert!(!result.should_return_value_to_escrow);
    }

    #[test]
    fn failure_keeps_retryable_and_returns_escrow() {
        let p = TxProcessor::new(COINBASE);
        let result =
            p.end_tx_retryable(&base_params(false, 30, INFRA), |_, _| {}, |_, _, _| Ok(()));
        assert!(!result.should_delete_retryable);
        assert!(result.should_return_value_to_escrow);
    }

    #[test]
    fn multi_dimensional_refund_pays_difference() {
        let p = TxProcessor::new(COINBASE);
        let mut params = base_params(true, 60, INFRA);
        let single = params.effective_base_fee * U256::from(params.gas_used);
        params.multi_dimensional_cost = Some(single - U256::from(20_000u64) * U256::from(ONE_GWEI));
        let (_, transfers) = run(&p, &params);
        let log = transfers.borrow();
        let refund = single - params.multi_dimensional_cost.unwrap();
        assert!(log
            .iter()
            .any(|(from, to, amt)| *from == NETWORK && *to == REFUND_TO && *amt == refund));
    }

    #[test]
    fn multi_dimensional_refund_skipped_when_estimating() {
        let p = TxProcessor::new(COINBASE);
        let mut params = base_params(true, 60, INFRA);
        params.effective_base_fee = U256::from(2 * ONE_GWEI);
        params.block_base_fee = U256::from(ONE_GWEI);
        let single = params.effective_base_fee * U256::from(params.gas_used);
        params.multi_dimensional_cost = Some(single / U256::from(2u64));
        let (_, transfers) = run(&p, &params);
        let log = transfers.borrow();
        let multi_refund = single - params.multi_dimensional_cost.unwrap();
        assert!(!log
            .iter()
            .any(|(_, to, amt)| *to == REFUND_TO && *amt == multi_refund));
    }

    #[test]
    fn multi_dimensional_no_refund_when_multi_equals_single() {
        let p = TxProcessor::new(COINBASE);
        let mut params = base_params(true, 60, INFRA);
        let single = params.effective_base_fee * U256::from(params.gas_used);
        params.multi_dimensional_cost = Some(single);
        let (_, transfers_before) = run(&p, &base_params(true, 60, INFRA));
        let (_, transfers_with) = run(&p, &params);
        assert_eq!(
            transfers_before.borrow().len(),
            transfers_with.borrow().len()
        );
    }

    #[test]
    fn multi_dimensional_no_refund_when_multi_above_single() {
        let p = TxProcessor::new(COINBASE);
        let mut params = base_params(true, 60, INFRA);
        let single = params.effective_base_fee * U256::from(params.gas_used);
        params.multi_dimensional_cost = Some(single * U256::from(2u64));
        let (_, transfers_before) = run(&p, &base_params(true, 60, INFRA));
        let (_, transfers_with) = run(&p, &params);
        assert_eq!(
            transfers_before.borrow().len(),
            transfers_with.borrow().len()
        );
    }
}

mod batch_poster_funds_due {
    use super::*;
    use alloy_primitives::address;

    /// `BatchPostersTable::set_funds_due` adjusts total funds due by
    /// `prev_total + value - prev`. Test the simple positive case.
    #[test]
    fn set_funds_due_updates_total_correctly() {
        let mut h = ArbosHarness::new().initialize();
        let l1 = h.l1_pricing_state();
        let bpt = l1.batch_poster_table();
        let a = address!("AAAA000000000000000000000000000000000000");
        let b = address!("BBBB000000000000000000000000000000000000");
        let bp_a = bpt.add_poster(a, a).unwrap();
        let bp_b = bpt.add_poster(b, b).unwrap();

        bp_a.set_funds_due(U256::from(100u64), &bpt.total_funds_due)
            .unwrap();
        bp_b.set_funds_due(U256::from(50u64), &bpt.total_funds_due)
            .unwrap();
        assert_eq!(bpt.total_funds_due().unwrap(), U256::from(150u64));

        bp_a.set_funds_due(U256::from(30u64), &bpt.total_funds_due)
            .unwrap();
        assert_eq!(bpt.total_funds_due().unwrap(), U256::from(80u64));
    }

    /// `set_funds_due(0)` reduces total by the previous funds_due value.
    #[test]
    fn set_funds_due_to_zero_decreases_total() {
        let mut h = ArbosHarness::new().initialize();
        let l1 = h.l1_pricing_state();
        let bpt = l1.batch_poster_table();
        let a = address!("AAAA000000000000000000000000000000000000");
        let bp = bpt.add_poster(a, a).unwrap();
        bp.set_funds_due(U256::from(500u64), &bpt.total_funds_due)
            .unwrap();
        assert_eq!(bpt.total_funds_due().unwrap(), U256::from(500u64));
        bp.set_funds_due(U256::ZERO, &bpt.total_funds_due).unwrap();
        assert_eq!(bpt.total_funds_due().unwrap(), U256::ZERO);
    }
}

mod l1_pricing_surplus {
    use super::*;

    /// Initial L1 pricing surplus is positive (or zero), and equals
    /// `l1_fees_available - total_funds_due`.
    #[test]
    fn surplus_at_genesis_is_zero() {
        let mut h = ArbosHarness::new().initialize();
        let l1 = h.l1_pricing_state();
        let (mag, neg) = l1.get_l1_pricing_surplus().unwrap();
        assert_eq!(mag, U256::ZERO);
        assert!(!neg);
    }

    /// After adding fees with no debt, surplus equals fees_available.
    #[test]
    fn surplus_grows_with_l1_fees_available() {
        let mut h = ArbosHarness::new().initialize();
        let l1 = h.l1_pricing_state();
        l1.add_to_l1_fees_available(U256::from(1_000_000u64))
            .unwrap();
        let (mag, neg) = l1.get_l1_pricing_surplus().unwrap();
        assert_eq!(mag, U256::from(1_000_000u64));
        assert!(!neg);
    }
}

mod retryable_submission_fee_overflow {
    use super::*;

    #[test]
    fn submission_fee_handles_large_calldata_without_overflow() {
        let l1 = U256::from(ONE_GWEI);
        let len = 1_000_000_000usize;
        let fee = retryable_submission_fee(len, l1);
        let expected = U256::from(1400u64 + 6 * len as u64) * l1;
        assert_eq!(fee, expected);
    }

    #[test]
    fn submission_fee_calldata_at_u64_max_does_not_panic() {
        let l1 = U256::from(1u64);
        let len = u64::MAX as usize;
        let _ = retryable_submission_fee(len, l1);
    }

    #[test]
    fn submission_fee_with_u256_max_l1_base_fee_does_not_panic() {
        let _ = retryable_submission_fee(100, U256::MAX);
    }
}

mod compute_poster_gas_overflow {
    use super::*;
    use arbos::tx_processor::compute_poster_gas;

    #[test]
    fn compute_poster_gas_with_u256_max_cost_does_not_panic() {
        let _ = compute_poster_gas(U256::MAX, U256::from(ONE_GWEI), false, U256::ZERO);
    }

    #[test]
    fn compute_poster_gas_with_u256_max_cost_estimation_mode_does_not_panic() {
        let _ = compute_poster_gas(
            U256::MAX,
            U256::from(ONE_GWEI),
            true,
            U256::from(ONE_GWEI / 2),
        );
    }

    #[test]
    fn compute_poster_gas_returns_u64_max_when_cost_overflows() {
        let huge = U256::MAX;
        let g = compute_poster_gas(huge, U256::from(1u64), false, U256::ZERO);
        assert_eq!(g, u64::MAX);
    }
}

mod compute_retryable_gas_split_overflow {
    use super::*;
    use alloy_primitives::Address;
    use arbos::tx_processor::compute_retryable_gas_split;

    #[test]
    fn retryable_gas_split_with_u64_max_gas_does_not_panic() {
        let _ = compute_retryable_gas_split(
            u64::MAX,
            U256::from(ONE_GWEI),
            Address::ZERO,
            U256::ZERO,
            10,
        );
    }

    #[test]
    fn retryable_gas_split_with_max_base_fee_does_not_panic() {
        let _ = compute_retryable_gas_split(1_000_000, U256::MAX, Address::ZERO, U256::ZERO, 10);
    }
}

mod retryable_lifecycle_edge_cases {
    use super::*;
    use alloy_primitives::{address, B256};

    #[test]
    fn open_retryable_at_u64_max_timestamp_does_not_panic() {
        let mut h = ArbosHarness::new().initialize();
        let rs = h.retryable_state();
        let id = B256::repeat_byte(0xCC);
        rs.create_retryable(
            id,
            u64::MAX,
            address!("AAAA000000000000000000000000000000000000"),
            None,
            U256::from(1u64),
            address!("BBBB000000000000000000000000000000000000"),
            &[],
        )
        .unwrap();
        let _ = rs.open_retryable(id, u64::MAX);
    }

    #[test]
    fn create_retryable_with_max_calldata_size() {
        let mut h = ArbosHarness::new().initialize();
        let rs = h.retryable_state();
        let id = B256::repeat_byte(0xDD);
        let big_data = vec![0xAB; 100_000];
        let result = rs.create_retryable(
            id,
            10_000,
            address!("AAAA000000000000000000000000000000000000"),
            None,
            U256::ZERO,
            address!("BBBB000000000000000000000000000000000000"),
            &big_data,
        );
        assert!(result.is_ok());
    }
}
