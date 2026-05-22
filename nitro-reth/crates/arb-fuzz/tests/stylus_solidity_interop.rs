//! Stylus -> Solidity -> Stylus reentrancy diff vs Nitro.
//!
//! Steps:
//!   1. Fund interop EOA.
//!   2. Deploy SolCaller (real stylus-sdk contract).
//!   3. activateProgram(SolCaller).
//!   4. Deploy Reentrant Solidity companion.
//!   5. EOA -> Reentrant.attack(sol_caller_addr) — Solidity calls back into Stylus's
//!      `forward(reentrant, "")`, which in turn re-enters Reentrant (no fallback). Tests
//!      cross-language gas / return-data / state plumbing.
//!
//! Run with:
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     NITRO_REF_IMAGE=offchainlabs/nitro-node:v3.10.0-rc.10-b1cf6db \
//!     cargo test -p arb-fuzz --test stylus_solidity_interop --release \
//!     -- --ignored --nocapture

use alloy_primitives::{keccak256, Address, Bytes, U256};
use arb_fuzz::{
    arbitrary_impls::{
        interop::{
            create_address, interop_eoa, interop_signing_key, reentrant_runtime, wrap_init_code,
            WhichProgram,
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

#[test]
#[ignore]
fn stylus_solidity_reentrancy_matches_canon() {
    let eoa = interop_eoa();
    let mut steps: Vec<ScenarioStep> = Vec::new();

    // 1. Fund EOA
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

    // 2. Deploy SolCaller (real stylus contract).
    let deploy_nonce = 0u64;
    let stylus_addr = create_address(eoa, deploy_nonce);
    let initcode = WhichProgram::SolCaller.initcode();
    let deploy = signed(
        deploy_nonce,
        None,
        Bytes::from(initcode),
        U256::ZERO,
        DEPLOY_GAS_CAP,
    )
    .build()
    .expect("stylus deploy");
    let idx = next_msg_idx();
    steps.push(message_step(idx, deploy, idx));

    // 3. activateProgram(stylus_addr).
    let mut activate_data = Vec::with_capacity(4 + 32);
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

    // 4. Deploy Reentrant solidity companion.
    let reentrant_nonce = 2u64;
    let reentrant_addr = create_address(eoa, reentrant_nonce);
    let reentrant = signed(
        reentrant_nonce,
        None,
        Bytes::from(wrap_init_code(&reentrant_runtime())),
        U256::ZERO,
        DEPLOY_GAS_CAP,
    )
    .build()
    .expect("reentrant deploy");
    let idx = next_msg_idx();
    steps.push(message_step(idx, reentrant, idx));

    // 5. EOA -> Reentrant.attack(stylus_addr)
    let mut attack_calldata = selector("attack(address)").to_vec();
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(stylus_addr.as_slice());
    attack_calldata.extend_from_slice(&pad);

    let attack = signed(
        3,
        Some(reentrant_addr),
        Bytes::from(attack_calldata),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("attack");
    let idx = next_msg_idx();
    steps.push(message_step(idx, attack, idx));

    let scen = Scenario {
        name: "stylus_solidity_reentrancy".into(),
        description: "Stylus -> Solidity -> Stylus reentrancy".into(),
        setup: ScenarioSetup {
            l2_chain_id: FUZZ_L2_CHAIN_ID,
            arbos_version: fuzz_arbos_version(),
            genesis: None,
        },
        steps,
    };

    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
    let report = nodes.run(&scen).expect("run reentrancy scenario");

    let real_block: Vec<_> = report.block_diffs.iter().collect();

    if !real_block.is_empty()
        || !report.tx_diffs.is_empty()
        || !report.state_diffs.is_empty()
        || !report.log_diffs.is_empty()
    {
        let payload = serde_json::json!({
            "block_diffs": format!("{:#?}", real_block),
            "tx_diffs": format!("{:#?}", report.tx_diffs),
            "state_diffs": format!("{:#?}", report.state_diffs),
            "log_diffs": format!("{:#?}", report.log_diffs),
        });
        let path = std::path::PathBuf::from("/tmp/stylus_solidity_interop.json");
        let _ = std::fs::write(&path, serde_json::to_vec_pretty(&payload).unwrap());
        panic!(
            "arbreth diverged from Nitro on stylus_solidity_interop; see {}",
            path.display()
        );
    }
}
