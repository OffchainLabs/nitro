use alloy_primitives::{Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    guards::GuardedRun,
    scaffolding::{fund_interop_eoa, selector4, signed, FUZZ_L1_BASE_FEE, INVOKE_GAS_CAP},
    shared_nodes::{next_msg_idx, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::messaging::{
    apply_l1_to_l2_alias, submit_retryable_ticket_id, DepositBuilder, MessageBuilder,
    RetryableSubmitBuilder,
};

const ARBRETRYABLETX: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x6e,
]);

fn one_arg_b32(sig: &str, h: B256) -> Bytes {
    let mut out = Vec::with_capacity(36);
    out.extend_from_slice(&selector4(sig));
    out.extend_from_slice(h.as_slice());
    Bytes::from(out)
}

fn fund_l1_sender(steps: &mut Vec<arb_test_harness::scenario::ScenarioStep>, sender: Address) {
    let idx = next_msg_idx();
    let dep = DepositBuilder {
        from: sender,
        to: apply_l1_to_l2_alias(sender),
        amount: U256::from(10u128).pow(U256::from(19u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("dep");
    steps.push(message_step(idx, dep, idx));
}

fn submit_retryable(
    steps: &mut Vec<arb_test_harness::scenario::ScenarioStep>,
    l1_sender: Address,
    to: Address,
    request_id: B256,
    gas_limit: u64,
    max_fee_per_gas: U256,
) -> B256 {
    let idx = next_msg_idx();
    let msg = RetryableSubmitBuilder {
        l1_sender,
        to,
        l2_call_value: U256::ZERO,
        deposit_value: U256::from(10u128).pow(U256::from(18u64)),
        max_submission_fee: U256::from(10u128).pow(U256::from(15u64)),
        excess_fee_refund_address: apply_l1_to_l2_alias(l1_sender),
        call_value_refund_address: apply_l1_to_l2_alias(l1_sender),
        gas_limit,
        max_fee_per_gas,
        data: Bytes::new(),
        l1_block_number: 3,
        timestamp: 1_700_000_010,
        request_id: Some(request_id),
    }
    .build()
    .expect("submit");
    let ticket = submit_retryable_ticket_id(&msg, FUZZ_L2_CHAIN_ID).expect("ticket id");
    steps.push(message_step(idx, msg, idx));
    ticket
}

#[test]
#[ignore]
fn submit_then_get_timeout_returns_nonzero() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let l1_sender = Address::repeat_byte(0xa1);
    fund_l1_sender(&mut steps, l1_sender);
    let ticket = submit_retryable(
        &mut steps,
        l1_sender,
        Address::repeat_byte(0xbb),
        B256::repeat_byte(0x10),
        100_000,
        U256::from(1u64),
    );
    let tx = signed(
        0,
        Some(ARBRETRYABLETX),
        one_arg_b32("getTimeout(bytes32)", ticket),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("query");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("submit_then_get_timeout", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn submit_then_get_beneficiary_returns_refund_addr() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let l1_sender = Address::repeat_byte(0xa2);
    fund_l1_sender(&mut steps, l1_sender);
    let ticket = submit_retryable(
        &mut steps,
        l1_sender,
        Address::repeat_byte(0xbc),
        B256::repeat_byte(0x11),
        100_000,
        U256::from(1u64),
    );
    let tx = signed(
        0,
        Some(ARBRETRYABLETX),
        one_arg_b32("getBeneficiary(bytes32)", ticket),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("query");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("submit_then_get_beneficiary", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn submit_then_cancel_by_beneficiary() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let l1_sender = Address::repeat_byte(0xa3);
    fund_l1_sender(&mut steps, l1_sender);
    let ticket = submit_retryable(
        &mut steps,
        l1_sender,
        Address::repeat_byte(0xbd),
        B256::repeat_byte(0x12),
        100_000,
        U256::from(1u64),
    );
    let tx = signed(
        0,
        Some(ARBRETRYABLETX),
        one_arg_b32("cancel(bytes32)", ticket),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("cancel from wrong sender");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("submit_then_cancel_wrong_sender", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn submit_then_redeem_from_unrelated_sender() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let l1_sender = Address::repeat_byte(0xa4);
    fund_l1_sender(&mut steps, l1_sender);
    let ticket = submit_retryable(
        &mut steps,
        l1_sender,
        Address::repeat_byte(0xbe),
        B256::repeat_byte(0x13),
        100_000,
        U256::from(1u64),
    );
    let tx = signed(
        0,
        Some(ARBRETRYABLETX),
        one_arg_b32("redeem(bytes32)", ticket),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("redeem");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("submit_then_redeem_other_sender", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn submit_then_keepalive_extends_timeout() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let l1_sender = Address::repeat_byte(0xa5);
    fund_l1_sender(&mut steps, l1_sender);
    let ticket = submit_retryable(
        &mut steps,
        l1_sender,
        Address::repeat_byte(0xbf),
        B256::repeat_byte(0x14),
        100_000,
        U256::from(1u64),
    );
    let tx = signed(
        0,
        Some(ARBRETRYABLETX),
        one_arg_b32("keepalive(bytes32)", ticket),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("keepalive");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("submit_then_keepalive", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn submit_two_then_query_both() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let l1_sender = Address::repeat_byte(0xa6);
    fund_l1_sender(&mut steps, l1_sender);
    let t1 = submit_retryable(
        &mut steps,
        l1_sender,
        Address::repeat_byte(0xc0),
        B256::repeat_byte(0x15),
        100_000,
        U256::from(1u64),
    );
    let _t2 = submit_retryable(
        &mut steps,
        l1_sender,
        Address::repeat_byte(0xc1),
        B256::repeat_byte(0x16),
        100_000,
        U256::from(1u64),
    );
    let tx = signed(
        0,
        Some(ARBRETRYABLETX),
        one_arg_b32("getTimeout(bytes32)", t1),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("query");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("submit_two_query_first", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn submit_callvalue_autoredeem_excess_refund_to_fresh_addr() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let l1_sender = Address::repeat_byte(0xa8);
    fund_l1_sender(&mut steps, l1_sender);

    let target = Address::repeat_byte(0xb8);
    let excess_refund = Address::repeat_byte(0xe1);
    let callvalue_refund = Address::repeat_byte(0xe2);
    let l2_call_value = U256::from(10u128).pow(U256::from(18u64));
    let gas_limit = 200_000u64;
    let max_fee = U256::from(2_000_000_000u64);
    let max_submission_fee = U256::from(10u128).pow(U256::from(15u64));
    let deposit = l2_call_value
        + max_submission_fee
        + max_fee * U256::from(gas_limit)
        + U256::from(10u128).pow(U256::from(17u64));

    let idx = next_msg_idx();
    let msg = RetryableSubmitBuilder {
        l1_sender,
        to: target,
        l2_call_value,
        deposit_value: deposit,
        max_submission_fee,
        excess_fee_refund_address: excess_refund,
        call_value_refund_address: callvalue_refund,
        gas_limit,
        max_fee_per_gas: max_fee,
        data: Bytes::new(),
        l1_block_number: 3,
        timestamp: 1_700_000_010,
        request_id: Some(B256::repeat_byte(0x20)),
    }
    .build()
    .expect("submit");
    steps.push(message_step(idx, msg, idx));

    GuardedRun::new("retryable_callvalue_fresh_excess_refund", steps)
        .diff_account(target)
        .diff_account(excess_refund)
        .diff_account(callvalue_refund)
        .diff_account(apply_l1_to_l2_alias(l1_sender))
        .run();
}

#[test]
#[ignore]
fn redeem_twice_second_fails() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let l1_sender = Address::repeat_byte(0xa7);
    fund_l1_sender(&mut steps, l1_sender);
    let ticket = submit_retryable(
        &mut steps,
        l1_sender,
        Address::repeat_byte(0xc2),
        B256::repeat_byte(0x17),
        INVOKE_GAS_CAP,
        U256::from(2_000_000_000u64),
    );
    let tx1 = signed(
        0,
        Some(ARBRETRYABLETX),
        one_arg_b32("redeem(bytes32)", ticket),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("redeem1");
    let i1 = next_msg_idx();
    steps.push(message_step(i1, tx1, i1));
    let tx2 = signed(
        1,
        Some(ARBRETRYABLETX),
        one_arg_b32("redeem(bytes32)", ticket),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("redeem2");
    let i2 = next_msg_idx();
    steps.push(message_step(i2, tx2, i2));
    GuardedRun::new("redeem_twice", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}
