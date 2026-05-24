//! Arbitrum-specific tx types driving Stylus + CREATE chains vs Nitro.
//!
//! Run:
//!   ARB_SPEC_BINARY=$(pwd)/target/fastdev/arb-reth \
//!     NITRO_REF_IMAGE=offchainlabs/nitro-node:v3.10.1-d7f07be \
//!     cargo test -p arb-fuzz --test arb_tx_types_create --release \
//!     -- --ignored --nocapture

use alloy_primitives::{keccak256, Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::{
        interop::{create_address, interop_eoa, interop_signing_key, wrap_init_code, WhichProgram},
        message_step,
    },
    shared_nodes::{fuzz_arbos_version, next_msg_idx, shared_dual_exec, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::{
    messaging::{
        apply_l1_to_l2_alias,
        signed_tx::{L2TxKind, SignedL2TxBuilder},
        ContractTxBuilder, DepositBuilder, MessageBuilder, RetryableSubmitBuilder,
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

fn factory_runtime() -> Vec<u8> {
    let sel = selector("deployCreate(bytes)");
    let mut out = Vec::with_capacity(96);
    out.extend_from_slice(&[0x60, 0x00, 0x35, 0x60, 0xe0, 0x1c]);
    out.push(0x63);
    out.extend_from_slice(&sel);
    out.push(0x14);
    out.extend_from_slice(&[0x60, 0x12, 0x57]);
    out.extend_from_slice(&[0x60, 0x00, 0x60, 0x00, 0xfd]);
    out.extend_from_slice(&[
        0x5b, 0x60, 0x24, 0x35, 0x80, 0x60, 0x44, 0x60, 0x80, 0x37, 0x60, 0x80, 0x34, 0xf0, 0x60,
        0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3,
    ]);
    out
}

fn encode_bytes_arg(data: &[u8]) -> Vec<u8> {
    let mut out = Vec::with_capacity(64 + data.len() + 32);
    let mut off32 = [0u8; 32];
    off32[31] = 0x20;
    let mut len32 = [0u8; 32];
    len32[24..32].copy_from_slice(&(data.len() as u64).to_be_bytes());
    out.extend_from_slice(&off32);
    out.extend_from_slice(&len32);
    out.extend_from_slice(data);
    while out.len() % 32 != 0 {
        out.push(0);
    }
    out
}

fn deploy_create_calldata(init_code: &[u8]) -> Vec<u8> {
    let mut out = Vec::with_capacity(4 + 64 + init_code.len());
    out.extend_from_slice(&selector("deployCreate(bytes)"));
    out.extend_from_slice(&encode_bytes_arg(init_code));
    out
}

fn forward_calldata(target: Address, inner: &[u8]) -> Bytes {
    let mut out = Vec::with_capacity(4 + 96 + inner.len());
    out.extend_from_slice(&selector("forward(address,bytes)"));
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(target.as_slice());
    out.extend_from_slice(&pad);
    let mut off32 = [0u8; 32];
    off32[31] = 0x40;
    out.extend_from_slice(&off32);
    let mut len32 = [0u8; 32];
    len32[24..32].copy_from_slice(&(inner.len() as u64).to_be_bytes());
    out.extend_from_slice(&len32);
    out.extend_from_slice(inner);
    while out.len() % 32 != 0 {
        out.push(0);
    }
    Bytes::from(out)
}

fn ctor_sload() -> Vec<u8> {
    vec![
        0x60, 0x00, 0x54, 0x50, 0x60, 0x00, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xf3,
    ]
}

fn stylus_addr() -> Address {
    create_address(interop_eoa(), 0)
}

fn factory_address() -> Address {
    create_address(interop_eoa(), 2)
}

fn run_named(name: &str, steps: Vec<ScenarioStep>) {
    let scen = Scenario {
        name: name.into(),
        description: format!("arb tx type: {name}"),
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
        let path = std::path::PathBuf::from(format!("/tmp/arb_tx_types_{name}.json"));
        let _ = std::fs::write(&path, serde_json::to_vec_pretty(&payload).unwrap());
        panic!(
            "arbreth diverged from Nitro on {name}; see {}",
            path.display()
        );
    }
}

fn baseline_steps_to_setup_stylus_and_factory() -> Vec<ScenarioStep> {
    let eoa = interop_eoa();
    let mut steps = Vec::new();

    let idx = next_msg_idx();
    let fund = DepositBuilder {
        from: eoa,
        to: eoa,
        amount: U256::from(10u128).pow(U256::from(20u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("fund");
    steps.push(message_step(idx, fund, idx));

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

    let stylus = stylus_addr();
    let mut act = Vec::with_capacity(36);
    act.extend_from_slice(&[0x58, 0xc7, 0x80, 0xc2]);
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(stylus.as_slice());
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

    let deploy_factory = signed(
        2,
        None,
        Bytes::from(wrap_init_code(&factory_runtime())),
        U256::ZERO,
        DEPLOY_GAS_CAP,
    )
    .build()
    .expect("deploy factory");
    let idx = next_msg_idx();
    steps.push(message_step(idx, deploy_factory, idx));

    steps
}

// ── ContractTx (L1->L2 contract call) drives Stylus -> CREATE ──────────────

#[test]
#[ignore]
fn contract_tx_triggers_stylus_then_create() {
    let mut steps = baseline_steps_to_setup_stylus_and_factory();
    let l1_sender = Address::repeat_byte(0x11);
    let fund_idx = next_msg_idx();
    let fund_aliased = DepositBuilder {
        from: l1_sender,
        to: apply_l1_to_l2_alias(l1_sender),
        amount: U256::from(10u128).pow(U256::from(18u64)),
        l1_block_number: 3,
        timestamp: 1_700_000_010,
        request_seq: fund_idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("fund aliased");
    steps.push(message_step(fund_idx, fund_aliased, fund_idx));

    let inner = deploy_create_calldata(&ctor_sload());
    let cdata = forward_calldata(factory_address(), &inner);
    let ct_idx = next_msg_idx();
    let contract_tx = ContractTxBuilder {
        from: l1_sender,
        gas_limit: INVOKE_GAS_CAP,
        max_fee_per_gas: U256::from(2_000_000_000u64),
        to: stylus_addr(),
        value: U256::ZERO,
        data: cdata,
        l1_block_number: 4,
        timestamp: 1_700_000_020,
        request_seq: ct_idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("contract tx");
    steps.push(message_step(ct_idx, contract_tx, ct_idx));

    run_named("contract_tx_stylus_create", steps);
}

// ── RetryableSubmit auto-redeem drives Stylus -> CREATE ────────────────────

#[test]
#[ignore]
fn retryable_submit_auto_redeem_to_stylus_create() {
    let mut steps = baseline_steps_to_setup_stylus_and_factory();
    let l1_sender = Address::repeat_byte(0x22);
    let inner = deploy_create_calldata(&ctor_sload());
    let cdata = forward_calldata(factory_address(), &inner);

    let submit = RetryableSubmitBuilder {
        l1_sender,
        to: stylus_addr(),
        l2_call_value: U256::ZERO,
        deposit_value: U256::from(10u128).pow(U256::from(18u64)),
        max_submission_fee: U256::from(10u128).pow(U256::from(15u64)),
        excess_fee_refund_address: apply_l1_to_l2_alias(l1_sender),
        call_value_refund_address: apply_l1_to_l2_alias(l1_sender),
        gas_limit: INVOKE_GAS_CAP,
        max_fee_per_gas: U256::from(2_000_000_000u64),
        data: cdata,
        l1_block_number: 5,
        timestamp: 1_700_000_030,
        request_id: Some(B256::repeat_byte(0xaa)),
    }
    .build()
    .expect("retryable submit");
    let idx = next_msg_idx();
    steps.push(message_step(idx, submit, idx));

    run_named("retryable_submit_stylus_create", steps);
}

// ── Direct deposit then signed tx that does Stylus -> CREATE2 collision ────

#[test]
#[ignore]
fn deposit_then_stylus_create_collision() {
    let mut steps = baseline_steps_to_setup_stylus_and_factory();
    let l1_sender = Address::repeat_byte(0x33);
    let deposit_idx = next_msg_idx();
    let dep = DepositBuilder {
        from: l1_sender,
        to: apply_l1_to_l2_alias(l1_sender),
        amount: U256::from(10u128).pow(U256::from(18u64)),
        l1_block_number: 6,
        timestamp: 1_700_000_040,
        request_seq: deposit_idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("deposit");
    steps.push(message_step(deposit_idx, dep, deposit_idx));

    let inner = deploy_create_calldata(&ctor_sload());
    let cdata = forward_calldata(factory_address(), &inner);
    let tx1 = signed(
        3,
        Some(stylus_addr()),
        cdata.clone(),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx1");
    let i1 = next_msg_idx();
    steps.push(message_step(i1, tx1, i1));

    let tx2 = signed(4, Some(stylus_addr()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx2");
    let i2 = next_msg_idx();
    steps.push(message_step(i2, tx2, i2));

    run_named("deposit_then_stylus_create2", steps);
}
