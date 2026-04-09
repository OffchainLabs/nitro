mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, B256, U256};
use arb_precompiles::{
    create_arbaddresstable_precompile,
    storage_slot::{
        derive_subspace_key, map_slot, map_slot_b256, ADDRESS_TABLE_SUBSPACE,
        ARBOS_STATE_ADDRESS, ROOT_STORAGE_KEY,
    },
};
use common::{calldata, decode_u256, word_address, PrecompileTest};

fn arbaddresstable() -> DynPrecompile {
    create_arbaddresstable_precompile()
}

fn table_key() -> B256 {
    derive_subspace_key(ROOT_STORAGE_KEY, ADDRESS_TABLE_SUBSPACE)
}

fn size_slot() -> U256 {
    map_slot(table_key().as_slice(), 0)
}

fn by_address_slot(addr: Address) -> U256 {
    let by_address_key = derive_subspace_key(table_key().as_slice(), &[]);
    map_slot_b256(
        by_address_key.as_slice(),
        &B256::left_padding_from(addr.as_slice()),
    )
}

#[test]
fn size_returns_zero_for_empty_table() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(&arbaddresstable(), &calldata("size()", &[]));
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn size_returns_stored_value() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .storage(ARBOS_STATE_ADDRESS, size_slot(), U256::from(42))
        .call(&arbaddresstable(), &calldata("size()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(42));
}

#[test]
fn address_exists_returns_false_for_unregistered() {
    let addr: Address = address!("00000000000000000000000000000000000000aa");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(
            &arbaddresstable(),
            &calldata("addressExists(address)", &[word_address(addr)]),
        );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn address_exists_returns_true_for_registered() {
    let addr: Address = address!("00000000000000000000000000000000000000aa");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .storage(ARBOS_STATE_ADDRESS, by_address_slot(addr), U256::from(1))
        .call(
            &arbaddresstable(),
            &calldata("addressExists(address)", &[word_address(addr)]),
        );
    assert_eq!(decode_u256(run.output()), U256::from(1));
}
