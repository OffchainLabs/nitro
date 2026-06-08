//! Differential test: ArbOwner fee-collector setters take effect within the
//! same block.
//!
//! The network and infrastructure fee collectors are read at each tx's
//! fee-distribution time, not at block start, so an owner that changes them
//! reroutes fees immediately — including the setting tx's own fee. Each setter
//! is driven by a chain owner with ample gas and compared against the reference
//! node; control setters that store unrelated values confirm there is no
//! regression on the common path.
//!
//! Run:
//!   ARB_SPEC_BINARY=$(pwd)/target/fastdev/arb-reth \
//!     NITRO_REF_IMAGE=offchainlabs/nitro-node:v3.10.1-d7f07be \
//!     cargo test -p arb-fuzz --test arbowner_zero_write_dual -- --ignored --nocapture

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

const ARBOWNER: Address = address!("0000000000000000000000000000000000000070");
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

fn word_arg(sel: &str, value: U256) -> Vec<u8> {
    let mut d = selector4(sel).to_vec();
    d.extend_from_slice(&value.to_be_bytes::<32>());
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

/// One owner-signed setter call (ample gas) compared against the reference node.
fn run_single(name: &str, sel: &str, value: U256) {
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

    let i = next();
    steps.push(msg_step(
        i,
        tx(
            0,
            Some(ARBOWNER),
            word_arg(sel, value),
            3_000_000,
            BASE_TS + i * BLOCK_SECS,
        )
        .build()
        .expect("setter call"),
        1,
    ));

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
        description: "ArbOwner setter fee-collector parity".into(),
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

// Changing the network fee collector reroutes the setting tx's own fee, so both
// nodes must agree on the new collector's credit (zero and nonzero targets).
#[test]
#[ignore]
fn set_network_fee_account_zero_matches_nitro() {
    run_single(
        "set_network_fee_account_zero",
        "setNetworkFeeAccount(address)",
        U256::ZERO,
    );
}

#[test]
#[ignore]
fn set_network_fee_account_nonzero_matches_nitro() {
    run_single(
        "set_network_fee_account_nonzero",
        "setNetworkFeeAccount(address)",
        U256::from_be_slice(Address::new([0xb7; 20]).as_slice()),
    );
}

// Setting the infra collector from the genesis zero to a nonzero address splits
// the setting tx's own fee between infra and network for the first time.
#[test]
#[ignore]
fn set_infra_fee_account_nonzero_matches_nitro() {
    run_single(
        "set_infra_fee_account_nonzero",
        "setInfraFeeAccount(address)",
        U256::from_be_slice(Address::new([0xc8; 20]).as_slice()),
    );
}

#[test]
#[ignore]
fn set_infra_fee_account_zero_matches_nitro() {
    run_single(
        "set_infra_fee_account_zero",
        "setInfraFeeAccount(address)",
        U256::ZERO,
    );
}

// Controls: setters whose value is unrelated to per-tx fee routing must stay
// clean at the zero edge (guards against an over-broad fix).
#[test]
#[ignore]
fn set_reward_recipient_zero_matches_nitro() {
    run_single(
        "set_reward_recipient_zero",
        "setL1PricingRewardRecipient(address)",
        U256::ZERO,
    );
}

#[test]
#[ignore]
fn set_l1_price_per_unit_zero_matches_nitro() {
    run_single(
        "set_l1_price_per_unit_zero",
        "setL1PricePerUnit(uint256)",
        U256::ZERO,
    );
}
