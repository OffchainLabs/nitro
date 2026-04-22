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
    let t = fresh_table(&mut h, 0xB0);
    assert_eq!(t.size().unwrap(), 0);
    assert_eq!(t.lookup(Address::ZERO).unwrap(), (0, false));
    assert!(t.lookup_index(0).unwrap().is_none());
    assert!(!t.address_exists(Address::ZERO).unwrap());
}

#[test]
fn register_returns_sequential_indices() {
    let mut h = ArbosHarness::new().initialize();
    let t = fresh_table(&mut h, 0xB1);
    let a = address!("AAAA000000000000000000000000000000000000");
    let b = address!("BBBB000000000000000000000000000000000000");
    let c = address!("CCCC000000000000000000000000000000000000");
    assert_eq!(t.register(a).unwrap(), 0);
    assert_eq!(t.register(b).unwrap(), 1);
    assert_eq!(t.register(c).unwrap(), 2);
    assert_eq!(t.register(a).unwrap(), 0);
    assert_eq!(t.size().unwrap(), 3);
}

#[test]
fn lookup_round_trips_index_and_address() {
    let mut h = ArbosHarness::new().initialize();
    let t = fresh_table(&mut h, 0xB2);
    let a = address!("DEADBEEF00000000000000000000000000000000");
    let idx = t.register(a).unwrap();
    assert_eq!(t.lookup(a).unwrap(), (idx, true));
    assert_eq!(t.lookup_index(idx).unwrap(), Some(a));
}

#[test]
fn compress_indexes_short_for_registered_addresses() {
    let mut h = ArbosHarness::new().initialize();
    let t = fresh_table(&mut h, 0xB5);
    for i in 0u8..10 {
        let mut bytes = [0u8; 20];
        bytes[19] = i;
        t.register(Address::from(bytes)).unwrap();
    }
    let mut bytes = [0u8; 20];
    bytes[19] = 5;
    let compressed = t.compress(Address::from(bytes)).unwrap();
    assert!(compressed.len() < 20);
}

#[test]
fn compress_full_address_for_unregistered() {
    let mut h = ArbosHarness::new().initialize();
    let t = fresh_table(&mut h, 0xB6);
    let a = address!("FACEFEED00000000000000000000000000000000");
    let compressed = t.compress(a).unwrap();
    assert!(compressed.len() >= 20);
}

#[test]
fn lookup_index_returns_none_out_of_range() {
    let mut h = ArbosHarness::new().initialize();
    let t = fresh_table(&mut h, 0xB7);
    let a = address!("AAAA000000000000000000000000000000000000");
    t.register(a).unwrap();
    assert_eq!(t.lookup_index(0).unwrap(), Some(a));
    assert!(t.lookup_index(1).unwrap().is_none());
    assert!(t.lookup_index(999_999).unwrap().is_none());
}
