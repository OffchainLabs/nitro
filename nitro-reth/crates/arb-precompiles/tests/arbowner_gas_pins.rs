//! Per-selector gas pins for ArbOwner (0x70).
//!
//! ArbOwner is owner-gated: every selector first runs `verify_owner`, which
//! reads the chain_owners membership for `caller` (2 SLOAD = 1600). After
//! that the dispatcher resets the per-call gas accumulator and runs the
//! selector body. The dispatcher then writes `gas_used = 0` into the output
//! struct on success or revert; the final billable cost is what
//! `gas_check` sees on the error path, i.e. the wrapper's `gas_used`.

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, B256, U256};
use arb_precompiles::{
    create_arbowner_precompile,
    storage_slot::{
        derive_subspace_key, map_slot_b256, ARBOS_STATE_ADDRESS, CHAIN_OWNER_SUBSPACE,
        ROOT_STORAGE_KEY,
    },
};
use common::{calldata, word_address, word_u256, PrecompileTest};

const OWNER: Address = address!("00000000000000000000000000000000000000aa");
const INTRUDER: Address = address!("00000000000000000000000000000000000000bb");

fn arbowner(ctx: std::sync::Arc<arb_context::ArbPrecompileCtx>) -> DynPrecompile {
    create_arbowner_precompile(ctx)
}

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

fn fixture(v: u64) -> PrecompileTest {
    install_owner(
        PrecompileTest::new()
            .arbos_version(v)
            .caller(OWNER)
            .arbos_state(),
        OWNER,
    )
}

// ── Owner-gating: caller not in chain_owners reverts ────────────────

#[test]
fn caller_not_owner_propagates_revert_err() {
    // verify_owner returns Err on non-owner — the dispatcher uses `?` so
    // the error escapes before the post-call wrapper would have converted
    // it via `gas_check`. The result reaches the caller as Err.
    let run = PrecompileTest::new()
        .arbos_version(30)
        .caller(INTRUDER)
        .arbos_state()
        .gas(50_000)
        .call(arbowner, &calldata("getNetworkFeeAccount()", &[]));
    assert!(run.result.is_err());
}

// ── Body returns gas, dispatcher overrides to 0 ─────────────────────
//
// The dispatcher discards the body's gas_used and writes 0 into the
// returned PrecompileOutput. These pins lock that observation in.

#[test]
fn get_network_fee_account_owner_dispatch_zeros_gas() {
    let run = fixture(30).call(arbowner, &calldata("getNetworkFeeAccount()", &[]));
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn get_infra_fee_account_owner_v6_zero_gas() {
    let run = fixture(6).call(arbowner, &calldata("getInfraFeeAccount()", &[]));
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn get_infra_fee_account_below_v5_reverts_burning_gas_limit() {
    let run = fixture(4)
        .gas(50_000)
        .call(arbowner, &calldata("getInfraFeeAccount()", &[]));
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 50_000);
}

#[test]
fn is_chain_owner_owner_zero_gas() {
    let run = fixture(30).call(
        arbowner,
        &calldata("isChainOwner(address)", &[word_address(OWNER)]),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn get_all_chain_owners_owner_zero_gas() {
    let run = fixture(30).call(arbowner, &calldata("getAllChainOwners()", &[]));
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn add_chain_owner_zero_gas() {
    let new_owner: Address = address!("00000000000000000000000000000000000000cc");
    let run = fixture(30).call(
        arbowner,
        &calldata("addChainOwner(address)", &[word_address(new_owner)]),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn remove_chain_owner_zero_gas() {
    // remove non-existing owner → may revert; but ArbOwner discards body
    // gas anyway. We pin the zero-output form.
    let target: Address = address!("00000000000000000000000000000000000000ff");
    let run = fixture(30).call(
        arbowner,
        &calldata("removeChainOwner(address)", &[word_address(target)]),
    );
    // Could be reverted (target not in set); either way gas_used = 0.
    assert_eq!(run.assert_ok().gas_used, 0);
}

#[test]
fn set_network_fee_account_zero_gas() {
    let new_account: Address = address!("00000000000000000000000000000000000000cc");
    let run = fixture(30).call(
        arbowner,
        &calldata(
            "setNetworkFeeAccount(address)",
            &[word_address(new_account)],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_infra_fee_account_v6_zero_gas() {
    let new_account: Address = address!("00000000000000000000000000000000000000dd");
    let run = fixture(6).call(
        arbowner,
        &calldata("setInfraFeeAccount(address)", &[word_address(new_account)]),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_l2_base_fee_zero_gas() {
    let run = fixture(30).call(
        arbowner,
        &calldata(
            "setL2BaseFee(uint256)",
            &[word_u256(U256::from(1_000_000u64))],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_minimum_l2_base_fee_zero_gas() {
    let run = fixture(30).call(
        arbowner,
        &calldata(
            "setMinimumL2BaseFee(uint256)",
            &[word_u256(U256::from(1u64))],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_speed_limit_zero_gas() {
    let run = fixture(30).call(
        arbowner,
        &calldata(
            "setSpeedLimit(uint64)",
            &[word_u256(U256::from(1_000_000u64))],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_max_tx_gas_limit_zero_gas() {
    let run = fixture(30).call(
        arbowner,
        &calldata(
            "setMaxTxGasLimit(uint64)",
            &[word_u256(U256::from(32_000_000u64))],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_l2_gas_pricing_inertia_zero_gas() {
    let run = fixture(30).call(
        arbowner,
        &calldata(
            "setL2GasPricingInertia(uint64)",
            &[word_u256(U256::from(102u64))],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_l2_gas_backlog_tolerance_zero_gas() {
    let run = fixture(30).call(
        arbowner,
        &calldata(
            "setL2GasBacklogTolerance(uint64)",
            &[word_u256(U256::from(10u64))],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_gas_backlog_v50_zero_gas() {
    let run = fixture(50).call(
        arbowner,
        &calldata("setGasBacklog(uint64)", &[word_u256(U256::from(7_777u64))]),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_gas_backlog_below_v50_reverts_burning_gas_limit() {
    let run = fixture(49).gas(50_000).call(
        arbowner,
        &calldata("setGasBacklog(uint64)", &[word_u256(U256::from(1u64))]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 50_000);
}

#[test]
fn set_l1_price_per_unit_zero_gas() {
    let run = fixture(30).call(
        arbowner,
        &calldata(
            "setL1PricePerUnit(uint256)",
            &[word_u256(U256::from(50_000_000_000u64))],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_per_batch_gas_charge_zero_gas() {
    let run = fixture(30).call(
        arbowner,
        &calldata(
            "setPerBatchGasCharge(int64)",
            &[word_u256(U256::from(210_000u64))],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_amortized_cost_cap_bips_zero_gas() {
    let run = fixture(30).call(
        arbowner,
        &calldata(
            "setAmortizedCostCapBips(uint64)",
            &[word_u256(U256::from(2_000u64))],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn schedule_arbos_upgrade_zero_gas() {
    let run = fixture(30).call(
        arbowner,
        &calldata(
            "scheduleArbOSUpgrade(uint64,uint64)",
            &[
                word_u256(U256::from(31u64)),
                word_u256(U256::from(2_000_000_000u64)),
            ],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_brotli_compression_level_v20_zero_gas() {
    let run = fixture(20).call(
        arbowner,
        &calldata(
            "setBrotliCompressionLevel(uint64)",
            &[word_u256(U256::from(1u64))],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_l1_pricing_equilibration_units_v20_zero_gas() {
    let run = fixture(20).call(
        arbowner,
        &calldata(
            "setL1PricingEquilibrationUnits(uint256)",
            &[word_u256(U256::from(1_000_000u64))],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_l1_pricing_inertia_zero_gas() {
    let run = fixture(30).call(
        arbowner,
        &calldata(
            "setL1PricingInertia(uint64)",
            &[word_u256(U256::from(10u64))],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_l1_pricing_reward_rate_v11_zero_gas() {
    let run = fixture(11).call(
        arbowner,
        &calldata(
            "setL1PricingRewardRate(uint64)",
            &[word_u256(U256::from(7u64))],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_l1_pricing_reward_recipient_v11_zero_gas() {
    let recipient: Address = address!("00000000000000000000000000000000000000ee");
    let run = fixture(11).call(
        arbowner,
        &calldata(
            "setL1PricingRewardRecipient(address)",
            &[word_address(recipient)],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_max_block_gas_limit_v50_zero_gas() {
    let run = fixture(50).call(
        arbowner,
        &calldata(
            "setMaxBlockGasLimit(uint64)",
            &[word_u256(U256::from(32_000_000u64))],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn set_wasm_max_stack_depth_v30_zero_gas() {
    let run = fixture(30).call(
        arbowner,
        &calldata(
            "setWasmMaxStackDepth(uint32)",
            &[word_u256(U256::from(1024u64))],
        ),
    );
    assert_eq!(run.gas_used(), 0);
}

#[test]
fn release_l1_pricer_surplus_funds_zero_gas() {
    let run = fixture(30).call(
        arbowner,
        &calldata(
            "releaseL1PricerSurplusFunds(uint256)",
            &[word_u256(U256::from(0u64))],
        ),
    );
    // Either succeeds or reverts; in both cases dispatcher writes 0.
    assert_eq!(run.assert_ok().gas_used, 0);
}
