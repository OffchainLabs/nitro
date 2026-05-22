use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia block 179,288,678. L2FundedByL1 message carries a ContractTx
/// (l2 sub-kind=1) with maxFeePerGas below the L2 basefee. Canon drops the
/// contract tx and keeps only the deposit. The previous fix (40c9980)
/// gated the basefee check on `!is_contract_tx`, leaving ContractTx
/// executing underpriced.
#[test]
fn sepolia_block_179_288_678() {
    let path = fixtures_root().join("stylus/regression/sepolia_block_179_288_678.json");
    if std::env::var("ARB_SPEC_BINARY").is_err() {
        eprintln!("skipping: set ARB_SPEC_BINARY=path/to/arb-reth");
        return;
    }
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
