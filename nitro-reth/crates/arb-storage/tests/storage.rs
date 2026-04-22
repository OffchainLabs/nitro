use alloy_primitives::{b256, keccak256, B256, U256};
use arb_storage::{
    storage_key_map, StorageBackedAddress, StorageBackedBigInt, StorageBackedBigUint,
    StorageBackedInt64, StorageBackedUint64,
};
use arb_test_utils::ArbosHarness;

const TEST_OFFSET: u64 = 42;

#[test]
fn storage_key_map_preserves_last_byte_and_hashes_first_31() {
    let storage_key = b"sub";
    let offset = 0xAABB_CCDD_EE11_2233u64;

    let mut key_bytes = [0u8; 32];
    key_bytes[24..32].copy_from_slice(&offset.to_be_bytes());

    let mut data = Vec::new();
    data.extend_from_slice(storage_key);
    data.extend_from_slice(&key_bytes[..31]);
    let h = keccak256(&data);

    let mut expected = [0u8; 32];
    expected[..31].copy_from_slice(&h.0[..31]);
    expected[31] = key_bytes[31];

    assert_eq!(
        storage_key_map(storage_key, offset),
        U256::from_be_bytes(expected)
    );
}

#[test]
fn storage_key_map_root_uses_empty_prefix() {
    let from_root = storage_key_map(&[], 7);
    let from_zero_subspace = storage_key_map(B256::ZERO.as_slice(), 7);
    assert_ne!(from_root, from_zero_subspace);
}

#[test]
fn sub_storage_keys_match_keccak_of_parent_and_sub_id() {
    let mut h = ArbosHarness::new().initialize();
    let root = h.root_storage();

    let sub_a = root.open_sub_storage(&[0x05]);
    let sub_b = root.open_sub_storage(&[0x05]);
    assert_eq!(sub_a.base_key(), sub_b.base_key());

    let sub_c = root.open_sub_storage(&[0x06]);
    assert_ne!(sub_a.base_key(), sub_c.base_key());

    let nested = sub_a.open_sub_storage(&[0x99]);
    let mut expected = Vec::from(sub_a.base_key().as_slice());
    expected.push(0x99);
    let expected_key = keccak256(&expected);
    assert_eq!(nested.base_key(), expected_key);
}

#[test]
fn uint64_round_trip() {
    let mut h = ArbosHarness::new().initialize();
    let storage = h.root_storage();
    let v = StorageBackedUint64::new(storage.state_ptr(), B256::repeat_byte(1), TEST_OFFSET);

    assert_eq!(v.get().unwrap(), 0);
    v.set(0xDEAD_BEEFu64).unwrap();
    assert_eq!(v.get().unwrap(), 0xDEAD_BEEFu64);
    v.set(u64::MAX).unwrap();
    assert_eq!(v.get().unwrap(), u64::MAX);
}

#[test]
fn int64_signed_round_trip() {
    let mut h = ArbosHarness::new().initialize();
    let storage = h.root_storage();
    let v = StorageBackedInt64::new(storage.state_ptr(), B256::repeat_byte(2), TEST_OFFSET);

    for x in [0i64, 1, -1, i64::MAX, i64::MIN, 31_591_083, -31_591_083] {
        v.set(x).unwrap();
        assert_eq!(v.get().unwrap(), x);
    }
}

#[test]
fn big_uint_round_trip_at_boundaries() {
    let mut h = ArbosHarness::new().initialize();
    let storage = h.root_storage();
    let v = StorageBackedBigUint::new(storage.state_ptr(), B256::repeat_byte(3), TEST_OFFSET);

    for x in [
        U256::ZERO,
        U256::from(1u64),
        U256::from(u64::MAX),
        U256::MAX,
    ] {
        v.set(x).unwrap();
        assert_eq!(v.get().unwrap(), x);
    }
}

#[test]
fn big_int_two_complement_encoding() {
    let mut h = ArbosHarness::new().initialize();
    let storage = h.root_storage();
    let v = StorageBackedBigInt::new(storage.state_ptr(), B256::repeat_byte(4), TEST_OFFSET);

    v.set(U256::from(33u64)).unwrap();
    let (mag, neg) = v.get_signed().unwrap();
    assert_eq!(mag, U256::from(33u64));
    assert!(!neg);
    assert!(!v.is_negative().unwrap());

    v.set_negative(U256::from(33u64)).unwrap();
    let (mag, neg) = v.get_signed().unwrap();
    assert_eq!(mag, U256::from(33u64));
    assert!(neg);
    assert!(v.is_negative().unwrap());
    assert_eq!(
        v.get_raw().unwrap(),
        U256::ZERO.wrapping_sub(U256::from(33u64))
    );

    let max_pos = (U256::from(1u64) << 255) - U256::from(1u64);
    v.set(max_pos).unwrap();
    let (mag, neg) = v.get_signed().unwrap();
    assert_eq!(mag, max_pos);
    assert!(!neg);

    let min_neg_magnitude = U256::from(1u64) << 255;
    v.set_negative(min_neg_magnitude).unwrap();
    let (mag, neg) = v.get_signed().unwrap();
    assert_eq!(mag, min_neg_magnitude);
    assert!(neg);
}

#[test]
fn address_round_trip() {
    let mut h = ArbosHarness::new().initialize();
    let storage = h.root_storage();
    let v = StorageBackedAddress::new(storage.state_ptr(), B256::repeat_byte(5), TEST_OFFSET);

    assert_eq!(v.get().unwrap(), alloy_primitives::Address::ZERO);
    let addr = alloy_primitives::address!("FFEEDDCCBBAA99887766554433221100AABBCCDD");
    v.set(addr).unwrap();
    assert_eq!(v.get().unwrap(), addr);
}

#[test]
fn distinct_offsets_produce_distinct_slots() {
    let mut h = ArbosHarness::new().initialize();
    let storage = h.root_storage();
    let key = B256::repeat_byte(7);

    let a = StorageBackedUint64::new(storage.state_ptr(), key, 1);
    let b = StorageBackedUint64::new(storage.state_ptr(), key, 2);
    a.set(111).unwrap();
    b.set(222).unwrap();
    assert_eq!(a.get().unwrap(), 111);
    assert_eq!(b.get().unwrap(), 222);
}

#[test]
fn known_storage_layout_constants() {
    let key_zero = b256!("0000000000000000000000000000000000000000000000000000000000000000");
    let _ = key_zero;
    let slot_root_5 = storage_key_map(&[], 5);
    let slot_subspace_5 = storage_key_map(&[5], 0);
    assert_ne!(slot_root_5, slot_subspace_5);
}
