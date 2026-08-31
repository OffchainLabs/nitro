use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia block 179,288,678. L2FundedByL1 message carries a ContractTx
/// (l2 sub-kind=1) with maxFeePerGas below the L2 basefee. Canon drops the
/// contract tx and keeps only the deposit. The previous fix (40c9980)
/// gated the basefee check on `!is_contract_tx`, leaving ContractTx
/// executing underpriced.
#[test]
#[cfg_attr(
    not(feature = "spec-binary"),
    ignore = "requires `--features spec-binary` plus ARB_SPEC_BINARY"
)]
fn sepolia_block_179_288_678() {
    let path = fixtures_root().join("stylus/regression/sepolia_block_179_288_678.json");
    assert!(
        std::env::var("ARB_SPEC_BINARY").is_ok(),
        "ARB_SPEC_BINARY must point at a built `arb-reth` binary"
    );
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
