use alloy_primitives::B256;
use arb_storage::{
    queue::{initialize_queue, open_queue},
    StorageError,
};
use arb_test_utils::ArbosHarness;

fn h_n(n: u8) -> B256 {
    B256::repeat_byte(n)
}

fn fresh_queue(
    h: &mut ArbosHarness,
    sub: u8,
) -> arb_storage::queue::Queue<arb_test_utils::EmptyDb> {
    let root = h.root_storage();
    let sto = root.open_sub_storage(&[sub]);
    initialize_queue(&sto).unwrap();
    open_queue(sto)
}

#[test]
fn empty_queue_basic_invariants() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let q = fresh_queue(&mut h, 0xE0);
    let b = unsafe { &mut *state_ptr };
    assert!(q.is_empty(b).unwrap());
    assert_eq!(q.size(b).unwrap(), 0);
    assert!(q.peek(b).unwrap().is_none());
    assert!(q.get(b).unwrap().is_none());
    assert!(q.shift(b).unwrap().is_none());
}

#[test]
fn put_increments_size() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let q = fresh_queue(&mut h, 0xE1);
    let b = unsafe { &mut *state_ptr };
    q.put(b, h_n(1)).unwrap();
    q.put(b, h_n(2)).unwrap();
    q.put(b, h_n(3)).unwrap();
    assert_eq!(q.size(b).unwrap(), 3);
    assert!(!q.is_empty(b).unwrap());
}

#[test]
fn fifo_get_order() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let q = fresh_queue(&mut h, 0xE2);
    let b = unsafe { &mut *state_ptr };
    for i in 1..=4u8 {
        q.put(b, h_n(i)).unwrap();
    }
    assert_eq!(q.get(b).unwrap(), Some(h_n(1)));
    assert_eq!(q.get(b).unwrap(), Some(h_n(2)));
    assert_eq!(q.get(b).unwrap(), Some(h_n(3)));
    assert_eq!(q.get(b).unwrap(), Some(h_n(4)));
    assert!(q.get(b).unwrap().is_none());
    assert!(q.is_empty(b).unwrap());
}

#[test]
fn peek_does_not_consume() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let q = fresh_queue(&mut h, 0xE3);
    let b = unsafe { &mut *state_ptr };
    q.put(b, h_n(7)).unwrap();
    assert_eq!(q.peek(b).unwrap(), Some(h_n(7)));
    assert_eq!(q.peek(b).unwrap(), Some(h_n(7)));
    assert_eq!(q.size(b).unwrap(), 1);
}

#[test]
fn for_each_visits_all_in_order() {
    let mut h = ArbosHarness::new().initialize();
    let state_ptr = h.state_ptr();
    let q = fresh_queue(&mut h, 0xE4);
    let b = unsafe { &mut *state_ptr };
    for i in 1..=5u8 {
        q.put(b, h_n(i)).unwrap();
    }
    let mut seen = Vec::new();
    q.for_each(b, |v| -> Result<(), StorageError> {
        seen.push(v);
        Ok(())
    })
    .unwrap();
    assert_eq!(seen, (1..=5u8).map(h_n).collect::<Vec<_>>());
    assert_eq!(q.size(b).unwrap(), 5);
}
