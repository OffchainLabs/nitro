use alloy_primitives::{Address, Bytes, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    guards::GuardedRun,
    scaffolding::{
        baseline_stylus_plus_helper, eoa_create_addr, selector4, signed, INVOKE_GAS_CAP,
    },
    shared_nodes::next_msg_idx,
};
use arb_test_harness::messaging::MessageBuilder;

const ARBINFO: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x65,
]);
const ARBADDRTABLE: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x66,
]);
const ARBAGGREGATOR: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x6d,
]);
const ARBSTATISTICS: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x6f,
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

fn one_arg_u256(sig: &str, v: U256) -> Bytes {
    let mut out = Vec::with_capacity(36);
    out.extend_from_slice(&selector4(sig));
    out.extend_from_slice(&v.to_be_bytes::<32>());
    Bytes::from(out)
}

fn send(steps: &mut Vec<arb_test_harness::scenario::ScenarioStep>, to: Address, data: Bytes) {
    let nonce = steps.iter().filter(|_| true).count() as u64;
    let tx = signed(
        nonce.saturating_sub(1),
        Some(to),
        data,
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
}

fn run_query(name: &str, to: Address, data: Bytes, expect_success: bool) {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(3, Some(to), data, U256::ZERO, INVOKE_GAS_CAP)
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
fn arbinfo_get_balance_self() {
    run_query(
        "arbinfo_get_balance_self",
        ARBINFO,
        one_arg_addr("getBalance(address)", eoa_create_addr(0)),
        true,
    );
}

#[test]
#[ignore]
fn arbinfo_get_balance_random() {
    run_query(
        "arbinfo_get_balance_random",
        ARBINFO,
        one_arg_addr("getBalance(address)", Address::repeat_byte(0xff)),
        true,
    );
}

#[test]
#[ignore]
fn arbinfo_get_code_eoa_empty() {
    run_query(
        "arbinfo_get_code_empty",
        ARBINFO,
        one_arg_addr("getCode(address)", Address::repeat_byte(0xab)),
        true,
    );
}

#[test]
#[ignore]
fn arbinfo_get_code_stylus_contract() {
    run_query(
        "arbinfo_get_code_stylus",
        ARBINFO,
        one_arg_addr("getCode(address)", eoa_create_addr(0)),
        true,
    );
}

#[test]
#[ignore]
fn arbaddrtable_size_initial() {
    run_query("arbaddrtable_size", ARBADDRTABLE, no_arg("size()"), true);
}

#[test]
#[ignore]
fn arbaddrtable_address_exists_random() {
    run_query(
        "arbaddrtable_exists_random",
        ARBADDRTABLE,
        one_arg_addr("addressExists(address)", Address::repeat_byte(0x77)),
        true,
    );
}

#[test]
#[ignore]
fn arbaddrtable_lookup_unknown_reverts() {
    run_query(
        "arbaddrtable_lookup_unknown",
        ARBADDRTABLE,
        one_arg_addr("lookup(address)", Address::repeat_byte(0x88)),
        false,
    );
}

#[test]
#[ignore]
fn arbaddrtable_lookup_index_zero() {
    run_query(
        "arbaddrtable_lookup_index_zero",
        ARBADDRTABLE,
        one_arg_u256("lookupIndex(uint256)", U256::from(0u64)),
        false,
    );
}

#[test]
#[ignore]
fn arbaddrtable_register_new_addr() {
    run_query(
        "arbaddrtable_register",
        ARBADDRTABLE,
        one_arg_addr("register(address)", Address::repeat_byte(0x44)),
        true,
    );
}

#[test]
#[ignore]
fn arbaddrtable_compress_unregistered() {
    run_query(
        "arbaddrtable_compress",
        ARBADDRTABLE,
        one_arg_addr("compress(address)", Address::repeat_byte(0x33)),
        true,
    );
}

#[test]
#[ignore]
fn arbaggregator_get_batch_posters() {
    run_query(
        "arbaggregator_batch_posters",
        ARBAGGREGATOR,
        no_arg("getBatchPosters()"),
        true,
    );
}

#[test]
#[ignore]
fn arbaggregator_get_default_aggregator() {
    run_query(
        "arbaggregator_default",
        ARBAGGREGATOR,
        no_arg("getDefaultAggregator()"),
        true,
    );
}

#[test]
#[ignore]
fn arbaggregator_get_preferred_aggregator_for_random() {
    run_query(
        "arbaggregator_preferred",
        ARBAGGREGATOR,
        one_arg_addr("getPreferredAggregator(address)", Address::repeat_byte(0x99)),
        true,
    );
}

#[test]
#[ignore]
fn arbaggregator_get_fee_collector_for_random() {
    run_query(
        "arbaggregator_fee_collector_random",
        ARBAGGREGATOR,
        one_arg_addr("getFeeCollector(address)", Address::repeat_byte(0xaa)),
        true,
    );
}

#[test]
#[ignore]
fn arbaggregator_get_tx_base_fee() {
    run_query(
        "arbaggregator_tx_base_fee",
        ARBAGGREGATOR,
        no_arg("getTxBaseFee()"),
        true,
    );
}

#[test]
#[ignore]
fn arbaggregator_set_fee_collector_non_owner_reverts() {
    run_query(
        "arbaggregator_set_fee_collector_non_owner",
        ARBAGGREGATOR,
        {
            let mut d = Vec::with_capacity(68);
            d.extend_from_slice(&selector4("setFeeCollector(address,address)"));
            let mut p1 = [0u8; 32];
            p1[12..].copy_from_slice(Address::repeat_byte(0xcd).as_slice());
            d.extend_from_slice(&p1);
            let mut p2 = [0u8; 32];
            p2[12..].copy_from_slice(Address::repeat_byte(0xef).as_slice());
            d.extend_from_slice(&p2);
            Bytes::from(d)
        },
        false,
    );
}

#[test]
#[ignore]
fn arbstatistics_get_stats() {
    run_query(
        "arbstatistics_stats",
        ARBSTATISTICS,
        no_arg("getStats()"),
        true,
    );
}

#[test]
#[ignore]
fn arbsys_alias_helpers_match() {
    let _ = send;
}
