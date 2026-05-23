use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0xec111095…9f72 at block 128,015,513 idx 3.
/// Stylus contract calls 0x1629…481 with value transfer that exceeds
/// the contract's balance. Canon's CALL returns Failure (out-of-funds);
/// arbreth (pre-fix) silently treated insufficient-balance transfer as
/// success, so the contract emitted a log and returned success.
#[test]
#[cfg_attr(
    not(feature = "spec-binary"),
    ignore = "requires `--features spec-binary` plus ARB_SPEC_BINARY"
)]
fn sepolia_block_128_015_513() {
    let path = fixtures_root().join("stylus/regression/sepolia_block_128_015_513.json");
    assert!(
        std::env::var("ARB_SPEC_BINARY").is_ok(),
        "ARB_SPEC_BINARY must point at a built `arb-reth` binary"
    );
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
