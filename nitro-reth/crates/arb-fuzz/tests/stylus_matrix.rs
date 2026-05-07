//! Deterministic Stylus differential matrix vs Nitro.
//!
//! Drives the same DualExec the libfuzzer target uses, but iterates over a
//! deterministic seed/calldata grid so we can run it under `cargo test` and
//! enumerate divergences.
//!
//! Requires Docker + the Nitro reference image. Marked `#[ignore]`; run with
//!   ARB_STYLUS_MATRIX_LIMIT=200 cargo test -p arb-fuzz --test stylus_matrix \
//!     --release -- --ignored --nocapture
//! Outputs to `/tmp/stylus_matrix/{summary.json, divergences/}`.

use std::{
    fs,
    path::PathBuf,
    sync::atomic::{AtomicU64, AtomicUsize, Ordering},
    time::Instant,
};

static GLOBAL_MSG_IDX: AtomicU64 = AtomicU64::new(1);
static GLOBAL_DELAYED: AtomicU64 = AtomicU64::new(0);

use alloy_primitives::{Address, Bytes, U256};
use arb_fuzz::{
    arbitrary_impls::{message_step, stylus::smith_wasm},
    shared_nodes::{shared_dual_exec, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::{
    messaging::{ContractTxBuilder, DepositBuilder, MessageBuilder},
    scenario::{Scenario, ScenarioSetup},
};

const FUZZ_ARBOS_VERSION: u64 = 60;
const FUZZ_L1_BASE_FEE: u64 = 30_000_000_000;
const FUZZ_GAS_CAP: u64 = 4_000_000;

fn deployer() -> Address {
    let mut b = [0u8; 20];
    b[0] = 0xab;
    b[19] = 0xcd;
    Address::new(b)
}

fn caller() -> Address {
    let mut b = [0u8; 20];
    b[0] = 0xfe;
    b[19] = 0xed;
    Address::new(b)
}

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
    use alloy_primitives::keccak256;
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
        ("zero64", vec![0; 64]),
        ("zero128", vec![0; 128]),
        ("zero512", vec![0; 512]),
    ];
    let gas_variants: &[(&str, u64)] = &[
        ("g_min", 500_000),
        ("g_med", 1_500_000),
        ("g_max", FUZZ_GAS_CAP),
    ];
    let wasm_seeds: Vec<u64> = (0..40u64).collect();
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

fn build_scenario(case: &Case) -> Option<(Scenario, Vec<u8>)> {
    let wasm = smith_wasm(case.wasm_seed).ok()?;
    if wasm.is_empty() {
        return None;
    }

    let mut steps = Vec::new();
    let mut idx: u64 = GLOBAL_MSG_IDX.load(Ordering::Relaxed);
    let mut delayed: u64 = GLOBAL_DELAYED.load(Ordering::Relaxed);
    let starting_idx = idx;

    for addr in [deployer(), caller()] {
        let dep = DepositBuilder {
            from: addr,
            to: addr,
            amount: U256::from(10u128).pow(U256::from(20u64)),
            l1_block_number: 1,
            timestamp: 1_700_000_000,
            request_seq: idx,
            base_fee_l1: FUZZ_L1_BASE_FEE,
        };
        if let Ok(msg) = dep.build() {
            delayed += 1;
            steps.push(message_step(idx, msg, delayed));
            idx += 1;
        }
    }

    let init_code = build_init_code(&wasm);
    let create = ContractTxBuilder {
        from: deployer(),
        gas_limit: case.gas_budget.clamp(500_000, FUZZ_GAS_CAP),
        max_fee_per_gas: U256::from(2_000_000_000u64),
        to: Address::ZERO,
        value: U256::ZERO,
        data: Bytes::from(init_code),
        l1_block_number: 1,
        timestamp: 1_700_000_001,
        request_seq: idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    };
    if let Ok(msg) = create.build() {
        delayed += 1;
        steps.push(message_step(idx, msg, delayed));
        idx += 1;
    }

    let probe_addr = create_address(deployer(), 0);
    let invoke = ContractTxBuilder {
        from: caller(),
        gas_limit: case.gas_budget.clamp(200_000, FUZZ_GAS_CAP),
        max_fee_per_gas: U256::from(2_000_000_000u64),
        to: probe_addr,
        value: U256::ZERO,
        data: Bytes::from(case.calldata.clone()),
        l1_block_number: 1,
        timestamp: 1_700_000_002,
        request_seq: idx,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    };
    if let Ok(msg) = invoke.build() {
        delayed += 1;
        steps.push(message_step(idx, msg, delayed));
        idx += 1;
    }

    if steps.is_empty() {
        return None;
    }
    GLOBAL_MSG_IDX.store(idx, Ordering::Relaxed);
    GLOBAL_DELAYED.store(delayed, Ordering::Relaxed);
    let _ = starting_idx;

    Some((
        Scenario {
            name: case.name.clone(),
            description: format!(
                "matrix scenario seed={} calldata_len={} gas={}",
                case.wasm_seed,
                case.calldata.len(),
                case.gas_budget
            ),
            setup: ScenarioSetup {
                l2_chain_id: FUZZ_L2_CHAIN_ID,
                arbos_version: FUZZ_ARBOS_VERSION,
                genesis: None,
            },
            steps,
        },
        wasm,
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
        let Some((scen, wasm)) = build_scenario(&case) else {
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
