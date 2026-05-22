//! End-to-end differential test for the Sepolia block 169,854,826 bug:
//! a Solidity factory makes a nested CALL into ArbWasm.activateProgram
//! with non-zero `value`. Pre-fix arbreth reverts the call with
//! `ProgramInsufficientValue` because the precompile reads its value from
//! a thread-local that `build.rs` only populates for outer-tx-to-0x71;
//! Nitro succeeds. Post-fix the precompile reads `input.value` (the
//! call-frame value revm already transferred to 0x71) and mirrors Nitro's
//! `payActivationDataFee` for the success path — so the two nodes match.
//!
//! Scenario:
//!   1. Fund EOA (deposit).
//!   2. Deploy a real Stylus program: a brotli-compressed program captured from Sepolia block
//!      115,184,744's alloc (already committed). Decompresses cleanly in the prover.
//!   3. Deploy a 34-byte raw-bytecode trampoline that forwards its calldata + `msg.value` to 0x71
//!      via a nested CALL.
//!   4. EOA → trampoline with `value = 0.001 ETH`, calldata = `activateProgram(stylus_addr)`.
//!
//! Pre-fix run shows tx-level divergence (status/gasUsed/logs); post-fix
//! run is clean.
//!
//! Run with:
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     cargo test -p arb-fuzz --test factory_activate_inner_call --release \
//!     -- --ignored --nocapture

use std::sync::{
    atomic::{AtomicU64, Ordering},
    OnceLock,
};

use alloy_primitives::{b256, keccak256, Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    shared_nodes::{fuzz_arbos_version, shared_dual_exec, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::{
    messaging::{
        signed_tx::{derive_address, L2TxKind, SignedL2TxBuilder},
        DepositBuilder, MessageBuilder,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
};

const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const FUZZ_GAS_CAP: u64 = 4_000_000;
const SEQUENCER_ALIAS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65, 0x71, 0x75, 0x65,
    0x6e, 0x63, 0x65, 0x72,
]);

static GLOBAL_MSG_IDX: AtomicU64 = AtomicU64::new(1);
static GLOBAL_DELAYED: AtomicU64 = AtomicU64::new(0);
static EOA_NONCE: AtomicU64 = AtomicU64::new(0);
static EOA_FUNDED: OnceLock<()> = OnceLock::new();

fn signing_key() -> B256 {
    b256!("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
}
fn eoa() -> Address {
    derive_address(signing_key())
}

// Real brotli-compressed Stylus contract runtime, captured from Sepolia
// block 115,184,744 (its sole Stylus program). Used as-is so activation
// reaches the data-fee check (uncompressed test WASM fails decompress and
// returns ProgramNotWasm well before the value check).
const STYLUS_RUNTIME_HEX: &str = include_str!("factory_activate_runtime.hex");

fn stylus_runtime() -> Vec<u8> {
    let s = STYLUS_RUNTIME_HEX.trim();
    (0..s.len())
        .step_by(2)
        .map(|i| u8::from_str_radix(&s[i..i + 2], 16).expect("hex"))
        .collect()
}

// Trampoline runtime: forwards 36-byte calldata + msg.value to 0x71.
// PC | bytes | op
//  0 | 36           | CALLDATASIZE
//  1 | 60 00        | PUSH1 0
//  3 | 60 00        | PUSH1 0
//  5 | 37           | CALLDATACOPY   ; mem[0..size] = calldata
//  6 | 60 00        | PUSH1 0        ; retsize
//  8 | 60 00        | PUSH1 0        ; retoff
// 10 | 36           | CALLDATASIZE   ; argsize
// 11 | 60 00        | PUSH1 0        ; argoff
// 13 | 34           | CALLVALUE      ; value
// 14 | 60 71        | PUSH1 0x71     ; addr
// 16 | 5a           | GAS
// 17 | f1           | CALL
// 18 | 15           | ISZERO
// 19 | 60 17        | PUSH1 0x17     ; revert handler
// 21 | 57           | JUMPI
// 22 | 00           | STOP
// 23 | 5b           | JUMPDEST 0x17
// 24 | 3d           | RETURNDATASIZE
// 25 | 60 00        | PUSH1 0
// 27 | 60 00        | PUSH1 0
// 29 | 3e           | RETURNDATACOPY
// 30 | 3d           | RETURNDATASIZE
// 31 | 60 00        | PUSH1 0
// 33 | fd           | REVERT
const TRAMPOLINE_RUNTIME: [u8; 34] = [
    0x36, 0x60, 0x00, 0x60, 0x00, 0x37, 0x60, 0x00, 0x60, 0x00, 0x36, 0x60, 0x00, 0x34, 0x60, 0x71,
    0x5a, 0xf1, 0x15, 0x60, 0x17, 0x57, 0x00, 0x5b, 0x3d, 0x60, 0x00, 0x60, 0x00, 0x3e, 0x3d, 0x60,
    0x00, 0xfd,
];

/// Deployer that RETURNs `runtime` as the contract's bytecode.
/// 14-byte deployer header + runtime; CODECOPY source-offset = 0x0e (== 14)
/// so the body bytes start at the correct index. The widely-copied template
/// in `stylus_matrix.rs` uses 0x0c, which prepends two stray bytes — that
/// breaks the Stylus prefix check downstream.
fn deploy_runtime_init_code(runtime: &[u8]) -> Vec<u8> {
    let size = runtime.len();
    let size_hi = ((size >> 8) & 0xFF) as u8;
    let size_lo = (size & 0xFF) as u8;
    let mut out = Vec::with_capacity(14 + size);
    out.extend_from_slice(&[
        0x61, size_hi, size_lo, 0x60, 0x0e, 0x60, 0x00, 0x39, 0x61, size_hi, size_lo, 0x60, 0x00,
        0xF3,
    ]);
    out.extend_from_slice(runtime);
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
    let hash = keccak256(&rlp);
    Address::from_slice(&hash.as_slice()[12..])
}

fn next_idx() -> u64 {
    GLOBAL_MSG_IDX.fetch_add(1, Ordering::Relaxed)
}
fn next_delayed_advance() -> u64 {
    GLOBAL_DELAYED.fetch_add(1, Ordering::Relaxed) + 1
}
fn current_delayed() -> u64 {
    GLOBAL_DELAYED.load(Ordering::Relaxed)
}
fn next_nonce() -> u64 {
    EOA_NONCE.fetch_add(1, Ordering::Relaxed)
}

fn signed_eip1559(
    nonce: u64,
    to: Option<Address>,
    data: Bytes,
    value: U256,
    gas: u64,
) -> SignedL2TxBuilder {
    SignedL2TxBuilder {
        chain_id: FUZZ_L2_CHAIN_ID,
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
        signing_key: signing_key(),
        l1_block_number: 2,
        timestamp: 1_700_000_000,
        request_id: None,
        sender: SEQUENCER_ALIAS,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
}

fn fund_step_once() -> Vec<ScenarioStep> {
    let mut out = Vec::new();
    if EOA_FUNDED.set(()).is_err() {
        return out;
    }
    let dep = DepositBuilder {
        from: eoa(),
        to: eoa(),
        amount: U256::from(10u128).pow(U256::from(20u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: 0,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    };
    if let Ok(msg) = dep.build() {
        let idx = next_idx();
        let delayed = next_delayed_advance();
        out.push(message_step(idx, msg, delayed));
    }
    out
}

#[test]
#[ignore]
fn factory_activate_inner_call_matches_canon() {
    let nodes = shared_dual_exec();

    // 1. Fund EOA.
    {
        let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
        let steps = fund_step_once();
        if !steps.is_empty() {
            let scen = Scenario {
                name: "fund_eoa".into(),
                description: "fund factory_activate_inner_call EOA".into(),
                setup: ScenarioSetup {
                    l2_chain_id: FUZZ_L2_CHAIN_ID,
                    arbos_version: fuzz_arbos_version(),
                    genesis: None,
                },
                steps,
            };
            nodes.run(&scen).expect("fund scenario");
        }
    }

    // 2. Deploy Stylus program (captured brotli-compressed runtime).
    let runtime = stylus_runtime();
    let deploy_nonce = next_nonce();
    let stylus_addr = create_address(eoa(), deploy_nonce);
    let deploy_msg = signed_eip1559(
        deploy_nonce,
        None,
        Bytes::from(deploy_runtime_init_code(&runtime)),
        U256::ZERO,
        FUZZ_GAS_CAP,
    )
    .build()
    .expect("build stylus-deploy tx");

    // 3. Deploy trampoline contract.
    let tramp_nonce = next_nonce();
    let tramp_addr = create_address(eoa(), tramp_nonce);
    let tramp_msg = signed_eip1559(
        tramp_nonce,
        None,
        Bytes::from(deploy_runtime_init_code(&TRAMPOLINE_RUNTIME)),
        U256::ZERO,
        FUZZ_GAS_CAP,
    )
    .build()
    .expect("build trampoline-deploy tx");

    // 4. EOA → trampoline {value: 0.001 ETH} (calldata = activateProgram selector + stylus_addr).
    let mut activate_data = Vec::with_capacity(4 + 32);
    activate_data.extend_from_slice(&[0x58, 0xc7, 0x80, 0xc2]); // activateProgram(address)
    let mut padded = [0u8; 32];
    padded[12..].copy_from_slice(stylus_addr.as_slice());
    activate_data.extend_from_slice(&padded);

    let activate_nonce = next_nonce();
    let activate_msg = signed_eip1559(
        activate_nonce,
        Some(tramp_addr),
        Bytes::from(activate_data),
        U256::from(10u128).pow(U256::from(15u64)), // 0.001 ETH
        FUZZ_GAS_CAP,
    )
    .build()
    .expect("build activate-via-trampoline tx");

    let mut steps = Vec::new();
    let idx = next_idx();
    let delayed = current_delayed();
    steps.push(message_step(idx, deploy_msg, delayed));
    let idx = next_idx();
    let delayed = current_delayed();
    steps.push(message_step(idx, tramp_msg, delayed));
    let idx = next_idx();
    let delayed = current_delayed();
    steps.push(message_step(idx, activate_msg, delayed));

    let scen = Scenario {
        name: "factory_activate_inner_call".into(),
        description: "trampoline -> ArbWasm.activateProgram{value:V}(stylus)".into(),
        setup: ScenarioSetup {
            l2_chain_id: FUZZ_L2_CHAIN_ID,
            arbos_version: fuzz_arbos_version(),
            genesis: None,
        },
        steps,
    };

    let report = {
        let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
        nodes.run(&scen).expect("run factory_activate scenario")
    };

    let real_block: Vec<_> = report.block_diffs.iter().collect();

    if !real_block.is_empty()
        || !report.tx_diffs.is_empty()
        || !report.state_diffs.is_empty()
        || !report.log_diffs.is_empty()
    {
        let payload = serde_json::json!({
            "block_diffs": format!("{:#?}", real_block),
            "tx_diffs": format!("{:#?}", report.tx_diffs),
            "state_diffs": format!("{:#?}", report.state_diffs),
            "log_diffs": format!("{:#?}", report.log_diffs),
        });
        let path = std::path::PathBuf::from("/tmp/factory_activate_inner_call.json");
        let _ = std::fs::write(&path, serde_json::to_vec_pretty(&payload).unwrap());
        panic!(
            "arbreth diverged from Nitro on factory-mediated activate; see {}",
            path.display()
        );
    }
}
