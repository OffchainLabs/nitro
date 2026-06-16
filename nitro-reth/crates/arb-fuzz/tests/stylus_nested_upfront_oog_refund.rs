//! A Stylus upfront-abort is dimensioned only when an outer Stylus frame
//! attributes it: under a Stylus ancestor no v60 refund is due; under a
//! plain-EVM ancestor (or top level) it is undimensioned and the sender is
//! refunded. A fresh callee is reached through an EVM forwarder capped below its
//! upfront cost, directly (EVM ancestor) or via a Stylus caller; all must match
//! the reference.
//!
//! Run with:
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     NITRO_REF_IMAGE=offchainlabs/nitro-node:v3.10.1-d7f07be \
//!     cargo test -p arb-fuzz --test stylus_nested_upfront_oog_refund \
//!     --release -- --ignored --nocapture

use std::sync::{
    atomic::{AtomicU64, Ordering},
    Mutex,
};

static SERIAL: Mutex<()> = Mutex::new(());

use alloy_primitives::{address, keccak256, Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::interop::{sol_caller_calldata, WhichProgram},
    scaffolding::selector4,
};
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

const L2_CHAIN_ID: u64 = 412_348;
const L1_CHAIN_ID: u64 = 11_155_111;
const ARBOS_VERSION: u64 = 60;

const ARBWASM: Address = Address::new([
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x71,
]);
const ARBOWNER: Address = Address::new([
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x70,
]);
const FUNDER: Address = Address::new([0xa1; 20]);
const SEQUENCER_ALIAS: Address = address!("a4b000000000000000000073657175656e636572");

/// Tight forwarded cap, below the program's init cost: the nested frame aborts
/// at its upfront gate and burns the whole cap before running.
const TIGHT_CAP: u16 = 2_500;

fn owner_key() -> B256 {
    B256::repeat_byte(0x42)
}

/// A non-owner sender, so a network→sender refund moves observable balance (the
/// owner is the network fee account, where an owner-sent tx's refund nets out).
fn user_key() -> B256 {
    B256::repeat_byte(0x77)
}

fn word(v: u64) -> [u8; 32] {
    let mut w = [0u8; 32];
    w[24..].copy_from_slice(&v.to_be_bytes());
    w
}

fn create_address(deployer: Address, nonce: u64) -> Address {
    let mut rlp = Vec::new();
    rlp.push(0xd6);
    rlp.push(0x94);
    rlp.extend_from_slice(deployer.as_slice());
    if nonce == 0 {
        rlp.push(0x80);
    } else if nonce < 0x80 {
        rlp.push(nonce as u8);
    } else {
        let b = nonce.to_be_bytes();
        let start = b.iter().position(|&x| x != 0).unwrap_or(7);
        let trimmed = &b[start..];
        rlp.push(0x80 + trimmed.len() as u8);
        rlp.extend_from_slice(trimmed);
        rlp[0] = 0xd6 + (trimmed.len() as u8);
    }
    Address::from_slice(&keccak256(&rlp)[12..])
}

/// Runtime that copies its calldata to memory, CALLs `target` with `gas_cap`
/// and a zero value, then STOPs regardless of the inner result.
fn forwarder_runtime(target: Address, gas_cap: u16) -> Vec<u8> {
    let mut c = Vec::new();
    c.extend_from_slice(&[0x36, 0x60, 0x00, 0x60, 0x00, 0x37]); // CALLDATACOPY(0,0,CALLDATASIZE)
    c.extend_from_slice(&[0x60, 0x00]); // retLen
    c.extend_from_slice(&[0x60, 0x00]); // retOff
    c.push(0x36); // argLen = CALLDATASIZE
    c.extend_from_slice(&[0x60, 0x00]); // argOff
    c.extend_from_slice(&[0x60, 0x00]); // value
    c.push(0x73); // PUSH20 addr
    c.extend_from_slice(target.as_slice());
    c.push(0x61); // PUSH2 gas_cap
    c.extend_from_slice(&gas_cap.to_be_bytes());
    c.push(0xf1); // CALL
    c.push(0x50); // POP
    c.push(0x00); // STOP
    c
}

/// Constructor that returns `runtime` (CODECOPY src offset 0x0e). The returned
/// `0xEF`-prefixed Stylus blob is stored verbatim, matching a real deployment.
fn wrap_init(runtime: &[u8]) -> Vec<u8> {
    let len = runtime.len();
    let mut c = vec![0x61, (len >> 8) as u8, len as u8];
    c.extend_from_slice(&[
        0x60,
        0x0e,
        0x60,
        0x00,
        0x39,
        0x61,
        (len >> 8) as u8,
        len as u8,
    ]);
    c.extend_from_slice(&[0x60, 0x00, 0xf3]);
    c.extend_from_slice(runtime);
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

struct Idx(AtomicU64);
impl Idx {
    fn new() -> Self {
        Self(AtomicU64::new(1))
    }
    fn next(&self) -> u64 {
        self.0.fetch_add(1, Ordering::SeqCst)
    }
}

fn owner_tx(
    nonce: u64,
    to: Option<Address>,
    data: Vec<u8>,
    gas: u64,
    ts: u64,
) -> SignedL2TxBuilder {
    SignedL2TxBuilder {
        chain_id: L2_CHAIN_ID,
        nonce,
        to,
        value: U256::ZERO,
        data: Bytes::from(data),
        gas_limit: gas,
        gas_price: 1_000_000_000,
        max_fee_per_gas: 1_000_000_000,
        max_priority_fee_per_gas: 0,
        access_list: Vec::new(),
        authorization_list: Vec::new(),
        kind: L2TxKind::Eip1559,
        signing_key: owner_key(),
        l1_block_number: 1,
        timestamp: ts,
        request_id: None,
        sender: SEQUENCER_ALIAS,
        base_fee_l1: 0,
    }
}

fn user_tx(nonce: u64, to: Option<Address>, data: Vec<u8>, gas: u64, ts: u64) -> SignedL2TxBuilder {
    let mut b = owner_tx(nonce, to, data, gas, ts);
    b.signing_key = user_key();
    b
}

fn msg_step(idx: u64, msg: arb_test_harness::messaging::L1Message, dmr: u64) -> ScenarioStep {
    ScenarioStep::Message {
        idx,
        message: msg,
        delayed_messages_read: dmr,
    }
}

/// Fund owner, set L1 price 0, deploy + activate the callee program, deploy the
/// forwarder, deploy + activate a Stylus caller, fund the user. Returns (steps,
/// program_addr, forwarder_addr, stylus_caller_addr).
fn build_prefix(
    owner: Address,
    idx: &Idx,
    gas_cap: u16,
) -> (Vec<ScenarioStep>, Address, Address, Address) {
    let mut steps = Vec::new();

    let dep_idx = idx.next();
    let dep = DepositBuilder {
        from: FUNDER,
        to: owner,
        amount: U256::from(10u128).pow(U256::from(21u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: dep_idx,
        base_fee_l1: 0,
    }
    .build()
    .expect("deposit");
    steps.push(msg_step(dep_idx, dep, 1));

    // L1 price 0 → receipts reflect pure L2 gas (no poster cost).
    let mut set_price = selector4("setL1PricePerUnit(uint256)").to_vec();
    set_price.extend_from_slice(&word(0));
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx(0, Some(ARBOWNER), set_price, 2_000_000, 1_700_000_000)
            .build()
            .expect("set price"),
        1,
    ));

    // Deploy the program at owner nonce 1.
    let program_addr = create_address(owner, 1);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx(
            1,
            None,
            WhichProgram::StorageStress.initcode(),
            500_000_000,
            1_700_000_002,
        )
        .build()
        .expect("deploy stylus"),
        1,
    ));

    // activateProgram(program_addr).
    let mut act = selector4("activateProgram(address)").to_vec();
    let mut arg = [0u8; 32];
    arg[12..].copy_from_slice(program_addr.as_slice());
    act.extend_from_slice(&arg);
    let mut activate = owner_tx(2, Some(ARBWASM), act, 200_000_000, 1_700_000_010);
    activate.value = U256::from(10u128).pow(U256::from(18u64));
    let i = idx.next();
    steps.push(msg_step(i, activate.build().expect("activate"), 1));

    // Deploy the forwarder at owner nonce 3. It CALLs the callee with the tight
    // cap regardless of who invokes it.
    let fwd_addr = create_address(owner, 3);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx(
            3,
            None,
            wrap_init(&forwarder_runtime(program_addr, gas_cap)),
            5_000_000,
            1_700_000_020,
        )
        .build()
        .expect("deploy forwarder"),
        1,
    ));

    // Deploy a Stylus caller at owner nonce 4 and activate it (nonce 5), so a
    // run can route the forwarder call through a Stylus ancestor frame.
    let caller_addr = create_address(owner, 4);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx(
            4,
            None,
            WhichProgram::SolCaller.initcode(),
            500_000_000,
            1_700_000_022,
        )
        .build()
        .expect("deploy caller"),
        1,
    ));
    let mut act2 = selector4("activateProgram(address)").to_vec();
    let mut arg2 = [0u8; 32];
    arg2[12..].copy_from_slice(caller_addr.as_slice());
    act2.extend_from_slice(&arg2);
    let mut activate2 = owner_tx(5, Some(ARBWASM), act2, 200_000_000, 1_700_000_024);
    activate2.value = U256::from(10u128).pow(U256::from(18u64));
    let i = idx.next();
    steps.push(msg_step(i, activate2.build().expect("activate caller"), 1));

    // Fund the non-owner sender of the invocation.
    let user_dep_idx = idx.next();
    let user_dep = DepositBuilder {
        from: FUNDER,
        to: derive_address(user_key()),
        amount: U256::from(10u128).pow(U256::from(20u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_026,
        request_seq: user_dep_idx,
        base_fee_l1: 0,
    }
    .build()
    .expect("user deposit");
    steps.push(msg_step(user_dep_idx, user_dep, 1));

    (steps, program_addr, fwd_addr, caller_addr)
}

#[derive(Clone, Copy, PartialEq)]
enum Parent {
    /// The forwarder is invoked directly: the callee's nearest ancestor is the
    /// plain-EVM forwarder, so its upfront-abort gas is undimensioned and the
    /// sender earns a refund.
    Evm,
    /// A Stylus caller invokes the forwarder, so the callee has a Stylus
    /// ancestor that attributes the abort gas; no refund is due.
    Stylus,
}

fn run(gas_cap: u16, parent: Parent) -> arb_test_harness::dual_exec::DiffReport {
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);
    let idx = Idx::new();

    let (mut steps, _program_addr, fwd_addr, caller_addr) = build_prefix(owner, &idx, gas_cap);

    // The non-owner user invokes the forwarder (directly, or through the Stylus
    // caller), which CALLs the callee with the tight cap, then STOPs.
    let user = derive_address(user_key());
    let (to, data) = match parent {
        Parent::Evm => {
            let mut inner = selector4("mint(address,uint256)").to_vec();
            inner.extend_from_slice(&[0u8; 64]);
            (fwd_addr, inner)
        }
        Parent::Stylus => (caller_addr, sol_caller_calldata(0, 1, Some(fwd_addr))),
    };
    let i = idx.next();
    steps.push(msg_step(
        i,
        user_tx(0, Some(to), data, 30_000_000, 1_700_000_030)
            .build()
            .expect("invoke"),
        1,
    ));

    // Trailing no-op deposit to seal the block before querying.
    let seal_idx = idx.next();
    let seal = DepositBuilder {
        from: FUNDER,
        to: owner,
        amount: U256::from(1u64),
        l1_block_number: 1,
        timestamp: 1_700_000_040,
        request_seq: seal_idx,
        base_fee_l1: 0,
    }
    .build()
    .expect("seal deposit");
    steps.push(msg_step(seal_idx, seal, 1));

    let scenario = Scenario {
        name: "stylus_nested_upfront_oog".into(),
        description: "nested Stylus upfront-cost abort refund attribution".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let checks = vec![StateCheck {
        address: user,
        slots: Vec::new(),
        check_balance: true,
        check_nonce: false,
        check_code: false,
    }];

    rig.dual
        .run_with_state_checks(&scenario, &checks)
        .expect("dual run")
}

/// A nested upfront-abort under a Stylus ancestor: the reference attributes the
/// abort gas to the ancestor's computation and gives no refund, so the sender's
/// net charge (and the block state root) must match.
#[test]
#[ignore]
fn nested_upfront_oog_under_stylus_matches_reference() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let report = run(TIGHT_CAP, Parent::Stylus);
    assert!(
        report.is_clean(),
        "nested upfront-OOG under a Stylus ancestor must match the reference: block={:?} tx={:?} state={:?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}

/// A nested upfront-abort under a plain-EVM ancestor: the abort gas is
/// undimensioned and the sender earns a refund on both nodes (the fix must not
/// suppress it).
#[test]
#[ignore]
fn nested_upfront_oog_under_evm_matches_reference() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let report = run(TIGHT_CAP, Parent::Evm);
    assert!(
        report.is_clean(),
        "nested upfront-OOG under a plain-EVM ancestor must match the reference: block={:?} tx={:?} state={:?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}

/// A generous forwarded cap lets the program run to completion (no upfront
/// abort); every observable agrees across nodes.
#[test]
#[ignore]
fn program_runs_control_clean() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let report = run(u16::MAX, Parent::Evm);
    assert!(
        report.is_clean(),
        "control (program runs) must be clean: block={:?} tx={:?} state={:?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}
