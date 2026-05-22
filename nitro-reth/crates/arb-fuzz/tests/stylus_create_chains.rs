//! Differential CREATE/CREATE2 coverage through Stylus dispatch vs Nitro.
//!
//! Run:
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     NITRO_REF_IMAGE=offchainlabs/nitro-node:v3.10.0-rc.10-b1cf6db \
//!     cargo test -p arb-fuzz --test stylus_create_chains --release \
//!     -- --ignored --nocapture

use alloy_primitives::{keccak256, Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::{
        interop::{
            create_address, interop_eoa, interop_signing_key, wrap_init_code, WhichProgram,
        },
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
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x71,
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

/// Factory with `deployCreate(bytes)` and `deployCreate2(bytes32,bytes)`,
/// forwarding msg.value as endowment.
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

fn ctor_sload_only() -> Vec<u8> {
    vec![
        0x60, 0x00, 0x54, 0x50, 0x60, 0x00, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xf3,
    ]
}

fn ctor_sstore() -> Vec<u8> {
    vec![
        0x60, 0x42, 0x60, 0x00, 0x55, 0x60, 0x00, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xf3,
    ]
}

fn ctor_subcall(target: Address) -> Vec<u8> {
    let mut out = Vec::with_capacity(40);
    out.extend_from_slice(&[0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x60, 0x00]);
    out.push(0x73);
    out.extend_from_slice(target.as_slice());
    out.extend_from_slice(&[
        0x5a, 0xf1, 0x50, 0x60, 0x00, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xf3,
    ]);
    out
}

fn ctor_nested_create() -> Vec<u8> {
    let inner: &[u8] = &[0x60, 0x00, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xf3];
    let inner_len = inner.len() as u8;
    let mut out = Vec::with_capacity(40);
    out.extend_from_slice(&[
        0x60, inner_len, 0x38, 0x03, 0x60, inner_len, 0x90, 0x60, 0x00, 0x39, 0x60, inner_len,
        0x60, 0x00, 0x60, 0x00, 0xf0, 0x50, 0x60, 0x00, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00,
        0xf3,
    ]);
    out.extend_from_slice(inner);
    out
}

fn encode_bytes_arg(data: &[u8]) -> Vec<u8> {
    let mut out = Vec::with_capacity(64 + data.len() + 32);
    let mut len32 = [0u8; 32];
    len32[24..32].copy_from_slice(&(data.len() as u64).to_be_bytes());
    let mut off32 = [0u8; 32];
    off32[31] = 0x20;
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
    out.extend_from_slice(&encode_bytes_arg(init_code));
    out
}

fn encode_create2_calldata(salt: B256, init_code: &[u8]) -> Vec<u8> {
    let mut out = Vec::with_capacity(4 + 96 + init_code.len());
    out.extend_from_slice(&selector("deployCreate2(bytes32,bytes)"));
    out.extend_from_slice(salt.as_slice());
    // bytes offset = 0x40
    let mut off32 = [0u8; 32];
    off32[31] = 0x40;
    out.extend_from_slice(&off32);
    // length
    let mut len32 = [0u8; 32];
    len32[24..32].copy_from_slice(&(init_code.len() as u64).to_be_bytes());
    out.extend_from_slice(&len32);
    // data
    out.extend_from_slice(init_code);
    while out.len() % 32 != 0 {
        out.push(0);
    }
    out
}

fn wrap_via_stylus_forward(stylus_addr: Address, target: Address, inner: &[u8]) -> Bytes {
    let mut out = Vec::with_capacity(4 + 32 + 32 + 32 + inner.len() + 32);
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
    let _ = stylus_addr;
    Bytes::from(out)
}

fn stylus_addr() -> Address {
    create_address(interop_eoa(), 0)
}

fn build_steps(target_tx_data: Bytes, target_to: Address) -> Vec<ScenarioStep> {
    let eoa = interop_eoa();
    let mut steps: Vec<ScenarioStep> = Vec::new();

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

    let stylus_nonce = 0u64;
    let stylus_initcode = WhichProgram::SolCaller.initcode();
    let deploy_stylus = signed(
        stylus_nonce,
        None,
        Bytes::from(stylus_initcode),
        U256::ZERO,
        DEPLOY_GAS_CAP,
    )
    .build()
    .expect("stylus deploy");
    let idx = next_msg_idx();
    steps.push(message_step(idx, deploy_stylus, idx));

    let stylus_addr = create_address(eoa, stylus_nonce);
    let mut activate_data = Vec::with_capacity(36);
    activate_data.extend_from_slice(&[0x58, 0xc7, 0x80, 0xc2]);
    let mut padded = [0u8; 32];
    padded[12..].copy_from_slice(stylus_addr.as_slice());
    activate_data.extend_from_slice(&padded);
    let activate = signed(
        1,
        Some(ARBWASM_ADDR),
        Bytes::from(activate_data),
        U256::from(10u128).pow(U256::from(15u64)),
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("activate");
    let idx = next_msg_idx();
    steps.push(message_step(idx, activate, idx));

    let factory_nonce = 2u64;
    let deploy_factory = signed(
        factory_nonce,
        None,
        Bytes::from(wrap_init_code(&factory_runtime())),
        U256::ZERO,
        DEPLOY_GAS_CAP,
    )
    .build()
    .expect("factory deploy");
    let idx = next_msg_idx();
    steps.push(message_step(idx, deploy_factory, idx));

    let tx = signed(3, Some(target_to), target_tx_data, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("target tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));

    steps
}

fn run_named(name: &str, steps: Vec<ScenarioStep>) {
    let scen = Scenario {
        name: name.into(),
        description: format!("CREATE-chain diff: {name}"),
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
        let path = std::path::PathBuf::from(format!("/tmp/stylus_create_chains_{name}.json"));
        let _ = std::fs::write(&path, serde_json::to_vec_pretty(&payload).unwrap());
        panic!("arbreth diverged from Nitro on {name}; see {}", path.display());
    }
}

fn factory_address() -> Address {
    create_address(interop_eoa(), /* factory nonce */ 2)
}

#[test]
#[ignore]
fn create_constructor_sload_only() {
    let inner = encode_create_calldata(&ctor_sload_only());
    let calldata = wrap_via_stylus_forward(stylus_addr(), factory_address(), &inner);
    let steps = build_steps(calldata, stylus_addr());
    run_named("create_sload_only", steps);
}

#[test]
#[ignore]
fn create2_constructor_sload_only() {
    let inner = encode_create2_calldata(B256::ZERO, &ctor_sload_only());
    let calldata = wrap_via_stylus_forward(stylus_addr(), factory_address(), &inner);
    let steps = build_steps(calldata, stylus_addr());
    run_named("create2_sload_only", steps);
}

#[test]
#[ignore]
fn create_constructor_sstore() {
    let inner = encode_create_calldata(&ctor_sstore());
    let calldata = wrap_via_stylus_forward(stylus_addr(), factory_address(), &inner);
    let steps = build_steps(calldata, stylus_addr());
    run_named("create_sstore", steps);
}

#[test]
#[ignore]
fn create_constructor_subcall() {
    let target = factory_address();
    let inner = encode_create_calldata(&ctor_subcall(target));
    let calldata = wrap_via_stylus_forward(stylus_addr(), factory_address(), &inner);
    let steps = build_steps(calldata, stylus_addr());
    run_named("create_subcall", steps);
}

#[test]
#[ignore]
fn create_constructor_nested_create() {
    let inner = encode_create_calldata(&ctor_nested_create());
    let calldata = wrap_via_stylus_forward(stylus_addr(), factory_address(), &inner);
    let steps = build_steps(calldata, stylus_addr());
    run_named("create_nested_create", steps);
}

// ── Stylus's own create/create2 hostios (parallel path to factory CREATE) ──

fn do_create_calldata(endowment: U256, init_code: &[u8]) -> Bytes {
    let mut out = Vec::with_capacity(4 + 96 + init_code.len());
    out.extend_from_slice(&selector("doCreate(uint256,bytes)"));
    out.extend_from_slice(&endowment.to_be_bytes::<32>());
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
    Bytes::from(out)
}

fn do_create2_calldata(endowment: U256, salt: B256, init_code: &[u8]) -> Bytes {
    let mut out = Vec::with_capacity(4 + 128 + init_code.len());
    out.extend_from_slice(&selector("doCreate2(uint256,bytes32,bytes)"));
    out.extend_from_slice(&endowment.to_be_bytes::<32>());
    out.extend_from_slice(salt.as_slice());
    let mut off32 = [0u8; 32];
    off32[31] = 0x60;
    out.extend_from_slice(&off32);
    let mut len32 = [0u8; 32];
    len32[24..32].copy_from_slice(&(init_code.len() as u64).to_be_bytes());
    out.extend_from_slice(&len32);
    out.extend_from_slice(init_code);
    while out.len() % 32 != 0 {
        out.push(0);
    }
    Bytes::from(out)
}

#[test]
#[ignore]
fn stylus_create_hostio_sload() {
    let calldata = do_create_calldata(U256::ZERO, &ctor_sload_only());
    let steps = build_steps(calldata, stylus_addr());
    run_named("stylus_create_sload", steps);
}

#[test]
#[ignore]
fn stylus_create_hostio_sstore() {
    let calldata = do_create_calldata(U256::ZERO, &ctor_sstore());
    let steps = build_steps(calldata, stylus_addr());
    run_named("stylus_create_sstore", steps);
}

#[test]
#[ignore]
fn stylus_create_hostio_with_endowment() {
    let calldata = do_create_calldata(U256::from(1000u64), &ctor_sload_only());
    let steps = build_steps(calldata, stylus_addr());
    run_named("stylus_create_endowment", steps);
}

#[test]
#[ignore]
fn stylus_create2_hostio_sload() {
    let calldata = do_create2_calldata(U256::ZERO, B256::ZERO, &ctor_sload_only());
    let steps = build_steps(calldata, stylus_addr());
    run_named("stylus_create2_sload", steps);
}

#[test]
#[ignore]
fn stylus_create2_hostio_collision() {
    let mut steps = build_steps(
        do_create2_calldata(U256::ZERO, B256::ZERO, &ctor_sload_only()),
        stylus_addr(),
    );
    let tx2 = signed(
        4,
        Some(stylus_addr()),
        do_create2_calldata(U256::ZERO, B256::ZERO, &ctor_sload_only()),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("second tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx2, idx));
    run_named("stylus_create2_collision", steps);
}

#[test]
#[ignore]
fn stylus_create_hostio_returns_ef_prefix() {
    let runtime: &[u8] = &[0xef, 0x00, 0x00];
    let init = wrap_init_code(runtime);
    let calldata = do_create_calldata(U256::ZERO, &init);
    let steps = build_steps(calldata, stylus_addr());
    run_named("stylus_create_ef_prefix", steps);
}

#[test]
#[ignore]
fn stylus_create_hostio_oversize() {
    let mut runtime = vec![0x00u8; 25_000];
    runtime[0] = 0xfe;
    let init = wrap_init_code(&runtime);
    let calldata = do_create_calldata(U256::ZERO, &init);
    let steps = build_steps(calldata, stylus_addr());
    run_named("stylus_create_oversize", steps);
}
