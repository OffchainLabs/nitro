mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, B256, U256};
use arb_precompiles::{
    create_arbnativetokenmanager_precompile,
    storage_slot::{
        derive_subspace_key, map_slot_b256, ARBOS_STATE_ADDRESS, NATIVE_TOKEN_SUBSPACE,
        ROOT_STORAGE_KEY,
    },
};
use common::{calldata, word_u256, PrecompileTest};

fn arbnativetokenmanager() -> DynPrecompile {
    create_arbnativetokenmanager_precompile()
}

fn owner_member_slot(addr: Address) -> U256 {
    let owner_key = derive_subspace_key(ROOT_STORAGE_KEY, NATIVE_TOKEN_SUBSPACE);
    let by_address_key = derive_subspace_key(owner_key.as_slice(), &[0]);
    map_slot_b256(
        by_address_key.as_slice(),
        &B256::left_padding_from(addr.as_slice()),
    )
}

#[test]
fn precompile_gated_below_v41() {
    let amount = word_u256(U256::from(1_000_000));
    let run = PrecompileTest::new().arbos_version(40).arbos_state().call(
        &arbnativetokenmanager(),
        &calldata("mintNativeToken(uint256)", &[amount]),
    );
    let out = run.assert_ok();
    assert!(out.bytes.is_empty());
}

#[test]
fn mint_rejects_non_owner_caller() {
    let intruder: Address = address!("00000000000000000000000000000000000000bb");
    let run = PrecompileTest::new()
        .arbos_version(41)
        .caller(intruder)
        .arbos_state()
        .call(
            &arbnativetokenmanager(),
            &calldata("mintNativeToken(uint256)", &[word_u256(U256::from(1))]),
        );
    assert!(run.assert_ok().reverted);
}

#[test]
fn mint_succeeds_for_owner_and_increments_balance() {
    let owner: Address = address!("00000000000000000000000000000000000000aa");
    let amount = U256::from(7_777_777_u64);
    let run = PrecompileTest::new()
        .arbos_version(41)
        .caller(owner)
        .arbos_state()
        .storage(ARBOS_STATE_ADDRESS, owner_member_slot(owner), U256::from(1))
        .call(
            &arbnativetokenmanager(),
            &calldata("mintNativeToken(uint256)", &[word_u256(amount)]),
        );
    run.assert_ok();
    assert_eq!(run.balance(owner), amount);
}

#[test]
fn burn_rejects_non_owner() {
    let intruder: Address = address!("00000000000000000000000000000000000000bb");
    let run = PrecompileTest::new()
        .arbos_version(41)
        .caller(intruder)
        .arbos_state()
        .call(
            &arbnativetokenmanager(),
            &calldata("burnNativeToken(uint256)", &[word_u256(U256::from(1))]),
        );
    assert!(run.assert_ok().reverted);
}
