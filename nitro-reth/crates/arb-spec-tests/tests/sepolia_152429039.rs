use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0x253bfee4bc… at block 152,429,039 (idx 6).
///
/// EOA -> Solidity factory that CREATEs a Stylus program, makes two
/// STATICCALLs into ArbWasm (codehashVersion + stylusVersion), then CALLs
/// the just-created Stylus program. Canon: status=1, gasUsed=1,385,389;
/// arbreth currently charges 203 gas less. Pins the divergence so the
/// eventual fix can be locked in.
#[test]
#[cfg_attr(
    not(feature = "spec-binary"),
    ignore = "requires `--features spec-binary` plus ARB_SPEC_BINARY"
)]
fn sepolia_block_152_429_039() {
    let path = fixtures_root().join("stylus/regression/sepolia_block_152_429_039.json");
    assert!(
        std::env::var("ARB_SPEC_BINARY").is_ok(),
        "ARB_SPEC_BINARY must point at a built `arb-reth` binary"
    );
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
