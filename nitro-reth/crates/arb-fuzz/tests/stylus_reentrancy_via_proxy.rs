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

fn sel(s: &str) -> [u8; 4] {
    let h = keccak256(s.as_bytes());
    [h[0], h[1], h[2], h[3]]
}

fn signed(nonce: u64, to: Option<Address>, data: Bytes, gas: u64) -> SignedL2TxBuilder {
    SignedL2TxBuilder {
        chain_id: FUZZ_L2_CHAIN_ID,
        nonce,
        to,
        value: U256::ZERO,
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

fn run_named(name: &str, steps: Vec<ScenarioStep>) {
    let scen = Scenario {
        name: name.into(),
        description: format!("reentrancy: {name}"),
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
    if !report.is_clean() {
        let payload = serde_json::json!({
            "scenario": name,
            "block_diffs": format!("{:#?}", report.block_diffs),
            "tx_diffs": format!("{:#?}", report.tx_diffs),
            "state_diffs": format!("{:#?}", report.state_diffs),
            "log_diffs": format!("{:#?}", report.log_diffs),
        });
        let path = std::path::PathBuf::from(format!("/tmp/stylus_reentrancy_{name}.json"));
        let _ = std::fs::write(&path, serde_json::to_vec_pretty(&payload).unwrap());
        panic!("diverged on {name}; see {}", path.display());
    }
}

fn proxy_runtime(impl_addr: Address) -> Vec<u8> {
    let mut out = Vec::with_capacity(64);
    out.extend_from_slice(&[0x60, 0x00, 0x60, 0x00, 0x37]);
    out.extend_from_slice(&[0x36, 0x60, 0x00, 0x60, 0x00, 0x37]);
    out.extend_from_slice(&[0x60, 0x00, 0x60, 0x00, 0x60, 0x00, 0x36, 0x60, 0x00]);
    out.push(0x73);
    out.extend_from_slice(impl_addr.as_slice());
    out.extend_from_slice(&[0x5a, 0xf4]);
    out.extend_from_slice(&[0x60, 0x00, 0x60, 0x00, 0xf3]);
    out
}

fn reentry_helper(proxy: Address, forward_target: Address) -> Vec<u8> {
    let mut inner = Vec::with_capacity(4 + 96);
    inner.extend_from_slice(&sel("forward(address,bytes)"));
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(forward_target.as_slice());
    inner.extend_from_slice(&pad);
    let mut off32 = [0u8; 32];
    off32[31] = 0x40;
    inner.extend_from_slice(&off32);
    let mut len32 = [0u8; 32];
    len32[31] = 0;
    inner.extend_from_slice(&len32);

    let inner_len = inner.len() as u8;
    let mut out = Vec::with_capacity(80 + inner.len());
    for (i, byte) in inner.iter().enumerate() {
        out.extend_from_slice(&[0x60, *byte, 0x60, i as u8, 0x52]);
    }
    let _ = inner_len;

    let mut prefix = Vec::with_capacity(80);
    prefix.push(0x7f);
    for i in 0..32 {
        prefix.push(if i < inner.len() { inner[i] } else { 0 });
    }
    prefix.extend_from_slice(&[0x60, 0x00, 0x52]);
    prefix.push(0x7f);
    for i in 0..32 {
        let idx = 32 + i;
        prefix.push(if idx < inner.len() { inner[idx] } else { 0 });
    }
    prefix.extend_from_slice(&[0x60, 0x20, 0x52]);
    prefix.push(0x7f);
    for i in 0..32 {
        let idx = 64 + i;
        prefix.push(if idx < inner.len() { inner[idx] } else { 0 });
    }
    prefix.extend_from_slice(&[0x60, 0x40, 0x52]);
    prefix.push(0x7f);
    for i in 0..32 {
        let idx = 96 + i;
        prefix.push(if idx < inner.len() { inner[idx] } else { 0 });
    }
    prefix.extend_from_slice(&[0x60, 0x60, 0x52]);
    prefix.extend_from_slice(&[0x60, 0x60, 0x52]);

    let mut out2 = Vec::with_capacity(120);
    out2.extend_from_slice(&prefix);
    out2.extend_from_slice(&[
        0x60, 0x00, 0x60, 0x00, 0x60, 0x20, 0x60, inner.len() as u8, 0x60, 0x00,
    ]);
    out2.push(0x73);
    out2.extend_from_slice(proxy.as_slice());
    out2.extend_from_slice(&[0x5a, 0xf1, 0x60, 0x00, 0x60, 0x00, 0xf3]);
    out2
}

#[test]
#[ignore]
fn reentry_via_proxy_must_fail_when_stylus_default() {
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

    let stylus_addr = create_address(eoa, 0);
    let deploy_stylus = signed(
        0,
        None,
        Bytes::from(WhichProgram::SolCaller.initcode()),
        DEPLOY_GAS_CAP,
    )
    .build()
    .expect("deploy stylus");
    let i = next_msg_idx();
    steps.push(message_step(i, deploy_stylus, i));

    let mut act = Vec::with_capacity(36);
    act.extend_from_slice(&[0x58, 0xc7, 0x80, 0xc2]);
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(stylus_addr.as_slice());
    act.extend_from_slice(&pad);
    let activate = signed(1, Some(ARBWASM_ADDR), Bytes::from(act), INVOKE_GAS_CAP)
        .build()
        .expect("activate");
    let i = next_msg_idx();
    steps.push(message_step(i, activate, i));

    let proxy_addr = create_address(eoa, 2);
    let helper_addr = create_address(eoa, 3);
    let helper_runtime = reentry_helper(proxy_addr, helper_addr);
    let deploy_proxy = signed(
        2,
        None,
        Bytes::from(wrap_init_code(&proxy_runtime(stylus_addr))),
        DEPLOY_GAS_CAP,
    )
    .build()
    .expect("deploy proxy");
    let i = next_msg_idx();
    steps.push(message_step(i, deploy_proxy, i));

    let deploy_helper = signed(
        3,
        None,
        Bytes::from(wrap_init_code(&helper_runtime)),
        DEPLOY_GAS_CAP,
    )
    .build()
    .expect("deploy helper");
    let i = next_msg_idx();
    steps.push(message_step(i, deploy_helper, i));

    let mut cdata = Vec::with_capacity(4 + 96);
    cdata.extend_from_slice(&sel("forward(address,bytes)"));
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(helper_addr.as_slice());
    cdata.extend_from_slice(&pad);
    let mut off32 = [0u8; 32];
    off32[31] = 0x40;
    cdata.extend_from_slice(&off32);
    cdata.extend_from_slice(&[0u8; 32]);
    let tx = signed(4, Some(proxy_addr), Bytes::from(cdata), INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let i = next_msg_idx();
    steps.push(message_step(i, tx, i));

    run_named("reentry_via_proxy", steps);
}
