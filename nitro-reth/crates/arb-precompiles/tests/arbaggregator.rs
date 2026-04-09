mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, U256};
use arb_precompiles::create_arbaggregator_precompile;
use common::{calldata, decode_address, decode_u256, decode_word, word_address, PrecompileTest};

fn arbaggregator() -> DynPrecompile {
    create_arbaggregator_precompile()
}

const BATCH_POSTER: Address = address!("a4b000000000000000000073657175656e636572");

#[test]
fn get_preferred_aggregator_returns_address_then_bool() {
    let probe: Address = address!("00000000000000000000000000000000000000ee");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(
            &arbaggregator(),
            &calldata("getPreferredAggregator(address)", &[word_address(probe)]),
        );
    let out = run.output();
    assert_eq!(out.len(), 64);
    let addr = decode_address(out);
    assert_eq!(addr, BATCH_POSTER);
    assert_eq!(decode_word(out, 1), common::word_u256(U256::from(1)));
}

#[test]
fn get_default_aggregator_returns_batch_poster() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(&arbaggregator(), &calldata("getDefaultAggregator()", &[]));
    assert_eq!(decode_address(run.output()), BATCH_POSTER);
}

#[test]
fn get_tx_base_fee_returns_zero() {
    let probe: Address = address!("00000000000000000000000000000000000000ee");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(
            &arbaggregator(),
            &calldata("getTxBaseFee(address)", &[word_address(probe)]),
        );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}
