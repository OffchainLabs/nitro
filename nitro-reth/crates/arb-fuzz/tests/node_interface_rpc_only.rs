//! NodeInterface (0xc8) / NodeInterfaceDebug (0xc9) must be RPC-only, not
//! consensus precompiles.
//!
//! 0xc8/0xc9 are wired only into the RPC-only message interceptor, never into
//! the consensus precompile set, so a committed on-chain `CALL` to 0xc8 must
//! behave like a `CALL` to an empty account (success=1, returndatasize=0, plain
//! CALL cost) rather than dispatching to a precompile that returns a 32-byte
//! word and charges SLOAD+COPY gas.
//!
//! A forwarder CALLs 0xc8 with the valid `nitroGenesisBlock()` selector and
//! records `success`→slot0, `returndatasize`→slot1; the empty-account behavior
//! writes 0 to slot1 (independent of the genesis block value). The same
//! forwarder against 0xc7 (a precompile in NEITHER node) is the control: it too
//! hits an empty account, so slot1=0 and the report is clean.

use std::sync::Mutex;

use alloy_primitives::{address, keccak256, Address, Bytes, B256, U256};
use arb_test_harness::{
    dual_exec::{DiffReport, DualExec, StateField},
    genesis::GenesisBuilder,
    messaging::{
        signed_tx::{derive_address, L2TxKind, SignedL2TxBuilder},
        DepositBuilder, MessageBuilder,
    },
    mock_l1::MockL1,
    node::{
        arbreth::ArbrethProcess, nitro_docker::NitroDocker, BlockId, ExecutionNode, NodeStartCtx,
        TxRequest,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep, StateCheck},
};

static SERIAL: Mutex<()> = Mutex::new(());

const L2_CHAIN_ID: u64 = 412_349;
const L1_CHAIN_ID: u64 = 11_155_111;
const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const ARBOS_VERSION: u64 = 60;
const BASE_TS: u64 = 1_700_000_000;
const BLOCK_SECS: u64 = 12;

const FUNDER: Address = Address::new([0xa1; 20]);

/// NodeInterface — RPC-only, never a consensus precompile.
const NODE_INTERFACE: u8 = 0xc8;
/// A precompile on NEITHER node — the control target.
const UNREGISTERED: u8 = 0xc7;

/// `nitroGenesisBlock()` — keccak selector, returns uint64.
const NITRO_GENESIS_BLOCK_SEL: [u8; 4] = [0x93, 0xa2, 0xfe, 0x21];

fn owner_key() -> B256 {
    B256::repeat_byte(0x42)
}

fn target_addr(byte: u8) -> Address {
    let mut a = [0u8; 20];
    a[19] = byte;
    Address::from(a)
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
        base_fee_l1: 0,
    }
}

fn msg_step(idx: u64, msg: arb_test_harness::messaging::L1Message, dmr: u64) -> ScenarioStep {
    ScenarioStep::Message {
        idx,
        message: msg,
        delayed_messages_read: dmr,
    }
}

/// Constructor returning `runtime` (runtime begins at offset 0x0e).
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

/// Runtime that `CALL`s `addr` forwarding all gas with `selector ++ args`, then
/// stores the call's success flag at slot 0 and `RETURNDATASIZE` at slot 1.
/// `args` must be a whole number of 32-byte words; total calldata < 256 bytes.
fn call_forwarder(addr: Address, selector: [u8; 4], args: &[u8]) -> Vec<u8> {
    debug_assert!(args.len().is_multiple_of(32) && 4 + args.len() < 256);
    let mut c = Vec::new();
    // PUSH4 selector; PUSH1 0xE0 SHL; PUSH1 0 MSTORE — selector left-aligned at mem[0..4].
    c.push(0x63);
    c.extend_from_slice(&selector);
    c.extend_from_slice(&[0x60, 0xE0, 0x1b, 0x60, 0x00, 0x52]);
    for (i, word) in args.chunks(32).enumerate() {
        c.push(0x7f); // PUSH32
        c.extend_from_slice(word);
        c.extend_from_slice(&[0x60, (4 + 32 * i) as u8, 0x52]); // PUSH1 off MSTORE
    }
    let arg_len = (4 + args.len()) as u8;
    // CALL operands (pushed reverse): retLen=0 retOff=0 argLen argOff=0 value=0 addr gas=GAS.
    c.extend_from_slice(&[
        0x60, 0x00, 0x60, 0x00, 0x60, arg_len, 0x60, 0x00, 0x60, 0x00,
    ]);
    c.push(0x73); // PUSH20 addr
    c.extend_from_slice(addr.as_slice());
    c.push(0x5a); // GAS — forward everything, never a tight inner budget
    c.extend_from_slice(&[0xf1]); // CALL
    c.extend_from_slice(&[0x60, 0x00, 0x55]); // PUSH1 0 SSTORE  (success → slot0)
    c.extend_from_slice(&[0x3d, 0x60, 0x01, 0x55]); // RETURNDATASIZE PUSH1 1 SSTORE  (→ slot1)
    c.push(0x00); // STOP
    c
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

/// Deploys a forwarder whose runtime CALLs `target` with `selector ++ args`,
/// invokes it, seals the block, and returns the cross-node diff report over the
/// forwarder's slots [0, 1] and the block/tx receipts.
fn run(target: Address, selector: [u8; 4], args: &[u8], name: &str) -> DiffReport {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let forwarder = create_address(owner, 0);
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
            None,
            wrap_init(&call_forwarder(target, selector, args)),
            3_000_000,
            BASE_TS + i * BLOCK_SECS,
        )
        .build()
        .expect("deploy forwarder"),
        1,
    ));

    let i = next();
    steps.push(msg_step(
        i,
        tx(
            1,
            Some(forwarder),
            Vec::new(),
            3_000_000,
            BASE_TS + i * BLOCK_SECS,
        )
        .build()
        .expect("invoke forwarder"),
        1,
    ));

    let i = next();
    steps.push(msg_step(
        i,
        DepositBuilder {
            from: FUNDER,
            to: FUNDER,
            amount: U256::from(1u64),
            l1_block_number: 1 + (i * BLOCK_SECS) / BLOCK_SECS,
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
        description: "forwarder CALLs target; record success+returndatasize".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let checks = [StateCheck {
        address: forwarder,
        slots: vec![B256::ZERO, B256::with_last_byte(1)],
        check_balance: false,
        check_nonce: false,
        check_code: false,
    }];

    rig.dual
        .run_with_state_checks(&scenario, &checks)
        .expect("dual run")
}

/// A committed forwarder CALL to 0xc8 with the valid `nitroGenesisBlock()`
/// selector must hit an empty account (returndatasize=0), so slot 1 records 0.
#[test]
#[ignore]
fn node_interface_committed_call_matches_empty_account() {
    let target = target_addr(NODE_INTERFACE);
    let forwarder = create_address(derive_address(owner_key()), 0);
    let report = run(target, NITRO_GENESIS_BLOCK_SEL, &[], "nodeinterface_0xc8");

    let slot1 = B256::with_last_byte(1);
    let storage_diff = report.state_diffs.iter().find(|d| {
        d.address == forwarder && matches!(d.field, StateField::Storage(s) if s == slot1)
    });

    eprintln!(
        "block_diffs={:#?}\ntx_diffs={:#?}\nstate_diffs={:#?}",
        report.block_diffs, report.tx_diffs, report.state_diffs,
    );

    let d = storage_diff.unwrap_or_else(|| {
        panic!(
            "expected a forwarder slot 1 (returndatasize) difference at 0xc8 \
             but it was absent\n  state_diffs={:#?}",
            report.state_diffs,
        )
    });
    let left: Option<B256> = serde_json::from_value(d.left.clone()).expect("decode left slot1");
    let right: Option<B256> = serde_json::from_value(d.right.clone()).expect("decode right slot1");
    assert_eq!(
        left,
        Some(B256::ZERO),
        "left returndatasize at slot 1 should be 0 (empty account)",
    );
    assert_eq!(
        right,
        Some(B256::from(U256::from(32u64))),
        "right returndatasize at slot 1 should be 32 (precompile returns a word)",
    );
    assert!(
        !report.is_clean(),
        "report must not be clean given the slot-1 difference",
    );
}

/// Control: target 0xc7 (a precompile on neither node). The CALL hits an empty
/// account → success=1, returndatasize=0, identical cost, and the report agrees
/// across nodes.
#[test]
#[ignore]
fn non_precompile_address_control_clean() {
    let target = target_addr(UNREGISTERED);
    let report = run(target, NITRO_GENESIS_BLOCK_SEL, &[], "control_0xc7");

    assert!(
        report.is_clean(),
        "control (0xc7) must be clean — both nodes CALL an empty account\n  \
         block_diffs={:#?}\n  tx_diffs={:#?}\n  state_diffs={:#?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}

/// A committed CALL to 0xc8 must behave exactly like a CALL to an empty account
/// on both nodes — identical success / returndatasize, receipt gas, and block
/// state.
#[test]
#[ignore]
fn nodeinterface_rpc_only_clean() {
    let target = target_addr(NODE_INTERFACE);
    let report = run(
        target,
        NITRO_GENESIS_BLOCK_SEL,
        &[],
        "nodeinterface_rpc_only",
    );

    assert!(
        report.is_clean(),
        "committed CALL to 0xc8 must match an empty-account CALL on both nodes\n  \
         block_diffs={:#?}\n  tx_diffs={:#?}\n  state_diffs={:#?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}

/// `eth_call` to 0xc8 must return NodeInterface data on both nodes — the RPC
/// layer services the selector directly rather than falling through to an empty
/// account.
#[test]
#[ignore]
fn nodeinterface_eth_call_matches_reference() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let rig = Rig::spawn(owner);

    let req = |node: &dyn ExecutionNode| {
        node.eth_call(
            TxRequest {
                from: Some(owner),
                to: Some(target_addr(NODE_INTERFACE)),
                data: Some(Bytes::from(NITRO_GENESIS_BLOCK_SEL.to_vec())),
                value: Some(U256::ZERO),
                gas: Some(3_000_000),
            },
            BlockId::Latest,
        )
        .expect("eth_call nitroGenesisBlock")
    };

    let nitro = req(&rig.dual.left);
    let arbreth = req(&rig.dual.right);
    assert_eq!(
        arbreth, nitro,
        "eth_call(0xc8.nitroGenesisBlock) must match the reference"
    );
    assert_eq!(
        arbreth.len(),
        32,
        "the RPC layer must service the selector (32-byte word), not return an empty account"
    );
}
