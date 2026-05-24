//! Per-selector gas pins for ArbOwnerPublic (0x6b).
//!
//! Method-level version gating means some selectors are only reachable at
//! v11+, v20+, v40+, v41+, v50+, or v60+. Each pin uses the first ArbOS
//! version that exposes the selector.

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address};
use arb_precompiles::create_arbownerpublic_precompile;
use common::{calldata, word_address, PrecompileTest};

// Constants used across the precompile's per-method schedules.
const SLOAD: u64 = 800;
const COPY: u64 = 3;
const WARM_SLOAD: u64 = 100;

fn arbownerpublic() -> DynPrecompile {
    create_arbownerpublic_precompile()
}

fn fixture(v: u64) -> PrecompileTest {
    PrecompileTest::new().arbos_version(v).arbos_state()
}

// ── field-read methods (2 SLOAD + COPY = 1603) ──────────────────────

#[test]
fn get_network_fee_account_v30_gas_pin() {
    let run = fixture(30).call(&arbownerpublic(), &calldata("getNetworkFeeAccount()", &[]));
    assert_eq!(run.gas_used(), 2 * SLOAD + COPY);
}

#[test]
fn get_infra_fee_account_v6_gas_pin() {
    let run = fixture(6).call(&arbownerpublic(), &calldata("getInfraFeeAccount()", &[]));
    assert_eq!(run.gas_used(), 2 * SLOAD + COPY);
}

#[test]
fn get_infra_fee_account_below_v5_reverts_burning_all_gas() {
    let run = fixture(4)
        .gas(50_000)
        .call(&arbownerpublic(), &calldata("getInfraFeeAccount()", &[]));
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 50_000);
}

#[test]
fn get_brotli_compression_level_v20_gas_pin() {
    let run = fixture(20).call(
        &arbownerpublic(),
        &calldata("getBrotliCompressionLevel()", &[]),
    );
    assert_eq!(run.gas_used(), 2 * SLOAD + COPY);
}

#[test]
fn get_brotli_compression_level_below_v20_reverts() {
    let run = fixture(19).gas(50_000).call(
        &arbownerpublic(),
        &calldata("getBrotliCompressionLevel()", &[]),
    );
    assert!(run.assert_ok().reverted);
}

#[test]
fn get_native_token_management_from_v50_gas_pin() {
    let run = fixture(50).call(
        &arbownerpublic(),
        &calldata("getNativeTokenManagementFrom()", &[]),
    );
    assert_eq!(run.gas_used(), 2 * SLOAD + COPY);
}

#[test]
fn get_transaction_filtering_from_v60_gas_pin() {
    let run = fixture(60).call(
        &arbownerpublic(),
        &calldata("getTransactionFilteringFrom()", &[]),
    );
    assert_eq!(run.gas_used(), 2 * SLOAD + COPY);
}

#[test]
fn get_filtered_funds_recipient_v60_gas_pin() {
    let run = fixture(60).call(
        &arbownerpublic(),
        &calldata("getFilteredFundsRecipient()", &[]),
    );
    assert_eq!(run.gas_used(), 2 * SLOAD + COPY);
}

#[test]
fn get_parent_gas_floor_per_token_v50_gas_pin() {
    let run = fixture(50).call(
        &arbownerpublic(),
        &calldata("getParentGasFloorPerToken()", &[]),
    );
    assert_eq!(run.gas_used(), 2 * SLOAD + COPY);
}

// ── set-membership probes (2 SLOAD + 2 COPY = 1606) ─────────────────

#[test]
fn is_chain_owner_v30_gas_pin() {
    let target: Address = address!("00000000000000000000000000000000000000aa");
    let run = fixture(30).call(
        &arbownerpublic(),
        &calldata("isChainOwner(address)", &[word_address(target)]),
    );
    assert_eq!(run.gas_used(), 2 * SLOAD + 2 * COPY);
}

#[test]
fn is_native_token_owner_v41_gas_pin() {
    let target: Address = address!("00000000000000000000000000000000000000bb");
    let run = fixture(41).call(
        &arbownerpublic(),
        &calldata("isNativeTokenOwner(address)", &[word_address(target)]),
    );
    assert_eq!(run.gas_used(), 2 * SLOAD + 2 * COPY);
}

#[test]
fn is_transaction_filterer_v60_gas_pin() {
    let target: Address = address!("00000000000000000000000000000000000000cc");
    let run = fixture(60).call(
        &arbownerpublic(),
        &calldata("isTransactionFilterer(address)", &[word_address(target)]),
    );
    assert_eq!(run.gas_used(), 2 * SLOAD + 2 * COPY);
}

// ── get-all set queries (variable, here pinning the empty-set form) ─

#[test]
fn get_all_chain_owners_empty_v30_gas_pin() {
    // Empty set: count=0 → (2 + 0) * SLOAD + (2 + 0) * COPY = 1606.
    let run = fixture(30).call(&arbownerpublic(), &calldata("getAllChainOwners()", &[]));
    assert_eq!(run.gas_used(), 2 * SLOAD + 2 * COPY);
}

#[test]
fn get_all_native_token_owners_empty_v41_gas_pin() {
    let run = fixture(41).call(
        &arbownerpublic(),
        &calldata("getAllNativeTokenOwners()", &[]),
    );
    assert_eq!(run.gas_used(), 2 * SLOAD + 2 * COPY);
}

#[test]
fn get_all_transaction_filterers_empty_v60_gas_pin() {
    let run = fixture(60).call(
        &arbownerpublic(),
        &calldata("getAllTransactionFilterers()", &[]),
    );
    assert_eq!(run.gas_used(), 2 * SLOAD + 2 * COPY);
}

// ── scheduled upgrade tuple: 3 SLOAD + 2 COPY = 2406 ────────────────

#[test]
fn get_scheduled_upgrade_v20_gas_pin() {
    let run = fixture(20).call(&arbownerpublic(), &calldata("getScheduledUpgrade()", &[]));
    assert_eq!(run.gas_used(), 3 * SLOAD + 2 * COPY);
}

// ── single-flag reads (2 SLOAD + COPY) ───────────────────────────────

#[test]
fn is_calldata_price_increase_enabled_v40_gas_pin() {
    let run = fixture(40).call(
        &arbownerpublic(),
        &calldata("isCalldataPriceIncreaseEnabled()", &[]),
    );
    assert_eq!(run.gas_used(), 2 * SLOAD + COPY);
}

#[test]
fn get_max_stylus_contract_fragments_v60_gas_pin() {
    // Uses a single Stylus params SLOAD + warm read of the next field.
    let run = fixture(60).call(
        &arbownerpublic(),
        &calldata("getMaxStylusContractFragments()", &[]),
    );
    assert_eq!(run.gas_used(), SLOAD + WARM_SLOAD + COPY);
}

#[test]
fn get_collect_tips_v60_gas_pin() {
    // OpenArbosState(800) via init_precompile_gas + body charge (800 + 3) = 1603.
    let run = fixture(60).call(&arbownerpublic(), &calldata("getCollectTips()", &[]));
    assert_eq!(run.gas_used(), 2 * SLOAD + COPY);
}

// ── rectifyChainOwner: requires a mapping-inconsistent owner.
//    Without that, the precompile reverts with the accumulated gas. ────

#[test]
fn rectify_chain_owner_below_v11_reverts() {
    let target: Address = address!("00000000000000000000000000000000000000aa");
    let run = fixture(10).gas(50_000).call(
        &arbownerpublic(),
        &calldata("rectifyChainOwner(address)", &[word_address(target)]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 50_000);
}

#[test]
fn rectify_chain_owner_v11_reverts_when_not_member_with_accumulated_gas() {
    // Not a member → empty_revert with the boilerplate gas accumulated so
    // far (OpenArbosState via init_precompile_gas + 1 word args).
    let target: Address = address!("00000000000000000000000000000000000000aa");
    let run = fixture(11).gas(50_000).call(
        &arbownerpublic(),
        &calldata("rectifyChainOwner(address)", &[word_address(target)]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    // init_precompile_gas(800 + 1*3) = 803.
    assert_eq!(out.gas_used, SLOAD + COPY);
}
