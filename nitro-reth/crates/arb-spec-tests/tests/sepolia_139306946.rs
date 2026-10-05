use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0xdb77b3b2… at block 139,306,946 (idx 1).
/// Locks the `stylus_call_trampoline` non-Revert sub-call gas fix.
#[test]
#[cfg_attr(
    not(feature = "spec-binary"),
    ignore = "requires `--features spec-binary` plus ARB_SPEC_BINARY"
)]
fn sepolia_block_139_306_946() {
    let path = fixtures_root().join("stylus/regression/sepolia_block_139_306_946.json");
    assert!(
        std::env::var("ARB_SPEC_BINARY").is_ok(),
        "ARB_SPEC_BINARY must point at a built `arb-reth` binary"
    );
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
