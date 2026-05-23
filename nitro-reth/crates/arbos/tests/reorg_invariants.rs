use alloy_primitives::{address, B256, U256};
use arb_test_utils::ArbosHarness;
use arbos::address_set::{initialize_address_set, open_address_set};

#[test]
fn fresh_harness_does_not_inherit_previous_state() {
    let addr = address!("AAAA000000000000000000000000000000000000");

    let mut h1 = ArbosHarness::new().initialize();
    let sp1 = h1.state_ptr();
    let root1 = h1.root_storage();
    let s1 = root1.open_sub_storage(&[0x77]);
    initialize_address_set(&s1).unwrap();
    let set1 = open_address_set(s1);
    let b1 = unsafe { &mut *sp1 };
    set1.add(b1, addr).unwrap();
    assert_eq!(set1.size(b1).unwrap(), 1);

    let mut h2 = ArbosHarness::new().initialize();
    let sp2 = h2.state_ptr();
    let root2 = h2.root_storage();
    let s2 = root2.open_sub_storage(&[0x77]);
    initialize_address_set(&s2).unwrap();
    let set2 = open_address_set(s2);
    let b2 = unsafe { &mut *sp2 };
    assert_eq!(set2.size(b2).unwrap(), 0);
    assert!(!set2.is_member(addr).unwrap());
}

#[test]
fn reopened_state_observes_prior_writes() {
    let addr = address!("BBBB000000000000000000000000000000000000");
    let mut h = ArbosHarness::new().initialize();
    let sp = h.state_ptr();

    {
        let root = h.root_storage();
        let s = root.open_sub_storage(&[0x88]);
        initialize_address_set(&s).unwrap();
        let set = open_address_set(s);
        set.add(unsafe { &mut *sp }, addr).unwrap();
    }

    {
        let root = h.root_storage();
        let set = open_address_set(root.open_sub_storage(&[0x88]));
        assert_eq!(set.size(unsafe { &mut *sp }).unwrap(), 1);
        assert!(set.is_member(addr).unwrap());
    }
}

#[test]
fn l1_pricing_state_is_isolated_per_harness() {
    {
        let mut h = ArbosHarness::new().initialize();
        let sp = h.state_ptr();
        let l1 = h.l1_pricing_state();
        let b = unsafe { &mut *sp };
        l1.set_units_since_update(b, 99_999).unwrap();
        l1.set_price_per_unit(U256::from(987_654_321u64)).unwrap();
    }

    let mut h = ArbosHarness::new().initialize();
    let sp = h.state_ptr();
    let l1 = h.l1_pricing_state();
    let b = unsafe { &mut *sp };
    assert_eq!(l1.units_since_update(b).unwrap(), 0);
    assert!(l1.price_per_unit().unwrap() < U256::from(987_654_321u64));
}

#[test]
fn repeated_initialize_yields_deterministic_state() {
    let read = || {
        let mut h = ArbosHarness::new()
            .with_arbos_version(30)
            .with_chain_id(421614)
            .with_l1_initial_base_fee(U256::from(500_000_000u64))
            .initialize();
        let sp = h.state_ptr();
        let l1 = h.l1_pricing_state();
        let l2 = h.l2_pricing_state();
        let b = unsafe { &mut *sp };
        (
            l1.price_per_unit().unwrap(),
            l1.inertia(b).unwrap(),
            l2.min_base_fee_wei().unwrap(),
            l2.speed_limit_per_second(b).unwrap(),
        )
    };
    let a = read();
    let b = read();
    let c = read();
    assert_eq!(a, b);
    assert_eq!(b, c);
}

#[test]
fn retryables_isolated_per_harness() {
    let id = B256::repeat_byte(0xAB);
    {
        let mut h = ArbosHarness::new().initialize();
        let sp = h.state_ptr();
        let rs = h.retryable_state();
        let b = unsafe { &mut *sp };
        rs.create_retryable(
            b,
            id,
            10_000,
            address!("CCCC000000000000000000000000000000000000"),
            None,
            U256::from(1u64),
            address!("DDDD000000000000000000000000000000000000"),
            &[],
        )
        .unwrap();
        assert!(rs.open_retryable(b, id, 100).unwrap().is_some());
    }
    let mut h = ArbosHarness::new().initialize();
    let sp = h.state_ptr();
    let rs = h.retryable_state();
    let b = unsafe { &mut *sp };
    assert!(rs.open_retryable(b, id, 100).unwrap().is_none());
}
