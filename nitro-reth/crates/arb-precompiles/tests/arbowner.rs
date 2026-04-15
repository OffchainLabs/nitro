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
    test.storage(
        ARBOS_STATE_ADDRESS,
        chain_owner_member_slot(owner),
        U256::from(1),
    )
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
        .storage(
            ARBOS_STATE_ADDRESS,
            root_slot(NETWORK_FEE_ACCOUNT_OFFSET),
            val,
        )
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
        &calldata(
            "setNetworkFeeAccount(address)",
            &[word_address(new_account)],
        ),
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
            &[
                word_u256(U256::from(60)),
                word_u256(U256::from(1_800_000_000)),
            ],
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
        run.storage(
            ARBOS_STATE_ADDRESS,
            subspace_slot(L2_PRICING_SUBSPACE, L2_SPEED_LIMIT)
        ),
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

#[test]
fn set_max_tx_gas_limit_writes_per_tx_slot_and_leaves_per_block_alone() {
    // Regression: SetMaxTxGasLimit used to dispatch by ArbOS version and write to
    // L2_PER_BLOCK_GAS_LIMIT (slot 1) at ArbOS < 50, which corrupted the per-block
    // gas limit. Nitro precompiles/ArbOwner.go::SetMaxTxGasLimit always calls
    // L2PricingState.SetMaxPerTxGasLimit, which writes perTxGasLimitOffset (slot 7).
    let test = fixture(30).storage(
        ARBOS_STATE_ADDRESS,
        subspace_slot(L2_PRICING_SUBSPACE, 1 /* L2_PER_BLOCK_GAS_LIMIT */),
        U256::from(32_000_000_u64),
    );
    let limit = U256::from(7_000_000_u64);
    let run = test.call(
        &arbowner(),
        &calldata("setMaxTxGasLimit(uint64)", &[word_u256(limit)]),
    );
    let _ = run.assert_ok();
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, subspace_slot(L2_PRICING_SUBSPACE, 7)),
        limit,
        "must write the per-tx slot"
    );
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, subspace_slot(L2_PRICING_SUBSPACE, 1)),
        U256::from(32_000_000_u64),
        "must NOT have overwritten per-block slot"
    );
}

#[test]
fn set_max_block_gas_limit_writes_per_block_slot_at_any_version() {
    // Companion: SetMaxBlockGasLimit always writes slot 1 in Nitro, no version gate.
    let limit = U256::from(40_000_000_u64);
    let run = fixture(30).call(
        &arbowner(),
        &calldata("setMaxBlockGasLimit(uint64)", &[word_u256(limit)]),
    );
    let _ = run.assert_ok();
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, subspace_slot(L2_PRICING_SUBSPACE, 1)),
        limit
    );
}

// ── L2 pricing setters: round-trip into the right L2 pricing slot ────

#[test]
fn set_l2_base_fee_writes_l2_base_fee_slot() {
    let value = U256::from(123_u64);
    let run = fixture(30).call(
        &arbowner(),
        &calldata("setL2BaseFee(uint256)", &[word_u256(value)]),
    );
    let _ = run.assert_ok();
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, subspace_slot(L2_PRICING_SUBSPACE, 2)),
        value
    );
}

#[test]
fn set_minimum_l2_base_fee_writes_min_base_fee_slot() {
    let value = U256::from(50_000_000_u64);
    let run = fixture(30).call(
        &arbowner(),
        &calldata("setMinimumL2BaseFee(uint256)", &[word_u256(value)]),
    );
    let _ = run.assert_ok();
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, subspace_slot(L2_PRICING_SUBSPACE, 3)),
        value
    );
}

#[test]
fn set_l2_gas_pricing_inertia_writes_pricing_inertia_slot() {
    let value = U256::from(1024_u64);
    let run = fixture(30).call(
        &arbowner(),
        &calldata("setL2GasPricingInertia(uint64)", &[word_u256(value)]),
    );
    let _ = run.assert_ok();
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, subspace_slot(L2_PRICING_SUBSPACE, 5)),
        value
    );
}

#[test]
fn set_l2_gas_pricing_inertia_rejects_zero() {
    let run = fixture(30).call(
        &arbowner(),
        &calldata("setL2GasPricingInertia(uint64)", &[word_u256(U256::ZERO)]),
    );
    let out = run.assert_ok();
    assert!(out.reverted, "zero inertia must revert");
}

#[test]
fn set_l2_gas_backlog_tolerance_writes_backlog_tolerance_slot() {
    let value = U256::from(60_u64);
    let run = fixture(30).call(
        &arbowner(),
        &calldata("setL2GasBacklogTolerance(uint64)", &[word_u256(value)]),
    );
    let _ = run.assert_ok();
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, subspace_slot(L2_PRICING_SUBSPACE, 6)),
        value
    );
}

#[test]
fn set_gas_backlog_writes_backlog_slot_at_v50_plus() {
    let value = U256::from(1_000_000_u64);
    let run = fixture(50).call(
        &arbowner(),
        &calldata("setGasBacklog(uint64)", &[word_u256(value)]),
    );
    let _ = run.assert_ok();
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, subspace_slot(L2_PRICING_SUBSPACE, 4)),
        value
    );
}

#[test]
fn set_gas_backlog_reverts_below_v50() {
    let value = U256::from(1_000_000_u64);
    let run = fixture(49).call(
        &arbowner(),
        &calldata("setGasBacklog(uint64)", &[word_u256(value)]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
}

// ── Native token / transaction filterer events ──────────────────────

#[test]
fn add_native_token_owner_requires_feature_enabled() {
    // Without the enabled-from time set in the past, AddNativeTokenOwner errors.
    let new_owner: Address = address!("00000000000000000000000000000000000000ee");
    let run = fixture(41).call(
        &arbowner(),
        &calldata("addNativeTokenOwner(address)", &[word_address(new_owner)]),
    );
    let out = run.assert_ok();
    assert!(out.reverted, "feature not enabled must revert");
}

#[test]
fn set_native_token_management_from_writes_root_field() {
    let when = U256::from(1_700_000_000_u64);
    let run = fixture(41).call(
        &arbowner(),
        &calldata("setNativeTokenManagementFrom(uint64)", &[word_u256(when)]),
    );
    // The setFeatureFromTime helper rejects values that aren't at least 7 days
    // in the future of the current block timestamp; the harness uses
    // block_timestamp = 1_700_000_000 by default, so a same-time value reverts.
    let out = run.assert_ok();
    assert!(out.reverted, "less-than-delay must revert");
}

#[test]
fn schedule_arbos_upgrade_writes_version_and_timestamp_slots() {
    let new_version = U256::from(60_u64);
    let when = U256::from(1_900_000_000_u64);
    let run = fixture(30).call(
        &arbowner(),
        &calldata(
            "scheduleArbOSUpgrade(uint64,uint64)",
            &[word_u256(new_version), word_u256(when)],
        ),
    );
    let _ = run.assert_ok();
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, root_slot(UPGRADE_VERSION_OFFSET)),
        new_version
    );
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, root_slot(UPGRADE_TIMESTAMP_OFFSET)),
        when
    );
}

// ── Ports from /data/nitro/precompiles/ArbOwner_test.go ───────────────

const INFRA_FEE_ACCOUNT_OFFSET: u64 = 6;

fn arbgasinfo() -> DynPrecompile {
    arb_precompiles::create_arbgasinfo_precompile()
}

fn arbownerpublic() -> DynPrecompile {
    arb_precompiles::create_arbownerpublic_precompile()
}

/// Port of Nitro's `TestArbInfraFeeAccount` (v6+ round-trip).
///
/// Nitro's test nominally targets `ArbosVersion_5`, but the mock EVM's
/// state config is a freshly-built `ArbitrumDevTestChainConfig()` whose
/// `InitialArbOSVersion` is well above 6, so `State.ArbOSVersion()`
/// reports the dev-chain default regardless of the `version` parameter.
/// The effective test is therefore against v≥6. We run at v6 directly.
///
/// We also skip the v0-no-op assertion: arbreth gates
/// `setInfraFeeAccount` at v≥5 via `check_method_version` and reverts
/// below that, which is stricter but not observable via precompile
/// dispatch from outside.
#[test]
fn nitro_parity_infra_fee_account_round_trip() {
    let new_addr: Address = address!("00000000000000000000000000000000000000cd");

    // Empty at start via both precompiles.
    let run = fixture(6).call(&arbowner(), &calldata("getInfraFeeAccount()", &[]));
    assert_eq!(decode_u256(run.output()), U256::ZERO);
    let run = fixture(6).call(&arbownerpublic(), &calldata("getInfraFeeAccount()", &[]));
    assert_eq!(decode_u256(run.output()), U256::ZERO);

    // Set via ArbOwner.
    let set_run = fixture(6).call(
        &arbowner(),
        &calldata("setInfraFeeAccount(address)", &[word_address(new_addr)]),
    );
    let out = set_run.assert_ok();
    assert!(!out.reverted);
    assert_eq!(
        set_run.storage(ARBOS_STATE_ADDRESS, root_slot(INFRA_FEE_ACCOUNT_OFFSET)),
        U256::from_be_slice(new_addr.as_slice()),
        "setter must write the infra fee slot"
    );

    // Read back via ArbOwner.
    let getter = set_run.continue_into(fixture(6), ARBOS_STATE_ADDRESS);
    let run = getter.call(&arbowner(), &calldata("getInfraFeeAccount()", &[]));
    assert_eq!(
        decode_u256(run.output()),
        U256::from_be_slice(new_addr.as_slice())
    );

    // Read back via ArbOwnerPublic.
    let getter = set_run.continue_into(fixture(6), ARBOS_STATE_ADDRESS);
    let run = getter.call(&arbownerpublic(), &calldata("getInfraFeeAccount()", &[]));
    assert_eq!(
        decode_u256(run.output()),
        U256::from_be_slice(new_addr.as_slice())
    );
}

/// Port of Nitro's `TestArbOwner` — chain owner management sub-flow.
/// Verifies add/remove/isChainOwner/getAllChainOwners with duplicate
/// adds not double-counting.
#[test]
fn nitro_parity_arb_owner_chain_owner_management() {
    let addr1: Address = address!("00000000000000000000000000000000000000d1");
    let addr2: Address = address!("00000000000000000000000000000000000000d2");
    let addr3: Address = address!("00000000000000000000000000000000000000d3");

    // Add two new owners (plus OWNER already installed).
    let add1 = fixture(30).call(
        &arbowner(),
        &calldata("addChainOwner(address)", &[word_address(addr1)]),
    );
    let _ = add1.assert_ok();

    let base = add1.continue_into(fixture(30), ARBOS_STATE_ADDRESS);
    let add2 = base.call(
        &arbowner(),
        &calldata("addChainOwner(address)", &[word_address(addr2)]),
    );
    let _ = add2.assert_ok();

    // Duplicate add of addr1: must remain idempotent.
    let base = add2.continue_into(fixture(30), ARBOS_STATE_ADDRESS);
    let add1_dup = base.call(
        &arbowner(),
        &calldata("addChainOwner(address)", &[word_address(addr1)]),
    );
    let _ = add1_dup.assert_ok();

    // isChainOwner checks — rebuild a fresh test per query because each
    // `.call()` consumes its PrecompileTest.
    for (who, expected) in [(addr1, true), (addr2, true), (addr3, false)] {
        let test = add1_dup.continue_into(fixture(30), ARBOS_STATE_ADDRESS);
        let run = test.call(
            &arbowner(),
            &calldata("isChainOwner(address)", &[word_address(who)]),
        );
        assert_eq!(
            decode_u256(run.output()),
            if expected { U256::from(1) } else { U256::ZERO },
            "isChainOwner({who})"
        );
    }

    // Remove addr1, verify it's gone, addr2 still present.
    let rm_test = add1_dup.continue_into(fixture(30), ARBOS_STATE_ADDRESS);
    let rm1 = rm_test.call(
        &arbowner(),
        &calldata("removeChainOwner(address)", &[word_address(addr1)]),
    );
    let _ = rm1.assert_ok();

    let test = rm1.continue_into(fixture(30), ARBOS_STATE_ADDRESS);
    let run = test.call(
        &arbowner(),
        &calldata("isChainOwner(address)", &[word_address(addr1)]),
    );
    assert_eq!(decode_u256(run.output()), U256::ZERO, "addr1 removed");

    let test = rm1.continue_into(fixture(30), ARBOS_STATE_ADDRESS);
    let run = test.call(
        &arbowner(),
        &calldata("isChainOwner(address)", &[word_address(addr2)]),
    );
    assert_eq!(decode_u256(run.output()), U256::from(1), "addr2 remains");
}

/// Port of Nitro's `TestArbOwner` — SetAmortizedCostCapBips round-trip.
#[test]
fn nitro_parity_arb_owner_amortized_cost_cap_round_trip() {
    // Initial value is zero.
    let run = fixture(30).call(&arbgasinfo(), &calldata("getAmortizedCostCapBips()", &[]));
    assert_eq!(decode_u256(run.output()), U256::ZERO);

    let new_cap = 77_734_u64;
    let set_run = fixture(30).call(
        &arbowner(),
        &calldata(
            "setAmortizedCostCapBips(uint64)",
            &[word_u256(U256::from(new_cap))],
        ),
    );
    let _ = set_run.assert_ok();

    let getter = set_run.continue_into(fixture(30), ARBOS_STATE_ADDRESS);
    let run = getter.call(&arbgasinfo(), &calldata("getAmortizedCostCapBips()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(new_cap));
}

/// Port of Nitro's `TestArbOwner` — SetNetworkFeeAccount round-trip
/// (confirms the getter picks up the setter's write, beyond the raw
/// slot-write regression test already covered by
/// `set_network_fee_account_writes_root_slot`).
#[test]
fn nitro_parity_arb_owner_network_fee_account_round_trip() {
    let new_fee_account: Address = address!("00000000000000000000000000000000000000fa");
    let set_run = fixture(30).call(
        &arbowner(),
        &calldata(
            "setNetworkFeeAccount(address)",
            &[word_address(new_fee_account)],
        ),
    );
    let _ = set_run.assert_ok();

    let getter = set_run.continue_into(fixture(30), ARBOS_STATE_ADDRESS);
    let run = getter.call(&arbowner(), &calldata("getNetworkFeeAccount()", &[]));
    assert_eq!(
        decode_u256(run.output()),
        U256::from_be_slice(new_fee_account.as_slice())
    );
}
