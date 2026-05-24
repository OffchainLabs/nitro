//! Per-selector gas pins for NodeInterface (0xc8).
//!
//! NodeInterface is RPC-only — most selectors are advisory and don't
//! touch consensus. Several methods unconditionally revert ("not reachable
//! from a precompile") with the accumulated boilerplate gas. The
//! executable selectors (`blockL1Num`, `getL1Confirmations`,
//! `findBatchContainingBlock`, `legacyLookupMessageBatchProof`,
//! `nitroGenesisBlock`, `gasEstimateComponents`, `gasEstimateL1Component`)
//! each return their pinned per-method gas.

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{Address, U256};
use arb_precompiles::create_nodeinterface_precompile;
use common::{calldata, calldata_estimate, word_u256, PrecompileTest};

const SLOAD: u64 = 800;
const COPY: u64 = 3;

fn nodeinterface() -> DynPrecompile {
    create_nodeinterface_precompile()
}

fn fixture() -> PrecompileTest {
    PrecompileTest::new().arbos_version(30).arbos_state()
}

#[test]
fn block_l1_num_v30_gas_pin() {
    // Pure read; returns COPY_GAS (no SLOAD, no init).
    let run = fixture().call(
        &nodeinterface(),
        &calldata("blockL1Num(uint64)", &[word_u256(U256::from(123u64))]),
    );
    assert_eq!(run.gas_used(), COPY);
}

#[test]
fn get_l1_confirmations_v30_gas_pin() {
    // Returns 0 with COPY_GAS.
    let run = fixture().call(
        &nodeinterface(),
        &calldata(
            "getL1Confirmations(bytes32)",
            &[alloy_primitives::B256::ZERO],
        ),
    );
    assert_eq!(run.gas_used(), COPY);
}

#[test]
fn find_batch_containing_block_v30_gas_pin() {
    let run = fixture().call(
        &nodeinterface(),
        &calldata(
            "findBatchContainingBlock(uint64)",
            &[word_u256(U256::from(1u64))],
        ),
    );
    assert_eq!(run.gas_used(), COPY);
}

#[test]
fn legacy_lookup_message_batch_proof_v30_gas_pin() {
    let run = fixture().call(
        &nodeinterface(),
        &calldata(
            "legacyLookupMessageBatchProof(uint256,uint64)",
            &[word_u256(U256::from(0u64)), word_u256(U256::from(0u64))],
        ),
    );
    assert_eq!(run.gas_used(), COPY);
}

#[test]
fn nitro_genesis_block_v30_gas_pin() {
    // 1 SLOAD + 1 COPY = 803.
    let run = fixture().call(&nodeinterface(), &calldata("nitroGenesisBlock()", &[]));
    assert_eq!(run.gas_used(), SLOAD + COPY);
}

#[test]
fn gas_estimate_components_v30_gas_pin() {
    // 2 * SLOAD + COPY = 1603 (handler returns this fixed cost).
    let run = fixture().call(
        &nodeinterface(),
        &calldata_estimate("gasEstimateComponents(address,bool,bytes)"),
    );
    assert_eq!(run.gas_used(), 2 * SLOAD + COPY);
}

#[test]
fn gas_estimate_l1_component_v30_gas_pin() {
    let run = fixture().call(
        &nodeinterface(),
        &calldata_estimate("gasEstimateL1Component(address,bool,bytes)"),
    );
    assert_eq!(run.gas_used(), 2 * SLOAD + COPY);
}

#[test]
fn l2_block_range_for_l1_reverts_with_init_gas_v30() {
    // RPC-only — handler returns empty_revert with the boilerplate gas
    // already accumulated by init_precompile_gas (SLOAD + argsCost).
    let run = fixture().gas(50_000).call(
        &nodeinterface(),
        &calldata("l2BlockRangeForL1(uint64)", &[word_u256(U256::from(1u64))]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    // init_precompile_gas: 800 + 1 word args = 803.
    assert_eq!(out.gas_used, SLOAD + COPY);
}

#[test]
fn estimate_retryable_ticket_reverts_with_init_gas_v30() {
    // estimateRetryableTicket(address,uint256,address,uint256,address,address,bytes)
    let mut buf = Vec::with_capacity(4 + 8 * 32);
    buf.extend_from_slice(&common::selector(
        "estimateRetryableTicket(address,uint256,address,uint256,address,address,bytes)",
    ));
    // 6 head args + 1 offset to bytes (224 = 7*32) + 1 length(0). 8 head words.
    for _ in 0..6 {
        buf.extend_from_slice(&[0u8; 32]);
    }
    buf.extend_from_slice(&word_u256(U256::from(7 * 32u64)).0);
    buf.extend_from_slice(&[0u8; 32]); // length=0
    let _ = Address::ZERO;
    let run = fixture()
        .gas(50_000)
        .call(&nodeinterface(), &alloy_primitives::Bytes::from(buf));
    let out = run.assert_ok();
    assert!(out.reverted);
    // init_precompile_gas: 800 + 8 words * 3 = 824.
    assert_eq!(out.gas_used, SLOAD + 8 * COPY);
}

#[test]
fn construct_outbox_proof_reverts_with_init_gas_v30() {
    let run = fixture().gas(50_000).call(
        &nodeinterface(),
        &calldata(
            "constructOutboxProof(uint64,uint64)",
            &[word_u256(U256::from(0u64)), word_u256(U256::from(0u64))],
        ),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    // init_precompile_gas: 800 + 2 words * 3 = 806.
    assert_eq!(out.gas_used, SLOAD + 2 * COPY);
}
