use alloy_primitives::{address, B256, U256};
use arb_test_utils::ArbosHarness;
use arbos::address_set::{initialize_address_set, open_address_set};

/// Two harnesses created back-to-back must not see each other's state.
/// Caught real bugs in ArbosState lifecycle when slot keys collide across
/// harness lifetimes.
#[test]
fn fresh_harness_does_not_inherit_previous_state() {
    let addr = address!("AAAA000000000000000000000000000000000000");

    let mut h1 = ArbosHarness::new().initialize();
    let root1 = h1.root_storage();
    let s1 = root1.open_sub_storage(&[0x77]);
    initialize_address_set(&s1).unwrap();
    let set1 = open_address_set(s1);
    set1.add(addr).unwrap();
    assert_eq!(set1.size().unwrap(), 1);

    let mut h2 = ArbosHarness::new().initialize();
    let root2 = h2.root_storage();
    let s2 = root2.open_sub_storage(&[0x77]);
    initialize_address_set(&s2).unwrap();
    let set2 = open_address_set(s2);
    assert_eq!(set2.size().unwrap(), 0);
    assert!(!set2.is_member(addr).unwrap());
}

/// Re-opening the same harness's state should observe prior writes.
#[test]
fn reopened_state_observes_prior_writes() {
    let addr = address!("BBBB000000000000000000000000000000000000");
    let mut h = ArbosHarness::new().initialize();

    {
        let root = h.root_storage();
        let s = root.open_sub_storage(&[0x88]);
        initialize_address_set(&s).unwrap();
        let set = open_address_set(s);
        set.add(addr).unwrap();
    }

    {
        let root = h.root_storage();
        let set = open_address_set(root.open_sub_storage(&[0x88]));
        assert_eq!(set.size().unwrap(), 1);
        assert!(set.is_member(addr).unwrap());
    }
}

/// L1 pricing state mutations from a prior harness must not appear in a
/// freshly-built one with the same configuration.
#[test]
fn l1_pricing_state_is_isolated_per_harness() {
    {
        let mut h = ArbosHarness::new().initialize();
        let l1 = h.l1_pricing_state();
        l1.set_units_since_update(99_999).unwrap();
        l1.set_price_per_unit(U256::from(987_654_321u64)).unwrap();
    }

    let mut h = ArbosHarness::new().initialize();
    let l1 = h.l1_pricing_state();
    assert_eq!(l1.units_since_update().unwrap(), 0);
    assert!(l1.price_per_unit().unwrap() < U256::from(987_654_321u64));
}

/// Repeated initialize() at the same arbos version yields identical
/// observable state for read-only fields. (Soft equivalence — slot
/// values are deterministic for given config.)
#[test]
fn repeated_initialize_yields_deterministic_state() {
    let read = || {
        let mut h = ArbosHarness::new()
            .with_arbos_version(30)
            .with_chain_id(421614)
            .with_l1_initial_base_fee(U256::from(500_000_000u64))
            .initialize();
        let l1 = h.l1_pricing_state();
        let l2 = h.l2_pricing_state();
        (
            l1.price_per_unit().unwrap(),
            l1.inertia().unwrap(),
            l2.min_base_fee_wei().unwrap(),
            l2.speed_limit_per_second().unwrap(),
        )
    };
    let a = read();
    let b = read();
    let c = read();
    assert_eq!(a, b);
    assert_eq!(b, c);
}

/// A retryable created in one harness must not be visible from a fresh one.
#[test]
fn retryables_isolated_per_harness() {
    let id = B256::repeat_byte(0xAB);
    {
        let mut h = ArbosHarness::new().initialize();
        let rs = h.retryable_state();
        rs.create_retryable(
            id,
            10_000,
            address!("CCCC000000000000000000000000000000000000"),
            None,
            U256::from(1u64),
            address!("DDDD000000000000000000000000000000000000"),
            &[],
        )
        .unwrap();
        assert!(rs.open_retryable(id, 100).unwrap().is_some());
    }
    let mut h = ArbosHarness::new().initialize();
    let rs = h.retryable_state();
    assert!(rs.open_retryable(id, 100).unwrap().is_none());
}
