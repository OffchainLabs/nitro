use std::path::Path;

use arb_bench::{
    capture::synthetic::generate,
    corpus::manifest::Manifest,
    report::{self, compare::Verdict},
    runner::{
        abba::{run_abba, AbbaConfig},
        in_process::InProcessRunner,
        RunnerConfig,
    },
};

#[test]
fn manifests_in_repo_are_well_formed() {
    let root = Path::new("../../bench/corpus/synthetic");
    if !root.exists() {
        // Workspace layout absorbed via cargo test path.
        return;
    }
    let entries = Manifest::discover(root).expect("discover");
    assert!(
        !entries.is_empty(),
        "expected at least one synthetic manifest"
    );
    for (path, m) in entries {
        m.validate()
            .unwrap_or_else(|e| panic!("manifest {} invalid: {e}", path.display()));
    }
}

#[test]
fn run_synthetic_workload_end_to_end() {
    let g = generate(
        "test/end_to_end",
        421614,
        30,
        "transfer_train",
        &serde_json::json!({ "block_count": 3, "txs_per_block": 4 }),
    )
    .unwrap();
    let mut r = InProcessRunner::new(RunnerConfig {
        rolling_window_blocks: 1,
        abort_on_block_error: false,
    });
    let result = r.run(g).unwrap();
    assert_eq!(result.blocks.len(), 3);
    assert_eq!(result.windows.len(), 3);
    assert!(result.summary.total_gas > 0);

    let dir = tempfile::tempdir().unwrap();
    let json_path = dir.path().join("r.json");
    let csv_path = dir.path().join("r.csv");
    let win_path = dir.path().join("r-windows.csv");
    report::json::write_run_result(&result, &json_path).unwrap();
    report::csv::write_run_blocks_csv(&result, &csv_path).unwrap();
    report::csv::write_run_windows_csv(&result, &win_path).unwrap();
    let read_back = report::json::read_run_result(&json_path).unwrap();
    assert_eq!(read_back.summary.total_gas, result.summary.total_gas);
}

#[test]
fn abba_full_round_trip() {
    let cfg = AbbaConfig {
        iterations: 1,
        bootstrap_iters: 200,
        tolerance_pct: 50.0,
        seed: 7,
        runner: RunnerConfig {
            rolling_window_blocks: 2,
            abort_on_block_error: false,
        },
    };
    let result = run_abba(
        &cfg,
        "test/abba_full",
        || {
            generate(
                "test/abba_full",
                421614,
                30,
                "transfer_train",
                &serde_json::json!({ "block_count": 3, "txs_per_block": 4 }),
            )
        },
        || {
            generate(
                "test/abba_full",
                421614,
                30,
                "transfer_train",
                &serde_json::json!({ "block_count": 3, "txs_per_block": 4 }),
            )
        },
    )
    .unwrap();

    assert_eq!(result.iterations, 1);
    assert!(!result.deltas.is_empty());
    assert!(!matches!(result.verdict, Verdict::Regression { .. }));

    let dir = tempfile::tempdir().unwrap();
    let json_path = dir.path().join("a.json");
    let csv_path = dir.path().join("a.csv");
    report::json::write_abba_result(&result, &json_path).unwrap();
    report::csv::write_abba_deltas_csv(&result, &csv_path).unwrap();
    let md = arb_bench::report::markdown::render_abba(&result);
    assert!(md.contains("Verdict"));
}
