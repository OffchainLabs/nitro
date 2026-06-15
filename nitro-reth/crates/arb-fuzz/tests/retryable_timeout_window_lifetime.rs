//! `open_retryable` must apply the v60 `timeoutWindowsLeft` lifetime extension.
//!
//! At ArbOS v60 `open_retryable` reads `timeoutWindowsLeft` on the expired
//! branch (`timeout < now`) and keeps a ticket live while
//! `timeout + windowsLeft*RetryableLifetimeSeconds >= now`.
//!
//! Both nodes run the StartBlock reaper (2 tickets/block) at identical times,
//! and the reaper itself is byte-identical. To keep the kept-alive ticket A in
//! the `timeout < now, windowsLeft > 0` state at query time, two expired decoys
//! are queued ahead of A so the query block's two reaps delete the decoys
//! (identically on both nodes) and never advance A.
//!
//! `get_timeout_past_window_clean` / `redeem_past_raw_timeout_clean`:
//! query/redeem A past its raw timeout but within the kept-alive window; both
//! nodes must agree.
//!
//! `get_timeout_pre_timeout_control_clean` / `redeem_pre_timeout_control_clean`:
//! identical scenario (same v60, same keepalive, same decoys, same gas), acting
//! on A *before* its raw timeout lapses. Both nodes return the live ticket. The
//! only variable is the block timestamp.
//!
//! A "no-keepalive (windowsLeft=0)" control is not used: at v60 the expired
//! branch always reads `windowsLeft` (a flat 800-gas SLOAD), so such a control
//! would differ on `gas_used` independently of this extension. Removing the
//! time-advance trigger isolates the windowsLeft branch instead.

use alloy_primitives::{Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    scaffolding::{fund_interop_eoa, selector4, signed, FUZZ_L1_BASE_FEE, INVOKE_GAS_CAP},
    shared_nodes::{next_msg_idx, shared_dual_exec, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::{
    messaging::{
        apply_l1_to_l2_alias, signed_l2_tx_hash, submit_retryable_ticket_id, DepositBuilder,
        MessageBuilder, RetryableSubmitBuilder,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
    ExecutionNode,
};

const ARBRETRYABLETX: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x6e,
]);

const LIFETIME: u64 = 7 * 24 * 60 * 60; // RetryableLifetimeSeconds = 604800

const SUBMIT_TS: u64 = 1_700_000_010;
const RAW_TIMEOUT: u64 = SUBMIT_TS + LIFETIME; // when A's stored timeout lapses
const PAST_TIMEOUT_TS: u64 = RAW_TIMEOUT + 86_400; // 1 day past raw, < RAW+LIFETIME
const PRE_TIMEOUT_TS: u64 = SUBMIT_TS + 600; // safely before raw timeout

fn one_arg_b32(sig: &str, h: B256) -> Bytes {
    let mut out = Vec::with_capacity(36);
    out.extend_from_slice(&selector4(sig));
    out.extend_from_slice(h.as_slice());
    Bytes::from(out)
}

fn fund_l1_sender(steps: &mut Vec<ScenarioStep>, sender: Address) {
    let idx = next_msg_idx();
    let dep = DepositBuilder {
        from: sender,
        to: apply_l1_to_l2_alias(sender),
        amount: U256::from(10u128).pow(U256::from(19u64)),
        l1_block_number: 1,
        timestamp: SUBMIT_TS,
        request_seq: idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("dep");
    steps.push(message_step(idx, dep, idx));
}

/// Submits a zero-call-value retryable at `timestamp` and returns its ticket id.
fn submit_retryable(
    steps: &mut Vec<ScenarioStep>,
    l1_sender: Address,
    to: Address,
    request_id: B256,
    timestamp: u64,
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
        gas_limit: 0,
        max_fee_per_gas: U256::from(1u64),
        data: Bytes::new(),
        l1_block_number: 3,
        timestamp,
        request_id: Some(request_id),
    }
    .build()
    .expect("submit");
    let ticket = submit_retryable_ticket_id(&msg, FUZZ_L2_CHAIN_ID).expect("ticket id");
    steps.push(message_step(idx, msg, idx));
    ticket
}

/// Builds: fund EOA, fund L1 sender, two expired decoys queued ahead of A,
/// submit A, keepalive(A) (nonce 0). Returns (steps, ticket_a).
fn submit_with_decoys_and_keepalive(eoa_nonce_base: u64) -> (Vec<ScenarioStep>, B256, u64) {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let l1_sender = Address::repeat_byte(0xa1);
    fund_l1_sender(&mut steps, l1_sender);

    // Two decoys submitted at SUBMIT_TS; both expire at SUBMIT_TS+LIFETIME and
    // sit ahead of A in the timeout queue.
    let _d1 = submit_retryable(
        &mut steps,
        l1_sender,
        Address::repeat_byte(0xd1),
        B256::repeat_byte(0x31),
        SUBMIT_TS,
    );
    let _d2 = submit_retryable(
        &mut steps,
        l1_sender,
        Address::repeat_byte(0xd2),
        B256::repeat_byte(0x32),
        SUBMIT_TS,
    );

    let ticket_a = submit_retryable(
        &mut steps,
        l1_sender,
        Address::repeat_byte(0xbb),
        B256::repeat_byte(0x10),
        SUBMIT_TS,
    );

    // keepalive(A): permissionless; at SUBMIT_TS the ticket is live so this
    // succeeds identically on both nodes and sets windowsLeft = 1.
    let mut ka = signed(
        eoa_nonce_base,
        Some(ARBRETRYABLETX),
        one_arg_b32("keepalive(bytes32)", ticket_a),
        U256::ZERO,
        INVOKE_GAS_CAP,
    );
    ka.timestamp = SUBMIT_TS + 1;
    let idx = next_msg_idx();
    steps.push(message_step(idx, ka.build().expect("keepalive"), idx));

    (steps, ticket_a, eoa_nonce_base + 1)
}

/// Appends a `getTimeout(A)` query at `timestamp` (its own block) and a trailing
/// no-op deposit to seal the block. Returns the query tx hash for the no-op guard.
fn append_query(
    steps: &mut Vec<ScenarioStep>,
    ticket_a: B256,
    nonce: u64,
    timestamp: u64,
) -> Option<B256> {
    let mut q = signed(
        nonce,
        Some(ARBRETRYABLETX),
        one_arg_b32("getTimeout(bytes32)", ticket_a),
        U256::ZERO,
        INVOKE_GAS_CAP,
    );
    q.timestamp = timestamp;
    let q_msg = q.build().expect("query");
    let q_hash = signed_l2_tx_hash(&q_msg);
    let idx = next_msg_idx();
    steps.push(message_step(idx, q_msg, idx));

    // Trailing no-op deposit seals the query block before state is read.
    let idx = next_msg_idx();
    let seal = DepositBuilder {
        from: Address::repeat_byte(0xee),
        to: Address::repeat_byte(0xee),
        amount: U256::from(1u64),
        l1_block_number: 4,
        timestamp: timestamp + 1,
        request_seq: idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("seal");
    steps.push(message_step(idx, seal, idx));

    q_hash
}

fn scenario(name: &str, steps: Vec<ScenarioStep>) -> Scenario {
    Scenario {
        name: name.to_string(),
        description: String::new(),
        setup: ScenarioSetup {
            l2_chain_id: FUZZ_L2_CHAIN_ID,
            arbos_version: arb_fuzz::shared_nodes::fuzz_arbos_version(),
            genesis: None,
        },
        steps,
    }
}

#[test]
#[ignore]
fn get_timeout_past_window_clean() {
    let (mut steps, ticket_a, next_nonce) = submit_with_decoys_and_keepalive(0);
    let q_hash = append_query(&mut steps, ticket_a, next_nonce, PAST_TIMEOUT_TS);
    let scen = scenario("get_timeout", steps);

    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().expect("dual_exec mutex");
    let report = nodes.run(&scen).expect("run");

    // No-op guard: the query tx must execute on both nodes (gas > 0) before any
    // cleanliness claim, so a dropped tx cannot masquerade as agreement.
    let q_hash = q_hash.expect("query tx hash");
    let lg = nodes.left.receipt(q_hash).map(|r| r.gas_used).unwrap_or(0);
    let rg = nodes.right.receipt(q_hash).map(|r| r.gas_used).unwrap_or(0);
    assert!(
        lg > 0 && rg > 0,
        "query tx did not execute on both nodes (left gas {lg}, right gas {rg})"
    );

    assert!(
        report.is_clean(),
        "getTimeout past raw timeout (windows-extended) must agree across nodes; report:\n{report:#?}"
    );
}

#[test]
#[ignore]
fn get_timeout_pre_timeout_control_clean() {
    let (mut steps, ticket_a, next_nonce) = submit_with_decoys_and_keepalive(0);
    // Only difference from the past-window case: query before the raw timeout lapses.
    let _q_hash = append_query(&mut steps, ticket_a, next_nonce, PRE_TIMEOUT_TS);
    let scen = scenario("get_timeout_control", steps);

    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().expect("dual_exec mutex");
    let report = nodes.run(&scen).expect("run");

    assert!(
        report.is_clean(),
        "control (query before raw timeout) must agree across nodes; report:\n{report:#?}"
    );
}

/// Appends `redeem(A)` at `timestamp` (its own block) + a trailing no-op deposit
/// to seal the block. Returns the redeem tx hash for the no-op guard.
fn append_redeem(
    steps: &mut Vec<ScenarioStep>,
    ticket_a: B256,
    nonce: u64,
    timestamp: u64,
) -> Option<B256> {
    let mut r = signed(
        nonce,
        Some(ARBRETRYABLETX),
        one_arg_b32("redeem(bytes32)", ticket_a),
        U256::ZERO,
        INVOKE_GAS_CAP,
    );
    r.timestamp = timestamp;
    let r_msg = r.build().expect("redeem");
    let r_hash = signed_l2_tx_hash(&r_msg);
    let idx = next_msg_idx();
    steps.push(message_step(idx, r_msg, idx));

    let idx = next_msg_idx();
    let seal = DepositBuilder {
        from: Address::repeat_byte(0xef),
        to: Address::repeat_byte(0xef),
        amount: U256::from(1u64),
        l1_block_number: 4,
        timestamp: timestamp + 1,
        request_seq: idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("seal");
    steps.push(message_step(idx, seal, idx));
    r_hash
}

#[test]
#[ignore]
fn redeem_past_raw_timeout_clean() {
    let (mut steps, ticket_a, next_nonce) = submit_with_decoys_and_keepalive(0);
    let r_hash = append_redeem(&mut steps, ticket_a, next_nonce, PAST_TIMEOUT_TS);
    let scen = scenario("redeem_past_raw_timeout", steps);

    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().expect("dual_exec mutex");
    let report = nodes.run(&scen).expect("run");

    let r_hash = r_hash.expect("redeem tx hash");
    let lg = nodes.left.receipt(r_hash).map(|r| r.gas_used).unwrap_or(0);
    let rg = nodes.right.receipt(r_hash).map(|r| r.gas_used).unwrap_or(0);
    assert!(
        lg > 0 && rg > 0,
        "redeem tx did not execute on both nodes (left gas {lg}, right gas {rg})"
    );

    assert!(
        report.is_clean(),
        "redeem past raw timeout (windows-extended) must agree across nodes; report:\n{report:#?}"
    );
}

#[test]
#[ignore]
fn redeem_pre_timeout_control_clean() {
    let (mut steps, ticket_a, next_nonce) = submit_with_decoys_and_keepalive(0);
    let _r = append_redeem(&mut steps, ticket_a, next_nonce, PRE_TIMEOUT_TS);
    let scen = scenario("redeem_control", steps);

    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().expect("dual_exec mutex");
    let report = nodes.run(&scen).expect("run");

    assert!(
        report.is_clean(),
        "control (redeem before raw timeout) must agree across nodes; report:\n{report:#?}"
    );
}
