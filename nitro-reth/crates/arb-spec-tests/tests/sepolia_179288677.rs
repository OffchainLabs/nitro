use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia block 179,288,677. L2FundedByL1 message carries an unsigned tx
/// with maxFeePerGas below the L2 basefee. Canon drops the unsigned tx and
/// keeps only the deposit; pre-fix arbreth executed the underpriced tx
/// because cfg.disable_base_fee was set chain-wide.
#[test]
fn sepolia_block_179_288_677() {
    let path = fixtures_root().join("stylus/regression/sepolia_block_179_288_677.json");
    if std::env::var("ARB_SPEC_BINARY").is_err() {
        eprintln!("skipping: set ARB_SPEC_BINARY=path/to/arb-reth");
        return;
    }
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
