use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0x40ad…c5c0 at block 112,694,675 (tx index 153).
/// Locks in the EIP-2929 cold/warm-on-revert fix in `wasm_call_cost`:
/// when a Stylus sub-call reverts, revm marks the account `Cold` again
/// rather than removing it from the state map, so a plain
/// `state.contains_key` warmth check misses the revert and under-charges
/// each subsequent re-access by 2,500 gas. Without the fix the tx
/// under-charges by exactly -7,500 (3 × 2,500) vs canon's 2,446,219.
#[test]
fn sepolia_block_112_694_675() {
    let path = fixtures_root().join("stylus/regression/sepolia_block_112_694_675.json");
    if std::env::var("ARB_SPEC_BINARY").is_err() {
        eprintln!("skipping: set ARB_SPEC_BINARY=path/to/arb-reth");
        return;
    }
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
