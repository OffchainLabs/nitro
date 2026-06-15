//! `pay_for_memory_grow` page-operand overflow handling (arbos >= 59).
//!
//! At ArbOS >= 59 `pay_for_memory_grow` takes `pages: u32`; when
//! `pages > u16::MAX` the whole gas budget is burned (`buy_gas(Gas(u64::MAX))`)
//! before truncation, so the program traps OutOfInk (failed receipt, full gas).
//! This pins that behavior against the reference, with a control whose only
//! change is a page operand within the `u16` range (no guard, no truncation).
//!
//! The WAT reads its page count via `i32.load` (little-endian), so both operands
//! MUST be `to_le_bytes`; big-endian 16 would become 16 << 24 > u16::MAX.

use std::sync::Mutex;

use alloy_primitives::{address, keccak256, Address, Bytes, B256, U256};
use arb_test_harness::{
    dual_exec::DualExec,
    genesis::GenesisBuilder,
    messaging::{
        signed_l2_tx_hash,
        signed_tx::{derive_address, L2TxKind, SignedL2TxBuilder},
        DepositBuilder, MessageBuilder,
    },
    mock_l1::MockL1,
    node::{
        arbreth::ArbrethProcess, nitro_docker::NitroDocker, BlockId, ExecutionNode, NodeStartCtx,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
};

static SERIAL: Mutex<()> = Mutex::new(());

const L2_CHAIN_ID: u64 = 412_350;
const L1_CHAIN_ID: u64 = 11_155_111;
const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const ARBOS_VERSION: u64 = 60;
const BASE_TS: u64 = 1_700_000_000;
const BLOCK_SECS: u64 = 12;
const GAS_CAP: u64 = 4_000_000;

const ARBWASM: Address = address!("0000000000000000000000000000000000000071");
const FUNDER: Address = Address::new([0xa1; 20]);
const SEQUENCER: Address = address!("a4b000000000000000000073657175656e636572");

// Direct-call `pay_for_memory_grow` program: reads the i32 page operand from
// calldata (little-endian `i32.load`) and forwards it verbatim to the hostio.
const WAT_PAY_FOR_MEMORY_GROW: &str = r#"
(module
    (import "vm_hooks" "pay_for_memory_grow" (func $pay_for_memory_grow (param i32)))
    (import "vm_hooks" "read_args"    (func $read_args    (param i32)))
    (import "vm_hooks" "write_result" (func $write_result (param i32 i32)))
    (memory (export "memory") 1)
    (func (export "user_entrypoint") (param $args_len i32) (result i32)
        (call $read_args (i32.const 0))
        (call $pay_for_memory_grow (i32.load (i32.const 0)))
        (call $write_result (i32.const 0) (i32.const 0))
        i32.const 0
    )
)
"#;

fn signing_key() -> B256 {
    B256::from(keccak256(b"memory-grow-eoa"))
}
fn eoa() -> Address {
    derive_address(signing_key())
}

fn build_init_code(wasm: &[u8]) -> Vec<u8> {
    let body = arb_stylus::compress_classic_program_code(wasm).expect("compress stylus program");
    let size = body.len();
    let size_hi = ((size >> 8) & 0xFF) as u8;
    let size_lo = (size & 0xFF) as u8;
    let mut out = Vec::with_capacity(14 + size);
    out.extend_from_slice(&[
        0x61, size_hi, size_lo, 0x60, 0x0e, 0x60, 0x00, 0x39, 0x61, size_hi, size_lo, 0x60, 0x00,
        0xF3,
    ]);
    out.extend_from_slice(&body);
    out
}

fn create_address(sender: Address, nonce: u64) -> Address {
    let nonce_rlp = if nonce == 0 {
        vec![0x80u8]
    } else {
        let bytes = nonce.to_be_bytes();
        let trimmed: &[u8] = bytes
            .iter()
            .position(|b| *b != 0)
            .map(|i| &bytes[i..])
            .unwrap_or(&bytes[..0]);
        if trimmed.len() == 1 && trimmed[0] < 0x80 {
            vec![trimmed[0]]
        } else {
            let mut v = vec![0x80 + trimmed.len() as u8];
            v.extend_from_slice(trimmed);
            v
        }
    };
    let mut payload = Vec::new();
    payload.push(0x80 + 20);
    payload.extend_from_slice(sender.as_slice());
    payload.extend_from_slice(&nonce_rlp);
    let mut rlp = vec![0xC0 + payload.len() as u8];
    rlp.extend_from_slice(&payload);
    Address::from_slice(&keccak256(&rlp).as_slice()[12..])
}

fn msg_step(idx: u64, msg: arb_test_harness::messaging::L1Message, dmr: u64) -> ScenarioStep {
    ScenarioStep::Message {
        idx,
        message: msg,
        delayed_messages_read: dmr,
    }
}

fn signed(
    nonce: u64,
    to: Option<Address>,
    data: Vec<u8>,
    value: U256,
    ts: u64,
) -> SignedL2TxBuilder {
    SignedL2TxBuilder {
        chain_id: L2_CHAIN_ID,
        nonce,
        to,
        value,
        data: Bytes::from(data),
        gas_limit: GAS_CAP,
        gas_price: 0,
        max_fee_per_gas: 2_000_000_000,
        max_priority_fee_per_gas: 0,
        access_list: Vec::new(),
        authorization_list: Vec::new(),
        kind: L2TxKind::Eip1559,
        signing_key: signing_key(),
        l1_block_number: 1 + (ts - BASE_TS) / BLOCK_SECS,
        timestamp: ts,
        request_id: None,
        sender: SEQUENCER,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
}

struct Rig {
    dual: DualExec<NitroDocker, ArbrethProcess>,
}

impl Rig {
    fn spawn() -> Self {
        let mock = MockL1::start(L1_CHAIN_ID).expect("mock l1 start");
        let genesis = GenesisBuilder::new(L2_CHAIN_ID, ARBOS_VERSION)
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

/// Tx hashes + program address needed to assert the setup landed before judging
/// the page-operand outcome.
struct Built {
    scenario: Scenario,
    program: Address,
    deploy_hash: B256,
    activate_hash: B256,
    invoke_hash: B256,
}

/// fund EOA -> deploy program -> activateProgram -> invoke(page_operand) ->
/// trailing no-op deposit to seal the block. Only the page operand differs
/// between the overflow and control scenarios. Each test spawns its own node
/// pair, so nonces, request seqs, and delayed-read counters reset cleanly.
fn build_scenario(name: &str, page_operand_le: [u8; 4]) -> Built {
    let wasm = wat::parse_bytes(WAT_PAY_FOR_MEMORY_GROW.as_bytes())
        .expect("WAT compiles")
        .into_owned();

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
            to: eoa(),
            amount: U256::from(10u128).pow(U256::from(20u64)),
            l1_block_number: 1,
            timestamp: BASE_TS,
            request_seq: i,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        }
        .build()
        .expect("deposit"),
        1,
    ));

    let deploy_addr = create_address(eoa(), 0);
    let i = next();
    let deploy = signed(
        0,
        None,
        build_init_code(&wasm),
        U256::ZERO,
        BASE_TS + i * BLOCK_SECS,
    )
    .build()
    .expect("deploy");
    let deploy_hash = signed_l2_tx_hash(&deploy).expect("deploy hash");
    steps.push(msg_step(i, deploy, 1));

    let mut activate_data = vec![0x58, 0xc7, 0x80, 0xc2]; // activateProgram(address)
    let mut padded = [0u8; 32];
    padded[12..].copy_from_slice(deploy_addr.as_slice());
    activate_data.extend_from_slice(&padded);
    let i = next();
    let activate = signed(
        1,
        Some(ARBWASM),
        activate_data,
        U256::from(10u128).pow(U256::from(15u64)),
        BASE_TS + i * BLOCK_SECS,
    )
    .build()
    .expect("activate");
    let activate_hash = signed_l2_tx_hash(&activate).expect("activate hash");
    steps.push(msg_step(i, activate, 1));

    let i = next();
    let invoke = signed(
        2,
        Some(deploy_addr),
        page_operand_le.to_vec(),
        U256::ZERO,
        BASE_TS + i * BLOCK_SECS,
    );
    let invoke_msg = invoke.build().expect("invoke");
    let invoke_hash = signed_l2_tx_hash(&invoke_msg).expect("invoke tx hash");
    steps.push(msg_step(i, invoke_msg, 1));

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
        description: "pay_for_memory_grow page-operand overflow".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };
    Built {
        scenario,
        program: deploy_addr,
        deploy_hash,
        activate_hash,
        invoke_hash,
    }
}

/// Assert the deploy + activate landed (status 1) on BOTH nodes and the program
/// carries Stylus bytecode, so the assertion can't pass trivially against a
/// setup that failed identically on both nodes.
fn assert_setup_landed(rig: &Rig, b: &Built) {
    for (node, code, dh, ah) in [
        (
            "reference",
            rig.dual.left.code(b.program, BlockId::Latest),
            rig.dual.left.receipt(b.deploy_hash),
            rig.dual.left.receipt(b.activate_hash),
        ),
        (
            "arbreth",
            rig.dual.right.code(b.program, BlockId::Latest),
            rig.dual.right.receipt(b.deploy_hash),
            rig.dual.right.receipt(b.activate_hash),
        ),
    ] {
        let dr = dh.expect("deploy receipt");
        assert_eq!(dr.status, 1, "[{node}] deploy must land");
        let ar = ah.expect("activate receipt");
        assert_eq!(ar.status, 1, "[{node}] activate must land");
        let code = code.expect("program code");
        assert!(
            arb_stylus::is_stylus_classic(&code),
            "[{node}] program must carry classic Stylus code, got {} bytes",
            code.len()
        );
    }
}

#[test]
#[ignore]
fn memory_grow_overflow_traps_out_of_ink() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());

    let mut rig = Rig::spawn();
    // 65536 > u16::MAX: the whole gas budget is burned (OutOfInk).
    let operand = 65536u32.to_le_bytes();
    assert_eq!(operand, [0x00, 0x00, 0x01, 0x00]);
    let built = build_scenario("grow_65536", operand);

    let report = rig.dual.run(&built.scenario).expect("dual run");

    assert_setup_landed(&rig, &built);

    let invoke_hash = built.invoke_hash;
    let status_differed = report
        .tx_diffs
        .iter()
        .any(|d| d.tx_hash == invoke_hash && d.field == "status");
    let state_root_differed = report.block_diffs.iter().any(|d| d.field == "state_root");

    let reference_invoke = rig
        .dual
        .left
        .receipt(invoke_hash)
        .expect("reference invoke");
    assert_eq!(
        reference_invoke.status, 0,
        "the reference must OOG the invoke (pages 65536 > u16::MAX at arbos>=59)"
    );
    assert!(
        status_differed || state_root_differed,
        "expected the invoke-tx status or block state_root to differ, found neither.\n  \
         invoke_hash={invoke_hash:?}\n  block_diffs={:#?}\n  tx_diffs={:#?}\n  state_diffs={:#?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}

#[test]
#[ignore]
fn memory_grow_in_range_control_clean() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());

    let mut rig = Rig::spawn();
    // 16 <= u16::MAX: no guard, no truncation; both hosts charge the same
    // free-page cost and succeed.
    let operand = 16u32.to_le_bytes();
    assert_eq!(operand, [0x10, 0x00, 0x00, 0x00]);
    let built = build_scenario("grow_16", operand);

    let report = rig.dual.run(&built.scenario).expect("dual run");

    assert_setup_landed(&rig, &built);

    let reference_invoke = rig
        .dual
        .left
        .receipt(built.invoke_hash)
        .expect("reference invoke");
    assert_eq!(
        reference_invoke.status, 1,
        "control: pages 16 <= u16::MAX so the invoke must succeed"
    );
    assert!(
        report.is_clean(),
        "control must be clean (isolates the page operand from the harness)\n  \
         block_diffs={:#?}\n  tx_diffs={:#?}\n  state_diffs={:#?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
}
