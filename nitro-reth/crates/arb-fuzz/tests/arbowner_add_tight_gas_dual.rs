//! Differential tests: ArbOwner methods at a tight forwarded budget.
//!
//! Owner methods bill the caller zero but revert if their computed cost
//! overruns the forwarded gas, so the cost must match the reference exactly at
//! that boundary. Storage writes are priced by value — clearing a slot to zero
//! costs less than setting it — so a method that writes a caller-supplied zero
//! (a reset fee account, an immediate upgrade timestamp) must charge the reset
//! price, not the set price, or it out-of-gases where the reference succeeds.

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

// Forwarded budget tight enough to bracket the method's body cost.
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

/// Runtime that `CALL`s 0x70 with `selector ++ args` forwarding exactly `gas`,
/// then stores the call's success flag at slot 0. `args` must be a whole number
/// of 32-byte words; the total calldata length stays below 256 bytes.
fn forward_runtime(selector: [u8; 4], args: &[u8], gas: u16) -> Vec<u8> {
    debug_assert!(args.len().is_multiple_of(32) && 4 + args.len() < 256);
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
        0x60, 0x00, 0x60, 0x00, 0x60, arg_len, 0x60, 0x00, 0x60, 0x00, 0x60, 0x70,
    ]);
    c.push(0x61);
    c.extend_from_slice(&gas.to_be_bytes()); // PUSH2 gas
    c.extend_from_slice(&[0xf1, 0x60, 0x00, 0x55, 0x00]); // CALL PUSH1 0 SSTORE STOP
    c
}

fn word(bytes: &[u8]) -> [u8; 32] {
    B256::left_padding_from(bytes).0
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

/// Deploys a forwarder whose runtime is `caller_runtime`, makes it a chain owner
/// (the method under test is access-controlled), invokes it so it forwards its
/// tight budget to ArbOwner, and asserts both nodes agree on the block, the
/// receipts, and the forwarder's recorded success flag.
fn run_forwarder(caller_runtime: Vec<u8>, name: &str) {
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
            wrap_init(&caller_runtime),
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
        name: name.into(),
        description: "owner method at a budget between the two nodes' body cost".into(),
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
        "differs from the reference on {name}\n  block_diffs={:#?}\n  tx_diffs={:#?}\n  state_diffs={:#?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}

fn env_gas(var: &str, default: u16) -> u16 {
    std::env::var(var)
        .ok()
        .and_then(|s| s.parse().ok())
        .unwrap_or(default)
}

#[test]
#[ignore]
fn add_wasm_cache_manager_tight_gas_matches_reference() {
    let sel = selector4("addWasmCacheManager(address)");
    let runtime = forward_runtime(sel, &word(MANAGER.as_slice()), TIGHT_CALL_GAS);
    run_forwarder(runtime, "addWasmCacheManager_tight_gas");
}

#[test]
#[ignore]
fn set_ink_price_tight_gas_matches_reference() {
    let sel = selector4("setInkPrice(uint32)");
    let gas = env_gas("ARB_SETTER_GAS", 21_000);
    let runtime = forward_runtime(sel, &word(&[0x12, 0x34]), gas);
    run_forwarder(runtime, "set_ink_price_tight_gas");
}

/// `scheduleArbOSUpgrade(50, 0)`: the zero timestamp is the value-dependent
/// reset write under test.
#[test]
#[ignore]
fn schedule_upgrade_zero_timestamp_tight_gas_matches_reference() {
    let sel = selector4("scheduleArbOSUpgrade(uint64,uint64)");
    let gas = env_gas("ARB_SCHEDULE_GAS", 30_000);
    let mut args = word(&[50]).to_vec();
    args.extend_from_slice(&[0u8; 32]); // timestamp = 0
    let runtime = forward_runtime(sel, &args, gas);
    run_forwarder(runtime, "schedule_upgrade_tight_gas");
}

/// `setNetworkFeeAccount(0)`: clearing the network fee account is a zero-value
/// write to an address slot, charged the reset price.
#[test]
#[ignore]
fn set_network_fee_account_zero_tight_gas_matches_reference() {
    let sel = selector4("setNetworkFeeAccount(address)");
    let gas = env_gas("ARB_SETTER0_GAS", 12_000);
    let runtime = forward_runtime(sel, &[0u8; 32], gas);
    run_forwarder(runtime, "set_network_fee_account_zero_tight_gas");
}

/// `setL2BaseFee(0)`: a zero-value write to a uint256 pricing slot.
#[test]
#[ignore]
fn set_l2_base_fee_zero_tight_gas_matches_reference() {
    let sel = selector4("setL2BaseFee(uint256)");
    let gas = env_gas("ARB_SETTER0_GAS", 12_000);
    let runtime = forward_runtime(sel, &[0u8; 32], gas);
    run_forwarder(runtime, "set_l2_base_fee_zero_tight_gas");
}

/// `setCollectTips(false)`: writes 0 to the collect-tips slot, charged the reset
/// price, so a tight budget that covers the reset write but not the set write
/// must still succeed.
#[test]
#[ignore]
fn set_collect_tips_false_tight_gas_matches_reference() {
    let sel = selector4("setCollectTips(bool)");
    let gas = env_gas("ARB_SETTER0_GAS", 12_000);
    let runtime = forward_runtime(sel, &[0u8; 32], gas);
    run_forwarder(runtime, "set_collect_tips_false_tight_gas");
}
