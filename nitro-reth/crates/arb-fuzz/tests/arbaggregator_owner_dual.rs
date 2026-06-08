//! Differential test: ArbAggregator owner-gated success paths.
//!
//! The precompile CALL matrix exercises these methods only from a non-owner
//! caller, so it covers the access-control rejection but never the success
//! body. `addBatchPoster` and `setFeeCollector` write the batch-poster table;
//! their storage-op count — and therefore their gas — must match the reference
//! node exactly when driven by a chain owner with ample gas.
//!
//! Run:
//!   ARB_SPEC_BINARY=$(pwd)/target/fastdev/arb-reth \
//!     NITRO_REF_IMAGE=offchainlabs/nitro-node:v3.10.1-d7f07be \
//!     cargo test -p arb-fuzz --test arbaggregator_owner_dual -- --ignored --nocapture

use std::sync::Mutex;

use alloy_primitives::{address, Address, Bytes, B256, U256};
use arb_fuzz::scaffolding::selector4;
use arb_test_harness::{
    dual_exec::DualExec,
    genesis::GenesisBuilder,
    messaging::{
        signed_tx::{derive_address, L2TxKind, SignedL2TxBuilder},
        DepositBuilder, MessageBuilder,
    },
    mock_l1::MockL1,
    node::{arbreth::ArbrethProcess, nitro_docker::NitroDocker, NodeStartCtx},
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
};

static SERIAL: Mutex<()> = Mutex::new(());

const L2_CHAIN_ID: u64 = 412_349;
const L1_CHAIN_ID: u64 = 11_155_111;
const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const ARBOS_VERSION: u64 = 60;
const BASE_TS: u64 = 1_700_000_000;
const BLOCK_SECS: u64 = 12;

const ARBAGGREGATOR: Address = address!("000000000000000000000000000000000000006d");
const FUNDER: Address = Address::new([0xa1; 20]);

fn owner_key() -> B256 {
    B256::repeat_byte(0x42)
}

fn tx(nonce: u64, to: Option<Address>, data: Vec<u8>, gas: u64, ts: u64) -> SignedL2TxBuilder {
    SignedL2TxBuilder {
        chain_id: L2_CHAIN_ID,
        nonce,
        to,
        value: U256::ZERO,
        data: Bytes::from(data),
        gas_limit: gas,
        gas_price: 10_000_000_000,
        max_fee_per_gas: 10_000_000_000,
        max_priority_fee_per_gas: 0,
        access_list: Vec::new(),
        authorization_list: Vec::new(),
        kind: L2TxKind::Eip1559,
        signing_key: owner_key(),
        l1_block_number: 1 + (ts - BASE_TS) / BLOCK_SECS,
        timestamp: ts,
        request_id: None,
        sender: address!("a4b000000000000000000073657175656e636572"),
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
}

fn msg_step(idx: u64, msg: arb_test_harness::messaging::L1Message, dmr: u64) -> ScenarioStep {
    ScenarioStep::Message {
        idx,
        message: msg,
        delayed_messages_read: dmr,
    }
}

fn addr_arg(sel: &str, a: Address) -> Vec<u8> {
    let mut d = selector4(sel).to_vec();
    d.extend_from_slice(B256::left_padding_from(a.as_slice()).as_slice());
    d
}

fn two_addr_arg(sel: &str, a: Address, b: Address) -> Vec<u8> {
    let mut d = addr_arg(sel, a);
    d.extend_from_slice(B256::left_padding_from(b.as_slice()).as_slice());
    d
}

struct Rig {
    dual: DualExec<NitroDocker, ArbrethProcess>,
}

impl Rig {
    fn spawn(owner: Address) -> Self {
        let mock = MockL1::start(L1_CHAIN_ID).expect("mock l1 start");
        let genesis = GenesisBuilder::new(L2_CHAIN_ID, ARBOS_VERSION)
            .with_initial_chain_owner(owner)
            .build()
            .expect("genesis build");
        let ctx = NodeStartCtx {
            binary: None,
            l2_chain_id: L2_CHAIN_ID,
            l1_chain_id: L1_CHAIN_ID,
            mock_l1_rpc: mock.rpc_url(),
            genesis,
            jwt_hex: String::new(),
            workdir: std::path::PathBuf::new(),
            http_port: 0,
            authrpc_port: 0,
        };
        let nitro = NitroDocker::start(&ctx).expect("nitro docker start");
        let arbreth = ArbrethProcess::start(&ctx).expect("arbreth start");
        std::mem::forget(mock);
        Rig {
            dual: DualExec::new(nitro, arbreth),
        }
    }
}

/// Drives a list of owner-signed ArbAggregator calls (ample gas) and asserts the
/// two nodes agree on every block, receipt, and state slot.
fn run_owner_calls(name: &str, calls: Vec<Vec<u8>>) {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);

    let mut steps = Vec::new();
    let mut idx = 0u64;
    let mut next = || {
        idx += 1;
        idx
    };

    let i = next();
    steps.push(msg_step(
        i,
        DepositBuilder {
            from: FUNDER,
            to: owner,
            amount: U256::from(10u128).pow(U256::from(21u64)),
            l1_block_number: 1,
            timestamp: BASE_TS,
            request_seq: i,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        }
        .build()
        .expect("deposit"),
        1,
    ));

    for (n, data) in calls.into_iter().enumerate() {
        let i = next();
        steps.push(msg_step(
            i,
            tx(
                n as u64,
                Some(ARBAGGREGATOR),
                data,
                3_000_000,
                BASE_TS + i * BLOCK_SECS,
            )
            .build()
            .expect("aggregator call"),
            1,
        ));
    }

    // Trailing no-op deposit so the last call's block is sealed.
    let i = next();
    steps.push(msg_step(
        i,
        DepositBuilder {
            from: FUNDER,
            to: FUNDER,
            amount: U256::from(1u64),
            l1_block_number: 1 + (BASE_TS + i * BLOCK_SECS - BASE_TS) / BLOCK_SECS,
            timestamp: BASE_TS + i * BLOCK_SECS,
            request_seq: i,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        }
        .build()
        .expect("trailing deposit"),
        2,
    ));

    let scenario = Scenario {
        name: name.into(),
        description: "ArbAggregator owner success path".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let report = rig.dual.run(&scenario).expect("dual run");
    assert!(
        report.is_clean(),
        "arbreth diverged from Nitro on {name}\n  block_diffs={:#?}\n  tx_diffs={:#?}\n  state_diffs={:#?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}

#[test]
#[ignore]
fn add_batch_poster_new_matches_nitro() {
    let poster = Address::new([0xc1; 20]);
    run_owner_calls(
        "add_batch_poster_new",
        vec![addr_arg("addBatchPoster(address)", poster)],
    );
}

#[test]
#[ignore]
fn add_batch_poster_already_present_matches_nitro() {
    let poster = Address::new([0xc2; 20]);
    run_owner_calls(
        "add_batch_poster_already_present",
        vec![
            addr_arg("addBatchPoster(address)", poster),
            addr_arg("addBatchPoster(address)", poster),
        ],
    );
}

#[test]
#[ignore]
fn set_fee_collector_matches_nitro() {
    let poster = Address::new([0xc3; 20]);
    let collector = Address::new([0xd4; 20]);
    run_owner_calls(
        "set_fee_collector",
        vec![
            addr_arg("addBatchPoster(address)", poster),
            two_addr_arg("setFeeCollector(address,address)", poster, collector),
        ],
    );
}

#[test]
#[ignore]
fn set_fee_collector_zero_matches_nitro() {
    let poster = Address::new([0xc5; 20]);
    run_owner_calls(
        "set_fee_collector_zero",
        vec![
            addr_arg("addBatchPoster(address)", poster),
            two_addr_arg("setFeeCollector(address,address)", poster, Address::ZERO),
        ],
    );
}

#[test]
#[ignore]
fn get_fee_collector_and_posters_matches_nitro() {
    let poster = Address::new([0xc6; 20]);
    run_owner_calls(
        "get_fee_collector_and_posters",
        vec![
            addr_arg("addBatchPoster(address)", poster),
            addr_arg("getFeeCollector(address)", poster),
            selector4("getBatchPosters()").to_vec(),
        ],
    );
}
