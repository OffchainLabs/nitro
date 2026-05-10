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
/// Background: `state_root` and `parent_hash` mismatches between the
/// stock-geth genesis state (arbreth) and Nitro's geth-fork genesis state
/// are a *known* issue documented in MEMORY.md ("Genesis state root
/// mismatch"). They cascade into every block hash. The test classifies
/// those as `genesis_skew` and surfaces only execution-field diffs
/// (`gas_used`, `tx_count`, `tx_diffs`, `state_diffs`, `log_diffs`) as
/// real findings.
#[test]
#[ignore]
fn live_against_nitro() {
    use arb_fuzz::shared_nodes::shared_dual_exec;
    use arb_test_harness::scenario::ScenarioStep;
    use arbitrary::{Arbitrary, Unstructured};
    use std::sync::atomic::{AtomicU64, Ordering};

    /// SplitMix64 produces well-distributed bytes deterministic from the
    /// iteration index. Replaces the prior low-entropy seed that locked
    /// every iteration into the same Legacy/1559 bias.
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

    /// Renumber every `Message` step's `idx` to a unique globally-monotonic
    /// value so the shared dual-exec keeps a consistent inbox sequence
    /// across fuzz iterations. Returns the new top idx.
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

    /// "Real" divergences exclude state_root/parent_hash mismatches that
    /// trace back to the v40 genesis-trie skew.
    fn is_genesis_skew(field: &str) -> bool {
        matches!(field, "state_root" | "parent_hash")
    }

    let iterations: usize = std::env::var("ARB_FUZZ_ITERATIONS")
        .ok()
        .and_then(|s| s.parse().ok())
        .unwrap_or(50);
    let mut clean = 0usize;
    let mut skipped = 0usize;
    let mut genesis_skew_only = 0usize;
    let mut diverged: Vec<String> = Vec::new();
    let mut errors: Vec<String> = Vec::new();

    let nodes = shared_dual_exec();
    // Start at idx=1; shared_dual_exec hasn't ingested any prior messages.
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
            Ok(report) => {
                let real_block_diffs: Vec<_> = report
                    .block_diffs
                    .iter()
                    .filter(|d| !is_genesis_skew(&d.field))
                    .collect();
                let has_real = !real_block_diffs.is_empty()
                    || !report.tx_diffs.is_empty()
                    || !report.state_diffs.is_empty()
                    || !report.log_diffs.is_empty();
                if !has_real {
                    if report.block_diffs.is_empty() {
                        clean += 1;
                        eprintln!("iter {i}: clean");
                    } else {
                        genesis_skew_only += 1;
                        eprintln!(
                            "iter {i}: clean apart from {} genesis-skew block diffs",
                            report.block_diffs.len()
                        );
                    }
                } else {
                    let summary = format!(
                        "iter {i} ({kind_label}): real_block={} tx_diffs={} state_diffs={} log_diffs={}\n{:#?}",
                        real_block_diffs.len(),
                        report.tx_diffs.len(),
                        report.state_diffs.len(),
                        report.log_diffs.len(),
                        report
                    );
                    eprintln!("DIVERGENCE: {summary}");
                    diverged.push(summary);
                }
            }
            Err(e) => {
                let msg = format!("iter {i} ({kind_label}): harness error: {e}");
                eprintln!("{msg}");
                errors.push(msg);
            }
        }
    }

    eprintln!(
        "\n=== diff_signed_tx live summary: {clean} clean, {genesis_skew_only} genesis-skew-only, {skipped} skipped, {} real divergences, {} errors ===",
        diverged.len(),
        errors.len()
    );
    if !diverged.is_empty() {
        for d in &diverged {
            eprintln!("--- {d}");
        }
        panic!(
            "{} real divergences across {iterations} iterations",
            diverged.len()
        );
    }
    assert!(
        clean + genesis_skew_only > 0,
        "expected at least one usable iteration"
    );
}
