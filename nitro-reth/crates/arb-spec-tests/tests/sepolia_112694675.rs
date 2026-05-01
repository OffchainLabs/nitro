use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Reproduces the gas divergence on Arbitrum Sepolia tx 0x40ad…c5c0
/// (block 112,694,675 / tx index 153). Canonical Sepolia v87 reports
/// gasUsed=2,446,219; our v60 chain reports 2,465,181 (Δ=+18,962).
///
/// Marked `#[ignore]` because the fixture is reproducing a known-unsolved
/// divergence — running it under the spec-tests CI would block merges.
/// Run manually:
///   ARB_SPEC_BINARY=$(pwd)/target/release/arb-reth \
///     cargo test -p arb-spec-tests --test sepolia_112694675 -- --ignored
#[test]
#[ignore]
fn sepolia_block_112_694_675() {
    let path = fixtures_root().join("stylus/regression/pending_sepolia_block_112_694_675.json");
    if std::env::var("ARB_SPEC_BINARY").is_err() {
        eprintln!("skipping: set ARB_SPEC_BINARY=path/to/arb-reth");
        return;
    }
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
