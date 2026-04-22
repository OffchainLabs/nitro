use alloy_primitives::{keccak256, B256};
use arb_test_utils::ArbosHarness;
use arbos::blockhash::open_blockhashes;

const ARBOS_V30: u64 = 30;

fn fresh(h: &mut ArbosHarness, sub: u8) -> arbos::blockhash::Blockhashes<arb_test_utils::EmptyDb> {
    let root = h.root_storage();
    open_blockhashes(root.open_sub_storage(&[sub]))
}

fn hash_n(n: u64) -> B256 {
    let mut buf = [0u8; 32];
    buf[24..32].copy_from_slice(&n.to_be_bytes());
    B256::from(buf)
}

#[test]
fn empty_blockhashes_have_zero_l1_number_and_no_hashes() {
    let mut h = ArbosHarness::new().initialize();
    let bh = fresh(&mut h, 0xC0);
    assert_eq!(bh.l1_block_number().unwrap(), 0);
    assert!(bh.block_hash(0).unwrap().is_none());
    assert!(bh.block_hash(100).unwrap().is_none());
}

#[test]
fn record_advances_l1_block_number() {
    let mut h = ArbosHarness::new().initialize();
    let bh = fresh(&mut h, 0xC1);
    bh.record_new_l1_block(5, hash_n(5), ARBOS_V30).unwrap();
    assert_eq!(bh.l1_block_number().unwrap(), 6);
}

#[test]
fn block_hash_returns_recorded_hash_for_recent_block() {
    let mut h = ArbosHarness::new().initialize();
    let bh = fresh(&mut h, 0xC2);
    bh.record_new_l1_block(10, hash_n(10), ARBOS_V30).unwrap();
    assert_eq!(bh.block_hash(10).unwrap(), Some(hash_n(10)));
    assert!(bh.block_hash(11).unwrap().is_none());
}

#[test]
fn block_hash_outside_256_window_returns_none() {
    let mut h = ArbosHarness::new().initialize();
    let bh = fresh(&mut h, 0xC3);
    bh.record_new_l1_block(500, hash_n(500), ARBOS_V30).unwrap();
    assert!(bh.block_hash(0).unwrap().is_none());
    assert!(bh.block_hash(243).unwrap().is_none());
    assert_eq!(bh.block_hash(500).unwrap(), Some(hash_n(500)));
}

#[test]
fn record_new_block_fills_gaps_with_derived_hashes() {
    let mut h = ArbosHarness::new().initialize();
    let bh = fresh(&mut h, 0xC4);
    bh.record_new_l1_block(0, hash_n(0), ARBOS_V30).unwrap();
    bh.record_new_l1_block(5, hash_n(5), ARBOS_V30).unwrap();

    assert_eq!(bh.block_hash(5).unwrap(), Some(hash_n(5)));
    let derived = bh.block_hash(2).unwrap().expect("gap-filled hash present");
    let mut expected_buf = Vec::new();
    expected_buf.extend_from_slice(hash_n(5).as_slice());
    expected_buf.extend_from_slice(&2u64.to_le_bytes());
    let expected = keccak256(&expected_buf);
    assert_eq!(derived, expected);
}

#[test]
fn record_with_lower_number_is_noop() {
    let mut h = ArbosHarness::new().initialize();
    let bh = fresh(&mut h, 0xC5);
    bh.record_new_l1_block(100, hash_n(100), ARBOS_V30).unwrap();
    bh.record_new_l1_block(50, hash_n(50), ARBOS_V30).unwrap();
    assert_eq!(bh.l1_block_number().unwrap(), 101);
}
