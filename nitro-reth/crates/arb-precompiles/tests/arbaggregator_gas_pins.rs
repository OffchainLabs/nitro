//! Per-selector gas pins for ArbAggregator (0x6d).

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, U256};
use arb_precompiles::create_arbaggregator_precompile;
use common::{calldata, word_address, word_u256, PrecompileTest};

const ARBOS_V30: u64 = 30;

fn arbaggregator(ctx: std::sync::Arc<arb_context::ArbPrecompileCtx>) -> DynPrecompile {
    create_arbaggregator_precompile(ctx)
}

fn fixture() -> PrecompileTest {
    PrecompileTest::new().arbos_version(ARBOS_V30).arbos_state()
}

#[test]
fn get_preferred_aggregator_v30_gas_pin() {
    let addr = Address::ZERO;
    let run = fixture().call(
        arbaggregator,
        &calldata("getPreferredAggregator(address)", &[word_address(addr)]),
    );
    assert_eq!(run.gas_used(), 809);
}

#[test]
fn get_default_aggregator_v30_gas_pin() {
    let run = fixture().call(arbaggregator, &calldata("getDefaultAggregator()", &[]));
    assert_eq!(run.gas_used(), 803);
}

#[test]
fn get_tx_base_fee_v30_gas_pin() {
    let addr = Address::ZERO;
    let run = fixture().call(
        arbaggregator,
        &calldata("getTxBaseFee(address)", &[word_address(addr)]),
    );
    assert_eq!(run.gas_used(), 806);
}

#[test]
fn set_tx_base_fee_v30_gas_pin() {
    let addr = Address::ZERO;
    let run = fixture().call(
        arbaggregator,
        &calldata(
            "setTxBaseFee(address,uint256)",
            &[word_address(addr), word_u256(U256::ZERO)],
        ),
    );
    assert_eq!(run.gas_used(), 806);
}

#[test]
fn get_fee_collector_unknown_poster_v30_revert_gas_pin() {
    let unknown_poster: Address = address!("00000000000000000000000000000000000000aa");
    let run = fixture().call(
        arbaggregator,
        &calldata("getFeeCollector(address)", &[word_address(unknown_poster)]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    // OpenArbosState (800) + args copy (3) + the poster-table read (800) that
    // finds the poster absent.
    assert_eq!(out.gas_used, 1603);
}
