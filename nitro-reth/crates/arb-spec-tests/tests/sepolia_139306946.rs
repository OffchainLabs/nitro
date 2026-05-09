use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0xdb77b3b2… at block 139,306,946 (idx 1).
/// Direct Stylus call (no proxy) where the Stylus contract makes a CALL into
/// a Solidity contract that nests another Solidity CALL, both reverting. Pre-fix
/// arbreth charges 858 gas less than canon for the Stylus→Solidity→Solidity chain.
/// Same family as blocks 112,694,675 (-7,500) and 130,021,029 (-1,906): Stylus
/// runtime / sub-call gas accounting drift. Fixture pins arbreth's current
/// gasUsed=189,960 as the regression target until the precise opcode-level root
/// cause is located.
#[test]
fn sepolia_block_139_306_946() {
    let path = fixtures_root().join("stylus/regression/sepolia_block_139_306_946.json");
    if std::env::var("ARB_SPEC_BINARY").is_err() {
        eprintln!("skipping: set ARB_SPEC_BINARY=path/to/arb-reth");
        return;
    }
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
