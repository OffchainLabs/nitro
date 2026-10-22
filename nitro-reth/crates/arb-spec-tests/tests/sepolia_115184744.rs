use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0x8856…79e at block 115,184,744 (tx index 3).
/// Locks in the ArbWASM `activateProgram` value-vs-data_fee fix:
/// when `msg.value < data_fee`, the precompile must revert with
/// `ProgramInsufficientValue(have, want)` (matching Go's
/// `payActivationDataFee`). Without the fix arbreth incorrectly
/// returned success, emitted a `ProgramActivated` log, and silently
/// burned a zero-balance sender to underflow.
#[test]
#[cfg_attr(
    not(feature = "spec-binary"),
    ignore = "requires `--features spec-binary` plus ARB_SPEC_BINARY"
)]
fn sepolia_block_115_184_744() {
    let path = fixtures_root().join("stylus/regression/sepolia_block_115_184_744.json");
    assert!(
        std::env::var("ARB_SPEC_BINARY").is_ok(),
        "ARB_SPEC_BINARY must point at a built `arb-reth` binary"
    );
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
