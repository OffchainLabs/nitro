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

#[test]
fn legacy_pricing_model_steady_state_and_escalation() {
    let mut h = ArbosHarness::new()
        .with_arbos_version(ARBOS_V30)
        .initialize();
    let p = h.l2_pricing_state();

    let min_price = p.min_base_fee_wei().unwrap();
    let limit = p.speed_limit_per_second().unwrap();
    assert_eq!(p.base_fee_wei().unwrap(), min_price);

    for seconds in 0u64..4 {
        let prev = p.gas_backlog().unwrap();
        p.set_gas_backlog(prev.saturating_add(seconds.saturating_mul(limit)))
            .unwrap();
        p.update_pricing_model(seconds, ARBOS_V30).unwrap();
        assert_eq!(p.base_fee_wei().unwrap(), min_price);
    }

    let mut last = p.base_fee_wei().unwrap();
    let mut escalated = false;
    for _ in 0..200 {
        let prev = p.gas_backlog().unwrap();
        p.set_gas_backlog(prev.saturating_add(8 * limit)).unwrap();
        p.update_pricing_model(1, ARBOS_V30).unwrap();
        let new_price = p.base_fee_wei().unwrap();
        assert!(new_price >= last);
        if new_price > last {
            escalated = true;
            break;
        }
        last = new_price;
    }
    assert!(escalated);

    let baseline = p.base_fee_wei().unwrap();
    p.set_gas_backlog(limit.saturating_mul(1000)).unwrap();
    p.update_pricing_model(0, ARBOS_V30).unwrap();
    p.update_pricing_model(1, ARBOS_V30).unwrap();
    assert!(p.base_fee_wei().unwrap() > baseline);
}

#[test]
fn gas_constraints_add_open_clear() {
    let mut h = ArbosHarness::new()
        .with_arbos_version(ARBOS_V60)
        .initialize();
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

#[test]
fn multi_gas_constraints_add_open_clear() {
    let mut h = ArbosHarness::new()
        .with_arbos_version(ARBOS_V60)
        .initialize();
    let p = h.l2_pricing_state();

    assert_eq!(p.multi_gas_constraints_length().unwrap(), 0);

    const N: u64 = 5;
    for i in 0..N {
        let w = weights(&[
            (ResourceKind::Computation, 10 + i),
            (ResourceKind::StorageAccessRead, 20 + i),
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
        assert_eq!(
            c.resource_weight(ResourceKind::Computation).unwrap(),
            10 + i
        );
        assert_eq!(
            c.resource_weight(ResourceKind::StorageAccessRead).unwrap(),
            20 + i
        );
    }

    p.clear_multi_gas_constraints().unwrap();
    assert_eq!(p.multi_gas_constraints_length().unwrap(), 0);
}

#[test]
fn multi_gas_constraints_exponents() {
    let mut h = ArbosHarness::new()
        .with_arbos_version(ARBOS_V60)
        .initialize();
    let p = h.l2_pricing_state();

    p.add_multi_gas_constraint(100, 10, 100, &weights(&[(ResourceKind::Computation, 1)]))
        .unwrap();
    p.add_multi_gas_constraint(40, 20, 200, &weights(&[(ResourceKind::StorageAccessRead, 2)]))
        .unwrap();

    let exps = p.calc_multi_gas_constraints_exponents().unwrap();
    assert_eq!(exps[ResourceKind::Computation as usize], 1000);
    assert_eq!(exps[ResourceKind::StorageAccessRead as usize], 2500);
}

#[test]
fn initial_base_fee_equals_min() {
    let mut h = ArbosHarness::new().initialize();
    let p = h.l2_pricing_state();
    let base = p.base_fee_wei().unwrap();
    let min = p.min_base_fee_wei().unwrap();
    assert_eq!(base, min);
    assert!(base > U256::ZERO);
}
