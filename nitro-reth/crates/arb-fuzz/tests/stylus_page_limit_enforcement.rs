//! The Stylus consensus `PageLimit` (ArbOS >= 59) must be enforced.
//!
//! At ArbOS >= 59 with `PageLimit > 0` and `newOpen > PageLimit` the page charge
//! saturates so the gas burn fails and the program OOGs.
//!
//! The memory-cost exponent table is `[0..128]` and returns `u64::MAX` at index
//! 129 and above, so at the default `PageLimit = 128` any `newOpen` above 128
//! saturates the cost anyway. Enforcement is only observable when `PageLimit` is lowered below
//! 128 (`ArbOwner.setWasmPageLimit`, selector 0x6595381a, gated v30+) and the
//! window `PageLimit < newOpen <= 128` keeps the exponent finite.
//!
//! Scenario: the chain owner lowers `PageLimit` to 4, deploys+activates a Stylus
//! program declaring `(memory 1)` (footprint 1 <= 4, so it activates), then
//! invokes it. The program calls `pay_for_memory_grow(8)` ->
//! `newOpen = 1 + 8 = 9 > 4` -> addPages OOGs. The grow amount (8) is far below
//! `u16::MAX`, so the page-operand truncation path is never exercised. The
//! control raises the limit to the default 128 so `newOpen = 9` passes and the
//! exponent stays finite; the two cases differ ONLY in the page-limit value.

use std::sync::Mutex;

use alloy_primitives::{keccak256, Address, Bytes, B256, U256};
use arb_test_harness::{
    dual_exec::DualExec,
    genesis::GenesisBuilder,
    messaging::{
        signed_l2_tx_hash,
        signed_tx::{derive_address, L2TxKind, SignedL2TxBuilder},
        DepositBuilder, L1Message, MessageBuilder,
    },
    mock_l1::MockL1,
    node::{
        arbreth::ArbrethProcess, nitro_docker::NitroDocker, BlockId, ExecutionNode, NodeStartCtx,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
};

static SERIAL: Mutex<()> = Mutex::new(());

const L2_CHAIN_ID: u64 = 412_351;
const L1_CHAIN_ID: u64 = 11_155_111;
const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const ARBOS_VERSION: u64 = 60;
const INVOKE_GAS_CAP: u64 = 30_000_000;
const DEPLOY_GAS_CAP: u64 = 150_000_000;

const ARBOWNER: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x70,
]);
const ARBWASM_ADDR: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x71,
]);
const SEQUENCER_ALIAS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65, 0x71, 0x75, 0x65,
    0x6e, 0x63, 0x65, 0x72,
]);
const FUNDER: Address = Address::new([0xa1; 20]);

/// `(memory 1)` footprint program that pays for an 8-page grow read from
/// calldata. At PageLimit=4 the cumulative open (1 footprint + 8 grow = 9)
/// exceeds the limit only inside the `addPages` host call. `i32.load` is
/// little-endian, so calldata is `pages.to_le_bytes()`.
const WAT_GROW: &str = r#"
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

/// Page operand the program grows by. 8 + footprint(1) = 9: above PageLimit=4
/// (positive) yet below the 128 exponent-saturation boundary, and far below
/// `u16::MAX` so the page-operand truncation path is never touched.
const GROW_PAGES: u32 = 8;
/// Lowered page limit that the cumulative open (9) exceeds.
const PAGE_LIMIT_POSITIVE: u16 = 4;
/// Ample page limit (the genesis default) that 9 stays under — the control.
const PAGE_LIMIT_CONTROL: u16 = 128;

fn owner_key() -> B256 {
    B256::from(keccak256(b"page-limit-owner"))
}

fn owner() -> Address {
    derive_address(owner_key())
}

fn selector4(sig: &str) -> [u8; 4] {
    let h = keccak256(sig.as_bytes());
    [h[0], h[1], h[2], h[3]]
}

fn signed(
    nonce: u64,
    to: Option<Address>,
    data: Bytes,
    value: U256,
    gas: u64,
    ts: u64,
) -> SignedL2TxBuilder {
    SignedL2TxBuilder {
        chain_id: L2_CHAIN_ID,
        nonce,
        to,
        value,
        data,
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
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
}

fn msg_step(idx: u64, msg: L1Message, dmr: u64) -> ScenarioStep {
    ScenarioStep::Message {
        idx,
        message: msg,
        delayed_messages_read: dmr,
    }
}

/// Stylus deploy init code: the classic program body
/// `0xEF 0xF0 0x00 0x00 ++ brotli(wasm)`, wrapped by a 14-byte deployer with
/// CODECOPY source-offset 0x0e. The body MUST be brotli-compressed: activation
/// reads byte 3 as the dictionary type and brotli-decompresses the remainder,
/// so a raw-WASM payload fails to decompress identically on both nodes and the
/// program never activates.
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

fn page_limit_calldata(limit: u16) -> Bytes {
    let mut d = selector4("setWasmPageLimit(uint16)").to_vec();
    let mut word = [0u8; 32];
    word[30..].copy_from_slice(&limit.to_be_bytes());
    d.extend_from_slice(&word);
    Bytes::from(d)
}

fn activate_calldata(addr: Address) -> Bytes {
    let mut d = vec![0x58, 0xc7, 0x80, 0xc2];
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(addr.as_slice());
    d.extend_from_slice(&pad);
    Bytes::from(d)
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

/// Tx hashes + addresses needed to assert the setup landed before judging the
/// page-limit outcome.
struct Built {
    scenario: Scenario,
    program: Address,
    deploy_hash: B256,
    activate_hash: B256,
    invoke_hash: B256,
}

/// fund owner -> setWasmPageLimit(limit) -> deploy Stylus -> activate ->
/// invoke(grow GROW_PAGES) -> trailing seal deposit. The program is deployed at
/// the owner EOA nonce 1 (nonce 0 is the page-limit setter).
fn build_scenario(name: &str, page_limit: u16) -> Built {
    let wasm = wat::parse_bytes(WAT_GROW.as_bytes())
        .expect("WAT compiles")
        .into_owned();
    let mut steps: Vec<ScenarioStep> = Vec::new();
    let o = owner();
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
            to: o,
            amount: U256::from(10u128).pow(U256::from(21u64)),
            l1_block_number: 1,
            timestamp: 1_700_000_000,
            request_seq: i,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        }
        .build()
        .expect("fund owner"),
        1,
    ));

    let i = next();
    steps.push(msg_step(
        i,
        signed(
            0,
            Some(ARBOWNER),
            page_limit_calldata(page_limit),
            U256::ZERO,
            3_000_000,
            1_700_000_000,
        )
        .build()
        .expect("set page limit"),
        1,
    ));

    let program = create_address(o, 1);
    let i = next();
    let deploy = signed(
        1,
        None,
        Bytes::from(build_init_code(&wasm)),
        U256::ZERO,
        DEPLOY_GAS_CAP,
        1_700_000_000,
    )
    .build()
    .expect("deploy stylus");
    let deploy_hash = signed_l2_tx_hash(&deploy).expect("deploy hash");
    steps.push(msg_step(i, deploy, 1));

    let i = next();
    let activate = signed(
        2,
        Some(ARBWASM_ADDR),
        activate_calldata(program),
        U256::from(10u128).pow(U256::from(15u64)),
        INVOKE_GAS_CAP,
        1_700_000_000,
    )
    .build()
    .expect("activate");
    let activate_hash = signed_l2_tx_hash(&activate).expect("activate hash");
    steps.push(msg_step(i, activate, 1));

    let i = next();
    let invoke = signed(
        3,
        Some(program),
        Bytes::from(GROW_PAGES.to_le_bytes().to_vec()),
        U256::ZERO,
        INVOKE_GAS_CAP,
        1_700_000_000,
    )
    .build()
    .expect("invoke grow");
    let invoke_hash = signed_l2_tx_hash(&invoke).expect("invoke hash");
    steps.push(msg_step(i, invoke, 1));

    let i = next();
    steps.push(msg_step(
        i,
        DepositBuilder {
            from: FUNDER,
            to: FUNDER,
            amount: U256::from(1u64),
            l1_block_number: 3,
            timestamp: 1_700_000_001,
            request_seq: i,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        }
        .build()
        .expect("seal deposit"),
        2,
    ));

    Built {
        scenario: Scenario {
            name: name.into(),
            description: "Stylus PageLimit enforcement parity".into(),
            setup: ScenarioSetup {
                l2_chain_id: L2_CHAIN_ID,
                arbos_version: ARBOS_VERSION,
                genesis: None,
            },
            steps,
        },
        program,
        deploy_hash,
        activate_hash,
        invoke_hash,
    }
}

/// Assert the deploy + activate landed (status 1) on BOTH nodes and the program
/// carries Stylus bytecode. Without this, a setup that fails identically on both
/// nodes would make the assertion pass trivially against a no-op.
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
fn page_limit_oog_matches_reference() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let mut rig = Rig::spawn(owner());
    let built = build_scenario("page_limit_4", PAGE_LIMIT_POSITIVE);
    let report = rig.dual.run(&built.scenario).expect("dual run");

    assert_setup_landed(&rig, &built);

    let invoke_status_differed = report
        .tx_diffs
        .iter()
        .any(|d| d.tx_hash == built.invoke_hash && d.field == "status");
    let state_root_differed = report.block_diffs.iter().any(|d| d.field == "state_root");

    let reference_invoke = rig
        .dual
        .left
        .receipt(built.invoke_hash)
        .expect("reference invoke");
    let arbreth_invoke = rig
        .dual
        .right
        .receipt(built.invoke_hash)
        .expect("arbreth invoke");

    if !(invoke_status_differed || state_root_differed) {
        eprintln!("page-limit difference not observed. Full report:");
        eprintln!("block_diffs:  {:#?}", report.block_diffs);
        eprintln!("tx_diffs:     {:#?}", report.tx_diffs);
        eprintln!("state_diffs:  {:#?}", report.state_diffs);
        eprintln!(
            "invoke: reference status={} gas={} | arbreth status={} gas={}",
            reference_invoke.status,
            reference_invoke.gas_used,
            arbreth_invoke.status,
            arbreth_invoke.gas_used,
        );
    }

    assert_eq!(
        reference_invoke.status, 0,
        "the reference must OOG the invoke (newOpen 9 > PageLimit 4 at arbos>=59)"
    );
    assert!(
        invoke_status_differed || state_root_differed,
        "expected the invoke tx status or the block state_root to differ when \
         cumulative open pages (9) exceed the lowered PageLimit (4) at arbos>=59"
    );
}

#[test]
#[ignore]
fn page_limit_control_clean() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let mut rig = Rig::spawn(owner());
    let built = build_scenario("page_limit_128", PAGE_LIMIT_CONTROL);
    let report = rig.dual.run(&built.scenario).expect("dual run");

    assert_setup_landed(&rig, &built);

    let reference_invoke = rig
        .dual
        .left
        .receipt(built.invoke_hash)
        .expect("reference invoke");
    assert_eq!(
        reference_invoke.status, 1,
        "control: newOpen 9 <= PageLimit 128 so the invoke must succeed"
    );
    assert!(
        report.is_clean(),
        "cumulative open pages (9) under the ample PageLimit (128) must agree on \
         both nodes; block_diffs={:#?} tx_diffs={:#?} state_diffs={:#?} log_diffs={:#?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
        report.log_diffs,
    );
}
