//! Deterministic smoke tests for `DiffSignedTxScenario::into_scenario`.
//! Confirms the fuzz scenario constructor produces a well-formed
//! deposit + signed-tx pair for each supported tx kind, without needing
//! libfuzzer or Docker.

use alloy_primitives::Address;
use arb_fuzz::arbitrary_impls::{
    AuthInput, BoundedBytes, DiffSignedTxScenario, SignedTxKind,
};
use arb_test_harness::scenario::ScenarioStep;

fn arbos_v40() -> arb_fuzz::arbitrary_impls::ArbosVersion {
    use arbitrary::{Arbitrary, Unstructured};
    let mut data = vec![40u8; 32];
    let mut u = Unstructured::new(&mut data);
    arb_fuzz::arbitrary_impls::ArbosVersion::arbitrary(&mut u).unwrap()
}

fn fixed_to() -> Option<Address> {
    Some(Address::repeat_byte(0xab))
}

fn make(kind: SignedTxKind, with_to: bool, auths: Vec<AuthInput>) -> DiffSignedTxScenario {
    DiffSignedTxScenario {
        arbos_version: arbos_v40(),
        kind,
        signing_key_low: [7u8; 32],
        to: if with_to { fixed_to() } else { None },
        data: BoundedBytes::default(),
        value_low: 0,
        gas: 200_000,
        max_fee: 1_000_000_000,
        max_priority_fee: 100_000_000,
        authorizations: auths,
    }
}

fn make_auth() -> AuthInput {
    AuthInput {
        signing_key: [3u8; 32],
        address: Address::repeat_byte(0xcd),
        nonce: 0,
    }
}

fn count_messages(steps: &[ScenarioStep]) -> usize {
    steps
        .iter()
        .filter(|s| matches!(s, ScenarioStep::Message { .. }))
        .count()
}

#[test]
fn legacy_tx_yields_deposit_plus_signed_tx() {
    let s = make(SignedTxKind::Legacy, true, Vec::new())
        .into_scenario()
        .expect("legacy scenario builds");
    assert_eq!(count_messages(&s.steps), 2, "expected deposit + signed tx");
}

#[test]
fn eip2930_tx_yields_deposit_plus_signed_tx() {
    let s = make(SignedTxKind::Eip2930, true, Vec::new())
        .into_scenario()
        .expect("eip2930 scenario builds");
    assert_eq!(count_messages(&s.steps), 2);
}

#[test]
fn eip1559_tx_yields_deposit_plus_signed_tx() {
    let s = make(SignedTxKind::Eip1559, true, Vec::new())
        .into_scenario()
        .expect("eip1559 scenario builds");
    assert_eq!(count_messages(&s.steps), 2);
}

#[test]
fn eip7702_with_one_auth_builds() {
    let s = make(SignedTxKind::Eip7702, true, vec![make_auth()])
        .into_scenario()
        .expect("eip7702 scenario builds");
    assert_eq!(count_messages(&s.steps), 2);
}

#[test]
fn eip7702_create_returns_none() {
    // EIP-7702 cannot be CREATE per spec.
    let scen = make(SignedTxKind::Eip7702, false, vec![make_auth()]);
    assert!(scen.into_scenario().is_none());
}

#[test]
fn eip7702_empty_auth_list_skipped_at_signed_step() {
    // Empty auth list makes SignedL2TxBuilder::build_envelope fail; the
    // scenario still builds (deposit succeeds) but only emits 1 message.
    let scen = make(SignedTxKind::Eip7702, true, Vec::new())
        .into_scenario()
        .expect("scenario still builds with deposit");
    assert_eq!(count_messages(&scen.steps), 1);
}

/// Live differential vs Nitro Docker. Runs `--ignored` because it spawns
/// a Docker container and a release arbreth process.
///
///   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
///     ARB_FUZZ_ARBOS_VERSION=40 \
///     cargo test -p arb-fuzz --test signed_tx_scenarios --release \
///     -- --ignored live_against_nitro --nocapture
///
/// Strict comparison: any block / tx / state / log diff fails the test.
#[test]
#[ignore]
fn live_against_nitro() {
    use arb_fuzz::shared_nodes::shared_dual_exec;
    use arb_test_harness::scenario::ScenarioStep;
    use arbitrary::{Arbitrary, Unstructured};
    use std::sync::atomic::{AtomicU64, Ordering};

    fn seed(i: usize) -> Vec<u8> {
        let mut state: u64 = 0x9E37_79B9_7F4A_7C15u64.wrapping_add(i as u64);
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

    fn renumber_steps(steps: &mut Vec<ScenarioStep>, next: &AtomicU64) {
        for step in steps.iter_mut() {
            if let ScenarioStep::Message {
                idx,
                delayed_messages_read,
                ..
            } = step
            {
                let new_idx = next.fetch_add(1, Ordering::SeqCst);
                *idx = new_idx;
                *delayed_messages_read = new_idx;
            }
        }
    }

    let iterations: usize = std::env::var("ARB_FUZZ_ITERATIONS")
        .ok()
        .and_then(|s| s.parse().ok())
        .unwrap_or(50);
    let mut clean = 0usize;
    let mut skipped = 0usize;
    let mut diverged: Vec<String> = Vec::new();
    let mut errors: Vec<String> = Vec::new();

    let nodes = shared_dual_exec();
    let next_msg_idx = AtomicU64::new(1);

    for i in 0..iterations {
        let bytes = seed(i);
        let mut u = Unstructured::new(&bytes);
        let scenario = match DiffSignedTxScenario::arbitrary(&mut u) {
            Ok(s) => s,
            Err(e) => {
                eprintln!("iter {i}: arbitrary input rejected: {e}");
                skipped += 1;
                continue;
            }
        };
        let mut scen = match scenario.clone().into_scenario() {
            Some(s) if !s.steps.is_empty() => s,
            _ => {
                skipped += 1;
                continue;
            }
        };
        renumber_steps(&mut scen.steps, &next_msg_idx);
        let kind_label = format!("{:?}", scenario.kind);
        eprintln!(
            "iter {i}: {kind_label} (auths={}) — running",
            scenario.authorizations.len()
        );

        let mut nodes = nodes.lock().expect("dual-exec mutex poisoned");
        match nodes.run(&scen) {
            Ok(report) if report.is_clean() => {
                clean += 1;
                eprintln!("iter {i}: clean");
            }
            Ok(report) => {
                let summary = format!(
                    "iter {i} ({kind_label}): block_diffs={} tx_diffs={} state_diffs={} log_diffs={}\n{:#?}",
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
                let msg = format!("iter {i} ({kind_label}): harness error: {e}");
                eprintln!("{msg}");
                errors.push(msg);
            }
        }
    }

    eprintln!(
        "\n=== diff_signed_tx live summary: {clean} clean, {skipped} skipped, {} divergences, {} errors ===",
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
