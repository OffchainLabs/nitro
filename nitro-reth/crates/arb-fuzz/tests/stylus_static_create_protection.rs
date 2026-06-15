//! A read-only CREATE/CREATE2 must halt the Stylus frame, not continue past it.
//!
//! When a Stylus program is entered through a STATICCALL its host runs in
//! `read_only` mode. The `create1`/`create2` EvmApi then takes the read-only
//! short-circuit and returns
//! `(CreateResponse::Fail("write protection"), 0, Gas(0), pages)`; the host
//! returns `Err` on `Fail` before `buy_gas`, halting the frame (`Failure`) so
//! the outer STATICCALL returns 0. The cost is 0 either way, so this is purely a
//! write-protection halt, not a gas-burn difference.
//!
//! A forwarder STATICCALLs `SolCaller.doCreate(uint256,bytes)` (read_only=true
//! inside the program), driving the create1 hostio, and SSTOREs the inner call's
//! success flag at slot 0: the write-protected create halts the program, so the
//! forwarder stores 0. The control is byte-for-byte identical except the
//! forwarder uses CALL (read_only=false), so the create1 hostio takes the normal
//! path — the 14-byte init code deploys its 1-byte runtime and the forwarder
//! stores 1. The two cases differ ONLY in the forwarder's call opcode (CALL
//! needs one extra zero-VALUE push that STATICCALL omits), isolating the
//! read_only create gate from the harness.

use std::sync::{
    atomic::{AtomicU64, Ordering},
    Mutex,
};

static SERIAL: Mutex<()> = Mutex::new(());

use alloy_primitives::{address, keccak256, Address, Bytes, B256, U256};
use arb_fuzz::{arbitrary_impls::interop::WhichProgram, scaffolding::selector4};
use arb_test_harness::{
    dual_exec::{DualExec, StateField},
    genesis::GenesisBuilder,
    messaging::{
        signed_tx::{derive_address, L2TxKind, SignedL2TxBuilder},
        DepositBuilder, MessageBuilder,
    },
    mock_l1::MockL1,
    node::{
        arbreth::ArbrethProcess, nitro_docker::NitroDocker, BlockId, ExecutionNode, NodeStartCtx,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep, StateCheck},
};

const L2_CHAIN_ID: u64 = 412_346;
const L1_CHAIN_ID: u64 = 11_155_111;
const ARBOS_VERSION: u64 = 60;

const ARBWASM: Address = Address::new([
    0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x71,
]);
const FUNDER: Address = Address::new([0xa1; 20]);
const SEQUENCER_ALIAS: Address = address!("a4b000000000000000000073657175656e636572");

const DEPLOY_GAS_CAP: u64 = 150_000_000;
const INVOKE_GAS_CAP: u64 = 30_000_000;

fn owner_key() -> B256 {
    B256::repeat_byte(0x42)
}

fn word(v: u64) -> [u8; 32] {
    let mut w = [0u8; 32];
    w[24..].copy_from_slice(&v.to_be_bytes());
    w
}

fn create_address(deployer: Address, nonce: u64) -> Address {
    let mut rlp = vec![0xd6u8, 0x94];
    rlp.extend_from_slice(deployer.as_slice());
    rlp.push(if nonce == 0 { 0x80 } else { nonce as u8 });
    Address::from_slice(&keccak256(&rlp)[12..])
}

/// Init code that returns a 1-byte runtime (SLOAD slot 0, POP, return 1 byte).
/// Deterministic + tiny so the inner deploy is byte-identical on both nodes in
/// the control; under read-only create the program halts before reaching it.
fn ctor_sload_only() -> Vec<u8> {
    vec![
        0x60, 0x00, 0x54, 0x50, 0x60, 0x00, 0x60, 0x00, 0x53, 0x60, 0x01, 0x60, 0x00, 0xf3,
    ]
}

/// `doCreate(uint256 endowment, bytes init_code)` calldata driving the Stylus
/// create1 hostio with a zero endowment.
fn do_create_calldata(init_code: &[u8]) -> Vec<u8> {
    let mut out = selector4("doCreate(uint256,bytes)").to_vec();
    out.extend_from_slice(&word(0)); // endowment = 0
    out.extend_from_slice(&word(0x40)); // bytes offset
    out.extend_from_slice(&word(init_code.len() as u64)); // bytes length
    out.extend_from_slice(init_code);
    while !out.len().is_multiple_of(32) {
        out.push(0);
    }
    out
}

/// STATICCALL forwarder runtime: copies its calldata to memory, STATICCALLs
/// `target` with it, and SSTOREs the call's success flag at slot 0.
///
/// STATICCALL pops: gas, addr, argOff, argLen, retOff, retLen (no value).
fn static_forwarder(target: Address) -> Vec<u8> {
    let mut c = Vec::new();
    // CALLDATACOPY(dest=0, off=0, size=CALLDATASIZE)
    c.extend_from_slice(&[0x36, 0x60, 0x00, 0x60, 0x00, 0x37]);
    // push (in reverse) retLen=0, retOff=0, argLen=CALLDATASIZE, argOff=0
    c.extend_from_slice(&[0x60, 0x00]); // retLen
    c.extend_from_slice(&[0x60, 0x00]); // retOff
    c.push(0x36); // argLen = CALLDATASIZE
    c.extend_from_slice(&[0x60, 0x00]); // argOff
                                        // addr
    c.push(0x73);
    c.extend_from_slice(target.as_slice());
    c.push(0x5a); // GAS
    c.push(0xfa); // STATICCALL
    c.extend_from_slice(&[0x60, 0x00, 0x55]); // PUSH1 0 SSTORE (success flag)
    c.push(0x00); // STOP
    c
}

/// CALL forwarder runtime: identical to `static_forwarder` but uses CALL, which
/// needs an extra zero-VALUE push that STATICCALL omits (ABI-mandated,
/// consensus-neutral). Inside the program read_only=false → normal create path.
///
/// CALL pops: gas, addr, value, argOff, argLen, retOff, retLen.
fn call_forwarder(target: Address) -> Vec<u8> {
    let mut c = Vec::new();
    // CALLDATACOPY(dest=0, off=0, size=CALLDATASIZE)
    c.extend_from_slice(&[0x36, 0x60, 0x00, 0x60, 0x00, 0x37]);
    // push (in reverse) retLen=0, retOff=0, argLen=CALLDATASIZE, argOff=0, value=0
    c.extend_from_slice(&[0x60, 0x00]); // retLen
    c.extend_from_slice(&[0x60, 0x00]); // retOff
    c.push(0x36); // argLen = CALLDATASIZE
    c.extend_from_slice(&[0x60, 0x00]); // argOff
    c.extend_from_slice(&[0x60, 0x00]); // value = 0 (ABI-mandated extra push)
                                        // addr
    c.push(0x73);
    c.extend_from_slice(target.as_slice());
    c.push(0x5a); // GAS
    c.push(0xf1); // CALL
    c.extend_from_slice(&[0x60, 0x00, 0x55]); // PUSH1 0 SSTORE (success flag)
    c.push(0x00); // STOP
    c
}

/// Constructor that returns `runtime` (CODECOPY src offset 0x0e).
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

fn eoa_tx(
    nonce: u64,
    to: Option<Address>,
    data: Vec<u8>,
    value: U256,
    gas: u64,
    ts: u64,
) -> SignedL2TxBuilder {
    SignedL2TxBuilder {
        chain_id: L2_CHAIN_ID,
        nonce,
        to,
        value,
        data: Bytes::from(data),
        gas_limit: gas,
        gas_price: 0,
        max_fee_per_gas: 2_000_000_000,
        max_priority_fee_per_gas: 0,
        access_list: Vec::new(),
        authorization_list: Vec::new(),
        kind: L2TxKind::Eip1559,
        signing_key: owner_key(),
        l1_block_number: 2,
        timestamp: ts,
        request_id: None,
        sender: SEQUENCER_ALIAS,
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

/// Setup tx hashes + addresses used to assert the prefix landed.
struct Prefix {
    steps: Vec<ScenarioStep>,
    stylus_addr: Address,
    fwd_addr: Address,
    activate_hash: B256,
    fwd_deploy_hash: B256,
}

/// Shared prefix: fund the EOA, deploy SolCaller (nonce 0), activate it
/// (nonce 1), deploy the forwarder built by `forwarder` (nonce 2).
fn build_prefix(owner: Address, idx: &Idx, forwarder: impl Fn(Address) -> Vec<u8>) -> Prefix {
    let mut steps = Vec::new();

    let dep_idx = idx.next();
    let dep = DepositBuilder {
        from: FUNDER,
        to: owner,
        amount: U256::from(10u128).pow(U256::from(20u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: dep_idx,
        base_fee_l1: 0,
    }
    .build()
    .expect("deposit");
    steps.push(msg_step(dep_idx, dep, 1));

    // Deploy SolCaller at owner nonce 0.
    let stylus_addr = create_address(owner, 0);
    let i = idx.next();
    steps.push(msg_step(
        i,
        eoa_tx(
            0,
            None,
            WhichProgram::SolCaller.initcode(),
            U256::ZERO,
            DEPLOY_GAS_CAP,
            1_700_000_001,
        )
        .build()
        .expect("deploy stylus"),
        1,
    ));

    // activateProgram(stylus_addr) at owner nonce 1.
    let mut act = selector4("activateProgram(address)").to_vec();
    let mut arg = [0u8; 32];
    arg[12..].copy_from_slice(stylus_addr.as_slice());
    act.extend_from_slice(&arg);
    let i = idx.next();
    let activate = eoa_tx(
        1,
        Some(ARBWASM),
        act,
        U256::from(10u128).pow(U256::from(15u64)),
        INVOKE_GAS_CAP,
        1_700_000_010,
    )
    .build()
    .expect("activate");
    let activate_hash =
        arb_test_harness::messaging::signed_l2_tx_hash(&activate).expect("act hash");
    steps.push(msg_step(i, activate, 1));

    // Deploy the forwarder at owner nonce 2.
    let fwd_addr = create_address(owner, 2);
    let i = idx.next();
    let fwd_deploy = eoa_tx(
        2,
        None,
        wrap_init(&forwarder(stylus_addr)),
        U256::ZERO,
        5_000_000,
        1_700_000_020,
    )
    .build()
    .expect("deploy forwarder");
    let fwd_deploy_hash =
        arb_test_harness::messaging::signed_l2_tx_hash(&fwd_deploy).expect("fwd hash");
    steps.push(msg_step(i, fwd_deploy, 1));

    Prefix {
        steps,
        stylus_addr,
        fwd_addr,
        activate_hash,
        fwd_deploy_hash,
    }
}

/// Outcome of a full run, with the per-node observables needed to judge the
/// read-only-create behavior directly (not just "some diff exists").
struct Outcome {
    report: arb_test_harness::dual_exec::DiffReport,
    fwd_addr: Address,
    reference_slot0: B256,
    arbreth_slot0: B256,
}

/// Builds + runs the full scenario with the given forwarder, asserting the
/// SolCaller activated + the forwarder deployed on BOTH nodes (so a no-op setup
/// can't make the assertion pass trivially), then reads the forwarder's slot-0
/// success flag from each node.
fn run(forwarder: impl Fn(Address) -> Vec<u8>) -> Outcome {
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);
    let idx = Idx::new();

    let prefix = build_prefix(owner, &idx, forwarder);
    let mut steps = prefix.steps;
    let fwd_addr = prefix.fwd_addr;

    // Invoke the forwarder (nonce 3): it forwards doCreate(0, ctor_sload_only)
    // into SolCaller, then SSTOREs the inner call's success flag at slot 0.
    let inner = do_create_calldata(&ctor_sload_only());
    let invoke = eoa_tx(
        3,
        Some(fwd_addr),
        inner,
        U256::ZERO,
        INVOKE_GAS_CAP,
        1_700_000_030,
    )
    .build()
    .expect("invoke forwarder");
    let invoke_hash = arb_test_harness::messaging::signed_l2_tx_hash(&invoke).expect("invoke hash");
    let i = idx.next();
    steps.push(msg_step(i, invoke, 1));

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
    steps.push(msg_step(seal_idx, seal, 2));

    let scenario = Scenario {
        name: "create_under_staticcall".into(),
        description: "CREATE under STATICCALL Gas(0) vs burn-all".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let checks = vec![StateCheck {
        address: fwd_addr,
        slots: vec![B256::ZERO],
        check_balance: false,
        check_nonce: false,
        check_code: false,
    }];

    let report = rig
        .dual
        .run_with_state_checks(&scenario, &checks)
        .expect("dual run");

    // Setup-landed guard: SolCaller activated + forwarder deployed + invoke
    // included, on BOTH nodes.
    for (node, code, ah, fh, ih) in [
        (
            "reference",
            rig.dual.left.code(prefix.stylus_addr, BlockId::Latest),
            rig.dual.left.receipt(prefix.activate_hash),
            rig.dual.left.receipt(prefix.fwd_deploy_hash),
            rig.dual.left.receipt(invoke_hash),
        ),
        (
            "arbreth",
            rig.dual.right.code(prefix.stylus_addr, BlockId::Latest),
            rig.dual.right.receipt(prefix.activate_hash),
            rig.dual.right.receipt(prefix.fwd_deploy_hash),
            rig.dual.right.receipt(invoke_hash),
        ),
    ] {
        assert_eq!(
            ah.expect("activate receipt").status,
            1,
            "[{node}] SolCaller activation must land"
        );
        assert_eq!(
            fh.expect("forwarder deploy receipt").status,
            1,
            "[{node}] forwarder deploy must land"
        );
        assert_eq!(
            ih.expect("invoke receipt").status,
            1,
            "[{node}] forwarder invoke (outer CALL) must land"
        );
        assert!(
            arb_stylus::is_stylus_classic(&code.expect("stylus code")),
            "[{node}] SolCaller must carry classic Stylus code"
        );
    }

    let reference_slot0 = rig
        .dual
        .left
        .storage(fwd_addr, B256::ZERO, BlockId::Latest)
        .expect("reference slot0");
    let arbreth_slot0 = rig
        .dual
        .right
        .storage(fwd_addr, B256::ZERO, BlockId::Latest)
        .expect("arbreth slot0");

    Outcome {
        report,
        fwd_addr,
        reference_slot0,
        arbreth_slot0,
    }
}

/// STATICCALL forwarder → read_only=true inside SolCaller drives the create1
/// hostio. The write-protected create halts the program frame (Failure), so the
/// outer STATICCALL returns 0 and the forwarder stores 0 at slot 0.
#[test]
#[ignore]
fn readonly_create_halts_frame() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let out = run(static_forwarder);

    eprintln!(
        "reference slot0={:?} arbreth slot0={:?}\n  block_diffs={:#?}\n  state_diffs={:#?}",
        out.reference_slot0, out.arbreth_slot0, out.report.block_diffs, out.report.state_diffs,
    );

    assert_eq!(
        out.reference_slot0,
        B256::ZERO,
        "the reference must store the failed STATICCALL flag (0) at forwarder slot 0"
    );

    let slot0_differs = out.reference_slot0 != out.arbreth_slot0;
    let storage_diff = out.report.state_diffs.iter().any(|d| {
        d.address == out.fwd_addr && matches!(d.field, StateField::Storage(s) if s == B256::ZERO)
    });
    let state_root_diff = out
        .report
        .block_diffs
        .iter()
        .any(|d| d.field == "state_root");

    // If the slot-0 values differ, the alternative outcome is exactly 1 (program
    // continued past the forbidden create), pinning the root cause rather than an
    // incidental difference.
    if slot0_differs {
        assert_eq!(
            out.arbreth_slot0,
            B256::from(U256::from(1u64)),
            "the surviving-program flag (1) at forwarder slot 0 pins the read_only create gate"
        );
    }

    assert!(
        slot0_differs && (storage_diff || state_root_diff),
        "expected a Storage(slot 0)@forwarder (left={:?} right={:?}) AND a state-level \
         difference from the read_only create gate; got block={:#?} state={:#?}",
        out.reference_slot0,
        out.arbreth_slot0,
        out.report.block_diffs,
        out.report.state_diffs,
    );
}

/// CALL forwarder → read_only=false → normal create path. The 1-byte runtime
/// deploys identically and the forwarder SSTOREs success=1; differs from the
/// read-only case ONLY in the call opcode.
#[test]
#[ignore]
fn mutable_create_control_clean() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let out = run(call_forwarder);

    assert_eq!(
        out.reference_slot0,
        B256::from(U256::from(1u64)),
        "control: the mutable CREATE succeeds, so the forwarder stores success flag 1"
    );
    assert!(
        out.report.is_clean(),
        "control (CALL, read_only=false) must be clean: block={:#?} tx={:#?} state={:#?}",
        out.report.block_diffs,
        out.report.tx_diffs,
        out.report.state_diffs,
    );
}
