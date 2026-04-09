mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, B256, U256};
use arb_precompiles::{
    create_arbfilteredtxmanager_precompile,
    storage_slot::{
        derive_subspace_key, map_slot_b256, ARBOS_STATE_ADDRESS, FILTERED_TX_STATE_ADDRESS,
        ROOT_STORAGE_KEY, TRANSACTION_FILTERER_SUBSPACE,
    },
};
use common::{calldata, decode_u256, PrecompileTest};

fn arbfilteredtxmanager() -> DynPrecompile {
    create_arbfilteredtxmanager_precompile()
}

fn filterer_member_slot(addr: Address) -> U256 {
    let filterer_key = derive_subspace_key(ROOT_STORAGE_KEY, TRANSACTION_FILTERER_SUBSPACE);
    let by_address_key = derive_subspace_key(filterer_key.as_slice(), &[0]);
    map_slot_b256(
        by_address_key.as_slice(),
        &B256::left_padding_from(addr.as_slice()),
    )
}

fn filtered_tx_slot(tx_hash: &B256) -> U256 {
    map_slot_b256(&[], tx_hash)
}

#[test]
fn precompile_gated_below_v60() {
    let tx = B256::from([0x42; 32]);
    let run = PrecompileTest::new()
        .arbos_version(59)
        .arbos_state()
        .call(
            &arbfilteredtxmanager(),
            &calldata("isTransactionFiltered(bytes32)", &[tx]),
        );
    let out = run.assert_ok();
    assert!(out.bytes.is_empty());
}

#[test]
fn is_filtered_returns_false_for_unknown_tx() {
    let tx = B256::from([0x33; 32]);
    let run = PrecompileTest::new()
        .arbos_version(60)
        .arbos_state()
        .call(
            &arbfilteredtxmanager(),
            &calldata("isTransactionFiltered(bytes32)", &[tx]),
        );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn is_filtered_returns_true_for_known_tx() {
    let tx = B256::from([0x99; 32]);
    let run = PrecompileTest::new()
        .arbos_version(60)
        .arbos_state()
        .storage(FILTERED_TX_STATE_ADDRESS, filtered_tx_slot(&tx), U256::from(1))
        .call(
            &arbfilteredtxmanager(),
            &calldata("isTransactionFiltered(bytes32)", &[tx]),
        );
    assert_eq!(decode_u256(run.output()), U256::from(1));
}

#[test]
fn add_rejects_non_filterer_caller() {
    let intruder: Address = address!("00000000000000000000000000000000000000bb");
    let tx = B256::from([0x77; 32]);
    let run = PrecompileTest::new()
        .arbos_version(60)
        .caller(intruder)
        .arbos_state()
        .call(
            &arbfilteredtxmanager(),
            &calldata("addFilteredTransaction(bytes32)", &[tx]),
        );
    assert!(run.assert_ok().reverted);
}

#[test]
fn add_succeeds_for_authorized_filterer() {
    let filterer: Address = address!("00000000000000000000000000000000000000aa");
    let tx = B256::from([0x44; 32]);
    let run = PrecompileTest::new()
        .arbos_version(60)
        .caller(filterer)
        .arbos_state()
        .storage(ARBOS_STATE_ADDRESS, filterer_member_slot(filterer), U256::from(1))
        .call(
            &arbfilteredtxmanager(),
            &calldata("addFilteredTransaction(bytes32)", &[tx]),
        );
    run.assert_ok();
    let stored = run.storage(FILTERED_TX_STATE_ADDRESS, filtered_tx_slot(&tx));
    assert_eq!(stored, U256::from(1));
}
