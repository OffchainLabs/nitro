//! A Stylus storage-cache flush that OOGs must not attribute its pre-OOG slots
//! to storage dimensions.
//!
//! On a Stylus storage-cache flush that runs out of gas, the pre-OOG slots are
//! attributed to WasmComputation, not to storage dimensions (StorageGrowth /
//! StorageAccessRead). The single-gas total is unchanged, but the v60
//! per-dimension `multiGasUsed` split — and, under an active WasmComputation
//! constraint that escalates that dimension's fee above base, the
//! multi-dimensional refund and the sender's balance — depend on this
//! attribution.
//!
//! A `StorageStress` program caches several fresh (0→nonzero) slots then
//! flushes, called from an EVM forwarder with a tight forwarded gas cap so the
//! flush OOGs on a late slot; the forwarder ignores the failed inner CALL and
//! returns success, so the outer tx commits and exposes the receipt. With a
//! generous forwarded budget the flush completes and every observable agrees;
//! the two scenarios differ only in the forwarded gas cap.

use std::sync::{
    atomic::{AtomicU64, Ordering},
    Mutex,
};

static SERIAL: Mutex<()> = Mutex::new(());

use alloy_primitives::{address, keccak256, Address, Bytes, B256, U256};
use arb_fuzz::{arbitrary_impls::interop::WhichProgram, scaffolding::selector4};
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

/// WasmComputation resource kind (`ResourceKind` discriminant).
const KIND_WASM_COMPUTATION: u8 = 8;

/// Number of fresh slots the inner program caches before flushing. Each fresh
/// (0→nonzero) write costs ~22_100 gas at flush, so a tight forwarded budget
/// covering the WASM execution plus part of the flush OOGs on a late slot,
/// leaving the earlier slots committed.
const SLOT_COUNT: u64 = 2;

/// Tight forwarded gas cap (positive): enough to enter the program and cache
/// all slots, but not enough to flush all of them — OOGs on a late slot.
/// Calibrate against the real flush cost if the inner CALL never enters the
/// flush loop or completes it (see module doc).
const TIGHT_FLUSH_BUDGET: u16 = 30_000;

fn owner_key() -> B256 {
    B256::repeat_byte(0x42)
}

fn word(v: u64) -> [u8; 32] {
    let mut w = [0u8; 32];
    w[24..].copy_from_slice(&v.to_be_bytes());
    w
}

fn word_u256(v: U256) -> [u8; 32] {
    v.to_be_bytes::<32>()
}

/// `setMultiGasPricingConstraints` with a single constraint weighting one
/// resource, matching the `(((uint8,uint64)[],uint32,uint64,uint64)[])` layout.
fn set_constraint_calldata(
    window: u32,
    target: u64,
    backlog: u64,
    resource: u8,
    weight: u64,
) -> Vec<u8> {
    let mut d = Vec::with_capacity(4 + 320);
    d.extend_from_slice(&selector4(
        "setMultiGasPricingConstraints(((uint8,uint64)[],uint32,uint64,uint64)[])",
    ));
    d.extend_from_slice(&word(0x20));
    d.extend_from_slice(&word(1));
    d.extend_from_slice(&word(0x20));
    d.extend_from_slice(&word(0x80));
    d.extend_from_slice(&word(window as u64));
    d.extend_from_slice(&word(target));
    d.extend_from_slice(&word(backlog));
    d.extend_from_slice(&word(1));
    d.extend_from_slice(&word(resource as u64));
    d.extend_from_slice(&word(weight));
    d
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

/// `writeRange(uint256 start, uint256 count, uint256 base)` — caches `count`
/// fresh slots from `start`, then flushes with `clear=false`. The flush is the
/// OOG site under a tight inner budget.
fn write_range_calldata(start: U256, count: U256, base: U256) -> Vec<u8> {
    let mut d = selector4("writeRange(uint256,uint256,uint256)").to_vec();
    d.extend_from_slice(&word_u256(start));
    d.extend_from_slice(&word_u256(count));
    d.extend_from_slice(&word_u256(base));
    d
}

/// Runtime that CALLDATACOPYs its calldata to memory and CALLs `target`
/// (value 0) with a fixed `gas_cap`, then STOPs regardless of the inner result
/// — so the outer tx always commits with status 1, exposing the receipt's
/// `multiGasUsed` from the failed inner flush.
///
/// CALL stack order popped: gas, addr, value, argOff, argLen, retOff, retLen.
fn forwarder_runtime(target: Address, gas_cap: u16) -> Vec<u8> {
    let mut c = Vec::new();
    // CALLDATACOPY(dest=0, off=0, size=CALLDATASIZE)
    c.extend_from_slice(&[0x36, 0x60, 0x00, 0x60, 0x00, 0x37]);
    // push (in reverse) retLen=0, retOff=0, argLen=CALLDATASIZE, argOff=0, value=0
    c.extend_from_slice(&[0x60, 0x00]); // retLen
    c.extend_from_slice(&[0x60, 0x00]); // retOff
    c.push(0x36); // argLen = CALLDATASIZE
    c.extend_from_slice(&[0x60, 0x00]); // argOff
    c.extend_from_slice(&[0x60, 0x00]); // value
                                        // addr
    c.push(0x73);
    c.extend_from_slice(target.as_slice());
    // gas = PUSH2 gas_cap
    c.push(0x61);
    c.extend_from_slice(&gas_cap.to_be_bytes());
    c.push(0xf1); // CALL
    c.push(0x50); // POP success flag
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

fn msg_step(idx: u64, msg: arb_test_harness::messaging::L1Message, dmr: u64) -> ScenarioStep {
    ScenarioStep::Message {
        idx,
        message: msg,
        delayed_messages_read: dmr,
    }
}

/// Build the shared step prefix: fund owner, set L1 price 0, install a
/// WasmComputation constraint, deploy + activate the StorageStress program,
/// deploy the forwarder. Returns (steps, storage_stress_addr, forwarder_addr).
fn build_prefix(owner: Address, idx: &Idx, gas_cap: u16) -> (Vec<ScenarioStep>, Address, Address) {
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

    // WasmComputation constraint with a small target so the residual WASM gas
    // escalates that dimension's fee above base — making the dimensional split
    // (storage vs wasm) move the multi-dimensional refund, hence sender balance.
    let cons = set_constraint_calldata(60, 100_000, 0, KIND_WASM_COMPUTATION, 10_000);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx(1, Some(ARBOWNER), cons, 2_000_000, 1_700_000_001)
            .build()
            .expect("set constraint"),
        1,
    ));

    // Deploy StorageStress at owner nonce 2.
    let stylus_addr = create_address(owner, 2);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx(
            2,
            None,
            WhichProgram::StorageStress.initcode(),
            500_000_000,
            1_700_000_002,
        )
        .build()
        .expect("deploy stylus"),
        1,
    ));

    // activateProgram(stylus_addr).
    let mut act = selector4("activateProgram(address)").to_vec();
    let mut arg = [0u8; 32];
    arg[12..].copy_from_slice(stylus_addr.as_slice());
    act.extend_from_slice(&arg);
    let mut activate = owner_tx(3, Some(ARBWASM), act, 200_000_000, 1_700_000_010);
    activate.value = U256::from(10u128).pow(U256::from(18u64));
    let i = idx.next();
    steps.push(msg_step(i, activate.build().expect("activate"), 1));

    // Deploy the forwarder at owner nonce 4 (knows stylus_addr + gas_cap).
    let fwd_addr = create_address(owner, 4);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx(
            4,
            None,
            wrap_init(&forwarder_runtime(stylus_addr, gas_cap)),
            5_000_000,
            1_700_000_020,
        )
        .build()
        .expect("deploy forwarder"),
        1,
    ));

    (steps, stylus_addr, fwd_addr)
}

fn run(gas_cap: u16) -> (arb_test_harness::dual_exec::DiffReport, Rig, Address) {
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);
    let idx = Idx::new();

    let (mut steps, stylus_addr, fwd_addr) = build_prefix(owner, &idx, gas_cap);

    // Invoke the forwarder: it CALLs writeRange(start, SLOT_COUNT, base) with the
    // tight cap, then STOPs. Fresh slots (base ^ i, all nonzero) start at 0, so
    // each is a 0→nonzero cold SSTORE attributed to WasmComputation on the OOG path.
    let start = U256::from(0x10u64);
    let base = U256::from(0x9e37_79b9_7f4a_7c15u64);
    let inner = write_range_calldata(start, U256::from(SLOT_COUNT), base);
    let i = idx.next();
    steps.push(msg_step(
        i,
        owner_tx(5, Some(fwd_addr), inner, 30_000_000, 1_700_000_030)
            .build()
            .expect("invoke forwarder"),
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
        name: "flush_oog_dim_attribution".into(),
        description: "flush_storage_cache OOG dimensional attribution".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    // Check the sender's (owner) balance — the refund-driven observable — plus
    // the StorageStress contract's first few slots, which the partial flush
    // commits (proving the flush wrote slots before OOGing).
    let mut slots = Vec::new();
    for i in 0..SLOT_COUNT {
        slots.push(B256::from(word_u256(U256::from(0x10u64) + U256::from(i))));
    }
    let checks = vec![
        StateCheck {
            address: owner,
            slots: Vec::new(),
            check_balance: true,
            check_nonce: false,
            check_code: false,
        },
        StateCheck {
            address: stylus_addr,
            slots,
            check_balance: false,
            check_nonce: false,
            check_code: false,
        },
    ];

    let report = rig
        .dual
        .run_with_state_checks(&scenario, &checks)
        .expect("dual run");
    (report, rig, owner)
}

/// The inner flush OOGs on a late slot, so its pre-OOG slots are attributed to
/// WasmComputation. The receipt's per-dimension `multiGasUsed` and, under the
/// active WasmComputation constraint, the refund (sender balance) follow that
/// attribution.
#[test]
#[ignore]
fn flush_oog_attributes_pre_oog_slots_to_wasm_computation() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let (report, _rig, owner) = run(TIGHT_FLUSH_BUDGET);

    let dim_diff = report.tx_diffs.iter().any(|d| {
        d.field == "mg_wasm_computation"
            || d.field == "mg_storage_growth"
            || d.field == "mg_storage_access_read"
            || d.field == "mg_refund"
    });
    let balance_diff = report.state_diffs.iter().any(|d| {
        d.address == owner && matches!(d.field, arb_test_harness::dual_exec::StateField::Balance)
    });
    let state_root_diff = report.block_diffs.iter().any(|d| d.field == "state_root");

    assert!(
        dim_diff || balance_diff || state_root_diff,
        "expected the dimensional multi-gas / refund / balance / state_root to reflect the \
         OOG flush attribution; got block={:?} tx={:?} state={:?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
    eprintln!(
        "block={:?}\ntx={:?}\nstate={:?}",
        report.block_diffs, report.tx_diffs, report.state_diffs
    );
}

/// With a generous forwarded budget the inner flush completes (no OOG), the
/// storage dimensions commit, and every observable agrees across nodes.
#[test]
#[ignore]
fn flush_completes_control_clean() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    // u16::MAX (65_535) caps the inner CALL well above the full flush cost
    // (~SLOT_COUNT * 22_100), so the flush completes successfully.
    let (report, _rig, _owner) = run(u16::MAX);
    assert!(
        report.is_clean(),
        "control (no OOG) must be clean: block={:?} tx={:?} state={:?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}
