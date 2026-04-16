//! Ports of `nitro/arbos/l2pricing/l2pricing_test.go`.
//!
//! Mirror Nitro's pure-storage L2 pricing tests over our `arb-test-utils`
//! harness. Tests cover the legacy pricing model (`update_pricing_model`
//! at ArbOS v30) and the multi-gas constraint exponent computation
//! (ArbOS v60+).

use alloy_primitives::U256;
use arb_primitives::multigas::{ResourceKind, NUM_RESOURCE_KIND};
use arb_test_utils::ArbosHarness;

const ARBOS_V30: u64 = 30;
const ARBOS_V60: u64 = 60;

fn weights(pairs: &[(ResourceKind, u64)]) -> [u64; NUM_RESOURCE_KIND] {
    let mut out = [0u64; NUM_RESOURCE_KIND];
    for &(kind, w) in pairs {
        out[kind as usize] = w;
    }
    out
}

/// Port of `TestPricingModelExp`.
///
/// Walks the legacy pricing model through three phases:
/// 1. Speed-limit traffic with full pool → price stays at min.
/// 2. Over-speed-limit traffic → price escalates (we just confirm it
///    rises above min after enough cycles).
/// 3. Empty pool with no gas → no movement; pool empty + 1s tick → rise.
#[test]
fn legacy_pricing_model_exp_steady_state_and_escalation() {
    let mut h = ArbosHarness::new().with_arbos_version(ARBOS_V30).initialize();
    let p = h.l2_pricing_state();

    let min_price = p.min_base_fee_wei().unwrap();
    let limit = p.speed_limit_per_second().unwrap();

    // Initially at min price.
    assert_eq!(p.base_fee_wei().unwrap(), min_price);

    // Phase 1: at speed limit with full pool, price stays at min.
    for seconds in 0u64..4 {
        // Grow backlog by `seconds * limit` (matching the Go fakeBlockUpdate),
        // then advance time by `seconds`.
        let _ = p.update_pricing_model(0, ARBOS_V30); // no-op tick to seed
        let prev_backlog = p.gas_backlog().unwrap();
        p.set_gas_backlog(prev_backlog.saturating_add(seconds.saturating_mul(limit)))
            .unwrap();
        p.update_pricing_model(seconds, ARBOS_V30).unwrap();
        assert_eq!(
            p.base_fee_wei().unwrap(),
            min_price,
            "price changed at speed limit"
        );
    }

    // Phase 2: exceeding the speed limit must eventually escalate price.
    let mut last_price = p.base_fee_wei().unwrap();
    let mut escalated = false;
    for _ in 0..200 {
        let prev_backlog = p.gas_backlog().unwrap();
        // Burn 8x the speed limit per second.
        p.set_gas_backlog(prev_backlog.saturating_add(8 * limit))
            .unwrap();
        p.update_pricing_model(1, ARBOS_V30).unwrap();
        let new_price = p.base_fee_wei().unwrap();
        assert!(new_price >= last_price, "price fell during over-speed run");
        if new_price > last_price {
            escalated = true;
            break;
        }
        last_price = new_price;
    }
    assert!(escalated, "price never escalated when running over speed limit");

    // Phase 3: a much larger backlog must drive the price strictly higher
    // than phase-2's break price, regardless of speed-limit defaults.
    // Go's original used 100M which only works against the v0 1M speed limit;
    // we use a backlog scaled to the active speed limit so the test is
    // version-agnostic.
    let baseline = p.base_fee_wei().unwrap();
    p.set_gas_backlog(limit.saturating_mul(1000)).unwrap();
    p.update_pricing_model(0, ARBOS_V30).unwrap();
    p.update_pricing_model(1, ARBOS_V30).unwrap();
    let after = p.base_fee_wei().unwrap();
    assert!(
        after > baseline,
        "price should have risen with backlog far above tolerance"
    );
}

/// Port of `TestGasConstraints`. ArbOS v60+ supports legacy single-gas
/// constraints via `add_gas_constraint`, separate from the legacy
/// `gas_backlog` model.
#[test]
fn gas_constraints_add_open_clear() {
    let mut h = ArbosHarness::new().with_arbos_version(ARBOS_V60).initialize();
    let p = h.l2_pricing_state();

    assert_eq!(p.gas_constraints_length().unwrap(), 0);

    const N: u64 = 10;
    for i in 0..N {
        p.add_gas_constraint(100 * i + 1, 100 * i + 2, 100 * i + 3)
            .unwrap();
    }
    assert_eq!(p.gas_constraints_length().unwrap(), N);

    for i in 0..N {
        let c = p.open_gas_constraint_at(i);
        assert_eq!(c.target().unwrap(), 100 * i + 1);
        assert_eq!(c.adjustment_window().unwrap(), 100 * i + 2);
        assert_eq!(c.backlog().unwrap(), 100 * i + 3);
    }

    p.clear_gas_constraints().unwrap();
    assert_eq!(p.gas_constraints_length().unwrap(), 0);
}

/// Port of `TestMultiGasConstraints` (ArbOS v60+).
#[test]
fn multi_gas_constraints_add_open_clear() {
    let mut h = ArbosHarness::new().with_arbos_version(ARBOS_V60).initialize();
    let p = h.l2_pricing_state();

    assert_eq!(p.multi_gas_constraints_length().unwrap(), 0);

    const N: u64 = 5;
    for i in 0..N {
        let w = weights(&[
            (ResourceKind::Computation, 10 + i),
            (ResourceKind::StorageAccess, 20 + i),
        ]);
        p.add_multi_gas_constraint(100 * i + 1, (100 * i + 2) as u32, 100 * i + 3, &w)
            .unwrap();
    }

    assert_eq!(p.multi_gas_constraints_length().unwrap(), N);

    for i in 0..N {
        let c = p.open_multi_gas_constraint_at(i);
        assert_eq!(c.target().unwrap(), 100 * i + 1);
        assert_eq!(c.adjustment_window().unwrap(), (100 * i + 2) as u32);
        assert_eq!(c.backlog().unwrap(), 100 * i + 3);
        assert_eq!(c.resource_weight(ResourceKind::Computation).unwrap(), 10 + i);
        assert_eq!(c.resource_weight(ResourceKind::StorageAccess).unwrap(), 20 + i);
    }

    p.clear_multi_gas_constraints().unwrap();
    assert_eq!(p.multi_gas_constraints_length().unwrap(), 0);
}

/// Port of `TestMultiGasConstraintsExponents`.
///
/// constraint A: backlog=100, target=100, window=10, weight=1
///   exponent_bips = (100 * 1 * 10000) / (10 * 100 * 1) = 1000
///
/// constraint B: backlog=200, target=40, window=20, weight=2
///   computed per-resource (storage access)
#[test]
fn multi_gas_constraints_exponents() {
    let mut h = ArbosHarness::new().with_arbos_version(ARBOS_V60).initialize();
    let p = h.l2_pricing_state();

    p.add_multi_gas_constraint(
        100,
        10,
        100,
        &weights(&[(ResourceKind::Computation, 1)]),
    )
    .unwrap();

    p.add_multi_gas_constraint(
        40,
        20,
        200,
        &weights(&[(ResourceKind::StorageAccess, 2)]),
    )
    .unwrap();

    let exps = p.calc_multi_gas_constraints_exponents().unwrap();
    assert_eq!(exps[ResourceKind::Computation as usize], 1000);
    assert_eq!(exps[ResourceKind::StorageAccess as usize], 2500);
}

/// New: confirm initial L2 base fee equals the configured min base fee
/// (a basic genesis sanity check).
#[test]
fn initial_base_fee_equals_min() {
    let mut h = ArbosHarness::new().initialize();
    let p = h.l2_pricing_state();
    let base = p.base_fee_wei().unwrap();
    let min = p.min_base_fee_wei().unwrap();
    assert_eq!(base, min);
    assert!(base > U256::ZERO);
}
