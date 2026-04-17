use std::collections::HashSet;

use alloy_primitives::{address, Address};
use arb_test_utils::ArbosHarness;
use arbos::address_set::{initialize_address_set, open_address_set};

const ARBOS_V30: u64 = 30;

fn fresh_set(
    harness: &mut ArbosHarness,
    sub_id: u8,
) -> arbos::address_set::AddressSet<arb_test_utils::EmptyDb> {
    let root = harness.root_storage();
    let sto = root.open_sub_storage(&[sub_id]);
    initialize_address_set(&sto).unwrap();
    open_address_set(sto)
}

#[test]
fn empty_set_size_zero_membership_false() {
    let mut h = ArbosHarness::new().initialize();
    let s = fresh_set(&mut h, 0xA0);
    assert_eq!(s.size().unwrap(), 0);
    assert!(!s.is_member(Address::ZERO).unwrap());
    s.remove(Address::ZERO, ARBOS_V30).unwrap();
    assert_eq!(s.size().unwrap(), 0);
    assert!(s.get_any_member().unwrap().is_none());
}

#[test]
fn add_remove_size_consistency() {
    let mut h = ArbosHarness::new().initialize();
    let s = fresh_set(&mut h, 0xA1);

    let a1 = address!("1111111111111111111111111111111111111111");
    let a2 = address!("2222222222222222222222222222222222222222");
    let a3 = address!("3333333333333333333333333333333333333333");

    s.add(a1).unwrap();
    assert_eq!(s.size().unwrap(), 1);
    s.add(a2).unwrap();
    assert_eq!(s.size().unwrap(), 2);
    s.add(a1).unwrap();
    assert_eq!(s.size().unwrap(), 2);

    assert!(s.is_member(a1).unwrap());
    assert!(s.is_member(a2).unwrap());
    assert!(!s.is_member(a3).unwrap());

    s.remove(a1, ARBOS_V30).unwrap();
    assert_eq!(s.size().unwrap(), 1);
    assert!(!s.is_member(a1).unwrap());
    assert!(s.is_member(a2).unwrap());

    s.add(a3).unwrap();
    assert_eq!(s.size().unwrap(), 2);
    s.remove(a3, ARBOS_V30).unwrap();
    assert_eq!(s.size().unwrap(), 1);

    s.add(a1).unwrap();
    let all: HashSet<Address> = s.all_members(u64::MAX).unwrap().into_iter().collect();
    assert_eq!(all, HashSet::from([a1, a2]));
}

#[test]
fn clear_resets_size_and_membership() {
    let mut h = ArbosHarness::new().initialize();
    let s = fresh_set(&mut h, 0xA2);
    let a1 = address!("1111111111111111111111111111111111111111");
    let a2 = address!("2222222222222222222222222222222222222222");
    s.add(a1).unwrap();
    s.add(a2).unwrap();
    assert_eq!(s.size().unwrap(), 2);
    s.clear().unwrap();
    assert_eq!(s.size().unwrap(), 0);
    assert!(!s.is_member(a1).unwrap());
    assert!(!s.is_member(a2).unwrap());
}

#[test]
fn all_members_caps_at_max_num() {
    let mut h = ArbosHarness::new().initialize();
    let s = fresh_set(&mut h, 0xA3);
    for i in 1u8..=5 {
        let mut bytes = [0u8; 20];
        bytes[19] = i;
        s.add(Address::from(bytes)).unwrap();
    }
    assert_eq!(s.all_members(3).unwrap().len(), 3);
    assert_eq!(s.all_members(10).unwrap().len(), 5);
}

#[test]
fn random_add_remove_keeps_invariants() {
    use std::collections::BTreeSet;
    let mut h = ArbosHarness::new().initialize();
    let s = fresh_set(&mut h, 0xA4);

    let pool: Vec<Address> = (1u8..=5)
        .map(|i| {
            let mut b = [0u8; 20];
            b[19] = i;
            Address::from(b)
        })
        .collect();
    let mut model: BTreeSet<Address> = BTreeSet::new();

    let mut state = 0xC0FFEEu64;
    for _ in 0..256 {
        state = state
            .wrapping_mul(6364136223846793005)
            .wrapping_add(1442695040888963407);
        let addr = pool[(state >> 33) as usize % pool.len()];
        let remove = (state & 1) == 0;
        if remove {
            s.remove(addr, ARBOS_V30).unwrap();
            model.remove(&addr);
        } else {
            s.add(addr).unwrap();
            model.insert(addr);
        }
        assert_eq!(s.size().unwrap() as usize, model.len());
        for a in &pool {
            assert_eq!(s.is_member(*a).unwrap(), model.contains(a));
        }
    }
}
