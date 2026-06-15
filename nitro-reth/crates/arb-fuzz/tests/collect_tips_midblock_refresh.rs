//! Mid-block `setCollectTips` must take effect for later transactions in the
//! same block. `collect_tips` is refreshed per transaction (v60+), so a tx that
//! follows `setCollectTips(true)` in the same block prices its tip at the full
//! gas price.
//!
//! The two transactions are delivered in one L2 batch message so they land in a
//! single block. The second transaction records `GASPRICE` into slot 0: with
//! tips collected it is the full gas price, otherwise the base fee.
//!
//! Control: an identical batch whose first transaction calls
//! `setCollectTips(false)` (a no-op — the genesis default is already false), so
//! both nodes price the tip at base fee and agree.

use std::sync::Mutex;

use alloy_primitives::{address, Address, Bytes, B256, U256};
use arb_fuzz::{arbitrary_impls::interop::create_address, scaffolding::selector4};
use arb_test_harness::{
    dual_exec::DualExec,
    genesis::GenesisBuilder,
    messaging::{
        b64_l2_msg, kinds,
        signed_tx::{derive_address, L2TxKind, SignedL2TxBuilder},
        DepositBuilder, L1Message, L1MessageHeader, MessageBuilder,
    },
    mock_l1::MockL1,
    node::{
        arbreth::ArbrethProcess, nitro_docker::NitroDocker, BlockId, ExecutionNode, NodeStartCtx,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep, StateCheck},
};

static SERIAL: Mutex<()> = Mutex::new(());

const L2_CHAIN_ID: u64 = 412_353;
const L1_CHAIN_ID: u64 = 11_155_111;
const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const ARBOS_VERSION: u64 = 60;
const BASE_TS: u64 = 1_700_000_000;
const BLOCK_SECS: u64 = 12;

const ARBOWNER: Address = address!("0000000000000000000000000000000000000070");
const FUNDER: Address = Address::new([0xa1; 20]);
const SEQUENCER_ALIAS: Address = address!("a4b000000000000000000073657175656e636572");

const MAX_FEE: u128 = 10_000_000_000;
const TIP: u128 = 1_000_000_000;

fn owner_key() -> B256 {
    B256::repeat_byte(0x42)
}

fn slot0() -> B256 {
    B256::ZERO
}

/// Runtime: `SSTORE(0, GASPRICE()); STOP`. Records the effective gas price the
/// transaction is charged, which depends on whether tips are collected.
fn gasprice_recorder_runtime() -> Vec<u8> {
    vec![0x3a, 0x60, 0x00, 0x55, 0x00]
}

fn set_collect_tips_calldata(enabled: bool) -> Vec<u8> {
    let mut d = selector4("setCollectTips(bool)").to_vec();
    let mut word = [0u8; 32];
    if enabled {
        word[31] = 1;
    }
    d.extend_from_slice(&word);
    d
}

#[allow(clippy::too_many_arguments)]
fn tx(
    nonce: u64,
    to: Option<Address>,
    data: Vec<u8>,
    value: U256,
    gas: u64,
    max_priority_fee_per_gas: u128,
    ts: u64,
) -> SignedL2TxBuilder {
    SignedL2TxBuilder {
        chain_id: L2_CHAIN_ID,
        nonce,
        to,
        value,
        data: Bytes::from(data),
        gas_limit: gas,
        gas_price: MAX_FEE,
        max_fee_per_gas: MAX_FEE,
        max_priority_fee_per_gas,
        access_list: Vec::new(),
        authorization_list: Vec::new(),
        kind: L2TxKind::Eip1559,
        signing_key: owner_key(),
        l1_block_number: 1 + (ts - BASE_TS) / BLOCK_SECS,
        timestamp: ts,
        request_id: None,
        sender: SEQUENCER_ALIAS,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
}

/// Pack several signed L2 transactions into one batch L2 message so they execute
/// in a single block. Body = `0x03 || (u64_be(len) || [0x04 || rlp(tx)])*`.
fn batch_message(subs: &[SignedL2TxBuilder], block_number: u64, timestamp: u64) -> L1Message {
    let mut body = vec![0x03u8];
    for s in subs {
        let sub = s.encode_body().expect("encode sub-message");
        body.extend_from_slice(&(sub.len() as u64).to_be_bytes());
        body.extend_from_slice(&sub);
    }
    L1Message {
        header: L1MessageHeader {
            kind: kinds::KIND_L2_MESSAGE,
            sender: SEQUENCER_ALIAS,
            block_number,
            timestamp,
            request_id: None,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        },
        l2_msg: b64_l2_msg(&Bytes::from(body)),
    }
}

fn msg_step(idx: u64, msg: L1Message, dmr: u64) -> ScenarioStep {
    ScenarioStep::Message {
        idx,
        message: msg,
        delayed_messages_read: dmr,
    }
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

/// Build the scenario. `collect` is the only knob differing between the positive
/// (true) and control (false): a batch containing `setCollectTips(collect)` then
/// a tip-bearing call to the GASPRICE recorder. Returns scenario + recorder.
fn build_scenario(name: &str, collect: bool) -> (Scenario, Address) {
    let owner = derive_address(owner_key());
    let recorder = create_address(owner, 0);

    let mut steps = Vec::new();
    let mut idx = 0u64;
    let mut next = || {
        idx += 1;
        idx
    };

    // Fund the owner.
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

    // Deploy the GASPRICE recorder at owner nonce 0.
    let i = next();
    steps.push(msg_step(
        i,
        tx(
            0,
            None,
            arb_fuzz::arbitrary_impls::interop::wrap_init_code(&gasprice_recorder_runtime()),
            U256::ZERO,
            3_000_000,
            0,
            BASE_TS + i * BLOCK_SECS,
        )
        .build()
        .expect("deploy recorder"),
        1,
    ));

    // One block (batch): setCollectTips(collect) at nonce 1, then a tip-bearing
    // call to the recorder at nonce 2.
    let i = next();
    let ts = BASE_TS + i * BLOCK_SECS;
    let set_tips = tx(
        1,
        Some(ARBOWNER),
        set_collect_tips_calldata(collect),
        U256::ZERO,
        3_000_000,
        0,
        ts,
    );
    let invoke = tx(
        2,
        Some(recorder),
        Vec::new(),
        U256::ZERO,
        3_000_000,
        TIP,
        ts,
    );
    steps.push(msg_step(
        i,
        batch_message(&[set_tips, invoke], 1 + (ts - BASE_TS) / BLOCK_SECS, ts),
        1,
    ));

    // Trailing no-op deposit seals the block before state is queried.
    let i = next();
    steps.push(msg_step(
        i,
        DepositBuilder {
            from: FUNDER,
            to: owner,
            amount: U256::from(1u64),
            l1_block_number: 2,
            timestamp: BASE_TS + i * BLOCK_SECS,
            request_seq: i,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        }
        .build()
        .expect("seal deposit"),
        2,
    ));

    (
        Scenario {
            name: name.into(),
            description: "mid-block setCollectTips refresh".into(),
            setup: ScenarioSetup {
                l2_chain_id: L2_CHAIN_ID,
                arbos_version: ARBOS_VERSION,
                genesis: None,
            },
            steps,
        },
        recorder,
    )
}

/// Assert the batch landed: the owner advanced past both batched txs and the
/// recorder stored a non-zero gas price on BOTH nodes. Without this a no-op
/// setup would make a clean report meaningless.
fn assert_setup_landed(rig: &Rig, recorder: Address, owner: Address) {
    for (node, n) in [
        ("reference", &rig.dual.left as &dyn ExecutionNode),
        ("arbreth", &rig.dual.right as &dyn ExecutionNode),
    ] {
        let nonce = n.nonce(owner, BlockId::Latest).expect("owner nonce");
        assert!(
            nonce >= 3,
            "[{node}] owner must advance past the batch, got nonce {nonce}"
        );
        let recorded = n
            .storage(recorder, slot0(), BlockId::Latest)
            .expect("recorder slot0");
        assert!(
            recorded != B256::ZERO,
            "[{node}] recorder must store a gas price"
        );
    }
}

#[test]
#[ignore]
fn collect_tips_midblock_refresh_clean() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);

    // setCollectTips(true) precedes the recorder call in the same batched block,
    // so the call must price its tip at the full gas price on both nodes.
    let (scenario, recorder) = build_scenario("collect_tips_midblock_refresh", true);
    let checks = [StateCheck {
        address: recorder,
        slots: vec![slot0()],
        check_balance: false,
        check_nonce: false,
        check_code: false,
    }];
    let report = rig
        .dual
        .run_with_state_checks(&scenario, &checks)
        .expect("dual run");

    assert_setup_landed(&rig, recorder, owner);

    assert!(
        report.is_clean(),
        "the recorder must observe the full gas price on both nodes after a same-block \
         setCollectTips(true)\n  block_diffs={:#?}\n  state_diffs={:#?}",
        report.block_diffs,
        report.state_diffs,
    );
}

#[test]
#[ignore]
fn collect_tips_midblock_control_clean() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);

    let (scenario, recorder) = build_scenario("collect_tips_midblock_control_clean", false);
    let checks = [StateCheck {
        address: recorder,
        slots: vec![slot0()],
        check_balance: false,
        check_nonce: false,
        check_code: false,
    }];
    let report = rig
        .dual
        .run_with_state_checks(&scenario, &checks)
        .expect("dual run");

    assert_setup_landed(&rig, recorder, owner);

    assert!(
        report.is_clean(),
        "setCollectTips(false) is a no-op so both nodes price the tip at base fee; \
         block_diffs={:#?} state_diffs={:#?}",
        report.block_diffs,
        report.state_diffs,
    );
}
