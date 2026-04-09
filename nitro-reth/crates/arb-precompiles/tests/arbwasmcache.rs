mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, B256, U256};
use arb_precompiles::{
    create_arbwasmcache_precompile,
    storage_slot::{
        derive_subspace_key, map_slot_b256, ARBOS_STATE_ADDRESS, CACHE_MANAGERS_KEY,
        PROGRAMS_DATA_KEY, PROGRAMS_SUBSPACE, ROOT_STORAGE_KEY,
    },
};
use common::{calldata, decode_u256, word_address, PrecompileTest};

fn arbwasmcache() -> DynPrecompile {
    create_arbwasmcache_precompile()
}

fn cache_manager_member_slot(addr: Address) -> U256 {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let cm_key = derive_subspace_key(programs_key.as_slice(), CACHE_MANAGERS_KEY);
    let by_addr_key = derive_subspace_key(cm_key.as_slice(), &[0]);
    let mut padded = [0u8; 32];
    padded[12..32].copy_from_slice(addr.as_slice());
    map_slot_b256(by_addr_key.as_slice(), &B256::from(padded))
}

#[test]
fn is_cache_manager_returns_false_for_unregistered() {
    let probe: Address = address!("00000000000000000000000000000000000000aa");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(
            &arbwasmcache(),
            &calldata("isCacheManager(address)", &[word_address(probe)]),
        );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn is_cache_manager_returns_true_for_member() {
    let probe: Address = address!("00000000000000000000000000000000000000aa");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .storage(ARBOS_STATE_ADDRESS, cache_manager_member_slot(probe), U256::from(1))
        .call(
            &arbwasmcache(),
            &calldata("isCacheManager(address)", &[word_address(probe)]),
        );
    assert_eq!(decode_u256(run.output()), U256::from(1));
}

#[test]
fn codehash_is_cached_reads_program_word_byte_14() {
    let codehash = B256::from([0x42; 32]);
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let data_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_DATA_KEY);
    let slot = map_slot_b256(data_key.as_slice(), &codehash);
    let mut word = [0u8; 32];
    word[14] = 1;
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .storage(ARBOS_STATE_ADDRESS, slot, U256::from_be_bytes(word))
        .call(
            &arbwasmcache(),
            &calldata("codehashIsCached(bytes32)", &[codehash]),
        );
    assert_eq!(decode_u256(run.output()), U256::from(1));
}

#[test]
fn codehash_is_cached_returns_false_when_byte_14_zero() {
    let codehash = B256::from([0x55; 32]);
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(
            &arbwasmcache(),
            &calldata("codehashIsCached(bytes32)", &[codehash]),
        );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}
