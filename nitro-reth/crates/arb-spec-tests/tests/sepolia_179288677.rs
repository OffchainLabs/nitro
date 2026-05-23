use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia block 179,288,677. L2FundedByL1 message carries an unsigned tx
/// with maxFeePerGas below the L2 basefee. Canon drops the unsigned tx and
/// keeps only the deposit; pre-fix arbreth executed the underpriced tx
/// because cfg.disable_base_fee was set chain-wide.
#[test]
#[cfg_attr(
    not(feature = "spec-binary"),
    ignore = "requires `--features spec-binary` plus ARB_SPEC_BINARY"
)]
fn sepolia_block_179_288_677() {
    let path = fixtures_root().join("stylus/regression/sepolia_block_179_288_677.json");
    assert!(
        std::env::var("ARB_SPEC_BINARY").is_ok(),
        "ARB_SPEC_BINARY must point at a built `arb-reth` binary"
    );
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
