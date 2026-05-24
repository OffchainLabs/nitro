//! Differential CALL/DELEGATECALL/STATICCALL coverage through Stylus dispatch vs Nitro.
//!
//! Categories:
//!   * Stylus -> Solidity -> Stylus round trips (re-entrancy)
//!   * STATICCALL violations (sstore, log, selfdestruct, create, value)
//!   * CALL value transfer edge cases
//!   * CALL targets: precompiles, ArbSys, EOAs, non-existent
//!   * CALL return data (empty, large, revert-with-data)
//!   * CALL gas (OOG in callee, stipend, depth)
//!
//! Run:
//!   ARB_SPEC_BINARY=$(pwd)/target/fastdev/arb-reth \
//!     NITRO_REF_IMAGE=offchainlabs/nitro-node:v3.10.1-d7f07be \
//!     cargo test -p arb-fuzz --test stylus_call_chains --release \
//!     -- --ignored --nocapture

use alloy_primitives::{keccak256, Address, Bytes, U256};
use arb_fuzz::{
    arbitrary_impls::{
        interop::{create_address, interop_eoa, interop_signing_key, wrap_init_code, WhichProgram},
        message_step,
    },
    shared_nodes::{fuzz_arbos_version, next_msg_idx, shared_dual_exec, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::{
    messaging::{
        signed_tx::{L2TxKind, SignedL2TxBuilder},
        DepositBuilder, MessageBuilder,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
};

const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const INVOKE_GAS_CAP: u64 = 30_000_000;
const DEPLOY_GAS_CAP: u64 = 150_000_000;
const SEQUENCER_ALIAS: Address = Address::new([
    0xa4, 0xb0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x73, 0x65, 0x71, 0x75, 0x65, 0x6e, 0x63, 0x65, 0x72,
]);
const ARBWASM_ADDR: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x71,
]);
const ARBSYS_ADDR: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x64,
]);

fn selector(sig: &str) -> [u8; 4] {
    let h = keccak256(sig.as_bytes());
    [h[0], h[1], h[2], h[3]]
}

fn signed(
    nonce: u64,
    to: Option<Address>,
    data: Bytes,
    value: U256,
    gas: u64,
) -> SignedL2TxBuilder {
    SignedL2TxBuilder {
        chain_id: FUZZ_L2_CHAIN_ID,
        nonce,
        to,
        value,
        data,
        gas_limit: gas,
        gas_price: 0,
        max_fee_per_gas: 2_000_000_000,
        max_priority_fee_per_gas: 0,
        access_list: Vec::new(),
        authorization_list: Vec::new(),
        kind: L2TxKind::Eip1559,
        signing_key: interop_signing_key(),
        l1_block_number: 2,
        timestamp: 1_700_000_000,
        request_id: None,
        sender: SEQUENCER_ALIAS,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
}

fn stylus_addr() -> Address {
    create_address(interop_eoa(), 0)
}

fn helper_addr() -> Address {
    create_address(interop_eoa(), 2)
}

fn run_named(name: &str, steps: Vec<ScenarioStep>) {
    let scen = Scenario {
        name: name.into(),
        description: format!("call-chain diff: {name}"),
        setup: ScenarioSetup {
            l2_chain_id: FUZZ_L2_CHAIN_ID,
            arbos_version: fuzz_arbos_version(),
            genesis: None,
        },
        steps,
    };
    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
    let report = nodes.run(&scen).expect("run scenario");
    if !report.block_diffs.is_empty()
        || !report.tx_diffs.is_empty()
        || !report.state_diffs.is_empty()
        || !report.log_diffs.is_empty()
    {
        let payload = serde_json::json!({
            "scenario": name,
            "block_diffs": format!("{:#?}", report.block_diffs),
            "tx_diffs": format!("{:#?}", report.tx_diffs),
            "state_diffs": format!("{:#?}", report.state_diffs),
            "log_diffs": format!("{:#?}", report.log_diffs),
        });
        let path = std::path::PathBuf::from(format!("/tmp/stylus_call_chains_{name}.json"));
        let _ = std::fs::write(&path, serde_json::to_vec_pretty(&payload).unwrap());
        panic!("arbreth diverged from Nitro on {name}; see {}", path.display());
    }
}

fn baseline_with_helper(helper_runtime: &[u8]) -> Vec<ScenarioStep> {
    let eoa = interop_eoa();
    let mut steps = Vec::new();

    let fund_idx = next_msg_idx();
    let fund = DepositBuilder {
        from: eoa,
        to: eoa,
        amount: U256::from(10u128).pow(U256::from(20u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: fund_idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("fund");
    steps.push(message_step(fund_idx, fund, fund_idx));

    let deploy_stylus = signed(
        0,
        None,
        Bytes::from(WhichProgram::SolCaller.initcode()),
        U256::ZERO,
        DEPLOY_GAS_CAP,
    )
    .build()
    .expect("deploy stylus");
    let idx = next_msg_idx();
    steps.push(message_step(idx, deploy_stylus, idx));

    let mut act = Vec::with_capacity(36);
    act.extend_from_slice(&[0x58, 0xc7, 0x80, 0xc2]);
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(stylus_addr().as_slice());
    act.extend_from_slice(&pad);
    let activate = signed(
        1,
        Some(ARBWASM_ADDR),
        Bytes::from(act),
        U256::from(10u128).pow(U256::from(15u64)),
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("activate");
    let idx = next_msg_idx();
    steps.push(message_step(idx, activate, idx));

    let deploy_helper = signed(
        2,
        None,
        Bytes::from(wrap_init_code(helper_runtime)),
        U256::ZERO,
        DEPLOY_GAS_CAP,
    )
    .build()
    .expect("deploy helper");
    let idx = next_msg_idx();
    steps.push(message_step(idx, deploy_helper, idx));

    steps
}

fn forward_call_calldata(target: Address, inner: &[u8]) -> Bytes {
    wrap_two_arg("forward(address,bytes)", target, inner)
}

fn forward_static_calldata(target: Address, inner: &[u8]) -> Bytes {
    wrap_two_arg("forwardStatic(address,bytes)", target, inner)
}

fn forward_delegate_calldata(target: Address, inner: &[u8]) -> Bytes {
    wrap_two_arg("forwardDelegate(address,bytes)", target, inner)
}

fn wrap_two_arg(sig: &str, addr: Address, data: &[u8]) -> Bytes {
    let mut out = Vec::with_capacity(4 + 96 + data.len());
    out.extend_from_slice(&selector(sig));
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(addr.as_slice());
    out.extend_from_slice(&pad);
    let mut off32 = [0u8; 32];
    off32[31] = 0x40;
    out.extend_from_slice(&off32);
    let mut len32 = [0u8; 32];
    len32[24..32].copy_from_slice(&(data.len() as u64).to_be_bytes());
    out.extend_from_slice(&len32);
    out.extend_from_slice(data);
    while out.len() % 32 != 0 {
        out.push(0);
    }
    Bytes::from(out)
}

// ── Solidity helper runtimes ──────────────────────────────────────────────

fn helper_sstore_then_return_zero() -> Vec<u8> {
    vec![
        0x60, 0x42, 0x60, 0x00, 0x55, 0x60, 0x00, 0x60, 0x00, 0xf3,
    ]
}

fn helper_log0() -> Vec<u8> {
    vec![
        0x60, 0xaa, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xa0, 0x60, 0x00, 0x60, 0x00, 0xf3,
    ]
}

fn helper_call_with_value(target: Address) -> Vec<u8> {
    let mut out = Vec::with_capacity(60);
    out.extend_from_slice(&[0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x01]);
    out.push(0x73);
    out.extend_from_slice(target.as_slice());
    out.extend_from_slice(&[0x5a, 0xf1, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3]);
    out
}

fn helper_revert_with_data() -> Vec<u8> {
    vec![
        0x60, 0xde, 0xad, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xfd,
    ]
}

fn helper_return_large(words: u16) -> Vec<u8> {
    let len_bytes = ((words as u32) * 32).to_be_bytes();
    let mut out = Vec::with_capacity(24);
    out.extend_from_slice(&[
        0x63, len_bytes[0], len_bytes[1], len_bytes[2], len_bytes[3], 0x60, 0x00, 0xf3,
    ]);
    out
}

fn helper_oog() -> Vec<u8> {
    let mut out = Vec::with_capacity(8);
    out.push(0x5b);
    out.extend_from_slice(&[0x60, 0x00, 0x56]);
    out
}

fn helper_callback_to_stylus(stylus: Address) -> Vec<u8> {
    let inner = selector("callCount()").to_vec();
    let mut out = Vec::with_capacity(60 + inner.len());
    let inner_len = inner.len();
    out.extend_from_slice(&[
        0x60, inner_len as u8, 0x60, 0x00, 0x60, 0x00, 0x37,
    ]);
    out.extend_from_slice(&[
        0x60, 0x00, 0x60, 0x00, 0x60, 0x20, 0x60, 0x00, 0x60, inner_len as u8, 0x60, 0x00, 0x60, 0x00,
    ]);
    out.push(0x73);
    out.extend_from_slice(stylus.as_slice());
    out.extend_from_slice(&[0x5a, 0xf1, 0x60, 0x00, 0x60, 0x00, 0x60, 0x20, 0x60, 0x00, 0xf3]);
    out
}

// ── Stylus -> Solidity -> Stylus round trip ─────────────────────────────────

#[test]
#[ignore]
fn stylus_call_helper_that_calls_stylus_back() {
    let helper_runtime = helper_callback_to_stylus(stylus_addr());
    let mut steps = baseline_with_helper(&helper_runtime);
    let inner: &[u8] = &[];
    let cdata = forward_call_calldata(helper_addr(), inner);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("stylus_round_trip", steps);
}

// ── STATICCALL violations ──────────────────────────────────────────────────

#[test]
#[ignore]
fn static_call_into_sstore_fails() {
    let mut steps = baseline_with_helper(&helper_sstore_then_return_zero());
    let cdata = forward_static_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("staticcall_sstore_fail", steps);
}

#[test]
#[ignore]
fn static_call_into_log_fails() {
    let mut steps = baseline_with_helper(&helper_log0());
    let cdata = forward_static_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("staticcall_log_fail", steps);
}

#[test]
#[ignore]
fn static_call_into_call_with_value_fails() {
    let helper = helper_call_with_value(Address::repeat_byte(0xab));
    let mut steps = baseline_with_helper(&helper);
    let cdata = forward_static_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("staticcall_call_with_value_fail", steps);
}

// ── CALL value transfer edge cases ─────────────────────────────────────────

#[test]
#[ignore]
fn call_value_transfer_to_fresh_eoa() {
    let helper = helper_call_with_value(Address::repeat_byte(0x99));
    let mut steps = baseline_with_helper(&helper);
    let cdata = forward_call_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("call_value_fresh_eoa", steps);
}

#[test]
#[ignore]
fn call_value_insufficient_balance() {
    let huge = U256::from(10u128).pow(U256::from(30u64));
    let big_val = huge.to_be_bytes::<32>();
    let mut helper_runtime = Vec::with_capacity(80);
    helper_runtime.extend_from_slice(&[
        0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x7f,
    ]);
    helper_runtime.extend_from_slice(&big_val);
    helper_runtime.push(0x73);
    helper_runtime.extend_from_slice(Address::repeat_byte(0x77).as_slice());
    helper_runtime.extend_from_slice(&[0x5a, 0xf1, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3]);
    let mut steps = baseline_with_helper(&helper_runtime);
    let cdata = forward_call_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("call_value_insufficient", steps);
}

// ── CALL targets ───────────────────────────────────────────────────────────

#[test]
#[ignore]
fn call_to_identity_precompile() {
    let mut steps = baseline_with_helper(&[0x60, 0x00, 0x60, 0x00, 0xf3]);
    let data = [0xde, 0xad, 0xbe, 0xef];
    let cdata = forward_call_calldata(Address::with_last_byte(0x04), &data);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("call_identity_precompile", steps);
}

#[test]
#[ignore]
fn call_to_ecrecover_precompile() {
    let mut steps = baseline_with_helper(&[0x60, 0x00, 0x60, 0x00, 0xf3]);
    let mut data = vec![0u8; 128];
    data[31] = 0x01;
    let cdata = forward_call_calldata(Address::with_last_byte(0x01), &data);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("call_ecrecover_precompile", steps);
}

#[test]
#[ignore]
fn call_to_arbsys_arb_block_number() {
    let mut steps = baseline_with_helper(&[0x60, 0x00, 0x60, 0x00, 0xf3]);
    let data = selector("arbBlockNumber()").to_vec();
    let cdata = forward_call_calldata(ARBSYS_ADDR, &data);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("call_arbsys", steps);
}

#[test]
#[ignore]
fn call_to_nonexistent_address() {
    let mut steps = baseline_with_helper(&[0x60, 0x00, 0x60, 0x00, 0xf3]);
    let cdata = forward_call_calldata(Address::repeat_byte(0xdd), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("call_nonexistent", steps);
}

// ── CALL return data ───────────────────────────────────────────────────────

#[test]
#[ignore]
fn call_returns_revert_with_data() {
    let mut steps = baseline_with_helper(&helper_revert_with_data());
    let cdata = forward_call_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("call_revert_with_data", steps);
}

#[test]
#[ignore]
fn call_returns_large_data() {
    let mut steps = baseline_with_helper(&helper_return_large(32));
    let cdata = forward_call_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("call_large_return", steps);
}

#[test]
#[ignore]
fn call_returns_empty() {
    let mut steps = baseline_with_helper(&[0x00]);
    let cdata = forward_call_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("call_empty_return", steps);
}

// ── CALL gas edge cases ────────────────────────────────────────────────────

#[test]
#[ignore]
fn call_into_oog_callee() {
    let mut steps = baseline_with_helper(&helper_oog());
    let cdata = forward_call_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("call_callee_oog", steps);
}

// ── DELEGATECALL semantics ─────────────────────────────────────────────────

#[test]
#[ignore]
fn delegate_call_into_sstore_writes_to_caller_storage() {
    let mut steps = baseline_with_helper(&helper_sstore_then_return_zero());
    let cdata = forward_delegate_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("delegatecall_sstore_caller", steps);
}

#[test]
#[ignore]
fn delegate_call_into_log_emits_from_caller() {
    let mut steps = baseline_with_helper(&helper_log0());
    let cdata = forward_delegate_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("delegatecall_log_caller", steps);
}

#[test]
#[ignore]
fn delegate_call_into_revert_with_data() {
    let mut steps = baseline_with_helper(&helper_revert_with_data());
    let cdata = forward_delegate_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("delegatecall_revert_data", steps);
}

// ── Stylus -> Stylus direct (no Solidity intermediary) ─────────────────────

#[test]
#[ignore]
fn stylus_calls_itself() {
    // Stylus contract forwards to itself: tests re-entrancy detection
    let mut steps = baseline_with_helper(&[0x00]);
    let inner = selector("callCount()").to_vec();
    let cdata = forward_call_calldata(stylus_addr(), &inner);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("stylus_self_call", steps);
}

#[test]
#[ignore]
fn stylus_delegate_to_self() {
    let mut steps = baseline_with_helper(&[0x00]);
    let inner = selector("callCount()").to_vec();
    let cdata = forward_delegate_calldata(stylus_addr(), &inner);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("stylus_self_delegate", steps);
}

// ── SELFDESTRUCT from a CALLed helper ──────────────────────────────────────

fn helper_selfdestruct(beneficiary: Address) -> Vec<u8> {
    let mut out = Vec::with_capacity(24);
    out.push(0x73);
    out.extend_from_slice(beneficiary.as_slice());
    out.push(0xff);
    out
}

#[test]
#[ignore]
fn call_into_selfdestruct() {
    let helper = helper_selfdestruct(Address::repeat_byte(0xee));
    let mut steps = baseline_with_helper(&helper);
    let cdata = forward_call_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("call_selfdestruct", steps);
}

// ── CALL with value from Stylus (uses do_call with value) ──────────────────
// SolCaller's `forward` always uses Call::new_mutating (CALL with value=0).
// Test value-carrying through a Solidity helper that itself calls with value.

fn helper_call_with_specified_value(target: Address, val_be: [u8; 32]) -> Vec<u8> {
    let mut out = Vec::with_capacity(80);
    out.extend_from_slice(&[0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x7f]);
    out.extend_from_slice(&val_be);
    out.push(0x73);
    out.extend_from_slice(target.as_slice());
    out.extend_from_slice(&[0x5a, 0xf1, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3]);
    out
}

#[test]
#[ignore]
fn call_value_to_existing_contract() {
    let target = Address::repeat_byte(0x55);
    let mut val = [0u8; 32];
    val[31] = 0x07;
    let helper = helper_call_with_specified_value(target, val);
    let mut steps = baseline_with_helper(&helper);
    let fund_helper = signed(
        3,
        Some(helper_addr()),
        Bytes::new(),
        U256::from(1000u64),
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("fund helper");
    let idx = next_msg_idx();
    steps.push(message_step(idx, fund_helper, idx));
    let cdata = forward_call_calldata(helper_addr(), &[]);
    let tx = signed(4, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("call_value_to_contract", steps);
}

// ── Nested DELEGATECALL chains ─────────────────────────────────────────────

fn helper_delegate_then_sstore(further: Address) -> Vec<u8> {
    let mut out = Vec::with_capacity(80);
    out.extend_from_slice(&[
        0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00,
    ]);
    out.push(0x73);
    out.extend_from_slice(further.as_slice());
    out.extend_from_slice(&[
        0x5a, 0xf4, 0x50, 0x60, 0xaa, 0x60, 0x01, 0x55, 0x60, 0x00, 0x60, 0x00, 0xf3,
    ]);
    out
}

#[test]
#[ignore]
fn delegate_chain_writes_observed_at_initial_caller() {
    let further_addr = Address::repeat_byte(0x88);
    let helper = helper_delegate_then_sstore(further_addr);
    let mut steps = baseline_with_helper(&helper);
    let further_runtime: &[u8] = &[0x60, 0xbb, 0x60, 0x00, 0x55, 0x60, 0x00, 0x60, 0x00, 0xf3];
    let deploy_further = signed(
        3,
        None,
        Bytes::from(wrap_init_code(further_runtime)),
        U256::ZERO,
        DEPLOY_GAS_CAP,
    )
    .build()
    .expect("deploy further");
    let idx = next_msg_idx();
    steps.push(message_step(idx, deploy_further, idx));
    let cdata = forward_delegate_calldata(helper_addr(), &[]);
    let tx = signed(4, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("delegate_chain_writes", steps);
}

// ── Stylus called via DELEGATECALL from Solidity ───────────────────────────

#[test]
#[ignore]
fn solidity_delegatecall_into_stylus() {
    let stylus = stylus_addr();
    let mut helper = Vec::with_capacity(60);
    helper.extend_from_slice(&[
        0x60, 0x04, 0x60, 0x00, 0x60, 0x00, 0x37,
        0x60, 0x20, 0x60, 0x00, 0x60, 0x04, 0x60, 0x00, 0x60, 0x00,
    ]);
    helper.push(0x73);
    helper.extend_from_slice(stylus.as_slice());
    helper.extend_from_slice(&[0x5a, 0xf4, 0x60, 0x00, 0x60, 0x00, 0xf3]);
    let mut steps = baseline_with_helper(&helper);
    let mut cdata = Vec::with_capacity(4);
    cdata.extend_from_slice(&selector("callCount()"));
    let tx = signed(3, Some(helper_addr()), Bytes::from(cdata), U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("sol_delegate_stylus", steps);
}

// ── Opcode coverage through Stylus -> Solidity helper ──────────────────────

fn helper_returns_opcode_result(opcode: u8) -> Vec<u8> {
    vec![opcode, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3]
}

fn helper_returns_balance(addr: Address) -> Vec<u8> {
    let mut out = Vec::with_capacity(28);
    out.push(0x73);
    out.extend_from_slice(addr.as_slice());
    out.extend_from_slice(&[0x31, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3]);
    out
}

#[test]
#[ignore]
fn opcode_timestamp_via_stylus_call() {
    let mut steps = baseline_with_helper(&helper_returns_opcode_result(0x42));
    let cdata = forward_call_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("op_timestamp", steps);
}

#[test]
#[ignore]
fn opcode_number_via_stylus_call() {
    let mut steps = baseline_with_helper(&helper_returns_opcode_result(0x43));
    let cdata = forward_call_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("op_number", steps);
}

#[test]
#[ignore]
fn opcode_coinbase_via_stylus_call() {
    let mut steps = baseline_with_helper(&helper_returns_opcode_result(0x41));
    let cdata = forward_call_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("op_coinbase", steps);
}

#[test]
#[ignore]
fn opcode_gasprice_via_stylus_call() {
    let mut steps = baseline_with_helper(&helper_returns_opcode_result(0x3a));
    let cdata = forward_call_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("op_gasprice", steps);
}

#[test]
#[ignore]
fn opcode_balance_of_stylus_via_helper() {
    let helper = helper_returns_balance(stylus_addr());
    let mut steps = baseline_with_helper(&helper);
    let cdata = forward_call_calldata(helper_addr(), &[]);
    let tx = signed(3, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("op_balance_stylus", steps);
}
