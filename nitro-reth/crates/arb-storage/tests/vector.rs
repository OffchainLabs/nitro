use alloy_primitives::B256;
use arb_storage::vector::open_sub_storage_vector;
use arb_test_utils::ArbosHarness;

fn fresh(
    h: &mut ArbosHarness,
    sub: u8,
) -> arb_storage::vector::SubStorageVector<arb_test_utils::EmptyDb> {
    let root = h.root_storage();
    open_sub_storage_vector(root.open_sub_storage(&[sub]))
}

#[test]
fn empty_vector_length_zero() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let v = fresh(&mut h, 0xF0);
    let b = unsafe { &mut *state_ptr };
    assert_eq!(v.length(b).unwrap(), 0);
}

#[test]
fn push_grows_length() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let v = fresh(&mut h, 0xF1);
    let b = unsafe { &mut *state_ptr };
    v.push(b).unwrap();
    v.push(b).unwrap();
    v.push(b).unwrap();
    assert_eq!(v.length(b).unwrap(), 3);
}

#[test]
fn pushed_items_are_distinct_storages() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let v = fresh(&mut h, 0xF2);
    let b = unsafe { &mut *state_ptr };
    let s0 = v.push(b).unwrap();
    let s1 = v.push(b).unwrap();

    s0.set_by_uint64(0, B256::repeat_byte(0xAA)).unwrap();
    s1.set_by_uint64(0, B256::repeat_byte(0xBB)).unwrap();

    assert_eq!(v.at(0).get_by_uint64(0).unwrap(), B256::repeat_byte(0xAA));
    assert_eq!(v.at(1).get_by_uint64(0).unwrap(), B256::repeat_byte(0xBB));
}

#[test]
fn pop_decrements_and_returns_index() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let v = fresh(&mut h, 0xF3);
    let b = unsafe { &mut *state_ptr };
    v.push(b).unwrap();
    v.push(b).unwrap();
    let popped = v.pop(b).unwrap();
    assert_eq!(popped, Some(1));
    assert_eq!(v.length(b).unwrap(), 1);
}

#[test]
fn pop_on_empty_returns_none() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let v = fresh(&mut h, 0xF4);
    let b = unsafe { &mut *state_ptr };
    assert_eq!(v.pop(b).unwrap(), None);
}
