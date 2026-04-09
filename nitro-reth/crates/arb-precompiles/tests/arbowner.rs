mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, B256, U256};
use arb_precompiles::{
    create_arbowner_precompile,
    storage_slot::{
        derive_subspace_key, map_slot_b256, root_slot, subspace_slot, ARBOS_STATE_ADDRESS,
        CHAIN_OWNER_SUBSPACE, L2_PRICING_SUBSPACE, ROOT_STORAGE_KEY,
    },
};
use common::{calldata, decode_u256, word_address, word_u256, PrecompileTest};

fn arbowner() -> DynPrecompile {
    create_arbowner_precompile()
}

const NETWORK_FEE_ACCOUNT_OFFSET: u64 = 3;
const UPGRADE_VERSION_OFFSET: u64 = 1;
const UPGRADE_TIMESTAMP_OFFSET: u64 = 2;
const L2_SPEED_LIMIT: u64 = 0;

fn chain_owner_member_slot(owner: Address) -> U256 {
    let set_key = derive_subspace_key(ROOT_STORAGE_KEY, CHAIN_OWNER_SUBSPACE);
    let by_address_key = derive_subspace_key(set_key.as_slice(), &[0]);
    let addr_as_b256 = B256::left_padding_from(owner.as_slice());
    map_slot_b256(by_address_key.as_slice(), &addr_as_b256)
}

fn install_owner(test: PrecompileTest, owner: Address) -> PrecompileTest {
    test.storage(ARBOS_STATE_ADDRESS, chain_owner_member_slot(owner), U256::from(1))
}

const OWNER: Address = address!("00000000000000000000000000000000000000aa");
const INTRUDER: Address = address!("00000000000000000000000000000000000000bb");

fn fixture(arbos_version: u64) -> PrecompileTest {
    install_owner(
        PrecompileTest::new()
            .arbos_version(arbos_version)
            .caller(OWNER)
            .arbos_state(),
        OWNER,
    )
}

#[test]
fn rejects_caller_not_in_chain_owners() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .caller(INTRUDER)
        .arbos_state()
        .gas(100_000)
        .call(&arbowner(), &calldata("getNetworkFeeAccount()", &[]));
    assert!(run.result.is_err());
}

#[test]
fn allows_owner_to_read_network_fee_account() {
    let fee_account: Address = address!("00000000000000000000000000000000000000ff");
    let val = U256::from_be_slice(fee_account.as_slice());
    let run = fixture(30)
        .storage(ARBOS_STATE_ADDRESS, root_slot(NETWORK_FEE_ACCOUNT_OFFSET), val)
        .call(&arbowner(), &calldata("getNetworkFeeAccount()", &[]));
    assert_eq!(decode_u256(run.output()), val);
}

#[test]
fn is_chain_owner_returns_true_for_owner() {
    let run = fixture(30).call(
        &arbowner(),
        &calldata("isChainOwner(address)", &[word_address(OWNER)]),
    );
    assert_eq!(decode_u256(run.output()), U256::from(1));
}

#[test]
fn is_chain_owner_returns_false_for_non_owner() {
    let run = fixture(30).call(
        &arbowner(),
        &calldata("isChainOwner(address)", &[word_address(INTRUDER)]),
    );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn set_network_fee_account_writes_root_slot() {
    let new_account: Address = address!("00000000000000000000000000000000000000cc");
    let run = fixture(30).call(
        &arbowner(),
        &calldata("setNetworkFeeAccount(address)", &[word_address(new_account)]),
    );
    let _ = run.assert_ok();
    let stored = run.storage(ARBOS_STATE_ADDRESS, root_slot(NETWORK_FEE_ACCOUNT_OFFSET));
    assert_eq!(stored, U256::from_be_slice(new_account.as_slice()));
}

#[test]
fn schedule_arbos_upgrade_writes_version_and_timestamp() {
    let run = fixture(30).call(
        &arbowner(),
        &calldata(
            "scheduleArbOSUpgrade(uint64,uint64)",
            &[word_u256(U256::from(60)), word_u256(U256::from(1_800_000_000))],
        ),
    );
    let _ = run.assert_ok();
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, root_slot(UPGRADE_VERSION_OFFSET)),
        U256::from(60)
    );
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, root_slot(UPGRADE_TIMESTAMP_OFFSET)),
        U256::from(1_800_000_000_u64)
    );
}

#[test]
fn set_speed_limit_writes_l2_pricing_field() {
    let run = fixture(30).call(
        &arbowner(),
        &calldata("setSpeedLimit(uint64)", &[word_u256(U256::from(2_000_000))]),
    );
    let _ = run.assert_ok();
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, subspace_slot(L2_PRICING_SUBSPACE, L2_SPEED_LIMIT)),
        U256::from(2_000_000_u64)
    );
}

#[test]
fn add_chain_owner_at_v60_succeeds() {
    let new_owner: Address = address!("00000000000000000000000000000000000000dd");
    let run = fixture(60).call(
        &arbowner(),
        &calldata("addChainOwner(address)", &[word_address(new_owner)]),
    );
    let _ = run.assert_ok();
    let added = run.storage(ARBOS_STATE_ADDRESS, chain_owner_member_slot(new_owner));
    assert_ne!(added, U256::ZERO);
}

#[test]
fn add_chain_owner_at_v30_succeeds_without_event() {
    let new_owner: Address = address!("00000000000000000000000000000000000000ee");
    let run = fixture(30).call(
        &arbowner(),
        &calldata("addChainOwner(address)", &[word_address(new_owner)]),
    );
    let _ = run.assert_ok();
    let added = run.storage(ARBOS_STATE_ADDRESS, chain_owner_member_slot(new_owner));
    assert_ne!(added, U256::ZERO);
}

#[test]
fn version_check_uses_raw_arbos_version_not_plus_55() {
    // Regression test for block 18,489,005: arbowner.rs added 55 to the raw
    // ArbOS version before the version gate, making raw=11 evaluate as 66 >= 60
    // and emit a spurious ChainOwnerAdded event.
    let new_owner: Address = address!("00000000000000000000000000000000000000ef");
    let run = fixture(11).call(
        &arbowner(),
        &calldata("addChainOwner(address)", &[word_address(new_owner)]),
    );
    let _ = run.assert_ok();
    let added = run.storage(ARBOS_STATE_ADDRESS, chain_owner_member_slot(new_owner));
    assert_ne!(added, U256::ZERO);
}
