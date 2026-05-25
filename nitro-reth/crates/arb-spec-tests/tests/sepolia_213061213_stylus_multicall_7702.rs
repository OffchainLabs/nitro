use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0x2bd2b083… at block 213,061,213 (idx 4). EOA -> Stylus
/// multicall contract 0xcd77… selector 0x88d695b2 multicall(address[],
/// uint256[]). Splits 0.002 ETH into 2x 0.001 ETH sub-CALLs to forwarders
/// 0x70997970… and 0x3c44cd…, both of which are EIP-7702 delegations to
/// the same implementation at 0x6bd9b71559….
///
/// Canon: status=1, gasUsed=94_441, 3 logs.
/// Local arbreth: status=0, gasUsed=97_317 (= entire gas limit consumed),
/// 0 logs — the Stylus program traps when its sub-CALL hostio targets a
/// 7702-delegated account.
#[test]
#[cfg_attr(
    not(feature = "spec-binary"),
    ignore = "requires `--features spec-binary` plus ARB_SPEC_BINARY"
)]
fn sepolia_block_213_061_213_stylus_multicall_7702() {
    let path = fixtures_root()
        .join("stylus/regression/sepolia_block_213_061_213_stylus_multicall_7702.json");
    assert!(
        std::env::var("ARB_SPEC_BINARY").is_ok(),
        "ARB_SPEC_BINARY must point at a built `arb-reth` binary"
    );
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
