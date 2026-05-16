use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0x3e6fd2df… at block 185,892,958 (idx 3). Direct EOA -> 0x71
/// `activateProgram(0x3ee64ada…)` at ArbOS v40. The target contract has
/// non-Stylus bytecode (no `[EF F0 00]` prefix) so the precompile reverts
/// via Nitro's "old ArbOS" path that does NOT charge the 3-gas result-copy.
/// Canon receipt: status=0, gasUsed=0x19abd0 (=1,682,384). Locks the
/// non-Solidity revert path fixed in a8a0d46.
#[test]
fn sepolia_block_185_892_958_failed_activate_v40() {
    let path = fixtures_root()
        .join("stylus/regression/sepolia_block_185_892_958_failed_activate_v40.json");
    if std::env::var("ARB_SPEC_BINARY").is_err() {
        eprintln!("skipping: set ARB_SPEC_BINARY=path/to/arb-reth");
        return;
    }
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
