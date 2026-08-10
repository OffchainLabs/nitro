use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0x97f87acd… at block 130,021,029 (idx 1).
/// Locks the ArbWasm `programInitGas`/`programVersion`/`programMemoryFootprint`/
/// `programTimeLeft` gas-charging fix: each method must charge
/// `StorageCodeHashCost` (2,600) for the codehash lookup and `WarmStorageReadCostEIP2929`
/// (100) for the StylusParams read instead of a full SLOAD (800), matching Nitro's
/// `getCodeHash` + `Params()` cost model. Pre-fix arbreth under-charged by 1,906 gas.
#[test]
#[cfg_attr(
    not(feature = "spec-binary"),
    ignore = "requires `--features spec-binary` plus ARB_SPEC_BINARY"
)]
fn sepolia_block_130_021_029() {
    let path = fixtures_root().join("stylus/regression/sepolia_block_130_021_029.json");
    assert!(
        std::env::var("ARB_SPEC_BINARY").is_ok(),
        "ARB_SPEC_BINARY must point at a built `arb-reth` binary"
    );
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
