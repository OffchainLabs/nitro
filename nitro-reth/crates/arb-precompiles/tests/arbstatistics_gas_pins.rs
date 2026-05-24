//! Per-selector gas pins for ArbStatistics (0x6f).

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::Bytes;
use arb_precompiles::create_arbstatistics_precompile;
use common::{calldata, PrecompileTest};

const ARBOS_V30: u64 = 30;
const GAS_LIMIT: u64 = 1_000_000;

fn arbstatistics(ctx: std::sync::Arc<arb_context::ArbPrecompileCtx>) -> DynPrecompile {
    create_arbstatistics_precompile(ctx)
}

fn fixture() -> PrecompileTest {
    PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .gas(GAS_LIMIT)
}

#[test]
fn get_stats_v30_gas_pin() {
    let run = fixture().call(arbstatistics, &calldata("getStats()", &[]));
    assert_eq!(run.gas_used(), 818);
}

#[test]
fn invalid_selector_v30_burns_all_gas() {
    let run = fixture().call(arbstatistics, &Bytes::from(vec![0xde, 0xad, 0xbe, 0xef]));
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, GAS_LIMIT);
}
