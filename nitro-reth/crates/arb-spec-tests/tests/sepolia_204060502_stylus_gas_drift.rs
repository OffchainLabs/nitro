use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0xe22b6570… at block 204,060,502 (idx 6). EOA -> Stylus
/// contract 0x68c7… selector 0xab11ec20. Canon receipt: status=0,
/// gasUsed=0xb226=45_606, gasUsedForL1=0.
///
/// Root cause: the contract at 0x68c7… was activated at block 204,059,808
/// with `MaxStackDepth=262_144` (ArbOS v40 default). It recurses deeply,
/// and Cranelift's compiled output uses different Rust call-stack per
/// frame on ARM vs x86 → recursion terminates at different ink levels →
/// different gas. Nitro committed canon on x86; ARM nodes diverge.
///
/// Nitro hardcoded the analogous tx 34s earlier (`0x58df300a…`, block
/// 204,060,366) — see `nitro/go-ethereum/core/reverted_tx_gas.go`
/// (original commit message: "*bypass transaction execution for
/// problematic txs execution on ARM architecture*"). This tx wasn't
/// added to Nitro's table (likely oversight). arbreth on arm64 runs the
/// same ARM-divergent path so we add it to our own `reverted_tx_gas`
/// table to keep consensus on sync.
#[test]
#[cfg_attr(
    not(feature = "spec-binary"),
    ignore = "requires `--features spec-binary` plus ARB_SPEC_BINARY"
)]
fn sepolia_block_204_060_502_stylus_gas_drift() {
    let path =
        fixtures_root().join("stylus/regression/sepolia_block_204_060_502_stylus_gas_drift.json");
    assert!(
        std::env::var("ARB_SPEC_BINARY").is_ok(),
        "ARB_SPEC_BINARY must point at a built `arb-reth` binary"
    );
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
