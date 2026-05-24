use alloy_primitives::{Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    guards::GuardedRun,
    scaffolding::{baseline_stylus_plus_helper, selector4, signed, INVOKE_GAS_CAP},
    shared_nodes::next_msg_idx,
};
use arb_test_harness::messaging::MessageBuilder;

const ARBRETRYABLETX: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x6e,
]);

fn one_arg_b32(sig: &str, h: B256) -> Bytes {
    let mut out = Vec::with_capacity(36);
    out.extend_from_slice(&selector4(sig));
    out.extend_from_slice(h.as_slice());
    Bytes::from(out)
}

fn no_arg(sig: &str) -> Bytes {
    Bytes::from(selector4(sig).to_vec())
}

#[test]
#[ignore]
fn get_lifetime_query_succeeds() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBRETRYABLETX),
        no_arg("getLifetime()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("retry_get_lifetime", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn get_current_redeemer_query_succeeds() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBRETRYABLETX),
        no_arg("getCurrentRedeemer()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("retry_get_current_redeemer", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn get_timeout_unknown_ticket_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBRETRYABLETX),
        one_arg_b32("getTimeout(bytes32)", B256::repeat_byte(0xab)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("retry_get_timeout_unknown", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn get_timeout_zero_ticket_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBRETRYABLETX),
        one_arg_b32("getTimeout(bytes32)", B256::ZERO),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("retry_get_timeout_zero", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn get_beneficiary_unknown_ticket_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBRETRYABLETX),
        one_arg_b32("getBeneficiary(bytes32)", B256::repeat_byte(0xcd)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("retry_get_beneficiary_unknown", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn redeem_unknown_ticket_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBRETRYABLETX),
        one_arg_b32("redeem(bytes32)", B256::repeat_byte(0xef)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("retry_redeem_unknown", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn cancel_unknown_ticket_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBRETRYABLETX),
        one_arg_b32("cancel(bytes32)", B256::repeat_byte(0x11)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("retry_cancel_unknown", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn keepalive_unknown_ticket_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBRETRYABLETX),
        one_arg_b32("keepalive(bytes32)", B256::repeat_byte(0x22)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("retry_keepalive_unknown", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn submit_retryable_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBRETRYABLETX),
        no_arg("submitRetryable(bytes32,uint256,uint256,uint256,uint256,uint64,uint256,address,address,address,bytes)"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("retry_submit_via_precompile", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn invalid_selector_arbretryable_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBRETRYABLETX),
        Bytes::from(vec![0xfa, 0xce, 0xb0, 0x0c]),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("retry_invalid_selector", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}
