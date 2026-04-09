mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::U256;
use arb_precompiles::create_arbstatistics_precompile;
use common::{calldata, decode_word, PrecompileTest};

fn arbstatistics() -> DynPrecompile {
    create_arbstatistics_precompile()
}

#[test]
fn get_stats_returns_block_number_and_zeros() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .block_number(7_654_321)
        .arbos_state()
        .call(&arbstatistics(), &calldata("getStats()", &[]));
    let out = run.output();
    assert_eq!(decode_word(out, 0), common::word_u64(7_654_321));
    for i in 1..6 {
        assert_eq!(decode_word(out, i), common::word_u256(U256::ZERO));
    }
}
