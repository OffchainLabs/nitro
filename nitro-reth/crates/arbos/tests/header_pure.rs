use std::{cell::RefCell, collections::HashMap};

use alloy_primitives::{Address, B256, U256};
use arbos::header::{
    compute_arbos_mixhash, derive_arb_header_info, extract_arbos_version_from_mix_hash,
    extract_l1_block_number_from_mix_hash, extract_send_count_from_mix_hash,
    extract_send_root_from_header_extra, merkle_root_from_partials, read_arbos_version,
    read_l2_base_fee, read_l2_per_block_gas_limit, read_storage_hash, read_storage_u64_be,
    ArbHeaderInfo, ARBOS_STATE_ADDRESS,
};

#[test]
fn mixhash_roundtrips_all_three_fields() {
    let mix = compute_arbos_mixhash(0x1122334455667788, 0xAABBCCDDEEFF0011, 30, false);
    assert_eq!(extract_send_count_from_mix_hash(mix), 0x1122334455667788);
    assert_eq!(
        extract_l1_block_number_from_mix_hash(mix),
        0xAABBCCDDEEFF0011
    );
    assert_eq!(extract_arbos_version_from_mix_hash(mix), 30);
}

#[test]
fn mixhash_is_zero_for_zero_inputs() {
    let mix = compute_arbos_mixhash(0, 0, 0, false);
    assert_eq!(mix, B256::ZERO);
}

#[test]
fn mixhash_layout_is_big_endian_send_count_first() {
    let mix = compute_arbos_mixhash(1, 0, 0, false);
    assert_eq!(mix.0[0..8], [0, 0, 0, 0, 0, 0, 0, 1]);
    assert_eq!(mix.0[8..16], [0u8; 8]);
    assert_eq!(mix.0[16..24], [0u8; 8]);
    assert_eq!(mix.0[24..32], [0u8; 8]);
}

#[test]
fn mixhash_collect_tips_sets_byte_25() {
    let mix = compute_arbos_mixhash(1, 2, 60, true);
    assert_eq!(mix.0[25], 1);
}

#[test]
fn mixhash_collect_tips_false_clears_byte_25() {
    let mix = compute_arbos_mixhash(1, 2, 60, false);
    assert_eq!(mix.0[25], 0);
}

#[test]
fn mixhash_old_arbos_version_9_ignores_collect_tips_flag() {
    use arb_chainspec::arbos_version::ARBOS_VERSION_COLLECT_TIPS_OLD;
    let mix = compute_arbos_mixhash(0, 0, ARBOS_VERSION_COLLECT_TIPS_OLD, true);
    assert_eq!(mix.0[25], 0);
}

#[test]
fn extract_send_root_shorter_than_32_returns_zero() {
    assert_eq!(extract_send_root_from_header_extra(&[]), B256::ZERO);
    assert_eq!(extract_send_root_from_header_extra(&[0xFF; 31]), B256::ZERO);
}

#[test]
fn extract_send_root_uses_first_32_bytes_only() {
    let mut extra = vec![0u8; 64];
    extra[0..32].copy_from_slice(&[0xAB; 32]);
    extra[32..64].copy_from_slice(&[0xCD; 32]);
    assert_eq!(
        extract_send_root_from_header_extra(&extra),
        B256::repeat_byte(0xAB)
    );
}

#[test]
fn compute_mix_hash_via_arb_header_info() {
    let info = ArbHeaderInfo {
        send_root: B256::ZERO,
        send_count: 42,
        l1_block_number: 100,
        arbos_format_version: 30,
        collect_tips: false,
    };
    let mix = info.compute_mix_hash();
    assert_eq!(mix, compute_arbos_mixhash(42, 100, 30, false));
}

#[test]
fn arbos_state_address_has_expected_bytes() {
    let bytes = ARBOS_STATE_ADDRESS.0;
    assert_eq!(bytes[0], 0xA4);
    assert_eq!(bytes[1], 0xB0);
    assert_eq!(bytes[2], 0x5F);
    for b in &bytes[3..] {
        assert_eq!(*b, 0xFF);
    }
}

#[derive(Default)]
struct MockStorage {
    map: RefCell<HashMap<(Address, B256), U256>>,
}

impl MockStorage {
    fn set(&self, addr: Address, slot: B256, value: U256) {
        self.map.borrow_mut().insert((addr, slot), value);
    }
    fn reader(
        &self,
    ) -> impl Fn(Address, B256) -> Result<Option<U256>, std::convert::Infallible> + '_ {
        |addr, slot| Ok(self.map.borrow().get(&(addr, slot)).copied())
    }
}

#[test]
fn read_storage_u64_be_returns_last_8_bytes() {
    let s = MockStorage::default();
    let addr = Address::ZERO;
    let slot = B256::ZERO;
    s.set(addr, slot, U256::from(0x1234567890ABCDEFu64));
    assert_eq!(
        read_storage_u64_be(&s.reader(), addr, slot).unwrap(),
        Some(0x1234567890ABCDEF)
    );
}

#[test]
fn read_storage_u64_be_returns_none_if_unset() {
    let s = MockStorage::default();
    assert_eq!(
        read_storage_u64_be(&s.reader(), Address::ZERO, B256::ZERO).unwrap(),
        None
    );
}

#[test]
fn read_storage_hash_converts_u256_to_b256() {
    let s = MockStorage::default();
    let addr = Address::ZERO;
    let slot = B256::ZERO;
    s.set(addr, slot, U256::from_be_slice(&[0xAA; 32]));
    assert_eq!(
        read_storage_hash(&s.reader(), addr, slot).unwrap(),
        Some(B256::repeat_byte(0xAA))
    );
}

#[test]
fn merkle_root_size_zero_is_zero_hash() {
    let s = MockStorage::default();
    assert_eq!(
        merkle_root_from_partials(&s.reader(), Address::ZERO, &[], 0).unwrap(),
        Some(B256::ZERO)
    );
}

#[test]
fn derive_arb_header_info_reads_version_from_storage() {
    let s = MockStorage::default();
    let reader = s.reader();
    let version = 30u64;
    let slot = {
        use alloy_primitives::keccak256;
        let key = [0u8; 32];
        let mut preimage = Vec::new();
        preimage.extend_from_slice(&key[..31]);
        let h = keccak256(&preimage);
        let mut mapped = [0u8; 32];
        mapped[..31].copy_from_slice(&h.0[..31]);
        mapped[31] = key[31];
        B256::from(mapped)
    };
    s.set(ARBOS_STATE_ADDRESS, slot, U256::from(version));
    let info = derive_arb_header_info(&reader, arbos::l1_pricing::BATCH_POSTER_ADDRESS)
        .unwrap()
        .expect("some");
    assert_eq!(info.arbos_format_version, version);
    assert_eq!(info.send_count, 0);
    assert_eq!(info.l1_block_number, 0);
    assert_eq!(info.send_root, B256::ZERO);
}

/// Root-level ArbOS slot for `offset` (`keccak256([0;31]) || offset`).
fn root_slot(offset: u8) -> B256 {
    use alloy_primitives::keccak256;
    let h = keccak256([0u8; 31]);
    let mut mapped = [0u8; 32];
    mapped[..31].copy_from_slice(&h.0[..31]);
    mapped[31] = offset;
    B256::from(mapped)
}

#[test]
fn derive_collect_tips_excluded_for_non_batch_poster_coinbase() {
    let s = MockStorage::default();
    s.set(ARBOS_STATE_ADDRESS, root_slot(0), U256::from(60u64)); // version
    s.set(ARBOS_STATE_ADDRESS, root_slot(11), U256::from(1u64)); // collectTips enabled
    let reader = s.reader();

    let on = derive_arb_header_info(&reader, arbos::l1_pricing::BATCH_POSTER_ADDRESS)
        .unwrap()
        .expect("some");
    assert!(
        on.collect_tips,
        "a batch-poster block collects tips when enabled"
    );

    let off = derive_arb_header_info(&reader, Address::ZERO)
        .unwrap()
        .expect("some");
    assert!(
        !off.collect_tips,
        "a non-batch-poster block never collects tips"
    );
}

#[test]
fn header_reads_propagate_backing_store_errors() {
    // A backing-store failure must propagate, never be swallowed into a
    // default/None that would read as a wrong (zero) header field.
    let failing = |_a: Address, _s: B256| -> Result<Option<U256>, &'static str> { Err("db down") };
    assert_eq!(
        read_storage_u64_be(&failing, Address::ZERO, B256::ZERO),
        Err("db down")
    );
    assert_eq!(read_l2_base_fee(&failing), Err("db down"));
    assert_eq!(
        derive_arb_header_info(&failing, arbos::l1_pricing::BATCH_POSTER_ADDRESS).err(),
        Some("db down")
    );
}

#[test]
fn read_arbos_version_returns_none_when_missing() {
    let s = MockStorage::default();
    assert_eq!(read_arbos_version(&s.reader()).unwrap(), None);
}

#[test]
fn read_l2_per_block_gas_limit_returns_none_when_missing() {
    let s = MockStorage::default();
    assert_eq!(read_l2_per_block_gas_limit(&s.reader()).unwrap(), None);
}

#[test]
fn read_l2_base_fee_returns_none_when_missing() {
    let s = MockStorage::default();
    assert_eq!(read_l2_base_fee(&s.reader()).unwrap(), None);
}
