use alloy_primitives::{address, Address};
use arb_test_utils::ArbosHarness;
use arbos::address_table::open_address_table;

fn fresh_table(
    h: &mut ArbosHarness,
    sub_id: u8,
) -> arbos::address_table::AddressTable<arb_test_utils::EmptyDb> {
    let root = h.root_storage();
    open_address_table(root.open_sub_storage(&[sub_id]))
}

#[test]
fn empty_table_size_zero_lookup_misses() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let t = fresh_table(&mut h, 0xB0);
    let b = unsafe { &mut *state_ptr };
    assert_eq!(t.size(b).unwrap(), 0);
    assert_eq!(t.lookup(b, Address::ZERO).unwrap(), (0, false));
    assert!(t.lookup_index(b, 0).unwrap().is_none());
    assert!(!t.address_exists(b, Address::ZERO).unwrap());
}

#[test]
fn register_returns_sequential_indices() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let t = fresh_table(&mut h, 0xB1);
    let b = unsafe { &mut *state_ptr };
    let a = address!("AAAA000000000000000000000000000000000000");
    let bb = address!("BBBB000000000000000000000000000000000000");
    let c = address!("CCCC000000000000000000000000000000000000");
    assert_eq!(t.register(b, a).unwrap(), (0, false));
    assert_eq!(t.register(b, bb).unwrap(), (1, false));
    assert_eq!(t.register(b, c).unwrap(), (2, false));
    assert_eq!(t.register(b, a).unwrap(), (0, true));
    assert_eq!(t.size(b).unwrap(), 3);
}

#[test]
fn lookup_round_trips_index_and_address() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let t = fresh_table(&mut h, 0xB2);
    let b = unsafe { &mut *state_ptr };
    let a = address!("DEADBEEF00000000000000000000000000000000");
    let (idx, _) = t.register(b, a).unwrap();
    assert_eq!(t.lookup(b, a).unwrap(), (idx, true));
    assert_eq!(t.lookup_index(b, idx).unwrap(), Some(a));
}

#[test]
fn compress_indexes_short_for_registered_addresses() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let t = fresh_table(&mut h, 0xB5);
    let b = unsafe { &mut *state_ptr };
    for i in 0u8..10 {
        let mut bytes = [0u8; 20];
        bytes[19] = i;
        t.register(b, Address::from(bytes)).unwrap();
    }
    let mut bytes = [0u8; 20];
    bytes[19] = 5;
    let compressed = t.compress(b, Address::from(bytes)).unwrap();
    assert!(compressed.len() < 20);
}

#[test]
fn compress_full_address_for_unregistered() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let t = fresh_table(&mut h, 0xB6);
    let b = unsafe { &mut *state_ptr };
    let a = address!("FACEFEED00000000000000000000000000000000");
    let compressed = t.compress(b, a).unwrap();
    assert!(compressed.len() >= 20);
}

#[test]
fn lookup_index_returns_none_out_of_range() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let t = fresh_table(&mut h, 0xB7);
    let b = unsafe { &mut *state_ptr };
    let a = address!("AAAA000000000000000000000000000000000000");
    t.register(b, a).unwrap();
    assert_eq!(t.lookup_index(b, 0).unwrap(), Some(a));
    assert!(t.lookup_index(b, 1).unwrap().is_none());
    assert!(t.lookup_index(b, 999_999).unwrap().is_none());
}
