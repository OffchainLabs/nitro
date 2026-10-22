use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0x40ab6a1c…7070 at block 115,187,469 idx 2.
/// Program at 0x509c…bc12 has codehash 0x67720af2…7924b which is
/// already activated as version=2 in the parent ArbOS state, so the
/// activateProgram call must short-circuit with ProgramUpToDate
/// (gasUsed=1,682,306) rather than running the full prover pipeline
/// and burning the full gas limit.
#[test]
#[cfg_attr(
    not(feature = "spec-binary"),
    ignore = "requires `--features spec-binary` plus ARB_SPEC_BINARY"
)]
fn sepolia_block_115_187_469() {
    let path = fixtures_root().join("stylus/regression/sepolia_block_115_187_469.json");
    assert!(
        std::env::var("ARB_SPEC_BINARY").is_ok(),
        "ARB_SPEC_BINARY must point at a built `arb-reth` binary"
    );
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
