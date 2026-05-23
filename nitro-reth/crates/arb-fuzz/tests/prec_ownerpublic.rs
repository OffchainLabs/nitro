use alloy_primitives::{Address, Bytes, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    guards::GuardedRun,
    scaffolding::{
        baseline_stylus_plus_helper, selector4, signed, INVOKE_GAS_CAP,
    },
    shared_nodes::next_msg_idx,
};
use arb_test_harness::messaging::MessageBuilder;

const ARBOWNERPUBLIC: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x6b,
]);

fn no_arg(sig: &str) -> Bytes {
    Bytes::from(selector4(sig).to_vec())
}

fn one_arg_addr(sig: &str, who: Address) -> Bytes {
    let mut out = Vec::with_capacity(36);
    out.extend_from_slice(&selector4(sig));
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(who.as_slice());
    out.extend_from_slice(&pad);
    Bytes::from(out)
}

fn run_query(name: &str, data: Bytes, expect_success: bool) {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(3, Some(ARBOWNERPUBLIC), data, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    let mut run = GuardedRun::new(name, steps).expect_last_tx_min_gas(25_000);
    if expect_success {
        run = run.expect_last_tx_status(true);
    }
    run.run();
}

#[test]
#[ignore]
fn ownerpublic_get_network_fee_account() {
    run_query("op_network_fee_account", no_arg("getNetworkFeeAccount()"), true);
}

#[test]
#[ignore]
fn ownerpublic_get_infra_fee_account() {
    run_query("op_infra_fee_account", no_arg("getInfraFeeAccount()"), true);
}

#[test]
#[ignore]
fn ownerpublic_get_all_chain_owners() {
    run_query("op_all_chain_owners", no_arg("getAllChainOwners()"), true);
}

#[test]
#[ignore]
fn ownerpublic_get_all_native_token_owners() {
    run_query("op_all_nt_owners", no_arg("getAllNativeTokenOwners()"), true);
}

#[test]
#[ignore]
fn ownerpublic_is_chain_owner_random_false() {
    run_query(
        "op_is_chain_owner_random",
        one_arg_addr("isChainOwner(address)", Address::repeat_byte(0xa1)),
        true,
    );
}

#[test]
#[ignore]
fn ownerpublic_is_native_token_owner_random_false() {
    run_query(
        "op_is_nt_owner_random",
        one_arg_addr("isNativeTokenOwner(address)", Address::repeat_byte(0xa2)),
        true,
    );
}

#[test]
#[ignore]
fn ownerpublic_get_brotli_compression_level() {
    run_query("op_brotli_level", no_arg("getBrotliCompressionLevel()"), true);
}

#[test]
#[ignore]
fn ownerpublic_get_scheduled_upgrade() {
    run_query("op_scheduled_upgrade", no_arg("getScheduledUpgrade()"), true);
}

#[test]
#[ignore]
fn ownerpublic_get_max_stylus_contract_fragments() {
    run_query(
        "op_max_stylus_fragments",
        no_arg("getMaxStylusContractFragments()"),
        true,
    );
}

#[test]
#[ignore]
fn ownerpublic_get_transaction_filtering_from() {
    run_query(
        "op_tx_filtering_from",
        no_arg("getTransactionFilteringFrom()"),
        true,
    );
}

#[test]
#[ignore]
fn ownerpublic_get_native_token_management_from() {
    run_query(
        "op_nt_management_from",
        no_arg("getNativeTokenManagementFrom()"),
        true,
    );
}

#[test]
#[ignore]
fn ownerpublic_get_parent_gas_floor_per_token() {
    run_query(
        "op_parent_gas_floor",
        no_arg("getParentGasFloorPerToken()"),
        true,
    );
}

#[test]
#[ignore]
fn ownerpublic_is_calldata_price_increase_enabled() {
    run_query(
        "op_calldata_price_inc",
        no_arg("isCalldataPriceIncreaseEnabled()"),
        true,
    );
}

#[test]
#[ignore]
fn ownerpublic_rectify_chain_owner_non_owner_reverts() {
    run_query(
        "op_rectify_non_owner",
        one_arg_addr("rectifyChainOwner(address)", Address::repeat_byte(0xa3)),
        false,
    );
}
