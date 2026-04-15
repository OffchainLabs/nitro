mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, B256, U256};
use arb_precompiles::{
    create_arbownerpublic_precompile,
    storage_slot::{
        derive_subspace_key, map_slot_b256, root_slot, subspace_slot, ARBOS_STATE_ADDRESS,
        CHAIN_OWNER_SUBSPACE, L1_PRICING_SUBSPACE, ROOT_STORAGE_KEY,
    },
};
use common::{calldata, decode_address, decode_u256, decode_word, word_address, PrecompileTest};

const NETWORK_FEE_ACCOUNT_OFFSET: u64 = 3;
const INFRA_FEE_ACCOUNT_OFFSET: u64 = 6;
const BROTLI_COMPRESSION_LEVEL_OFFSET: u64 = 7;
const UPGRADE_VERSION_OFFSET: u64 = 1;
const UPGRADE_TIMESTAMP_OFFSET: u64 = 2;
const L1_GAS_FLOOR_PER_TOKEN: u64 = 12;

fn arbownerpublic() -> DynPrecompile {
    create_arbownerpublic_precompile()
}

fn chain_owner_member_slot(owner: Address) -> U256 {
    let set_key = derive_subspace_key(ROOT_STORAGE_KEY, CHAIN_OWNER_SUBSPACE);
    let by_address_key = derive_subspace_key(set_key.as_slice(), &[0]);
    let addr_as_b256 = B256::left_padding_from(owner.as_slice());
    map_slot_b256(by_address_key.as_slice(), &addr_as_b256)
}

fn fixture(arbos_version: u64) -> PrecompileTest {
    PrecompileTest::new()
        .arbos_version(arbos_version)
        .arbos_state()
}

#[test]
fn get_network_fee_account_returns_root_field() {
    let fee: Address = address!("00000000000000000000000000000000000000ff");
    let val = U256::from_be_slice(fee.as_slice());
    let run = fixture(30)
        .storage(
            ARBOS_STATE_ADDRESS,
            root_slot(NETWORK_FEE_ACCOUNT_OFFSET),
            val,
        )
        .call(&arbownerpublic(), &calldata("getNetworkFeeAccount()", &[]));
    assert_eq!(decode_address(run.output()), fee);
}

#[test]
fn get_infra_fee_account_returns_root_field_at_v6() {
    let fee: Address = address!("00000000000000000000000000000000000000fe");
    let val = U256::from_be_slice(fee.as_slice());
    let run = fixture(6)
        .storage(
            ARBOS_STATE_ADDRESS,
            root_slot(INFRA_FEE_ACCOUNT_OFFSET),
            val,
        )
        .call(&arbownerpublic(), &calldata("getInfraFeeAccount()", &[]));
    assert_eq!(decode_address(run.output()), fee);
}

#[test]
fn get_infra_fee_account_falls_back_to_network_fee_account_below_v6() {
    // Nitro precompiles/ArbOwnerPublic.go::GetInfraFeeAccount returns
    // NetworkFeeAccount when ArbOSVersion < 6, not a revert.
    let network: Address = address!("00000000000000000000000000000000000000bb");
    let run = fixture(5)
        .storage(
            ARBOS_STATE_ADDRESS,
            root_slot(NETWORK_FEE_ACCOUNT_OFFSET),
            U256::from_be_slice(network.as_slice()),
        )
        .call(&arbownerpublic(), &calldata("getInfraFeeAccount()", &[]));
    assert_eq!(decode_address(run.output()), network);
}

#[test]
fn get_brotli_compression_level_at_v20() {
    let run = fixture(20)
        .storage(
            ARBOS_STATE_ADDRESS,
            root_slot(BROTLI_COMPRESSION_LEVEL_OFFSET),
            U256::from(11),
        )
        .call(
            &arbownerpublic(),
            &calldata("getBrotliCompressionLevel()", &[]),
        );
    assert_eq!(decode_u256(run.output()), U256::from(11));
}

#[test]
fn get_brotli_compression_level_returns_zero_below_v20_when_unset() {
    // Nitro doesn't gate this; just reads the field, which is 0 if uninit.
    let run = fixture(19).call(
        &arbownerpublic(),
        &calldata("getBrotliCompressionLevel()", &[]),
    );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn get_scheduled_upgrade_at_v20_returns_pair() {
    let run = fixture(20)
        .storage(
            ARBOS_STATE_ADDRESS,
            root_slot(UPGRADE_VERSION_OFFSET),
            U256::from(60),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            root_slot(UPGRADE_TIMESTAMP_OFFSET),
            U256::from(1_800_000_000_u64),
        )
        .call(&arbownerpublic(), &calldata("getScheduledUpgrade()", &[]));
    let out = run.output();
    assert_eq!(decode_word(out, 0), common::word_u64(60));
    assert_eq!(decode_word(out, 1), common::word_u64(1_800_000_000));
}

#[test]
fn is_chain_owner_returns_true_for_member() {
    let owner: Address = address!("00000000000000000000000000000000000000aa");
    let run = fixture(30)
        .storage(
            ARBOS_STATE_ADDRESS,
            chain_owner_member_slot(owner),
            U256::from(1),
        )
        .call(
            &arbownerpublic(),
            &calldata("isChainOwner(address)", &[word_address(owner)]),
        );
    assert_eq!(decode_u256(run.output()), U256::from(1));
}

#[test]
fn is_chain_owner_returns_false_for_non_member() {
    let stranger: Address = address!("00000000000000000000000000000000000000bb");
    let run = fixture(30).call(
        &arbownerpublic(),
        &calldata("isChainOwner(address)", &[word_address(stranger)]),
    );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn rectify_chain_owner_errors_when_caller_not_owner_at_any_version() {
    // Nitro precompiles/ArbOwnerPublic.go::RectifyChainOwner doesn't gate by
    // version. It calls ChainOwners().RectifyMapping(addr), which errors with
    // "RectifyMapping: Address is not an owner" when addr isn't tracked.
    let target: Address = address!("00000000000000000000000000000000000000cc");
    let run = fixture(10).call(
        &arbownerpublic(),
        &calldata("rectifyChainOwner(address)", &[word_address(target)]),
    );
    assert!(run.result.is_err(), "non-owner rectify must error");
}

#[test]
fn get_parent_gas_floor_per_token_at_v50() {
    let run = fixture(50)
        .storage(
            ARBOS_STATE_ADDRESS,
            subspace_slot(L1_PRICING_SUBSPACE, L1_GAS_FLOOR_PER_TOKEN),
            U256::from(4),
        )
        .call(
            &arbownerpublic(),
            &calldata("getParentGasFloorPerToken()", &[]),
        );
    assert_eq!(decode_u256(run.output()), U256::from(4));
}

#[test]
fn get_parent_gas_floor_per_token_returns_zero_below_v50_when_unset() {
    // Nitro precompiles/ArbOwnerPublic.go::GetParentGasFloorPerToken is not
    // version-gated; it just reads the L1PricingState field. Returning 0 from
    // an uninitialised field is correct.
    let run = fixture(49).call(
        &arbownerpublic(),
        &calldata("getParentGasFloorPerToken()", &[]),
    );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn is_native_token_owner_returns_false_below_v41_when_unset() {
    let target: Address = address!("00000000000000000000000000000000000000dd");
    let run = fixture(40).call(
        &arbownerpublic(),
        &calldata("isNativeTokenOwner(address)", &[word_address(target)]),
    );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn get_max_stylus_contract_fragments_returns_zero_below_v60() {
    let run = fixture(59).call(
        &arbownerpublic(),
        &calldata("getMaxStylusContractFragments()", &[]),
    );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}
