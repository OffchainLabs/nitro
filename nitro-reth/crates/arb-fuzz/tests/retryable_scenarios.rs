//! Live SubmitRetryable differential vs Nitro Docker.
//!
//!   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
//!     ARB_FUZZ_ARBOS_VERSION=40 ARB_FUZZ_ITERATIONS=80 \
//!     cargo test -p arb-fuzz --test retryable_scenarios --release \
//!     -- --ignored live_against_nitro --nocapture

use arb_fuzz::arbitrary_impls::DiffRetryableScenario;

fn seed(i: usize) -> Vec<u8> {
    let mut state: u64 = 0xD1B5_4A32_D192_ED03u64.wrapping_add(i as u64);
    let mut out = Vec::with_capacity(256);
    while out.len() < 256 {
        state = state.wrapping_add(0x9E37_79B9_7F4A_7C15);
        let mut z = state;
        z = (z ^ (z >> 30)).wrapping_mul(0xBF58_476D_1CE4_E5B9);
        z = (z ^ (z >> 27)).wrapping_mul(0x94D0_49BB_1331_11EB);
        z ^= z >> 31;
        out.extend_from_slice(&z.to_le_bytes());
    }
    out
}

#[test]
#[ignore]
fn live_against_nitro() {
    use arb_fuzz::shared_nodes::shared_dual_exec;
    use arbitrary::{Arbitrary, Unstructured};

    let iterations: usize = std::env::var("ARB_FUZZ_ITERATIONS")
        .ok()
        .and_then(|s| s.parse().ok())
        .unwrap_or(50);

    let mut clean = 0usize;
    let mut skipped = 0usize;
    let mut diverged: Vec<String> = Vec::new();
    let mut errors: Vec<String> = Vec::new();

    let nodes = shared_dual_exec();

    for i in 0..iterations {
        let bytes = seed(i);
        let mut u = Unstructured::new(&bytes);
        let scenario = match DiffRetryableScenario::arbitrary(&mut u) {
            Ok(s) => s,
            Err(_) => {
                skipped += 1;
                continue;
            }
        };
        let scen = match scenario.clone().into_scenario() {
            Some(s) if !s.steps.is_empty() => s,
            _ => {
                skipped += 1;
                continue;
            }
        };
        eprintln!(
            "iter {i}: SubmitRetryable to={:?} l2_call_value={} deposit={} gas_limit={} — running",
            scenario.to,
            scenario.l2_call_value,
            scenario.deposit_value,
            scenario.gas_limit
        );

        let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
        match nodes.run(&scen) {
            Ok(report) if report.is_clean() => {
                clean += 1;
                eprintln!("iter {i}: clean");
            }
            Ok(report) => {
                let summary = format!(
                    "iter {i}: block_diffs={} tx_diffs={} state_diffs={} log_diffs={}\n{:#?}",
                    report.block_diffs.len(),
                    report.tx_diffs.len(),
                    report.state_diffs.len(),
                    report.log_diffs.len(),
                    report
                );
                eprintln!("DIVERGENCE: {summary}");
                diverged.push(summary);
            }
            Err(e) => {
                let msg = format!("iter {i}: harness error: {e}");
                eprintln!("{msg}");
                errors.push(msg);
            }
        }
    }

    eprintln!(
        "\n=== retryable live summary: {clean} clean, {skipped} skipped, {} divergences, {} errors ===",
        diverged.len(),
        errors.len()
    );
    if !diverged.is_empty() {
        for d in &diverged {
            eprintln!("--- {d}");
        }
        panic!(
            "{} divergences across {iterations} iterations",
            diverged.len()
        );
    }
    assert!(clean > 0, "expected at least one clean iteration");
}
