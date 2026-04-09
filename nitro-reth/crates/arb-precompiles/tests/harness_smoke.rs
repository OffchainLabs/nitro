mod common;

use alloy_primitives::U256;
use arb_precompiles::create_arbsys_precompile;
use common::{calldata, decode_u256, PrecompileTest};

#[test]
fn arbsys_arb_block_number_returns_configured_block() {
    let precompile = create_arbsys_precompile();
    let input = calldata("arbBlockNumber()", &[]);
    let run = PrecompileTest::new()
        .arbos_version(30)
        .block_number(123_456)
        .arbos_state()
        .call(&precompile, &input);
    let value = decode_u256(run.output());
    assert_eq!(value, U256::from(123_456));
}
