use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0xb0d3394d… at block 152,472,573 (idx 6). Direct EOA -> 0x71
/// `activateProgram(0x30f9...)` at ArbOS v40 with non-zero value, exercising
/// the outer EOA branch: tx_env.value is zeroed so revm skips the transfer,
/// the data_fee is burnt from the sender post-commit, and NetworkFeeAccount
/// is SLOADed inside the precompile. Canon gasUsed = 0x259ae4 = 2,464,484.
#[test]
fn sepolia_block_152_472_573_direct_activate_v40() {
    let path = fixtures_root()
        .join("stylus/regression/sepolia_block_152_472_573_direct_activate_v40.json");
    if std::env::var("ARB_SPEC_BINARY").is_err() {
        eprintln!("skipping: set ARB_SPEC_BINARY=path/to/arb-reth");
        return;
    }
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
