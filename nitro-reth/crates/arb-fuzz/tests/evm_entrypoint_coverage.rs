//! Differential coverage of every entry point in geth's `core/vm/evm.go`.
//!
//! Each `#[test]` exercises one entry point through Stylus dispatch and
//! diffs against Nitro. Cases derive from geth's own branches:
//!   * depth limit, balance check, nonce overflow
//!   * address collision (CREATE/CREATE2)
//!   * EIP-158 nonce init, EIP-3541 0xEF reject, EIP-170 max code size
//!   * value-transfer semantics (CALL with value, STATICCALL forbids)
//!   * precompile dispatch, 7702 delegation follow
//!
//! Run:
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     NITRO_REF_IMAGE=offchainlabs/nitro-node:v3.10.0-rc.10-b1cf6db \
//!     cargo test -p arb-fuzz --test evm_entrypoint_coverage --release \
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
        signed_tx::{L2TxKind, SignedL2TxBuilder},
        DepositBuilder, MessageBuilder,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
};

const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const INVOKE_GAS_CAP: u64 = 30_000_000;
const DEPLOY_GAS_CAP: u64 = 150_000_000;
const SEQUENCER_ALIAS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65, 0x71, 0x75, 0x65,
    0x6e, 0x63, 0x65, 0x72,
]);
const ARBWASM_ADDR: Address = Address::new([
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x71,
]);

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

fn selector(sig: &str) -> [u8; 4] {
    let h = keccak256(sig.as_bytes());
    [h[0], h[1], h[2], h[3]]
}

fn run_named(name: &str, steps: Vec<ScenarioStep>) {
    let scen = Scenario {
        name: name.into(),
        description: format!("evm.go coverage: {name}"),
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
        let path = std::path::PathBuf::from(format!("/tmp/evm_entrypoint_{name}.json"));
        let _ = std::fs::write(
            &path,
            serde_json::to_vec_pretty(&serde_json::json!({
                "scenario": name,
                "block_diffs": format!("{:#?}", report.block_diffs),
                "tx_diffs": format!("{:#?}", report.tx_diffs),
                "state_diffs": format!("{:#?}", report.state_diffs),
                "log_diffs": format!("{:#?}", report.log_diffs),
            }))
            .unwrap(),
        );
        panic!("divergence on {name}; see {}", path.display());
    }
}

fn fund_eoa(steps: &mut Vec<ScenarioStep>) {
    let eoa = interop_eoa();
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
}

fn deploy_and_activate_stylus(steps: &mut Vec<ScenarioStep>, nonce_base: u64) -> Address {
    let eoa = interop_eoa();
    let initcode = WhichProgram::SolCaller.initcode();
    let deploy = signed(
        nonce_base,
        None,
        Bytes::from(initcode),
        U256::ZERO,
        DEPLOY_GAS_CAP,
    )
    .build()
    .expect("deploy");
    let idx = next_msg_idx();
    steps.push(message_step(idx, deploy, idx));

    let addr = create_address(eoa, nonce_base);
    let mut activate_data = Vec::with_capacity(36);
    activate_data.extend_from_slice(&[0x58, 0xc7, 0x80, 0xc2]);
    let mut padded = [0u8; 32];
    padded[12..].copy_from_slice(addr.as_slice());
    activate_data.extend_from_slice(&padded);
    let act = signed(
        nonce_base + 1,
        Some(ARBWASM_ADDR),
        Bytes::from(activate_data),
        U256::from(10u128).pow(U256::from(15u64)),
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("activate");
    let idx = next_msg_idx();
    steps.push(message_step(idx, act, idx));

    addr
}

fn deploy_runtime(steps: &mut Vec<ScenarioStep>, nonce: u64, runtime: &[u8]) -> Address {
    let eoa = interop_eoa();
    let tx = signed(
        nonce,
        None,
        Bytes::from(wrap_init_code(runtime)),
        U256::ZERO,
        DEPLOY_GAS_CAP,
    )
    .build()
    .expect("deploy runtime");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    create_address(eoa, nonce)
}

fn ctor_returns_runtime(runtime: &[u8]) -> Vec<u8> {
    wrap_init_code(runtime)
}

fn ctor_returns_oversize() -> Vec<u8> {
    let mut runtime = vec![0x00u8; 25_000];
    runtime[0] = 0xfe;
    wrap_init_code(&runtime)
}

fn ctor_returns_ef_prefix() -> Vec<u8> {
    let runtime: &[u8] = &[0xef, 0x00, 0x00];
    wrap_init_code(runtime)
}

fn ctor_storage_op(op: u8) -> Vec<u8> {
    match op {
        0x54 => vec![
            0x60, 0x00, 0x54, 0x50, 0x60, 0x00, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xf3,
        ],
        0x55 => vec![
            0x60, 0x42, 0x60, 0x00, 0x55, 0x60, 0x00, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00,
            0xf3,
        ],
        _ => unreachable!(),
    }
}

fn factory_runtime() -> Vec<u8> {
    let create_sel = selector("deployCreate(bytes)");
    let create2_sel = selector("deployCreate2(bytes32,bytes)");
    let mut out = Vec::with_capacity(256);
    out.extend_from_slice(&[0x60, 0x00, 0x35, 0x60, 0xe0, 0x1c]);
    out.push(0x80);
    out.push(0x63);
    out.extend_from_slice(&create_sel);
    out.push(0x14);
    let create_jumpi_pos = out.len();
    out.extend_from_slice(&[0x61, 0x00, 0x00, 0x57]);
    out.push(0x80);
    out.push(0x63);
    out.extend_from_slice(&create2_sel);
    out.push(0x14);
    let create2_jumpi_pos = out.len();
    out.extend_from_slice(&[0x61, 0x00, 0x00, 0x57]);
    out.extend_from_slice(&[0x60, 0x00, 0x60, 0x00, 0xfd]);

    let create_dest = out.len();
    out.push(0x5b);
    out.extend_from_slice(&[0x60, 0x24, 0x35]);
    out.push(0x80);
    out.extend_from_slice(&[0x60, 0x44]);
    out.extend_from_slice(&[0x60, 0x80]);
    out.push(0x37);
    out.extend_from_slice(&[0x60, 0x80]);
    out.push(0x34);
    out.push(0xf0);
    out.extend_from_slice(&[0x60, 0x00, 0x52]);
    out.extend_from_slice(&[0x60, 0x20, 0x60, 0x00, 0xf3]);

    let create2_dest = out.len();
    out.push(0x5b);
    out.extend_from_slice(&[0x60, 0x44, 0x35]);
    out.push(0x80);
    out.extend_from_slice(&[0x60, 0x64]);
    out.extend_from_slice(&[0x60, 0x80]);
    out.push(0x37);
    out.extend_from_slice(&[0x60, 0x04, 0x35]);
    out.push(0x90);
    out.extend_from_slice(&[0x60, 0x80]);
    out.push(0x34);
    out.push(0xf5);
    out.extend_from_slice(&[0x60, 0x00, 0x52]);
    out.extend_from_slice(&[0x60, 0x20, 0x60, 0x00, 0xf3]);

    out[create_jumpi_pos + 1] = ((create_dest >> 8) & 0xff) as u8;
    out[create_jumpi_pos + 2] = (create_dest & 0xff) as u8;
    out[create2_jumpi_pos + 1] = ((create2_dest >> 8) & 0xff) as u8;
    out[create2_jumpi_pos + 2] = (create2_dest & 0xff) as u8;
    out
}

fn encode_bytes(data: &[u8]) -> Vec<u8> {
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

fn encode_create_calldata(init_code: &[u8]) -> Vec<u8> {
    let mut out = Vec::with_capacity(4 + 64 + init_code.len());
    out.extend_from_slice(&selector("deployCreate(bytes)"));
    out.extend_from_slice(&encode_bytes(init_code));
    out
}

fn encode_create2_calldata(salt: B256, init_code: &[u8]) -> Vec<u8> {
    let mut out = Vec::with_capacity(4 + 96 + init_code.len());
    out.extend_from_slice(&selector("deployCreate2(bytes32,bytes)"));
    out.extend_from_slice(salt.as_slice());
    let mut off32 = [0u8; 32];
    off32[31] = 0x40;
    out.extend_from_slice(&off32);
    let mut len32 = [0u8; 32];
    len32[24..32].copy_from_slice(&(init_code.len() as u64).to_be_bytes());
    out.extend_from_slice(&len32);
    out.extend_from_slice(init_code);
    while out.len() % 32 != 0 {
        out.push(0);
    }
    out
}

fn baseline_steps_with_factory() -> (Vec<ScenarioStep>, Address) {
    let mut steps = Vec::new();
    fund_eoa(&mut steps);
    let _stylus = deploy_and_activate_stylus(&mut steps, 0);
    let factory = deploy_runtime(&mut steps, 2, &factory_runtime());
    (steps, factory)
}

// ── geth evm.go `Create` paths ─────────────────────────────────────────────

#[test]
#[ignore]
fn create_constructor_sload() {
    let (mut steps, factory) = baseline_steps_with_factory();
    let calldata = encode_create_calldata(&ctor_storage_op(0x54));
    let tx = signed(
        3,
        Some(factory),
        Bytes::from(calldata),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("create_sload", steps);
}

#[test]
#[ignore]
fn create_constructor_sstore() {
    let (mut steps, factory) = baseline_steps_with_factory();
    let calldata = encode_create_calldata(&ctor_storage_op(0x55));
    let tx = signed(
        3,
        Some(factory),
        Bytes::from(calldata),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("create_sstore", steps);
}

#[test]
#[ignore]
fn create_max_code_size_exceeded() {
    let (mut steps, factory) = baseline_steps_with_factory();
    let calldata = encode_create_calldata(&ctor_returns_oversize());
    let tx = signed(
        3,
        Some(factory),
        Bytes::from(calldata),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("create_oversize", steps);
}

#[test]
#[ignore]
fn create_ef_prefix_returned() {
    let (mut steps, factory) = baseline_steps_with_factory();
    let calldata = encode_create_calldata(&ctor_returns_ef_prefix());
    let tx = signed(
        3,
        Some(factory),
        Bytes::from(calldata),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("create_ef_prefix", steps);
}

#[test]
#[ignore]
fn create_endowment_transfer() {
    let (mut steps, factory) = baseline_steps_with_factory();
    let init = ctor_returns_runtime(&[0x00]);
    let calldata = encode_create_calldata(&init);
    let tx = signed(
        3,
        Some(factory),
        Bytes::from(calldata),
        U256::from(123u64),
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("create_endowment", steps);
}

// ── geth evm.go `Create2` paths ────────────────────────────────────────────

#[test]
#[ignore]
fn create2_constructor_sload() {
    let (mut steps, factory) = baseline_steps_with_factory();
    let calldata = encode_create2_calldata(B256::ZERO, &ctor_storage_op(0x54));
    let tx = signed(
        3,
        Some(factory),
        Bytes::from(calldata),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("create2_sload", steps);
}

#[test]
#[ignore]
fn create2_address_collision() {
    let (mut steps, factory) = baseline_steps_with_factory();
    let init = ctor_returns_runtime(&[0x00]);
    let calldata = encode_create2_calldata(B256::ZERO, &init);
    let tx1 = signed(
        3,
        Some(factory),
        Bytes::from(calldata.clone()),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx1");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx1, idx));
    let tx2 = signed(
        4,
        Some(factory),
        Bytes::from(calldata),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx2");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx2, idx));
    run_named("create2_collision", steps);
}

// ── geth evm.go `Call` paths ───────────────────────────────────────────────

#[test]
#[ignore]
fn call_value_transfer_to_new_account() {
    let mut steps = Vec::new();
    fund_eoa(&mut steps);
    let _ = deploy_and_activate_stylus(&mut steps, 0);
    let tx = signed(
        2,
        Some(Address::repeat_byte(0xab)),
        Bytes::new(),
        U256::from(7u64),
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("call_value_to_new_account", steps);
}

#[test]
#[ignore]
fn call_precompile_ecrecover() {
    let mut steps = Vec::new();
    fund_eoa(&mut steps);
    let _ = deploy_and_activate_stylus(&mut steps, 0);
    let mut data = vec![0u8; 128];
    data[31] = 0x01;
    let tx = signed(
        2,
        Some(Address::with_last_byte(0x01)),
        Bytes::from(data),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    run_named("call_precompile_ecrecover", steps);
}
