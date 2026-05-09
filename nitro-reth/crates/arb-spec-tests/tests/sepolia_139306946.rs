use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0xdb77b3b2… at block 139,306,946 (idx 1).
/// Locks the `stylus_call_trampoline` non-Revert sub-call gas fix.
#[test]
fn sepolia_block_139_306_946() {
    let path = fixtures_root().join("stylus/regression/sepolia_block_139_306_946.json");
    if std::env::var("ARB_SPEC_BINARY").is_err() {
        eprintln!("skipping: set ARB_SPEC_BINARY=path/to/arb-reth");
        return;
    }
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
