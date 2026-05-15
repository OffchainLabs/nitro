use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0xb0d3394d… at block 152,472,573 (idx 6). Direct EOA -> 0x71
/// `activateProgram(0x30f9...)` at ArbOS v40. Canonical gasUsed=2,464,996;
/// arbreth currently produces 2,464,484 (Δ = -512). Locks the divergence
/// so the eventual fix can be promoted out of `pending_` to enforce parity.
///
/// `#[ignore]` so the normal suite stays green; run explicitly with
/// `cargo test --test pending_sepolia_152472573_direct_activate_v40 -- --ignored`.
#[test]
#[ignore]
fn sepolia_block_152_472_573_direct_activate_v40() {
    let path = fixtures_root()
        .join("stylus/regression/pending_sepolia_block_152_472_573_direct_activate_v40.json");
    if std::env::var("ARB_SPEC_BINARY").is_err() {
        eprintln!("skipping: set ARB_SPEC_BINARY=path/to/arb-reth");
        return;
    }
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
