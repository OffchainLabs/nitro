//! Deep differential matrix: Stylus -> 7702-delegated EOA, sweeping
//! call type × delegate shape × calldata at ArbOS v50 and v51.
//!
//! Goes beyond `stylus_calls_7702.rs` (which only covers the 213M baseline
//! CALL + empty-calldata + log-emitter-delegate case) to surface edge cases
//! in the 7702 sub-call path my fix touches.
//!
//! Matrix axes:
//!   call_type:   CALL (sol_caller.forward), STATICCALL (forward_static)
//!   delegate:    log-emitter, reverter, storage-writer, return-data
//!   calldata:    empty, junk bytes, abi-encoded selector
//!
//! Skipped combinations: STATICCALL + value (forbidden by EVM),
//! STATICCALL + storage-writer (would revert; uninteresting).
//!
//! Run command:
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     NITRO_REF_IMAGE=offchainlabs/nitro-node:v3.10.0-rc.10-b1cf6db \
//!     ARB_FUZZ_ARBOS_VERSION=50 \
//!     cargo test -p arb-fuzz --test stylus_7702_matrix --release \
//!     -- --ignored --nocapture
//!
//! Re-run with ARB_FUZZ_ARBOS_VERSION=51 for the v51 sweep.

use alloy_primitives::{keccak256, Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::{interop::wrap_init_code, message_step},
    shared_nodes::{fuzz_arbos_version, next_msg_idx, shared_dual_exec, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::{
    messaging::{
        signed_tx::{derive_address, AuthorizationItem, L2TxKind, SignedL2TxBuilder},
        DepositBuilder, MessageBuilder,
    },
    scenario::{Scenario, ScenarioSetup, ScenarioStep},
};

const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const INVOKE_GAS_CAP: u64 = 30_000_000;
const DEPLOY_GAS_CAP: u64 = 150_000_000;
const ARBWASM_ADDR: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x71,
]);
const SEQUENCER_ALIAS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65, 0x71, 0x75, 0x65,
    0x6e, 0x63, 0x65, 0x72,
]);

/// Deterministic key from a seed byte. Keeps scenarios isolated.
fn key_from_seed(seed: u8) -> B256 {
    let mut k = [0u8; 32];
    // Avoid leading-zero / order-overflow risk by ensuring first byte is 1
    // and last byte is the seed; secp256k1 secret keys are valid for any
    // 32 bytes in (0, N).
    k[0] = 0x01;
    k[31] = seed | 0x01;
    B256::from(k)
}

fn selector(sig: &str) -> [u8; 4] {
    let h = keccak256(sig.as_bytes());
    [h[0], h[1], h[2], h[3]]
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
            trimmed.to_vec()
        } else {
            let mut v = vec![0x80 + trimmed.len() as u8];
            v.extend_from_slice(trimmed);
            v
        }
    };
    let mut rlp = Vec::with_capacity(2 + 20 + nonce_rlp.len());
    rlp.push(0xc0 + (21 + nonce_rlp.len() - 1) as u8);
    rlp.push(0x94);
    rlp.extend_from_slice(sender.as_slice());
    rlp.extend_from_slice(&nonce_rlp);
    Address::from_slice(&keccak256(&rlp)[12..])
}

// ── Delegate-shape runtimes ────────────────────────────────────────────

/// LOG1 (topic=0xaa, no data) then RETURN. Always succeeds.
fn delegate_log_emitter() -> Vec<u8> {
    vec![0x60, 0xaa, 0x60, 0x00, 0x60, 0x00, 0xa1, 0x60, 0x00, 0x60, 0x00, 0xf3]
}

/// REVERT with empty data.
fn delegate_reverter() -> Vec<u8> {
    vec![0x60, 0x00, 0x60, 0x00, 0xfd]
}

/// Write 0x42 to storage slot 0, then RETURN. Tests storage flowing to the
/// 7702 EOA (not the delegate) per EIP-7702.
fn delegate_storage_writer() -> Vec<u8> {
    // PUSH1 0x42 (value)
    // PUSH1 0x00 (slot)
    // SSTORE
    // PUSH1 0x00 (size)
    // PUSH1 0x00 (offset)
    // RETURN
    vec![0x60, 0x42, 0x60, 0x00, 0x55, 0x60, 0x00, 0x60, 0x00, 0xf3]
}

/// Returns 32 bytes of 0xab. Tests return-data plumbing.
fn delegate_return_data() -> Vec<u8> {
    // PUSH32 0xababab... (push 0xab 32 times - need a PUSH32 of 32 bytes of 0xab)
    // PUSH1 0   MSTORE
    // PUSH1 32  PUSH1 0  RETURN
    let mut out = Vec::with_capacity(64);
    out.push(0x7f); // PUSH32
    out.extend_from_slice(&[0xabu8; 32]);
    out.extend_from_slice(&[0x60, 0x00, 0x52]); // PUSH1 0 MSTORE
    out.extend_from_slice(&[0x60, 0x20, 0x60, 0x00, 0xf3]); // PUSH1 32 PUSH1 0 RETURN
    out
}

#[derive(Debug, Clone, Copy)]
enum CallType {
    Call,
    Static,
}

impl CallType {
    fn forward_selector(&self) -> [u8; 4] {
        match self {
            CallType::Call => selector("forward(address,bytes)"),
            CallType::Static => selector("forward_static(address,bytes)"),
        }
    }
}

struct Variant {
    label: &'static str,
    delegate_runtime: Vec<u8>,
    call_type: CallType,
    inner_calldata: Vec<u8>,
}

fn build_matrix() -> Vec<Variant> {
    // 9 combinations covering all interesting cells.
    vec![
        Variant {
            label: "CALL_log_emitter_empty",
            delegate_runtime: delegate_log_emitter(),
            call_type: CallType::Call,
            inner_calldata: vec![],
        },
        Variant {
            label: "CALL_log_emitter_junk",
            delegate_runtime: delegate_log_emitter(),
            call_type: CallType::Call,
            inner_calldata: vec![0xde, 0xad, 0xbe, 0xef, 0xff, 0x00, 0x11, 0x22],
        },
        Variant {
            label: "CALL_reverter_empty",
            delegate_runtime: delegate_reverter(),
            call_type: CallType::Call,
            inner_calldata: vec![],
        },
        Variant {
            label: "CALL_reverter_selector",
            delegate_runtime: delegate_reverter(),
            call_type: CallType::Call,
            inner_calldata: selector("dummy()").to_vec(),
        },
        Variant {
            label: "CALL_storage_writer",
            delegate_runtime: delegate_storage_writer(),
            call_type: CallType::Call,
            inner_calldata: vec![],
        },
        Variant {
            label: "CALL_return_data",
            delegate_runtime: delegate_return_data(),
            call_type: CallType::Call,
            inner_calldata: vec![],
        },
        Variant {
            label: "STATIC_log_emitter_empty",
            delegate_runtime: delegate_log_emitter(),
            call_type: CallType::Static,
            inner_calldata: vec![],
        },
        Variant {
            label: "STATIC_reverter_empty",
            delegate_runtime: delegate_reverter(),
            call_type: CallType::Static,
            inner_calldata: vec![],
        },
        Variant {
            label: "STATIC_return_data",
            delegate_runtime: delegate_return_data(),
            call_type: CallType::Static,
            inner_calldata: vec![],
        },
    ]
}

fn signed_with_key(
    signing_key: B256,
    nonce: u64,
    to: Option<Address>,
    data: Bytes,
    value: U256,
    gas: u64,
    auths: Vec<AuthorizationItem>,
) -> SignedL2TxBuilder {
    let kind = if auths.is_empty() {
        L2TxKind::Eip1559
    } else {
        L2TxKind::Eip7702
    };
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
        authorization_list: auths,
        kind,
        signing_key,
        l1_block_number: 2,
        timestamp: 1_700_000_000,
        request_id: None,
        sender: SEQUENCER_ALIAS,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
}

fn run_variant(variant: &Variant, seed: u8) {
    let caller_key = key_from_seed(seed);
    let victim_key = key_from_seed(seed.wrapping_add(0x80));
    let caller = derive_address(caller_key);
    let victim = derive_address(victim_key);
    assert_ne!(caller, victim);

    let mut steps: Vec<ScenarioStep> = Vec::new();

    // 1. Fund caller
    let idx = next_msg_idx();
    let fund_caller = DepositBuilder {
        from: caller,
        to: caller,
        amount: U256::from(10u128).pow(U256::from(20u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("fund caller");
    steps.push(message_step(idx, fund_caller, idx));

    // 2. Fund victim
    let idx = next_msg_idx();
    let fund_victim = DepositBuilder {
        from: victim,
        to: victim,
        amount: U256::from(10u128).pow(U256::from(18u64)),
        l1_block_number: 1,
        timestamp: 1_700_000_000,
        request_seq: idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
    .build()
    .expect("fund victim");
    steps.push(message_step(idx, fund_victim, idx));

    // 3. Deploy delegate runtime
    let delegate_nonce = 0u64;
    let delegate_addr = create_address(caller, delegate_nonce);
    let deploy_delegate = signed_with_key(
        caller_key,
        delegate_nonce,
        None,
        Bytes::from(wrap_init_code(&variant.delegate_runtime)),
        U256::ZERO,
        4_000_000,
        Vec::new(),
    )
    .build()
    .expect("deploy delegate");
    let idx = next_msg_idx();
    steps.push(message_step(idx, deploy_delegate, idx));

    // 4. SetCodeTx from victim authorising delegate
    let auth = AuthorizationItem {
        chain_id: FUZZ_L2_CHAIN_ID,
        address: delegate_addr,
        nonce: 0,
        signing_key: victim_key,
    };
    let set_code = signed_with_key(
        victim_key,
        0,
        Some(delegate_addr),
        Bytes::new(),
        U256::ZERO,
        INVOKE_GAS_CAP,
        vec![auth],
    )
    .build()
    .expect("set code");
    let idx = next_msg_idx();
    steps.push(message_step(idx, set_code, idx));

    // 5. Deploy SolCaller
    let sol_caller_nonce = 1u64;
    let sol_caller_addr = create_address(caller, sol_caller_nonce);
    let sol_caller_hex = include_str!("../prebuilt/sol_caller.hex").trim();
    let sol_caller_initcode: Vec<u8> = (0..sol_caller_hex.len())
        .step_by(2)
        .map(|i| u8::from_str_radix(&sol_caller_hex[i..i + 2], 16).expect("hex"))
        .collect();
    let deploy_stylus = signed_with_key(
        caller_key,
        sol_caller_nonce,
        None,
        Bytes::from(sol_caller_initcode),
        U256::ZERO,
        DEPLOY_GAS_CAP,
        Vec::new(),
    )
    .build()
    .expect("deploy stylus");
    let idx = next_msg_idx();
    steps.push(message_step(idx, deploy_stylus, idx));

    // 6. activateProgram
    let mut activate_data = Vec::with_capacity(36);
    activate_data.extend_from_slice(&selector("activateProgram(address)"));
    let mut padded = [0u8; 32];
    padded[12..].copy_from_slice(sol_caller_addr.as_slice());
    activate_data.extend_from_slice(&padded);
    let activate = signed_with_key(
        caller_key,
        2,
        Some(ARBWASM_ADDR),
        Bytes::from(activate_data),
        U256::from(10u128).pow(U256::from(15u64)),
        INVOKE_GAS_CAP,
        Vec::new(),
    )
    .build()
    .expect("activate");
    let idx = next_msg_idx();
    steps.push(message_step(idx, activate, idx));

    // 7. caller -> SolCaller.forward[_static](victim, inner_calldata)
    let mut forward_data = Vec::with_capacity(4 + 32 + 32 + 32 + variant.inner_calldata.len());
    forward_data.extend_from_slice(&variant.call_type.forward_selector());
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(victim.as_slice());
    forward_data.extend_from_slice(&pad); // address target
    forward_data.extend_from_slice(&{
        let mut b = [0u8; 32];
        b[31] = 0x40;
        b
    }); // offset to bytes
    let mut len_word = [0u8; 32];
    let len = variant.inner_calldata.len() as u64;
    len_word[24..].copy_from_slice(&len.to_be_bytes());
    forward_data.extend_from_slice(&len_word);
    forward_data.extend_from_slice(&variant.inner_calldata);
    // ABI pads bytes to 32-byte boundary
    let pad_len = (32 - (variant.inner_calldata.len() % 32)) % 32;
    forward_data.extend_from_slice(&vec![0u8; pad_len]);

    let forward = signed_with_key(
        caller_key,
        3,
        Some(sol_caller_addr),
        Bytes::from(forward_data),
        U256::ZERO,
        INVOKE_GAS_CAP,
        Vec::new(),
    )
    .build()
    .expect("forward");
    let idx = next_msg_idx();
    steps.push(message_step(idx, forward, idx));

    let scen = Scenario {
        name: format!("matrix_{}_seed{}", variant.label, seed),
        description: format!(
            "Stylus 7702 matrix: variant={}",
            variant.label
        ),
        setup: ScenarioSetup {
            l2_chain_id: FUZZ_L2_CHAIN_ID,
            arbos_version: fuzz_arbos_version(),
            genesis: None,
        },
        steps,
    };

    let nodes = shared_dual_exec();
    let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
    let report = nodes.run(&scen).expect("run matrix variant");

    if !report.block_diffs.is_empty()
        || !report.state_diffs.is_empty()
        || !report.log_diffs.is_empty()
    {
        let payload = serde_json::json!({
            "variant": variant.label,
            "arbos_version": fuzz_arbos_version(),
            "block_diffs": format!("{:#?}", report.block_diffs),
            "tx_diffs": format!("{:#?}", report.tx_diffs),
            "state_diffs": format!("{:#?}", report.state_diffs),
            "log_diffs": format!("{:#?}", report.log_diffs),
        });
        let path = std::path::PathBuf::from(format!(
            "/tmp/stylus_7702_matrix_v{}_{}.json",
            fuzz_arbos_version(),
            variant.label
        ));
        let _ = std::fs::write(&path, serde_json::to_vec_pretty(&payload).unwrap());
        panic!(
            "consensus divergence on {} at v{}; see {}",
            variant.label,
            fuzz_arbos_version(),
            path.display()
        );
    }
    if !report.tx_diffs.is_empty() {
        // Only RPC-level differences (e.g. effective_gas_price) — log and
        // continue. State, block, and logs all match Nitro.
        eprintln!(
            "[{}] {} tx-level diffs (non-consensus): {:#?}",
            variant.label,
            report.tx_diffs.len(),
            report.tx_diffs
        );
    }
}

#[test]
#[ignore]
fn deep_stylus_7702_matrix_matches_canon() {
    let matrix = build_matrix();
    let mut seed: u8 = 1;
    for variant in &matrix {
        eprintln!(
            "==== {} (v{}) ====",
            variant.label,
            fuzz_arbos_version()
        );
        run_variant(variant, seed);
        seed = seed.wrapping_add(1);
    }
    eprintln!(
        "ALL {} variants clean (consensus) at v{}",
        matrix.len(),
        fuzz_arbos_version()
    );
}
