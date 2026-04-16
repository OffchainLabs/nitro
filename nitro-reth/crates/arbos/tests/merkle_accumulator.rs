use alloy_primitives::B256;
use arb_test_utils::ArbosHarness;
use arbos::merkle_accumulator::{calc_num_partials, open_merkle_accumulator, InMemoryMerkleAccumulator};

fn item(n: u64) -> B256 {
    let mut buf = [0u8; 32];
    buf[24..32].copy_from_slice(&n.to_be_bytes());
    B256::from(buf)
}

#[test]
fn calc_num_partials_matches_bit_length() {
    assert_eq!(calc_num_partials(0), 0);
    assert_eq!(calc_num_partials(1), 1);
    assert_eq!(calc_num_partials(2), 2);
    assert_eq!(calc_num_partials(3), 2);
    assert_eq!(calc_num_partials(4), 3);
    assert_eq!(calc_num_partials(7), 3);
    assert_eq!(calc_num_partials(8), 4);
    assert_eq!(calc_num_partials(u64::MAX), 64);
}

#[test]
fn empty_in_memory_root_is_zero() {
    let m = InMemoryMerkleAccumulator::new();
    assert_eq!(m.size(), 0);
    assert_eq!(m.root(), B256::ZERO);
}

#[test]
fn in_memory_append_grows_size() {
    let mut m = InMemoryMerkleAccumulator::new();
    for i in 1..=10 {
        m.append(item(i));
        assert_eq!(m.size(), i);
    }
}

#[test]
fn in_memory_root_changes_on_append() {
    let mut m = InMemoryMerkleAccumulator::new();
    let mut prev = m.root();
    for i in 1..=8 {
        m.append(item(i));
        let now = m.root();
        assert_ne!(now, prev);
        prev = now;
    }
}

#[test]
fn in_memory_and_persistent_match_for_same_inputs() {
    let mut h = ArbosHarness::new().initialize();
    let root = h.root_storage();
    let persistent = open_merkle_accumulator(root.open_sub_storage(&[0xD0]));

    let mut in_mem = InMemoryMerkleAccumulator::new();
    for i in 1..=12u64 {
        let it = item(i);
        persistent.append(it).unwrap();
        in_mem.append(it);
    }
    assert_eq!(persistent.size().unwrap(), in_mem.size());
    assert_eq!(persistent.root().unwrap(), in_mem.root());
}

#[test]
fn in_memory_from_partials_reconstructs_size() {
    let mut m1 = InMemoryMerkleAccumulator::new();
    for i in 1..=11u64 {
        m1.append(item(i));
    }
    let partials = m1.partials().to_vec();
    let m2 = InMemoryMerkleAccumulator::from_partials(partials);
    assert_eq!(m2.size(), m1.size());
    assert_eq!(m2.root(), m1.root());
}

#[test]
fn append_emits_events_at_higher_levels() {
    let mut m = InMemoryMerkleAccumulator::new();
    let _ = m.append(item(1));
    let events = m.append(item(2));
    assert!(!events.is_empty());
    assert_eq!(events[0].level, 1);
    assert_eq!(events[0].num_leaves, 1);
}
