use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0x58df300a… at block 204,060,366 (idx 2). This tx hits a
/// Stylus path that produced divergent gas across Nitro versions, so OCL
/// hardcoded `gasUsed = 45174` in `go-ethereum/core/reverted_tx_gas.go`.
/// arbreth must apply the same override via `RevertedTxHook` or it will
/// diverge on Sepolia replay at this block. Canon receipt: status=0,
/// gasUsed=0xb226 (=45606).
#[test]
fn sepolia_block_204_060_366_pre_recorded_revert() {
    let path = fixtures_root()
        .join("stylus/regression/sepolia_block_204_060_366_pre_recorded_revert.json");
    if std::env::var("ARB_SPEC_BINARY").is_err() {
        eprintln!("skipping: set ARB_SPEC_BINARY=path/to/arb-reth");
        return;
    }
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
