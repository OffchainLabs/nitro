//! `ArbWasmCache.cacheProgram` (and `cacheCodehash` / `evictCodehash`) must
//! attribute every gas charge to its correct multi-gas resource dimension:
//!
//!   * argsCost                                      → L2Calldata
//!   * OpenArbosState / cache-manager / chain-owner / getProgram / module-hash reads + GetCodeHash
//!     → StorageAccessRead
//!   * the `params` warm read                        → Computation
//!   * the program `initCost`                        → StorageAccessRead
//!   * the UpdateProgramCache log                     → HistoryGrowth
//!   * the program SSTORE                             → StorageAccessWrite
//!
//! Both cases share one scenario (deposit → setL1PricePerUnit(0) → install one
//! constraint weighting StorageAccessRead → deploy SolCaller → activateProgram →
//! cacheProgram) and differ only in the constraint's starting backlog:
//!
//!   * floor (backlog 0): every dimension prices at the floor `base_fee`, so the per-tx fee is
//!     value-neutral. The constraint still grows the per-dimension backlog by the weighted gas, so
//!     the committed backlog (and block `state_root`) depends on the attribution.
//!   * escalated (nonzero backlog): the StorageAccessRead fee rises above the floor while
//!     Computation stays at it, so the attribution also moves the v60 multi-dimensional refund and
//!     the owner balance.

use std::sync::Mutex;

use alloy_primitives::{address, keccak256, Address, Bytes, B256, U256};
use arb_fuzz::{arbitrary_impls::interop::WhichProgram, scaffolding::selector4};
use arb_test_harness::{
    dual_exec::DualExec,
    genesis::GenesisBuilder,
    messaging::{
        signed_tx::{derive_address, L2TxKind, SignedL2TxBuilder},
        DepositBuilder, L1Message, MessageBuilder,
    },
    mock_l1::MockL1,
    node::{
        arbreth::ArbrethProcess, nitro_docker::NitroDocker, BlockId, ExecutionNode, NodeStartCtx,
        TxRequest,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep, StateCheck},
};

static SERIAL: Mutex<()> = Mutex::new(());

const L2_CHAIN_ID: u64 = 412_357;
const L1_CHAIN_ID: u64 = 11_155_111;
const ARBOS_VERSION: u64 = 60;

const ARBOWNER: Address = address!("0000000000000000000000000000000000000070");
const ARBOWNERPUBLIC: Address = address!("000000000000000000000000000000000000006b");
const ARBWASM: Address = address!("0000000000000000000000000000000000000071");
const ARBWASMCACHE: Address = address!("0000000000000000000000000000000000000072");
const ARBGASINFO: Address = address!("000000000000000000000000000000000000006c");
const FUNDER: Address = Address::new([0xa1; 20]);
const SEQUENCER: Address = address!("a4b000000000000000000073657175656e636572");

const BASE_TS: u64 = 1_700_000_000;
const DEPLOY_GAS_CAP: u64 = 1_000_000_000;
const INVOKE_GAS_CAP: u64 = 30_000_000;

/// StorageAccessRead resource kind (`ResourceKind` discriminant).
const KIND_STORAGE_READ: u8 = 3;

fn owner_key() -> B256 {
    B256::repeat_byte(0x42)
}

fn word(v: u64) -> [u8; 32] {
    let mut w = [0u8; 32];
    w[24..].copy_from_slice(&v.to_be_bytes());
    w
}

fn msg_step(idx: u64, msg: L1Message, dmr: u64) -> ScenarioStep {
    ScenarioStep::Message {
        idx,
        message: msg,
        delayed_messages_read: dmr,
    }
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

/// Hand-encoded `setMultiGasPricingConstraints` with one constraint and one
/// resource weight, matching the
/// `(((uint8,uint64)[],uint32,uint64,uint64)[])` layout the precompile decodes.
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
    d.extend_from_slice(&word(0x20)); // offset to constraints array
    d.extend_from_slice(&word(1)); // constraints.length
    d.extend_from_slice(&word(0x20)); // element[0] offset (relative to array data)
    d.extend_from_slice(&word(0x80)); // resources offset (relative to struct)
    d.extend_from_slice(&word(window as u64));
    d.extend_from_slice(&word(target));
    d.extend_from_slice(&word(backlog));
    d.extend_from_slice(&word(1)); // resources.length
    d.extend_from_slice(&word(resource as u64));
    d.extend_from_slice(&word(weight));
    d
}

/// An owner-signed EIP-1559 tx; `base_fee_l1: 0` keeps poster cost out of the
/// receipt so the refund reflects pure L2 multi-gas pricing.
fn owner_tx(
    nonce: u64,
    to: Option<Address>,
    value: U256,
    data: Vec<u8>,
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
        sender: SEQUENCER,
        base_fee_l1: 0,
    }
}

/// `cacheProgram(address)` calldata: selector 0xe73ac9f2 ++ pad32(addr).
fn cache_program_calldata(addr: Address) -> Vec<u8> {
    let mut d = vec![0xe7u8, 0x3a, 0xc9, 0xf2];
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(addr.as_slice());
    d.extend_from_slice(&pad);
    d
}

/// `activateProgram(address)` calldata: selector 0x58c780c2 ++ pad32(addr).
fn activate_calldata(addr: Address) -> Vec<u8> {
    let mut d = vec![0x58u8, 0xc7, 0x80, 0xc2];
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(addr.as_slice());
    d.extend_from_slice(&pad);
    d
}

/// `programMemoryFootprint(address)` calldata: exercises `load_params_and_program`.
fn footprint_calldata(addr: Address) -> Vec<u8> {
    let mut d = selector4("programMemoryFootprint(address)").to_vec();
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(addr.as_slice());
    d.extend_from_slice(&pad);
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

/// A nonzero starting backlog escalates the weighted dimension's fee above the
/// floor in the positive case; `backlog == 0` neutralizes it for the control.
fn scenario_for(name: &str, read_backlog: u64) -> Scenario {
    let owner = derive_address(owner_key());

    // Each step its own message + stepped timestamp so it lands in a fresh block.
    let dep = DepositBuilder {
        from: FUNDER,
        to: owner,
        amount: U256::from(10u128).pow(U256::from(20u64)),
        l1_block_number: 1,
        timestamp: BASE_TS,
        request_seq: 1,
        base_fee_l1: 0,
    }
    .build()
    .expect("deposit");

    // Zero L1 price so receipts/refund reflect pure L2 gas.
    let mut set_price = selector4("setL1PricePerUnit(uint256)").to_vec();
    set_price.extend_from_slice(&word(0));
    let set_price = owner_tx(
        0,
        Some(ARBOWNER),
        U256::ZERO,
        set_price,
        2_000_000,
        BASE_TS + 10,
    )
    .build()
    .expect("set price");

    // One constraint weighting StorageAccessRead (kind 3). A nonzero backlog
    // escalates that dimension's next-block fee above the floor; Computation
    // (unweighted) stays at the floor. backlog=0 (control) escalates nothing.
    let cons = set_constraint_calldata(60, 100_000, read_backlog, KIND_STORAGE_READ, 10_000);
    let set_cons = owner_tx(1, Some(ARBOWNER), U256::ZERO, cons, 2_000_000, BASE_TS + 20)
        .build()
        .expect("set constraint");

    // Deploy the real SolCaller SDK program (nonzero init_gas).
    let deploy = owner_tx(
        2,
        None,
        U256::ZERO,
        WhichProgram::SolCaller.initcode(),
        DEPLOY_GAS_CAP,
        BASE_TS + 30,
    )
    .build()
    .expect("deploy");
    let stylus_addr = create_address(owner, 2);

    // Activate: sets program.init_cost = info.init_gas, cached = false.
    let activate = owner_tx(
        3,
        Some(ARBWASM),
        U256::from(10u128).pow(U256::from(15u64)),
        activate_calldata(stylus_addr),
        INVOKE_GAS_CAP,
        BASE_TS + 40,
    )
    .build()
    .expect("activate");

    // cacheProgram flips the cache bit false→true, burning initCost.
    let cache = owner_tx(
        4,
        Some(ARBWASMCACHE),
        U256::ZERO,
        cache_program_calldata(stylus_addr),
        INVOKE_GAS_CAP,
        BASE_TS + 50,
    )
    .build()
    .expect("cache");

    // Trailing no-op deposit to seal the cacheProgram block before querying.
    let seal = DepositBuilder {
        from: FUNDER,
        to: FUNDER,
        amount: U256::from(1u64),
        l1_block_number: 1,
        timestamp: BASE_TS + 60,
        request_seq: 7,
        base_fee_l1: 0,
    }
    .build()
    .expect("seal");

    Scenario {
        name: name.into(),
        description: "cacheProgram initCost dimensional attribution under asymmetric pricing"
            .into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps: vec![
            msg_step(1, dep, 1),
            msg_step(2, set_price, 1),
            msg_step(3, set_cons, 1),
            msg_step(4, deploy, 1),
            msg_step(5, activate, 1),
            msg_step(6, cache, 1),
            msg_step(7, seal, 2),
        ],
    }
}

fn owner_check() -> StateCheck {
    StateCheck {
        address: derive_address(owner_key()),
        slots: Vec::new(),
        check_balance: true,
        check_nonce: false,
        check_code: false,
    }
}

/// Read the per-resource multi-gas base fees (`uint256[]`) and the floor, asserting
/// the StorageAccessRead dimension escalated above both the floor and Computation —
/// otherwise the attribution would be value-neutral and the test insensitive.
fn assert_read_dim_escalated(rig: &mut Rig, owner: Address) {
    let fees_bytes = rig
        .dual
        .right
        .eth_call(
            TxRequest {
                from: Some(owner),
                to: Some(ARBGASINFO),
                data: Some(Bytes::from(selector4("getMultiGasBaseFee()").to_vec())),
                value: Some(U256::ZERO),
                gas: Some(3_000_000),
            },
            BlockId::Latest,
        )
        .expect("getMultiGasBaseFee");
    let fees = decode_uint256_array(&fees_bytes);
    let floor = rig
        .dual
        .right
        .eth_call(
            TxRequest {
                from: Some(owner),
                to: Some(ARBGASINFO),
                data: Some(Bytes::from(selector4("getMinimumGasPrice()").to_vec())),
                value: Some(U256::ZERO),
                gas: Some(3_000_000),
            },
            BlockId::Latest,
        )
        .ok()
        .map(|b| U256::from_be_slice(&b))
        .unwrap_or(U256::ZERO);
    let read_fee = fees
        .get(KIND_STORAGE_READ as usize)
        .copied()
        .unwrap_or(U256::ZERO);
    let comp_fee = fees.get(1usize).copied().unwrap_or(U256::ZERO);
    eprintln!("floor={floor} read_fee={read_fee} comp_fee={comp_fee} fees={fees:?}");
    assert!(
        read_fee > floor && read_fee > comp_fee,
        "StorageAccessRead fee did not escalate above the floor/Computation; the \
         constraint must bite for the comparison to be sensitive"
    );
}

/// Decode an ABI `uint256[]` return (offset, length, then words).
fn decode_uint256_array(b: &[u8]) -> Vec<U256> {
    if b.len() < 64 {
        return Vec::new();
    }
    let len = U256::from_be_slice(&b[32..64]).to::<usize>();
    (0..len)
        .filter_map(|i| {
            let s = 64 + i * 32;
            b.get(s..s + 32).map(U256::from_be_slice)
        })
        .collect()
}

// Escalated pricing: a nonzero StorageAccessRead backlog lifts that dimension's
// fee above the floor (sensitivity-gated below), so the attribution moves both
// the per-tx fee and the v60 multi-dimensional refund.
#[test]
#[ignore]
fn cacheprogram_attribution_matches_when_escalated() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);

    let scenario = scenario_for("cacheprogram_attribution_escalated", 50_000_000);
    let report = rig
        .dual
        .run_with_state_checks(&scenario, &[owner_check()])
        .expect("dual run");

    assert_read_dim_escalated(&mut rig, owner);

    assert!(
        report.is_clean(),
        "escalated attribution mismatch\n  block_diffs={:#?}\n  tx_diffs={:#?}\n  state_diffs={:#?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}

// Floor pricing: backlog 0, so no dimension escalates and every kind prices at
// the floor base_fee. The per-tx fee is value-neutral, but the constraint still
// grows the per-dimension backlog, so the committed backlog and state_root
// depend on the attribution. Differs from the escalated case only in the
// constraint's backlog argument.
#[test]
#[ignore]
fn cacheprogram_attribution_matches_at_floor() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);

    let scenario = scenario_for("cacheprogram_attribution_floor", 0);
    let report = rig
        .dual
        .run_with_state_checks(&scenario, &[owner_check()])
        .expect("dual run");

    assert!(
        report.is_clean(),
        "floor-priced attribution mismatch\n  block_diffs={:#?}\n  tx_diffs={:#?}\n  state_diffs={:#?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}

/// Same constraint setup as [`scenario_for`], but after deploy+activate it calls
/// three StylusParams-reading queries as txs: `ArbWasm.inkPrice()` (load_params),
/// `ArbWasm.programMemoryFootprint(addr)` (load_params_and_program), and
/// `ArbOwnerPublic.getMaxStylusContractFragments()`. Each reads the params slot,
/// whose warm read must be billed to Computation, not StorageAccessRead.
fn scenario_param_queries(name: &str, read_backlog: u64) -> Scenario {
    let owner = derive_address(owner_key());

    let dep = DepositBuilder {
        from: FUNDER,
        to: owner,
        amount: U256::from(10u128).pow(U256::from(20u64)),
        l1_block_number: 1,
        timestamp: BASE_TS,
        request_seq: 1,
        base_fee_l1: 0,
    }
    .build()
    .expect("deposit");

    let mut set_price = selector4("setL1PricePerUnit(uint256)").to_vec();
    set_price.extend_from_slice(&word(0));
    let set_price = owner_tx(
        0,
        Some(ARBOWNER),
        U256::ZERO,
        set_price,
        2_000_000,
        BASE_TS + 10,
    )
    .build()
    .expect("set price");

    let cons = set_constraint_calldata(60, 100_000, read_backlog, KIND_STORAGE_READ, 10_000);
    let set_cons = owner_tx(1, Some(ARBOWNER), U256::ZERO, cons, 2_000_000, BASE_TS + 20)
        .build()
        .expect("set constraint");

    let deploy = owner_tx(
        2,
        None,
        U256::ZERO,
        WhichProgram::SolCaller.initcode(),
        DEPLOY_GAS_CAP,
        BASE_TS + 30,
    )
    .build()
    .expect("deploy");
    let stylus_addr = create_address(owner, 2);

    let activate = owner_tx(
        3,
        Some(ARBWASM),
        U256::from(10u128).pow(U256::from(15u64)),
        activate_calldata(stylus_addr),
        INVOKE_GAS_CAP,
        BASE_TS + 40,
    )
    .build()
    .expect("activate");

    // load_params: a param getter that takes no program argument.
    let ink = owner_tx(
        4,
        Some(ARBWASM),
        U256::ZERO,
        selector4("inkPrice()").to_vec(),
        INVOKE_GAS_CAP,
        BASE_TS + 50,
    )
    .build()
    .expect("inkPrice");

    // load_params_and_program: reads the params slot and the program slot.
    let footprint = owner_tx(
        5,
        Some(ARBWASM),
        U256::ZERO,
        footprint_calldata(stylus_addr),
        INVOKE_GAS_CAP,
        BASE_TS + 60,
    )
    .build()
    .expect("footprint");

    // ArbOwnerPublic params read.
    let fragments = owner_tx(
        6,
        Some(ARBOWNERPUBLIC),
        U256::ZERO,
        selector4("getMaxStylusContractFragments()").to_vec(),
        INVOKE_GAS_CAP,
        BASE_TS + 70,
    )
    .build()
    .expect("fragments");

    let seal = DepositBuilder {
        from: FUNDER,
        to: FUNDER,
        amount: U256::from(1u64),
        l1_block_number: 1,
        timestamp: BASE_TS + 80,
        request_seq: 7,
        base_fee_l1: 0,
    }
    .build()
    .expect("seal");

    Scenario {
        name: name.into(),
        description: "stylus params query resource attribution".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps: vec![
            msg_step(1, dep, 1),
            msg_step(2, set_price, 1),
            msg_step(3, set_cons, 1),
            msg_step(4, deploy, 1),
            msg_step(5, activate, 1),
            msg_step(6, ink, 1),
            msg_step(7, footprint, 1),
            msg_step(8, fragments, 1),
            msg_step(9, seal, 2),
        ],
    }
}

// Floor pricing: a StorageAccessRead-weighting constraint with backlog 0. The
// query methods read StylusParams (computation-dimensioned); the warm params
// read must be billed to Computation, not StorageAccessRead, so the query
// blocks' committed backlog and state_root match.
#[test]
#[ignore]
fn stylus_params_query_attribution_at_floor() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
    let mut rig = Rig::spawn(owner);

    let scenario = scenario_param_queries("stylus_params_query_floor", 0);
    let report = rig
        .dual
        .run_with_state_checks(&scenario, &[owner_check()])
        .expect("dual run");

    assert!(
        report.is_clean(),
        "params-query attribution mismatch\n  block_diffs={:#?}\n  tx_diffs={:#?}\n  state_diffs={:#?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}
