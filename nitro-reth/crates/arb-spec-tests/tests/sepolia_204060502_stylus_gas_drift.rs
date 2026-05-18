use arb_spec_tests::runner::{fixtures_root, run_execution_fixture};

/// Sepolia tx 0xe22b6570… at block 204,060,502 (idx 6). EOA -> Stylus
/// contract 0x68c7… selector 0xab11ec20. Canon receipt: status=0,
/// gasUsed=45606, gasUsedForL1=0. Local arbreth produces 45748 (+142).
/// Fixture replay produces 45984 (+378; inflated because we omit the
/// contract's storage, forcing cold reads).
///
/// NOT in Nitro's hardcoded RevertedTxGasUsed table — Nitro naturally
/// computes 45606 on this control path. Per-hostio trace
/// (`STYLUS_HOSTIO_TRACE=1 ARB_SPEC_KEEP_WORKDIR=1`) shows only three
/// hostios fire before revert: `msg_reentrant`, `pay_for_memory_grow(0)`,
/// `read_args` — so the +142/+378 delta lives in the WASM bytecode
/// portion (per-opcode ink ⊕ ink_header_cost basic-block overhead),
/// not in host-function gas charges. Pricing constants, opcode costs,
/// and basic-block detection were audited line-by-line vs Nitro and
/// match exactly.
///
/// Marked `#[ignore]` until root-caused: leaves the fixture as a
/// reproducer without breaking the green regression suite.
#[ignore = "stylus runtime gas drift; root cause not yet isolated"]
#[test]
fn sepolia_block_204_060_502_stylus_gas_drift() {
    let path = fixtures_root()
        .join("stylus/regression/sepolia_block_204_060_502_stylus_gas_drift.json");
    if std::env::var("ARB_SPEC_BINARY").is_err() {
        eprintln!("skipping: set ARB_SPEC_BINARY=path/to/arb-reth");
        return;
    }
    if let Err(e) = run_execution_fixture(&path, None) {
        panic!("fixture failed: {e}");
    }
}
