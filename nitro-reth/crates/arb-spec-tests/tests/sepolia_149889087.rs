use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0x4f940a5b… at block 149,889,087 (idx 1).
/// Locks the ArbOS v40-49 → SpecId::PRAGUE mapping so EIP-7702 type 0x04 txs
/// pass revm validation; pre-fix arbreth maps v40 → CANCUN and revm rejects
/// type 0x04 with `Eip7702NotSupported`, dropping the tx from the block.
#[test]
fn sepolia_block_149_889_087() {
    let path = fixtures_root().join("stylus/regression/sepolia_block_149_889_087.json");
    if std::env::var("ARB_SPEC_BINARY").is_err() {
        eprintln!("skipping: set ARB_SPEC_BINARY=path/to/arb-reth");
        return;
    }
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
