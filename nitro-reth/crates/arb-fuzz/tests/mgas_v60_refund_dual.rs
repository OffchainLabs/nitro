//! Differential test for the v60 multi-gas refund against Nitro.
//!
//! The per-transaction refund (`max(0, header*gasUsed - sum(fee*rawGas))`) only
//! fires when the header base fee exceeds the next-block `base_fee_wei` floor —
//! i.e. when the base fee is falling. This lowers the speed limit and bursts gas
//! to lift the base fee above the floor, then runs a stream of refund-sensitive
//! transactions (storage sets, storage clears that mint an EIP-3529 refund, and
//! calls that perform an internal create) across the blocks where it relaxes
//! toward the floor. State root, receipts and per-resource multi-gas must match
//! Nitro on every block; the owner's balance (the refund's sink) must match.
//!
//! Run:
//!   ARB_SPEC_BINARY=$(pwd)/target/fastdev/arb-reth \
//!     cargo test -p arb-fuzz --test mgas_v60_refund_dual --release \
//!     -- --ignored --nocapture

use std::sync::Mutex;

static SERIAL: Mutex<()> = Mutex::new(());

use alloy_primitives::{address, keccak256, Address, Bytes, B256, U256};
use arb_fuzz::{arbitrary_impls::interop::wrap_init_code, scaffolding::selector4};
use arb_test_harness::{
    dual_exec::DualExec,
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

const L2_CHAIN_ID: u64 = 412_348;
const L1_CHAIN_ID: u64 = 11_155_111;
const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const ARBOS_VERSION: u64 = 60;
const MIN_BASE_FEE: u128 = 100_000_000; // genesis floor (0.1 Gwei)

const ARBOWNER: Address = address!("0000000000000000000000000000000000000070");
const FUNDER: Address = Address::new([0xa1; 20]);

fn owner_key() -> B256 {
    B256::repeat_byte(0x42)
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

/// Stores calldata[0..32] at slot 0: a nonzero value is a set (growth on the
/// first write), a zero value clears the slot and mints an EIP-3529 refund.
const STORE_RUNTIME: [u8; 7] = [0x60, 0x00, 0x35, 0x60, 0x00, 0x55, 0x00];

/// Performs one internal CREATE of a 32-byte-runtime child, then STOP — a
/// nested create frame, whose gas must stay attributed to this transaction.
const FACTORY_RUNTIME: [u8; 18] = [
    0x64, 0x60, 0x20, 0x60, 0x00, 0xf3, // PUSH5 <init: PUSH1 0x20 PUSH1 0 RETURN>
    0x60, 0x00, 0x52, // PUSH1 0 MSTORE
    0x60, 0x05, 0x60, 0x1b, 0x60, 0x00, 0xf0, // PUSH1 5 PUSH1 27 PUSH1 0 CREATE
    0x50, 0x00, // POP STOP
];

/// Burns gas by doing `calldata[0]` fresh SSTOREs (slot i = i). Used to grow the
/// gas backlog so the base fee climbs above the floor.
const BURNER_RUNTIME: [u8; 21] = [
    0x60, 0x00, 0x35, // PUSH1 0 CALLDATALOAD  (N)
    0x5b, // JUMPDEST (L=3)
    0x80, 0x15, 0x60, 0x13, 0x57, // DUP1 ISZERO PUSH1 0x13 JUMPI (end)
    0x80, 0x80, 0x55, // DUP1 DUP1 SSTORE  (slot N = N)
    0x60, 0x01, 0x90, 0x03, // PUSH1 1 SWAP1 SUB  (N-1)
    0x60, 0x03, 0x56, // PUSH1 3 JUMP  (loop)
    0x5b, 0x00, // JUMPDEST (end=0x13) STOP
];

fn set_speed_limit(limit: u64) -> Vec<u8> {
    let mut d = selector4("setSpeedLimit(uint64)").to_vec();
    d.extend_from_slice(&word(limit));
    d
}

const BASE_TS: u64 = 1_700_000_000;
// Seconds between blocks: the backlog decays by speed_limit * this each block,
// so the base fee relaxes toward the floor and the header exceeds the next
// block's slot — the window where the refund fires.
const BLOCK_SECS: u64 = 12;

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
        // The L1 block advances with the timestamp so the L2 block time (which
        // drives backlog decay, hence the falling base fee) advances too.
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
fn v60_refund_stream_matches_nitro() {
    let _serial = SERIAL.lock().unwrap_or_else(|e| e.into_inner());
    let owner = derive_address(owner_key());
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

    // Lower the speed limit so a single burst grows the backlog above the
    // tolerance and the base fee climbs above the floor; the refund stream then
    // uses less gas than the per-block decay, so the base fee relaxes and the
    // header exceeds the next-block slot on every block (the refund window).
    let i = next();
    steps.push(msg_step(
        i,
        tx(
            0,
            Some(ARBOWNER),
            set_speed_limit(10_000),
            3_000_000,
            BASE_TS + i * BLOCK_SECS,
        )
        .build()
        .expect("setSpeedLimit"),
        1,
    ));

    // Deploy the storage, factory and gas-burner contracts.
    let i = next();
    steps.push(msg_step(
        i,
        tx(
            1,
            None,
            wrap_init_code(&STORE_RUNTIME),
            3_000_000,
            BASE_TS + i * BLOCK_SECS,
        )
        .build()
        .expect("deploy store"),
        1,
    ));
    let store = create_address(owner, 1);
    let i = next();
    steps.push(msg_step(
        i,
        tx(
            2,
            None,
            wrap_init_code(&FACTORY_RUNTIME),
            3_000_000,
            BASE_TS + i * BLOCK_SECS,
        )
        .build()
        .expect("deploy factory"),
        1,
    ));
    let factory = create_address(owner, 2);
    let i = next();
    steps.push(msg_step(
        i,
        tx(
            3,
            None,
            wrap_init_code(&BURNER_RUNTIME),
            3_000_000,
            BASE_TS + i * BLOCK_SECS,
        )
        .build()
        .expect("deploy burner"),
        1,
    ));
    let burner = create_address(owner, 3);

    // One burst (~2M gas of fresh SSTOREs) to lift the backlog above tolerance.
    let i = next();
    steps.push(msg_step(
        i,
        tx(
            4,
            Some(burner),
            word(100).to_vec(),
            4_000_000,
            BASE_TS + i * BLOCK_SECS,
        )
        .build()
        .expect("burn"),
        1,
    ));

    // Refund-sensitive stream across the relaxing base fee: storage set, an
    // internal-create call (the next tx must not inherit its gas), a storage
    // clear (EIP-3529 refund — the term that flips the refund's sign), repeated.
    let mut nonce = 5u64;
    let mut push_call = |steps: &mut Vec<ScenarioStep>, to: Address, data: Vec<u8>, i: u64| {
        steps.push(msg_step(
            i,
            tx(nonce, Some(to), data, 3_000_000, BASE_TS + i * BLOCK_SECS)
                .build()
                .expect("call"),
            1,
        ));
        nonce += 1;
    };
    for round in 0..4u64 {
        let i = next();
        push_call(&mut steps, store, word(7 + round).to_vec(), i);
        let i = next();
        push_call(&mut steps, factory, Vec::new(), i);
        let i = next();
        push_call(&mut steps, store, word(0).to_vec(), i); // clear → EIP-3529 refund
        let i = next();
        push_call(&mut steps, factory, Vec::new(), i);
    }

    let scenario = Scenario {
        name: "v60_refund_stream".into(),
        description: "refund across a relaxing base fee with creates and clears".into(),
        setup: ScenarioSetup {
            l2_chain_id: L2_CHAIN_ID,
            arbos_version: ARBOS_VERSION,
            genesis: None,
        },
        steps,
    };

    let checks = [StateCheck {
        address: owner,
        slots: vec![],
        check_balance: true,
        check_nonce: true,
        check_code: false,
    }];
    let report = rig
        .dual
        .run_with_state_checks(&scenario, &checks)
        .expect("dual run");

    // Diagnostics: confirm the refund regime was genuinely exercised (some
    // block's base fee above the floor), otherwise the comparison is vacuous.
    let max_n = rig.dual.right.block(BlockId::Latest).expect("head").number;
    let mut saw_above_floor = false;
    for n in 1..=max_n {
        let l = rig.dual.left.block(BlockId::Number(n)).ok();
        let r = rig.dual.right.block(BlockId::Number(n)).ok();
        let lf = l.as_ref().and_then(|b| b.base_fee_per_gas).unwrap_or(0);
        let rf = r.as_ref().and_then(|b| b.base_fee_per_gas).unwrap_or(0);
        if rf > MIN_BASE_FEE {
            saw_above_floor = true;
        }
        if lf != rf {
            println!("block {n}: base_fee DIVERGE nitro={lf} arbreth={rf}");
        }
    }

    assert!(
        report.is_clean(),
        "v60 refund stream diverged from Nitro:\n  blocks={:?}\n  txs={:?}\n  state={:?}",
        report.block_diffs,
        report.tx_diffs,
        report.state_diffs,
    );
    assert!(
        saw_above_floor,
        "base fee never rose above the floor — refund path not exercised",
    );

    // Preconditions: the contracts deployed and the stream executed.
    assert!(
        !rig.dual
            .right
            .code(store, BlockId::Latest)
            .expect("code")
            .is_empty(),
        "store contract did not deploy",
    );
    let final_nonce = rig.dual.right.nonce(owner, BlockId::Latest).expect("nonce");
    assert!(
        final_nonce >= 21,
        "stream did not fully execute (nonce {final_nonce})"
    );
}
