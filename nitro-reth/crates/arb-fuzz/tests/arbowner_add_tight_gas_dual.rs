//! Differential test: ArbOwner.addWasmCacheManager at a tight forwarded budget.
//!
//! `AddressSet::Add` reads membership, size, and the increment's count (three
//! reads) and performs three set-writes, returning nothing; an access-controlled
//! body that overruns its forwarded gas reverts and bills the owner zero. A
//! caller forwarding a budget right at that boundary must add the manager — and
//! charge gas — identically on arbreth and the Nitro reference.
//!
//! Run:
//!   ARB_SPEC_BINARY=$(pwd)/target/fastdev/arb-reth \
//!     cargo test -p arb-fuzz --test arbowner_add_tight_gas_dual --profile fastdev \
//!     -- --ignored --nocapture

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

const ARBOWNER: Address = address!("0000000000000000000000000000000000000070");
const FUNDER: Address = Address::new([0xa1; 20]);
const MANAGER: Address = Address::new([0xbb; 20]);

// Forwarded budget inside `[arbreth body = 62406, nitro body = 63203)`.
const TIGHT_CALL_GAS: u16 = 62_800;

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

/// Constructor that returns `runtime`.
fn wrap_init(runtime: &[u8]) -> Vec<u8> {
    let len = runtime.len();
    let mut c = Vec::new();
    // 14-byte prefix: PUSH2 len; PUSH1 0x0e; PUSH1 0x00; CODECOPY; PUSH2 len;
    // PUSH1 0x00; RETURN — runtime begins at offset 0x0e.
    c.extend_from_slice(&[0x61, (len >> 8) as u8, len as u8]);
    c.extend_from_slice(&[0x60, 0x0e, 0x60, 0x00, 0x39]);
    c.extend_from_slice(&[0x61, (len >> 8) as u8, len as u8]);
    c.extend_from_slice(&[0x60, 0x00, 0xf3]);
    c.extend_from_slice(runtime);
    c
}

/// Runtime that `CALL`s 0x70 with `addWasmCacheManager(MANAGER)` forwarding
/// exactly `gas`, then stores the call's success flag at slot 0.
fn caller_runtime(manager: Address, gas: u16) -> Vec<u8> {
    let sel = selector4("addWasmCacheManager(address)");
    let mut c = Vec::new();
    c.push(0x63);
    c.extend_from_slice(&sel);
    c.extend_from_slice(&[0x60, 0xE0, 0x1b, 0x60, 0x00, 0x52]); // PUSH1 224 SHL PUSH1 0 MSTORE
    c.push(0x73);
    c.extend_from_slice(manager.as_slice());
    c.extend_from_slice(&[0x60, 0x04, 0x52]); // PUSH1 4 MSTORE
                                              // CALL operands (pushed reverse): retLen retOff argLen argOff value addr gas
    c.extend_from_slice(&[
        0x60, 0x00, 0x60, 0x00, 0x60, 0x24, 0x60, 0x00, 0x60, 0x00, 0x60, 0x70,
    ]);
    c.push(0x61);
    c.extend_from_slice(&gas.to_be_bytes()); // PUSH2 gas
    c.extend_from_slice(&[0xf1, 0x60, 0x00, 0x55, 0x00]); // CALL PUSH1 0 SSTORE STOP
    c
}

fn add_chain_owner(addr: Address) -> Vec<u8> {
    let mut d = selector4("addChainOwner(address)").to_vec();
    d.extend_from_slice(B256::left_padding_from(addr.as_slice()).as_slice());
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

#[test]
#[ignore]
fn add_wasm_cache_manager_tight_gas_matches_nitro() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let caller = create_address(owner, 0);
    let mut rig = Rig::spawn(owner);

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

    // Deploy the caller (owner nonce 0).
    let i = next();
    steps.push(msg_step(
        i,
        tx(
            0,
            None,
            wrap_init(&caller_runtime(MANAGER, TIGHT_CALL_GAS)),
            3_000_000,
            BASE_TS + i * BLOCK_SECS,
        )
        .build()
        .expect("deploy caller"),
        1,
    ));

    // Make the caller a chain owner (owner nonce 1, ample gas → both succeed).
    let i = next();
    steps.push(msg_step(
        i,
        tx(
            1,
            Some(ARBOWNER),
            add_chain_owner(caller),
            3_000_000,
            BASE_TS + i * BLOCK_SECS,
        )
        .build()
        .expect("addChainOwner"),
        1,
    ));

    // Invoke the caller (owner nonce 2): it forwards the tight budget to
    // addWasmCacheManager and records the call's success flag at slot 0.
    let i = next();
    steps.push(msg_step(
        i,
        tx(
            2,
            Some(caller),
            Vec::new(),
            3_000_000,
            BASE_TS + i * BLOCK_SECS,
        )
        .build()
        .expect("invoke caller"),
        1,
    ));

    // Trailing no-op deposit so the invoke's block is sealed before we query.
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
        name: "addWasmCacheManager_tight_gas".into(),
        description: "owner-method add at a budget between arbreth and nitro body cost".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let checks = [StateCheck {
        address: caller,
        slots: vec![B256::ZERO],
        check_balance: false,
        check_nonce: false,
        check_code: false,
    }];

    let report = rig
        .dual
        .run_with_state_checks(&scenario, &checks)
        .expect("dual run");

    assert!(
        report.is_clean(),
        "arbreth diverged from Nitro on the tight-gas add\n  block_diffs={:#?}\n  tx_diffs={:#?}\n  state_diffs={:#?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}

// Forwarded budget inside the window where the setter's StylusParams read being
// a warm access (100) rather than a cold SLOAD (800) decides the outcome: the
// reference node sets the param while a node charging the cold cost out-of-gases.
fn setter_tight_gas() -> u16 {
    std::env::var("ARB_SETTER_GAS")
        .ok()
        .and_then(|s| s.parse().ok())
        .unwrap_or(21_000)
}

/// Runtime that `CALL`s 0x70 with `setInkPrice(0x1234)` forwarding exactly
/// `gas`, then stores the call's success flag at slot 0.
fn set_ink_price_runtime(gas: u16) -> Vec<u8> {
    let sel = selector4("setInkPrice(uint32)");
    let mut c = Vec::new();
    c.push(0x63);
    c.extend_from_slice(&sel);
    c.extend_from_slice(&[0x60, 0xE0, 0x1b, 0x60, 0x00, 0x52]); // PUSH1 224 SHL PUSH1 0 MSTORE
    c.extend_from_slice(&[0x61, 0x12, 0x34]); // PUSH2 0x1234 (the uint32 value)
    c.extend_from_slice(&[0x60, 0x04, 0x52]); // PUSH1 4 MSTORE
                                              // CALL operands (reverse): retLen retOff argLen argOff value addr gas
    c.extend_from_slice(&[
        0x60, 0x00, 0x60, 0x00, 0x60, 0x24, 0x60, 0x00, 0x60, 0x00, 0x60, 0x70,
    ]);
    c.push(0x61);
    c.extend_from_slice(&gas.to_be_bytes()); // PUSH2 gas
    c.extend_from_slice(&[0xf1, 0x60, 0x00, 0x55, 0x00]); // CALL PUSH1 0 SSTORE STOP
    c
}

#[test]
#[ignore]
fn set_ink_price_tight_gas_matches_nitro() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let caller = create_address(owner, 0);
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
            wrap_init(&set_ink_price_runtime(setter_tight_gas())),
            3_000_000,
            BASE_TS + i * BLOCK_SECS,
        )
        .build()
        .expect("deploy caller"),
        1,
    ));

    let i = next();
    steps.push(msg_step(
        i,
        tx(
            1,
            Some(ARBOWNER),
            add_chain_owner(caller),
            3_000_000,
            BASE_TS + i * BLOCK_SECS,
        )
        .build()
        .expect("addChainOwner"),
        1,
    ));

    let i = next();
    steps.push(msg_step(
        i,
        tx(
            2,
            Some(caller),
            Vec::new(),
            3_000_000,
            BASE_TS + i * BLOCK_SECS,
        )
        .build()
        .expect("invoke caller"),
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
        name: "set_ink_price_tight_gas".into(),
        description: "owner setter at a budget between arbreth and nitro body cost".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let checks = [StateCheck {
        address: caller,
        slots: vec![B256::ZERO],
        check_balance: false,
        check_nonce: false,
        check_code: false,
    }];

    let report = rig
        .dual
        .run_with_state_checks(&scenario, &checks)
        .expect("dual run");

    assert!(
        report.is_clean(),
        "arbreth diverged from Nitro on the tight-gas setter\n  block_diffs={:#?}\n  tx_diffs={:#?}\n  state_diffs={:#?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}
