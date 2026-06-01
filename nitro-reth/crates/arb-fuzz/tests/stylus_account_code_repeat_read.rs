//! Regression test for the Stylus `account_code` repeat-read gas charge.
//!
//! A Stylus program that reads a large data contract's code in many
//! consecutive chunks must charge the account-touch cost only on the first
//! read of an address; subsequent reads of the same address are free. This
//! genesis-injects the real contracts at their on-chain addresses and replays
//! the transaction, diffing arbreth vs Nitro.
//!
//! Run:
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     cargo test -p arb-fuzz --test stylus_account_code_repeat_read --release -- --ignored

use std::path::PathBuf;

use alloy_primitives::{address, Address, Bytes, U256};
use arb_fuzz::{
    arbitrary_impls::{interop::interop_eoa, message_step},
    scaffolding::{activate_program, signed},
    shared_nodes::{fuzz_arbos_version, next_msg_idx, FUZZ_L1_CHAIN_ID, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::{
    dual_exec::DualExec,
    genesis::GenesisBuilder,
    messaging::MessageBuilder,
    mock_l1::MockL1,
    node::{arbreth::ArbrethProcess, nitro_docker::NitroDocker, NodeStartCtx},
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
};
use serde_json::json;

const STYLUS_HEX: &str = include_str!("fixtures/repro254596805/stylus.hex");
const DATA1_HEX: &str = include_str!("fixtures/repro254596805/data1.hex");
const DATA2_HEX: &str = include_str!("fixtures/repro254596805/data2.hex");

const STYLUS_ADDR: Address = address!("51d5d578c1544af73b6a3056669a6275b1714d00");
const DATA1_ADDR: Address = address!("6cc702154ad937654c851e71400c9009e42c3b53");
const DATA2_ADDR: Address = address!("e2e7ecd57daf408e0875a943305c703602ac972b");

fn decode(h: &str) -> Vec<u8> {
    hex::decode(h.trim().trim_start_matches("0x")).expect("hex")
}

fn alloc_code(code: &[u8]) -> serde_json::Value {
    json!({ "balance": "0x0", "code": format!("0x{}", hex::encode(code)) })
}

#[test]
#[ignore]
fn account_code_repeat_read_matches_nitro() {
    let mock = MockL1::start(FUZZ_L1_CHAIN_ID).expect("mock l1");
    let mut genesis = GenesisBuilder::new(FUZZ_L2_CHAIN_ID, fuzz_arbos_version())
        .build()
        .expect("genesis build");
    {
        let alloc = genesis
            .get_mut("alloc")
            .and_then(|a| a.as_object_mut())
            .expect("alloc object");
        alloc.insert(
            format!("0x{:x}", STYLUS_ADDR),
            alloc_code(&decode(STYLUS_HEX)),
        );
        alloc.insert(
            format!("0x{:x}", DATA1_ADDR),
            alloc_code(&decode(DATA1_HEX)),
        );
        alloc.insert(
            format!("0x{:x}", DATA2_ADDR),
            alloc_code(&decode(DATA2_HEX)),
        );
        alloc.insert(
            format!("0x{:x}", interop_eoa()),
            json!({ "balance": "0xd3c21bcecceda1000000" }),
        );
    }

    let ctx = NodeStartCtx {
        binary: None,
        l2_chain_id: FUZZ_L2_CHAIN_ID,
        l1_chain_id: FUZZ_L1_CHAIN_ID,
        mock_l1_rpc: mock.rpc_url(),
        genesis,
        jwt_hex: String::new(),
        workdir: PathBuf::new(),
        http_port: 0,
        authrpc_port: 0,
    };
    let nitro = NitroDocker::start(&ctx).expect("nitro start");
    let arbreth = ArbrethProcess::start(&ctx).expect("arbreth start");
    let mut dual = DualExec::new(nitro, arbreth);

    let mut steps: Vec<ScenarioStep> = Vec::new();
    activate_program(&mut steps, 0, STYLUS_ADDR);

    let mut calldata = Vec::with_capacity(16_001);
    calldata.push(0x01);
    calldata.extend(std::iter::repeat_n(0x42, 16_000));
    let invoke = signed(
        1,
        Some(STYLUS_ADDR),
        Bytes::from(calldata),
        U256::ZERO,
        50_000_000,
    )
    .build()
    .expect("invoke");
    let idx = next_msg_idx();
    steps.push(message_step(idx, invoke, idx));

    let scen = Scenario {
        name: "stylus_account_code_repeat_read".into(),
        description: "Stylus account_code repeat-read gas parity".into(),
        setup: ScenarioSetup {
            l2_chain_id: FUZZ_L2_CHAIN_ID,
            arbos_version: fuzz_arbos_version(),
            genesis: None,
        },
        steps,
    };

    let report = dual.run(&scen).expect("run");
    for d in &report.block_diffs {
        eprintln!(
            "BLOCK {} {} nitro={} arbreth={}",
            d.number, d.field, d.left, d.right
        );
    }
    for d in &report.tx_diffs {
        eprintln!(
            "TX {:?} {} nitro={} arbreth={}",
            d.tx_hash, d.field, d.left, d.right
        );
    }
    std::mem::forget(mock);
    assert!(
        report.is_clean(),
        "arbreth diverged from Nitro (see diffs above)"
    );
}
