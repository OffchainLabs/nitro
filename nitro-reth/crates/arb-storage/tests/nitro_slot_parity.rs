//! Real Nitro storage slot keys (extracted from `testing/genesis-from-nitro.json`)
//! must round-trip through our `storage_key_map`. If our slot derivation drifts,
//! every dual-exec block-replay test would fail; this is the cheap pre-flight.

use alloy_primitives::{b256, keccak256, B256, U256};
use arb_storage::storage_key_map;

#[test]
fn root_chain_id_offset_4_matches_nitro_slot() {
    let observed = b256!("15fed0451499512d95f3ec5a41c878b9de55f21878b5b4e190d4667ec709b404");
    let computed = U256::from(storage_key_map(&[], 4));
    assert_eq!(B256::from(computed.to_be_bytes::<32>()), observed);
}

#[test]
fn root_version_offset_0_matches_nitro_slot() {
    let observed = b256!("15fed0451499512d95f3ec5a41c878b9de55f21878b5b4e190d4667ec709b400");
    let computed = U256::from(storage_key_map(&[], 0));
    assert_eq!(B256::from(computed.to_be_bytes::<32>()), observed);
}

#[test]
fn root_network_fee_offset_3_matches_nitro_slot() {
    let observed = b256!("15fed0451499512d95f3ec5a41c878b9de55f21878b5b4e190d4667ec709b403");
    let computed = U256::from(storage_key_map(&[], 3));
    assert_eq!(B256::from(computed.to_be_bytes::<32>()), observed);
}

#[test]
fn chain_owners_subspace_size_offset_matches_nitro_slot() {
    let chain_owners_subspace = keccak256([4u8]);
    let observed = b256!("41e0d7d38ffe0727248ee6ed6ea1250b08279ad004e3ab07b7ffe78352d8c400");
    let computed = U256::from(storage_key_map(chain_owners_subspace.as_slice(), 0));
    assert_eq!(B256::from(computed.to_be_bytes::<32>()), observed);
}

#[test]
fn chain_owners_subspace_first_owner_slot_matches_nitro() {
    let chain_owners_subspace = keccak256([4u8]);
    let observed = b256!("41e0d7d38ffe0727248ee6ed6ea1250b08279ad004e3ab07b7ffe78352d8c401");
    let computed = U256::from(storage_key_map(chain_owners_subspace.as_slice(), 1));
    assert_eq!(B256::from(computed.to_be_bytes::<32>()), observed);
}

#[test]
fn batch_posters_nested_subspace_first_poster_slot_matches_nitro() {
    let l1_pricing = keccak256([0u8]);
    let mut buf = Vec::with_capacity(33);
    buf.extend_from_slice(l1_pricing.as_slice());
    buf.push(0u8);
    let bpt = keccak256(&buf);
    let mut buf = Vec::with_capacity(33);
    buf.extend_from_slice(bpt.as_slice());
    buf.push(0u8);
    let poster_addrs = keccak256(&buf);

    let observed = b256!("19cc27d300234c0bd290398e6aaf8b7680a1006d08dc01e871b6d473bc9d6001");
    let computed = U256::from(storage_key_map(poster_addrs.as_slice(), 1));
    assert_eq!(B256::from(computed.to_be_bytes::<32>()), observed);
}

#[test]
fn chain_config_subspace_seq_slot_0_matches_nitro_slot() {
    let chain_config_subspace = keccak256([7u8]);
    let observed = b256!("696e3678057072f0c8dba2395c9474f6a52565714cff46262e4548533b097700");
    let computed = U256::from(storage_key_map(chain_config_subspace.as_slice(), 0));
    assert_eq!(B256::from(computed.to_be_bytes::<32>()), observed);
}

#[test]
fn last_byte_of_offset_is_preserved_in_slot() {
    for offset in [0u64, 1, 2, 0x42, 0xFE, 0xFF] {
        let slot = storage_key_map(&[], offset);
        let bytes = slot.to_be_bytes::<32>();
        assert_eq!(bytes[31], (offset & 0xFF) as u8);
    }
}
