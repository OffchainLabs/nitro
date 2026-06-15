//! ArbFilteredTransactionsManager (`0x74`) free-access add must out-of-gas at the
//! 22_728 threshold (argsCost + OpenArbosState + IsMember + SSTORE + log) and
//! surface that OOG as a revert that rolls back the SSTORE and log.
//!
//! A *filterer* contract calls `0x74.addFilteredTransaction(h)` through an inner
//! `CALL{gas: inner}` and records the inner CALL's success flag at its slot 0.
//! Below the threshold both nodes must revert (slot 0, no log); at/above both
//! must succeed (slot 1, log). The cases differ ONLY in the forwarded inner-gas
//! budget:
//!   - `filtered_add_window_clean` (22_000) and `filtered_add_boundary_below_…` (22_727): in
//!     (21_928, 22_727] — both revert; 22_727 locks the OpenArbosState +800.
//!   - `filtered_add_boundary_at_threshold_clean` (22_728): exact fit — both succeed.
//!   - `filtered_add_ample_gas_control_clean` (50_000): ample — both succeed.

use std::sync::Mutex;

use alloy_primitives::{address, keccak256, Address, Bytes, B256, U256};
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
    scenario::{Scenario, ScenarioSetup, ScenarioStep, StateCheck},
};

static SERIAL: Mutex<()> = Mutex::new(());

const L2_CHAIN_ID: u64 = 412_349;
const L1_CHAIN_ID: u64 = 11_155_111;
const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const ARBOS_VERSION: u64 = 60;
const BASE_TS: u64 = 1_700_000_000;
const BLOCK_SECS: u64 = 12;
const FEATURE_ENABLE_DELAY: u64 = 7 * 24 * 60 * 60;

const ARBOWNER: Address = address!("0000000000000000000000000000000000000070");
const ARBFILTEREDTX: Address = address!("0000000000000000000000000000000000000074");
const FUNDER: Address = Address::new([0xa1; 20]);

// add = argsCost(3) + OpenArbosState(800) + IsMember(800) + SSTORE(20_000) + log(1125) = 22_728.
const ADD_WINDOW_GAS: u16 = 22_000; // in (21_928, 22_727]: below the 22_728 add threshold -> reverts.
const ADD_BOUNDARY_BELOW: u16 = 22_727; // last in-window value: locks the OpenArbosState +800.
const ADD_BOUNDARY_AT: u16 = 22_728; // exact threshold: both succeed.
const AMPLE_INNER_GAS: u16 = 50_000;

const FILTERED_HASH: B256 = B256::repeat_byte(0x5a);

fn owner_key() -> B256 {
    B256::repeat_byte(0x42)
}

fn create_address(deployer: Address, nonce: u64) -> Address {
    let mut rlp = vec![0xd6u8, 0x94];
    rlp.extend_from_slice(deployer.as_slice());
    if nonce == 0 {
        rlp.push(0x80);
    } else {
        rlp.push(nonce as u8);
    }
    Address::from_slice(&keccak256(&rlp)[12..])
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

/// Constructor that returns `runtime` (runtime begins at offset 0x0e).
fn wrap_init(runtime: &[u8]) -> Vec<u8> {
    let len = runtime.len();
    let mut c = Vec::new();
    c.extend_from_slice(&[0x61, (len >> 8) as u8, len as u8]);
    c.extend_from_slice(&[0x60, 0x0e, 0x60, 0x00, 0x39]);
    c.extend_from_slice(&[0x61, (len >> 8) as u8, len as u8]);
    c.extend_from_slice(&[0x60, 0x00, 0xf3]);
    c.extend_from_slice(runtime);
    c
}

/// Runtime that `CALL`s `target` with `selector ++ args` forwarding exactly
/// `gas`, then stores the call's success flag at slot 0.
fn forward_runtime(target: Address, selector: [u8; 4], args: &[u8], gas: u16) -> Vec<u8> {
    debug_assert!(args.len().is_multiple_of(32) && 4 + args.len() < 256);
    let target_byte = target.0[19];
    let mut c = Vec::new();
    c.push(0x63);
    c.extend_from_slice(&selector);
    c.extend_from_slice(&[0x60, 0xE0, 0x1b, 0x60, 0x00, 0x52]); // PUSH1 224 SHL PUSH1 0 MSTORE
    for (i, word) in args.chunks(32).enumerate() {
        c.push(0x7f); // PUSH32
        c.extend_from_slice(word);
        c.extend_from_slice(&[0x60, (4 + 32 * i) as u8, 0x52]); // PUSH1 off MSTORE
    }
    let arg_len = (4 + args.len()) as u8;
    // CALL operands (reverse): retLen retOff argLen argOff value addr gas
    c.extend_from_slice(&[
        0x60,
        0x00,
        0x60,
        0x00,
        0x60,
        arg_len,
        0x60,
        0x00,
        0x60,
        0x00,
        0x60,
        target_byte,
    ]);
    c.push(0x61);
    c.extend_from_slice(&gas.to_be_bytes()); // PUSH2 gas
    c.extend_from_slice(&[0xf1, 0x60, 0x00, 0x55, 0x00]); // CALL PUSH1 0 SSTORE STOP
    c
}

fn word(bytes: &[u8]) -> [u8; 32] {
    B256::left_padding_from(bytes).0
}

fn set_filtering_from(timestamp: u64) -> Vec<u8> {
    let mut d = selector4("setTransactionFilteringFrom(uint64)").to_vec();
    d.extend_from_slice(&word(&timestamp.to_be_bytes()));
    d
}

fn add_transaction_filterer(addr: Address) -> Vec<u8> {
    let mut d = selector4("addTransactionFilterer(address)").to_vec();
    d.extend_from_slice(&word(addr.as_slice()));
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

/// Builds the full scenario: fund owner → deploy forwarder → enable filtering
/// (one-week delay, then advance time) → make the forwarder a filterer →
/// forwarder invokes `0x74.addFilteredTransaction(h)` with `inner_gas` → seal.
fn run(inner_gas: u16, name: &str) -> (arb_test_harness::dual_exec::DiffReport, Address) {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let forwarder = create_address(owner, 0);
    let mut rig = Rig::spawn(owner);

    // `setTransactionFilteringFrom` (step 3, block time `BASE_TS + 3*BLOCK_SECS`)
    // requires the enable time to be at least `now + FeatureEnableDelay`, so it
    // must clear the step-3 block time plus the full one-week delay; one extra
    // block of margin keeps it strictly above the boundary.
    let enable_ts = BASE_TS + 3 * BLOCK_SECS + FEATURE_ENABLE_DELAY + BLOCK_SECS;
    // `addTransactionFilterer` (step 4) gates on `enabled_time <= block.time`.
    let live_ts = enable_ts + BLOCK_SECS;

    let runtime = forward_runtime(
        ARBFILTEREDTX,
        selector4("addFilteredTransaction(bytes32)"),
        FILTERED_HASH.as_slice(),
        inner_gas,
    );

    let steps = vec![
        // 1) Fund the owner.
        msg_step(
            1,
            DepositBuilder {
                from: FUNDER,
                to: owner,
                amount: U256::from(10u128).pow(U256::from(21u64)),
                l1_block_number: 1,
                timestamp: BASE_TS,
                request_seq: 1,
                base_fee_l1: FUZZ_L1_BASE_FEE,
            }
            .build()
            .expect("deposit"),
            1,
        ),
        // 2) Deploy the forwarder (nonce 0 → CREATE address == `forwarder`).
        msg_step(
            2,
            tx(
                0,
                None,
                wrap_init(&runtime),
                3_000_000,
                BASE_TS + 2 * BLOCK_SECS,
            )
            .build()
            .expect("deploy forwarder"),
            1,
        ),
        // 3) Owner enables transaction filtering one week out.
        msg_step(
            3,
            tx(
                1,
                Some(ARBOWNER),
                set_filtering_from(enable_ts),
                3_000_000,
                BASE_TS + 3 * BLOCK_SECS,
            )
            .build()
            .expect("setTransactionFilteringFrom"),
            1,
        ),
        // 4) Owner registers the forwarder as a filterer (block time >= enable_ts).
        msg_step(
            4,
            tx(
                2,
                Some(ARBOWNER),
                add_transaction_filterer(forwarder),
                3_000_000,
                live_ts,
            )
            .build()
            .expect("addTransactionFilterer"),
            1,
        ),
        // 5) Trigger: forwarder calls 0x74.addFilteredTransaction(h) with inner_gas.
        msg_step(
            5,
            tx(
                3,
                Some(forwarder),
                Vec::new(),
                3_000_000,
                live_ts + BLOCK_SECS,
            )
            .build()
            .expect("invoke forwarder"),
            1,
        ),
        // 6) Trailing deposit to seal the block before querying state.
        msg_step(
            6,
            DepositBuilder {
                from: FUNDER,
                to: FUNDER,
                amount: U256::from(1u64),
                l1_block_number: 1 + (live_ts + 2 * BLOCK_SECS - BASE_TS) / BLOCK_SECS,
                timestamp: live_ts + 2 * BLOCK_SECS,
                request_seq: 6,
                base_fee_l1: FUZZ_L1_BASE_FEE,
            }
            .build()
            .expect("trailing deposit"),
            2,
        ),
    ];

    let scenario = Scenario {
        name: name.into(),
        description: "filterer contract calls addFilteredTransaction with a bounded inner budget"
            .into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    // Forwarder slot 0 = recorded inner-CALL success flag.
    let checks = [StateCheck {
        address: forwarder,
        slots: vec![B256::ZERO],
        check_balance: false,
        check_nonce: false,
        check_code: false,
    }];

    let report = rig
        .dual
        .run_with_state_checks(&scenario, &checks)
        .expect("dual run");
    (report, forwarder)
}

#[test]
#[ignore]
fn filtered_add_window_clean() {
    let (report, _) = run(ADD_WINDOW_GAS, "add_window");
    assert!(
        report.is_clean(),
        "add@22000 must agree (both revert below the threshold):\n{report:#?}"
    );
}

#[test]
#[ignore]
fn filtered_add_boundary_below_locks_open_arbos_state_clean() {
    let (report, _) = run(ADD_BOUNDARY_BELOW, "add_boundary_below");
    assert!(
        report.is_clean(),
        "add@22727 must agree (both revert; locks the OpenArbosState +800):\n{report:#?}"
    );
}

#[test]
#[ignore]
fn filtered_add_boundary_at_threshold_clean() {
    let (report, _) = run(ADD_BOUNDARY_AT, "add_boundary_at");
    assert!(
        report.is_clean(),
        "add@22728 must agree (both succeed at the exact threshold):\n{report:#?}"
    );
}

#[test]
#[ignore]
fn filtered_add_ample_gas_control_clean() {
    let (report, _forwarder) = run(AMPLE_INNER_GAS, "ample_inner_gas");
    assert!(
        report.is_clean(),
        "control mismatch (ample inner gas should let both nodes complete the SSTORE + log)\n  \
         block_diffs={:#?}\n  tx_diffs={:#?}\n  state_diffs={:#?}\n  log_diffs={:#?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
        report.log_diffs,
    );
}
