use arb_primitives::multigas::{MultiGas, ResourceKind, NUM_RESOURCE_KIND};
use proptest::prelude::*;

fn multigas_strategy() -> impl Strategy<Value = MultiGas> {
    (
        prop::array::uniform8(any::<u64>()),
        any::<u64>(),
        any::<u64>(),
    )
        .prop_map(|(gas, total, refund)| MultiGas::from_raw(gas, total, refund))
}

#[test]
fn zero_is_identity_for_add() {
    let z = MultiGas::zero();
    let g = MultiGas::computation_gas(1_000);
    assert_eq!(g.saturating_add(z), g);
    assert_eq!(z.saturating_add(g), g);
}

#[test]
fn safe_add_overflow_returns_original() {
    let a = MultiGas::computation_gas(u64::MAX);
    let b = MultiGas::computation_gas(1);
    let (res, overflow) = a.safe_add(b);
    assert!(overflow);
    assert_eq!(res, a);
}

#[test]
fn safe_sub_underflow_returns_original() {
    let a = MultiGas::computation_gas(10);
    let b = MultiGas::computation_gas(20);
    let (res, underflow) = a.safe_sub(b);
    assert!(underflow);
    assert_eq!(res, a);
}

#[test]
fn saturating_add_caps_at_max() {
    let a = MultiGas::computation_gas(u64::MAX);
    let b = MultiGas::computation_gas(100);
    let r = a.saturating_add(b);
    assert_eq!(r.get(ResourceKind::Computation), u64::MAX);
}

#[test]
fn saturating_sub_caps_at_zero() {
    let a = MultiGas::computation_gas(5);
    let b = MultiGas::computation_gas(100);
    let r = a.saturating_sub(b);
    assert_eq!(r.get(ResourceKind::Computation), 0);
}

#[test]
fn from_pairs_aggregates_kinds() {
    let g = MultiGas::from_pairs(&[
        (ResourceKind::Computation, 100),
        (ResourceKind::StorageAccess, 200),
    ]);
    assert_eq!(g.get(ResourceKind::Computation), 100);
    assert_eq!(g.get(ResourceKind::StorageAccess), 200);
    assert_eq!(g.get(ResourceKind::HistoryGrowth), 0);
}

proptest! {
    #[test]
    fn add_then_sub_round_trips(a in multigas_strategy(), b in multigas_strategy()) {
        let (sum, overflowed) = a.safe_add(b);
        if !overflowed {
            let (back, underflowed) = sum.safe_sub(b);
            prop_assert!(!underflowed);
            for i in 0..NUM_RESOURCE_KIND {
                let kind = ResourceKind::from_u8(i as u8).unwrap();
                prop_assert_eq!(back.get(kind), a.get(kind));
            }
        }
    }

    #[test]
    fn saturating_add_is_associative_for_disjoint_kinds(a in 0u64..1_000_000, b in 0u64..1_000_000, c in 0u64..1_000_000) {
        let g = MultiGas::computation_gas(a)
            .saturating_add(MultiGas::computation_gas(b))
            .saturating_add(MultiGas::computation_gas(c));
        let g2 = MultiGas::computation_gas(a)
            .saturating_add(MultiGas::computation_gas(b).saturating_add(MultiGas::computation_gas(c)));
        prop_assert_eq!(g.get(ResourceKind::Computation), g2.get(ResourceKind::Computation));
    }

    #[test]
    fn saturating_increment_matches_saturating_add(kind_idx in 0u8..(NUM_RESOURCE_KIND as u8), val in any::<u64>()) {
        let kind = ResourceKind::from_u8(kind_idx).unwrap();
        let inc = MultiGas::zero().saturating_increment(kind, val);
        let added = MultiGas::zero().saturating_add(MultiGas::new(kind, val));
        prop_assert_eq!(inc.get(kind), added.get(kind));
        prop_assert_eq!(inc.total(), added.total());
    }
}
