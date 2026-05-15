//! Sweep real Stylus-SDK programs × calldata variants × ArbOS versions.
//!
//! For each program (counter, erc20_mini, sol_caller, storage_stress) we
//! deploy once, activate once, then fire a handful of calldata patterns and
//! diff against the Nitro Docker reference via `shared_dual_exec`. Any
//! divergence (no field-level filters) is dumped to
//! `/tmp/stylus_real_programs/divergences/`.
//!
//! Run with:
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     NITRO_REF_IMAGE=offchainlabs/nitro-node:v3.10.0-rc.10-b1cf6db \
//!     cargo test -p arb-fuzz --test stylus_real_programs --release \
//!     -- --ignored --nocapture

use std::{
    collections::HashSet,
    fs,
    path::PathBuf,
    sync::{
        atomic::{AtomicUsize, Ordering},
        Mutex, OnceLock,
    },
    time::Instant,
};

use arb_fuzz::{
    arbitrary_impls::{
        interop::{
            counter_calldata, erc20_calldata, sol_caller_calldata, storage_stress_calldata,
        },
        ArbosVersion, DiffStylusInteropScenario, WhichProgram,
    },
    shared_nodes::shared_dual_exec,
};

static SEEN_BLOCK_DIFFS: OnceLock<Mutex<HashSet<(u64, String)>>> = OnceLock::new();
static SEEN_TX_DIFFS: OnceLock<Mutex<HashSet<(String, String)>>> = OnceLock::new();

fn seen_block() -> &'static Mutex<HashSet<(u64, String)>> {
    SEEN_BLOCK_DIFFS.get_or_init(|| Mutex::new(HashSet::new()))
}
fn seen_tx() -> &'static Mutex<HashSet<(String, String)>> {
    SEEN_TX_DIFFS.get_or_init(|| Mutex::new(HashSet::new()))
}

#[test]
#[ignore]
fn stylus_real_programs_diff_matrix() {
    let out_dir = PathBuf::from(
        std::env::var("ARB_STYLUS_REAL_OUT")
            .unwrap_or_else(|_| "/tmp/stylus_real_programs".to_string()),
    );
    let _ = fs::remove_dir_all(&out_dir);
    let div_dir = out_dir.join("divergences");
    fs::create_dir_all(&div_dir).expect("mkdir");

    let arbos_versions: Vec<u64> = std::env::var("ARB_STYLUS_REAL_VERSIONS")
        .ok()
        .map(|s| s.split(',').filter_map(|t| t.trim().parse().ok()).collect())
        .unwrap_or_else(|| vec![40, 50, 60]);

    let programs = [
        WhichProgram::Counter,
        WhichProgram::Erc20Mini,
        WhichProgram::SolCaller,
        WhichProgram::StorageStress,
    ];

    let calldata_seeds = [0u64, 1, 2, 3, 7, 42, 123, 1_000];

    let total = AtomicUsize::new(0);
    let diverged = AtomicUsize::new(0);
    let harness_errs = AtomicUsize::new(0);

    let nodes = shared_dual_exec();
    let start = Instant::now();

    for arbos_version in &arbos_versions {
        for program in &programs {
            for seed in &calldata_seeds {
                let scen = DiffStylusInteropScenario {
                    arbos_version: ArbosVersion(*arbos_version),
                    program: *program,
                    action_seed: *seed,
                    interop_seed: *seed ^ 0x1234,
                };
                let Some(s) = scen.clone().into_scenario() else {
                    continue;
                };
                let case_name = format!(
                    "{}_v{}_seed_{}",
                    program.name(),
                    arbos_version,
                    seed
                );
                eprintln!("[stylus_real_programs] running {case_name}");

                let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
                match nodes.run(&s) {
                    Ok(report) => {
                        let mut sb = seen_block().lock().unwrap();
                        let new_block_diffs: Vec<_> = report
                            .block_diffs
                            .iter()
                            .filter(|d| sb.insert((d.number, d.field.clone())))
                            .collect();
                        drop(sb);
                        let mut st = seen_tx().lock().unwrap();
                        let new_tx_diffs: Vec<_> = report
                            .tx_diffs
                            .iter()
                            .filter(|d| st.insert((format!("{:?}", d.tx_hash), d.field.clone())))
                            .collect();
                        drop(st);
                        let real = !new_block_diffs.is_empty()
                            || !new_tx_diffs.is_empty()
                            || !report.state_diffs.is_empty()
                            || !report.log_diffs.is_empty();
                        if real {
                            diverged.fetch_add(1, Ordering::Relaxed);
                            let calldata = match program {
                                WhichProgram::Counter => counter_calldata(*seed),
                                WhichProgram::Erc20Mini => erc20_calldata(*seed),
                                WhichProgram::SolCaller => {
                                    sol_caller_calldata(*seed, *seed ^ 0x1234, None)
                                }
                                WhichProgram::StorageStress => storage_stress_calldata(*seed),
                            };
                            let payload = serde_json::json!({
                                "case": case_name,
                                "program": program.name(),
                                "arbos_version": arbos_version,
                                "action_seed": seed,
                                "calldata_hex": format!("0x{}", hex::encode(&calldata)),
                                "block_diffs_new": format!("{:#?}", new_block_diffs),
                                "tx_diffs_new": format!("{:#?}", new_tx_diffs),
                                "state_diffs": format!("{:#?}", report.state_diffs),
                                "log_diffs": format!("{:#?}", report.log_diffs),
                            });
                            let path = div_dir.join(format!("{case_name}.json"));
                            let _ =
                                fs::write(&path, serde_json::to_vec_pretty(&payload).unwrap());
                            eprintln!("[stylus_real_programs] DIVERGE {case_name}");
                        }
                    }
                    Err(e) => {
                        harness_errs.fetch_add(1, Ordering::Relaxed);
                        eprintln!("[stylus_real_programs] HARNESS ERR {case_name}: {e}");
                    }
                }
                total.fetch_add(1, Ordering::Relaxed);
            }
        }
    }

    let summary = serde_json::json!({
        "total_runs": total.load(Ordering::Relaxed),
        "diverged": diverged.load(Ordering::Relaxed),
        "harness_errors": harness_errs.load(Ordering::Relaxed),
        "elapsed_secs": start.elapsed().as_secs(),
    });
    fs::write(
        out_dir.join("summary.json"),
        serde_json::to_vec_pretty(&summary).unwrap(),
    )
    .expect("write summary");
    eprintln!("[stylus_real_programs] done: {:#}", summary);
}
