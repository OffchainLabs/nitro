//! Tight-budget pins for `ArbOwner.removeWasmCacheManager`, modelled on the
//! canonical Arbitrum Sepolia call that forwarded only 20,111 gas to the
//! precompile. `AddressSet::remove` clears slots, so its writes bill a storage
//! reset (5,000) rather than a full set (20,000); the dispatcher must reflect
//! that or the access-controlled call out-of-gases inside the owner's budget.

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, B256, U256};
use arb_precompiles::create_arbowner_precompile;
use arb_storage::{
    layout::{
        derive_subspace_key, map_slot, map_slot_b256, programs::CACHE_MANAGERS_KEY,
        CHAIN_OWNER_SUBSPACE, PROGRAMS_SUBSPACE, ROOT_STORAGE_KEY,
    },
    ARBOS_STATE_ADDRESS,
};
use common::{calldata, word_address, PrecompileTest};

const OWNER: Address = address!("00000000000000000000000000000000000000aa");
const MANAGER: Address = address!("d01c86379c53650b02ae259ae4b608684687c73a");

// Gas the precompile body bills for this single-member removal:
// open(800) + args(3) + IsMember(800) + Remove(3 reads + 3 resets) = 19,003.
const REMOVE_GAS: u64 = 19_003;
// Gas the canonical block forwarded to the precompile frame.
const CANONICAL_BUDGET: u64 = 20_111;

fn arbowner(ctx: std::sync::Arc<arb_context::ArbPrecompileCtx>) -> DynPrecompile {
    create_arbowner_precompile(ctx)
}

fn cache_managers_base() -> B256 {
    let programs = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    derive_subspace_key(programs.as_slice(), CACHE_MANAGERS_KEY)
}

fn cache_managers_size_slot() -> U256 {
    map_slot(cache_managers_base().as_slice(), 0)
}

fn cache_managers_by_address_slot(member: Address) -> U256 {
    let by_address = derive_subspace_key(cache_managers_base().as_slice(), &[0]);
    map_slot_b256(
        by_address.as_slice(),
        &B256::left_padding_from(member.as_slice()),
    )
}

fn chain_owner_member_slot(owner: Address) -> U256 {
    let set_key = derive_subspace_key(ROOT_STORAGE_KEY, CHAIN_OWNER_SUBSPACE);
    let by_address = derive_subspace_key(set_key.as_slice(), &[0]);
    map_slot_b256(
        by_address.as_slice(),
        &B256::left_padding_from(owner.as_slice()),
    )
}

/// `OWNER` is a chain owner; the cache-manager set holds `MANAGER` alone.
fn fixture(gas: u64) -> PrecompileTest {
    let base = cache_managers_base();
    PrecompileTest::new()
        .arbos_version(30)
        .caller(OWNER)
        .arbos_state()
        .gas(gas)
        .storage(
            ARBOS_STATE_ADDRESS,
            chain_owner_member_slot(OWNER),
            U256::from(1),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            cache_managers_size_slot(),
            U256::from(1),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            map_slot(base.as_slice(), 1),
            U256::from_be_bytes(B256::left_padding_from(MANAGER.as_slice()).0),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            cache_managers_by_address_slot(MANAGER),
            U256::from(1),
        )
}

fn run(gas: u64) -> common::PrecompileRun {
    fixture(gas).call(
        arbowner,
        &calldata("removeWasmCacheManager(address)", &[word_address(MANAGER)]),
    )
}

#[test]
fn fits_canonical_budget_and_empties_the_set() {
    let r = run(CANONICAL_BUDGET);
    let out = r.assert_ok();
    assert!(
        !out.reverted,
        "must succeed within the canonical 20,111-gas budget"
    );
    assert_eq!(
        out.gas_used, 0,
        "access-controlled methods bill the caller zero"
    );
    assert_eq!(
        r.storage(ARBOS_STATE_ADDRESS, cache_managers_size_slot()),
        U256::ZERO,
        "the set must be emptied",
    );
    assert_eq!(
        r.storage(ARBOS_STATE_ADDRESS, cache_managers_by_address_slot(MANAGER)),
        U256::ZERO,
        "the membership mapping must be cleared",
    );
}

#[test]
fn body_gas_threshold_pins_reset_pricing() {
    assert!(
        !run(REMOVE_GAS).assert_ok().reverted,
        "succeeds at exactly the billed body gas",
    );
    // One gas short, the access-controlled body overruns its budget: it reverts
    // (the removal rolls back) but bills the owner zero, never consuming the
    // forwarded gas.
    let short = run(REMOVE_GAS - 1);
    let out = short.assert_ok();
    assert!(out.reverted, "one gas short reverts");
    assert_eq!(out.gas_used, 0, "the owner is billed zero even over budget");
}
