use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia block 235,386,091 tx [1]: EOA -> 0x6e ArbRetryableTx.redeem(bytes32).
/// Gas limit 1,200,000. Canon gasUsed = 1,200,000 (the full limit); local was
/// 1,110,000 — under by exactly 90,000 = 6 * 15,000.
///
/// Chain is at v51 <= arbos < v60 with 6 single-dim gas constraints. Redeem
/// follows the donate-all-remaining-gas pattern: on success the precompile
/// reports gasUsed = gas_limit when actual ShrinkBacklog cost equals the
/// reservation. compute_actual_backlog_cost hard-coded SSTORE_RESET (5000)
/// per constraint, while the reservation uses SSTORE_SET (20000) and Nitro's
/// storage.WriteCost(value) only switches to SSTORE_RESET when the post-shrink
/// backlog is zero — which it isn't here. Off by 15,000 per non-draining
/// constraint, 6 constraints, 90,000 total.
///
/// Fixed in arb-precompiles/src/arbretryabletx.rs::compute_actual_backlog_cost
/// (per-constraint backlog read + WriteCost(new_value)).
///
/// Capture instructions: needs the chain state at block 235,386,090. With
/// `ARB_SPEC_RPC_URL` pointing at a Sepolia archive that has it, drop the
/// fixture JSON at the path below and the test picks it up.
#[test]
#[cfg_attr(
    not(feature = "spec-binary"),
    ignore = "requires `--features spec-binary` plus ARB_SPEC_BINARY"
)]
fn sepolia_block_235_386_091_redeem_six_constraints() {
    let path = fixtures_root()
        .join("retryables/regression/sepolia_block_235_386_091_redeem_six_constraints.json");
    assert!(
        std::env::var("ARB_SPEC_BINARY").is_ok(),
        "ARB_SPEC_BINARY must point at a built `arb-reth` binary"
    );
    assert!(
        path.exists(),
        "fixture not yet captured at {}",
        path.display()
    );
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
