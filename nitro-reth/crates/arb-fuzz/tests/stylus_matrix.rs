//! Deterministic Stylus differential matrix vs Nitro.
//!
//! Drives the same DualExec the libfuzzer target uses, but iterates over a
//! deterministic seed/calldata grid so we can run it under `cargo test` and
//! enumerate divergences. Uses kind=3 sequencer-batch SignedL2Tx (EIP-1559)
//! for the deploy + activate + invoke trio, matching the shape of the
//! existing stylus/ fixtures.
//!
//! Marked `#[ignore]`. Run with:
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     ARB_STYLUS_MATRIX_LIMIT=200 \
//!     cargo test -p arb-fuzz --test stylus_matrix --release \
//!     -- --ignored --nocapture
//! Outputs to `/tmp/stylus_matrix/{summary.json, divergences/}`.

use std::{
    fs,
    path::PathBuf,
    sync::atomic::{AtomicU64, AtomicUsize, Ordering},
    time::Instant,
};

use alloy_primitives::{b256, keccak256, Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::{message_step, stylus::smith_wasm},
    shared_nodes::{shared_dual_exec, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::{
    messaging::{
        signed_tx::{derive_address, L2TxKind, SignedL2TxBuilder},
        DepositBuilder, MessageBuilder,
    },
    scenario::{Scenario, ScenarioSetup},
};

const FUZZ_ARBOS_VERSION: u64 = 60;
const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const FUZZ_GAS_CAP: u64 = 4_000_000;
const SEQUENCER_ALIAS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65, 0x71, 0x75, 0x65,
    0x6e, 0x63, 0x65, 0x72,
]);

static GLOBAL_MSG_IDX: AtomicU64 = AtomicU64::new(1);
static GLOBAL_DELAYED: AtomicU64 = AtomicU64::new(0);
static EOA_NONCE: AtomicU64 = AtomicU64::new(0);
static EOA_FUNDED: std::sync::OnceLock<()> = std::sync::OnceLock::new();

fn signing_key() -> B256 {
    b256!("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
}

fn eoa() -> Address {
    derive_address(signing_key())
}

const ARBWASM_ADDR: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x71,
]);

fn build_init_code(wasm: &[u8]) -> Vec<u8> {
    let mut body = Vec::with_capacity(3 + wasm.len());
    body.extend_from_slice(&[0xEF, 0xF0, 0x00]);
    body.extend_from_slice(wasm);
    let size = body.len();
    let size_hi = ((size >> 8) & 0xFF) as u8;
    let size_lo = (size & 0xFF) as u8;
    let mut out = Vec::with_capacity(12 + size);
    out.extend_from_slice(&[
        0x61, size_hi, size_lo, 0x60, 0x0c, 0x60, 0x00, 0x39, 0x61, size_hi, size_lo, 0x60, 0x00,
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
    let hash = keccak256(&rlp);
    Address::from_slice(&hash.as_slice()[12..])
}

#[derive(Clone)]
struct Case {
    name: String,
    wasm_seed: u64,
    calldata: Vec<u8>,
    gas_budget: u64,
}

fn matrix() -> Vec<Case> {
    let calldata_variants: &[(&str, Vec<u8>)] = &[
        ("empty", vec![]),
        ("zero4", vec![0; 4]),
        ("ones4", vec![0xff; 4]),
        ("zero32", vec![0; 32]),
        ("ones32", vec![0xff; 32]),
        ("seq32", (0..32).collect()),
        ("zero128", vec![0; 128]),
    ];
    let gas_variants: &[(&str, u64)] = &[("g_med", 2_000_000), ("g_max", FUZZ_GAS_CAP)];
    let wasm_seeds: Vec<u64> = (1..41u64).collect();
    let mut out = Vec::new();
    for seed in wasm_seeds {
        for (cd_name, cd) in calldata_variants {
            for (g_name, gas) in gas_variants {
                out.push(Case {
                    name: format!("seed{seed}_{cd_name}_{g_name}"),
                    wasm_seed: seed,
                    calldata: cd.clone(),
                    gas_budget: *gas,
                });
            }
        }
    }
    out
}

fn next_idx() -> u64 {
    GLOBAL_MSG_IDX.fetch_add(1, Ordering::Relaxed)
}

fn next_delayed() -> u64 {
    GLOBAL_DELAYED.fetch_add(1, Ordering::Relaxed) + 1
}

fn current_delayed() -> u64 {
    GLOBAL_DELAYED.load(Ordering::Relaxed)
}

fn next_nonce() -> u64 {
    EOA_NONCE.fetch_add(1, Ordering::Relaxed)
}

fn fund_step_once(steps: &mut Vec<arb_test_harness::scenario::ScenarioStep>) {
    if EOA_FUNDED.set(()).is_err() {
        return;
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
        let delayed = next_delayed();
        steps.push(message_step(idx, msg, delayed));
    }
}

fn signed_eip1559(nonce: u64, to: Option<Address>, data: Bytes, value: U256, gas: u64) -> SignedL2TxBuilder {
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
        kind: L2TxKind::Eip1559,
        signing_key: signing_key(),
        l1_block_number: 2,
        timestamp: 1_700_000_000,
        request_id: None,
        sender: SEQUENCER_ALIAS,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
}

fn build_scenario(case: &Case) -> Option<(Scenario, Vec<u8>, Address)> {
    let wasm = smith_wasm(case.wasm_seed).ok()?;
    if wasm.is_empty() {
        return None;
    }

    let mut steps = Vec::new();
    fund_step_once(&mut steps);

    let init_code = build_init_code(&wasm);
    let deploy_nonce = next_nonce();
    let activate_nonce = deploy_nonce + 1;
    let invoke_nonce = deploy_nonce + 2;
    let _ = (next_nonce(), next_nonce()); // consume the activate + invoke nonces

    let deploy_addr = create_address(eoa(), deploy_nonce);

    // Deploy
    let deploy = signed_eip1559(
        deploy_nonce,
        None,
        Bytes::from(init_code),
        U256::ZERO,
        case.gas_budget.clamp(1_000_000, FUZZ_GAS_CAP),
    );
    if let Ok(msg) = deploy.build() {
        let idx = next_idx();
        let delayed = current_delayed();
        steps.push(message_step(idx, msg, delayed));
    } else {
        return None;
    }

    // Activate (call ArbWASM.activateProgram(deploy_addr) with value)
    let mut activate_data = Vec::with_capacity(4 + 32);
    activate_data.extend_from_slice(&[0x58, 0xc7, 0x80, 0xc2]); // selector
    let mut padded_addr = [0u8; 32];
    padded_addr[12..32].copy_from_slice(deploy_addr.as_slice());
    activate_data.extend_from_slice(&padded_addr);
    let activate = signed_eip1559(
        activate_nonce,
        Some(ARBWASM_ADDR),
        Bytes::from(activate_data),
        U256::from(10u128).pow(U256::from(15u64)), // 0.001 ETH
        case.gas_budget.clamp(2_000_000, FUZZ_GAS_CAP),
    );
    if let Ok(msg) = activate.build() {
        let idx = next_idx();
        let delayed = current_delayed();
        steps.push(message_step(idx, msg, delayed));
    } else {
        return None;
    }

    // Invoke
    let invoke = signed_eip1559(
        invoke_nonce,
        Some(deploy_addr),
        Bytes::from(case.calldata.clone()),
        U256::ZERO,
        case.gas_budget.clamp(500_000, FUZZ_GAS_CAP),
    );
    if let Ok(msg) = invoke.build() {
        let idx = next_idx();
        let delayed = current_delayed();
        steps.push(message_step(idx, msg, delayed));
    } else {
        return None;
    }

    Some((
        Scenario {
            name: case.name.clone(),
            description: format!(
                "matrix scenario seed={} calldata_len={} gas={} deploy_addr={}",
                case.wasm_seed,
                case.calldata.len(),
                case.gas_budget,
                deploy_addr
            ),
            setup: ScenarioSetup {
                l2_chain_id: FUZZ_L2_CHAIN_ID,
                arbos_version: FUZZ_ARBOS_VERSION,
                genesis: None,
            },
            steps,
        },
        wasm,
        deploy_addr,
    ))
}

#[test]
#[ignore]
fn stylus_diff_matrix() {
    let out_dir = PathBuf::from(
        std::env::var("ARB_STYLUS_MATRIX_OUT")
            .unwrap_or_else(|_| "/tmp/stylus_matrix".to_string()),
    );
    let _ = fs::remove_dir_all(&out_dir);
    let div_dir = out_dir.join("divergences");
    fs::create_dir_all(&div_dir).expect("mkdir");

    let limit: usize = std::env::var("ARB_STYLUS_MATRIX_LIMIT")
        .ok()
        .and_then(|s| s.parse().ok())
        .unwrap_or(usize::MAX);

    let cases = matrix();
    eprintln!(
        "[stylus_matrix] {} cases, limit={}",
        cases.len(),
        if limit == usize::MAX { 0 } else { limit }
    );

    let nodes = shared_dual_exec();
    let total = AtomicUsize::new(0);
    let diverged = AtomicUsize::new(0);
    let harness_errs = AtomicUsize::new(0);

    let start = Instant::now();
    for (i, case) in cases.into_iter().enumerate().take(limit) {
        let Some((scen, wasm, deploy_addr)) = build_scenario(&case) else {
            continue;
        };

        let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
        match nodes.run(&scen) {
            Ok(report) => {
                let real_block_diffs: Vec<_> = report
                    .block_diffs
                    .iter()
                    .filter(|d| d.field != "state_root" && d.field != "parent_hash")
                    .collect();
                let real = !real_block_diffs.is_empty()
                    || !report.tx_diffs.is_empty()
                    || !report.state_diffs.is_empty()
                    || !report.log_diffs.is_empty();
                if real {
                    diverged.fetch_add(1, Ordering::Relaxed);
                    let payload = serde_json::json!({
                        "case": case.name,
                        "wasm_seed": case.wasm_seed,
                        "calldata_len": case.calldata.len(),
                        "gas_budget": case.gas_budget,
                        "calldata_hex": format!("0x{}", hex::encode(&case.calldata)),
                        "wasm_len": wasm.len(),
                        "wasm_hex": format!("0x{}", hex::encode(&wasm)),
                        "deploy_addr": format!("{deploy_addr}"),
                        "block_diffs_filtered": format!("{:#?}", real_block_diffs),
                        "tx_diffs": format!("{:#?}", report.tx_diffs),
                        "state_diffs": format!("{:#?}", report.state_diffs),
                        "log_diffs": format!("{:#?}", report.log_diffs),
                    });
                    let path = div_dir.join(format!("{}.json", case.name));
                    let _ = fs::write(&path, serde_json::to_vec_pretty(&payload).unwrap());
                    eprintln!(
                        "[stylus_matrix] DIVERGE [{i:04}] {} -> {}",
                        case.name,
                        path.display()
                    );
                }
            }
            Err(e) => {
                harness_errs.fetch_add(1, Ordering::Relaxed);
                eprintln!("[stylus_matrix] HARNESS ERR [{i:04}] {}: {e}", case.name);
            }
        }
        total.fetch_add(1, Ordering::Relaxed);

        let cur = total.load(Ordering::Relaxed);
        if cur % 25 == 0 {
            let dv = diverged.load(Ordering::Relaxed);
            let elapsed = start.elapsed().as_secs();
            eprintln!("[stylus_matrix] progress {cur} cases, {dv} diverged, {elapsed}s elapsed");
        }
    }

    let summary = serde_json::json!({
        "total_cases": total.load(Ordering::Relaxed),
        "diverged": diverged.load(Ordering::Relaxed),
        "harness_errors": harness_errs.load(Ordering::Relaxed),
        "elapsed_secs": start.elapsed().as_secs(),
    });
    fs::write(
        out_dir.join("summary.json"),
        serde_json::to_vec_pretty(&summary).unwrap(),
    )
    .expect("write summary");

    eprintln!("[stylus_matrix] done: {:#}", summary);
}
