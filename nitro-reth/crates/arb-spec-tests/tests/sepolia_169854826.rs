use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0xedfff06f… at block 169,854,826 (idx 9).
///
/// EOA -> Solidity factory that CREATEs a Stylus program then CALLs
/// ArbWasm.activateProgram (0x71) with a non-zero value budget. Pre-fix
/// arbreth read STYLUS_CALL_VALUE inside the precompile, which build.rs
/// only sets when the *outer* tx target is 0x71; the inner factory ->
/// 0x71 frame saw 0 and the check tripped with ProgramInsufficientValue
/// (selector 0x09781ab7). Post-fix reads input.value (the call-frame
/// value revm transferred) and the tx succeeds. Canon: status=1,
/// gasUsed=8,835,707.
#[test]
fn sepolia_block_169_854_826() {
    let path = fixtures_root().join("stylus/regression/sepolia_block_169_854_826.json");
    if std::env::var("ARB_SPEC_BINARY").is_err() {
        eprintln!("skipping: set ARB_SPEC_BINARY=path/to/arb-reth");
        return;
    }
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
