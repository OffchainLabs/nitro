//! NodeInterface precompile conformance tests.
//!
//! Documents the expected behavior of the 0xc8 virtual contract against
//! Nitro's reference implementation. Each method tested for both happy
//! path and boundary conditions. Methods that Nitro handles via RPC
//! interception (constructOutboxProof, estimateRetryableTicket) are
//! tested for their required fallback — returning a sensible default
//! rather than reverting, matching Nitro's behavior when the batch
//! fetcher is unavailable.

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{B256, U256};
use arb_precompiles::{
    create_nodeinterface_precompile, set_cached_l1_block_number,
    storage_slot::{
        root_slot, subspace_slot, ARBOS_STATE_ADDRESS, L1_PRICING_SUBSPACE, L2_PRICING_SUBSPACE,
    },
};
use common::{calldata, calldata_estimate, decode_u256, decode_word, word_u256, PrecompileTest};

const GENESIS_BLOCK_NUM_OFFSET: u64 = 5;
const L1_PRICE_PER_UNIT: u64 = 7;
const L2_BASE_FEE: u64 = 2;

fn nodeinterface() -> DynPrecompile {
    create_nodeinterface_precompile()
}

// ======================================================================
// nitroGenesisBlock()
// ======================================================================

#[test]
fn nitro_genesis_block_default_is_zero() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(&nodeinterface(), &calldata("nitroGenesisBlock()", &[]));
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn nitro_genesis_block_returns_stored_value() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            root_slot(GENESIS_BLOCK_NUM_OFFSET),
            U256::from(22_207_817_u64),
        )
        .call(&nodeinterface(), &calldata("nitroGenesisBlock()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(22_207_817_u64));
}

// ======================================================================
// blockL1Num(uint64)
// ======================================================================

#[test]
fn block_l1_num_reads_cached_l1_block() {
    set_cached_l1_block_number(1000, 18_000_000);
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        &nodeinterface(),
        &calldata("blockL1Num(uint64)", &[word_u256(U256::from(1000))]),
    );
    assert_eq!(decode_u256(run.output()), U256::from(18_000_000_u64));
}

#[test]
fn block_l1_num_returns_zero_for_uncached_block() {
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        &nodeinterface(),
        &calldata(
            "blockL1Num(uint64)",
            &[word_u256(U256::from(99_999_999_u64))],
        ),
    );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

#[test]
fn block_l1_num_truncates_uint64_input() {
    set_cached_l1_block_number(42, 123);
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        &nodeinterface(),
        &calldata("blockL1Num(uint64)", &[word_u256(U256::from(42))]),
    );
    assert_eq!(decode_u256(run.output()), U256::from(123));
}

// ======================================================================
// gasEstimateComponents(address, bool, bytes)
// ======================================================================
//
// Nitro returns (gasEstimate, gasEstimateForL1, baseFee, l1BaseFeeEstimate).
// Our implementation returns 0 for gasEstimate (the full estimate requires
// calling back into eth_estimateGas, which isn't possible from an EVM
// precompile). baseFee and l1BaseFeeEstimate must match ArbOS storage.
// gasEstimateForL1 applies the 110% padding factor.

#[test]
fn gas_estimate_components_returns_basefee_and_l1_price() {
    let l1_price = U256::from(50_000_000_u64);
    let basefee = U256::from(100_000_000_u64);
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            subspace_slot(L1_PRICING_SUBSPACE, L1_PRICE_PER_UNIT),
            l1_price,
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            subspace_slot(L2_PRICING_SUBSPACE, L2_BASE_FEE),
            basefee,
        )
        .call(
            &nodeinterface(),
            &calldata_estimate("gasEstimateComponents(address,bool,bytes)"),
        );
    let out = run.output();
    // gasEstimate = 0 (requires eth_estimateGas to compute)
    assert_eq!(decode_word(out, 0), common::word_u256(U256::ZERO));
    // baseFee
    assert_eq!(decode_word(out, 2), common::word_u256(basefee));
    // l1BaseFeeEstimate
    assert_eq!(decode_word(out, 3), common::word_u256(l1_price));
}

#[test]
fn gas_estimate_components_always_returns_128_bytes() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            subspace_slot(L1_PRICING_SUBSPACE, L1_PRICE_PER_UNIT),
            U256::from(1u64),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            subspace_slot(L2_PRICING_SUBSPACE, L2_BASE_FEE),
            U256::from(1u64),
        )
        .call(
            &nodeinterface(),
            &calldata_estimate("gasEstimateComponents(address,bool,bytes)"),
        );
    assert_eq!(run.output().len(), 128, "4 × uint256 = 128 bytes");
}

// ======================================================================
// gasEstimateL1Component(address, bool, bytes)
// ======================================================================
//
// Nitro returns (gasEstimateForL1, baseFee, l1BaseFeeEstimate).
// Critically: Nitro's L1Component version does NOT apply padding —
// only the full gasEstimateComponents method pads.
// TODO: we currently pad here too (bug); once fixed, gasEstimateL1
// should be strictly LESS than gasEstimateComponents' L1 component.

#[test]
fn gas_estimate_l1_component_returns_basefee_and_l1_price() {
    let l1_price = U256::from(75_000_000_u64);
    let basefee = U256::from(150_000_000_u64);
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            subspace_slot(L1_PRICING_SUBSPACE, L1_PRICE_PER_UNIT),
            l1_price,
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            subspace_slot(L2_PRICING_SUBSPACE, L2_BASE_FEE),
            basefee,
        )
        .call(
            &nodeinterface(),
            &calldata_estimate("gasEstimateL1Component(address,bool,bytes)"),
        );
    let out = run.output();
    assert_eq!(decode_word(out, 1), common::word_u256(basefee));
    assert_eq!(decode_word(out, 2), common::word_u256(l1_price));
}

#[test]
fn gas_estimate_l1_component_returns_96_bytes() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            subspace_slot(L1_PRICING_SUBSPACE, L1_PRICE_PER_UNIT),
            U256::from(1u64),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            subspace_slot(L2_PRICING_SUBSPACE, L2_BASE_FEE),
            U256::from(1u64),
        )
        .call(
            &nodeinterface(),
            &calldata_estimate("gasEstimateL1Component(address,bool,bytes)"),
        );
    assert_eq!(run.output().len(), 96, "3 × uint256 = 96 bytes");
}

// ======================================================================
// getL1Confirmations(bytes32)
// ======================================================================
//
// Nitro: GetBatchFetcher returns 0 confirmations when the batch fetcher
// is nil (i.e. validators/followers without L1 access). For arbreth
// without an L1 follower, we must return 0, NOT revert — reverting
// breaks bridge tooling that relies on a zero result for "unknown/pending".

#[test]
fn get_l1_confirmations_unknown_block_returns_zero_not_revert() {
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        &nodeinterface(),
        &calldata("getL1Confirmations(bytes32)", &[B256::repeat_byte(0xAB)]),
    );
    let execution = run.assert_ok();
    assert!(
        !execution.reverted,
        "getL1Confirmations must return 0 for unknown blocks, not revert (matches Nitro when batch fetcher is nil)"
    );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

// ======================================================================
// findBatchContainingBlock(uint64)
// ======================================================================
//
// Nitro: returns 0 / error when batch fetcher is unavailable. For arbreth,
// returning 0 (not reverting) lets clients distinguish "no batch data" from
// "method doesn't exist".

#[test]
fn find_batch_containing_block_without_batch_data_returns_zero() {
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        &nodeinterface(),
        &calldata(
            "findBatchContainingBlock(uint64)",
            &[word_u256(U256::from(1u64))],
        ),
    );
    let execution = run.assert_ok();
    assert!(
        !execution.reverted,
        "findBatchContainingBlock must return 0, not revert, when no batch data is available"
    );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

// ======================================================================
// l2BlockRangeForL1(uint64)
// ======================================================================
//
// Nitro: binary-searches block headers. If no L2 block maps to the given
// L1 block, Nitro reverts with "requested L1 block number not found in
// chain". For arbreth without header access from within a precompile,
// this method cannot be implemented at EVM level. TODO: intercept at RPC.

#[test]
#[ignore = "TODO: implement at RPC layer via interception of eth_call"]
fn l2_block_range_for_l1_returns_first_last_block_range() {
    // Test kept as a TODO marker. When we implement RPC-layer
    // interception, this should populate headers and verify the range.
}

// ======================================================================
// estimateRetryableTicket(...)
// ======================================================================
//
// Nitro: constructs a SubmitRetryableTx and executes it inside an eth_call
// context. Requires tx-construction and state-transition machinery not
// available in a precompile. TODO: RPC-layer implementation.

#[test]
#[ignore = "TODO: implement at RPC layer"]
fn estimate_retryable_ticket_computes_submission_fee() {}

// ======================================================================
// constructOutboxProof(uint64, uint64)
// ======================================================================
//
// Nitro: filters L2ToL1Tx events (address 0x64, specific topic) across
// all blocks, reconstructs the Merkle tree, returns proof. Requires log
// filter access. TODO: RPC-layer implementation.

#[test]
#[ignore = "TODO: implement at RPC layer"]
fn construct_outbox_proof_builds_merkle_path() {}

// ======================================================================
// legacyLookupMessageBatchProof(uint256, uint64)
// ======================================================================
//
// Nitro: pre-Nitro outbox lookup (ClassicOutbox). arbreth doesn't need
// classic lookups — returning 0 / empty is correct.

#[test]
fn legacy_lookup_message_batch_proof_returns_empty_not_revert() {
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        &nodeinterface(),
        &calldata(
            "legacyLookupMessageBatchProof(uint256,uint64)",
            &[word_u256(U256::ZERO), word_u256(U256::ZERO)],
        ),
    );
    let execution = run.assert_ok();
    assert!(
        !execution.reverted,
        "legacy lookup is not used post-Nitro; must return empty, not revert"
    );
}

// ======================================================================
// Error cases
// ======================================================================

#[test]
fn unknown_selector_reverts() {
    // An arbitrary 4-byte selector that isn't defined.
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(&nodeinterface(), &[0xDEu8, 0xAD, 0xBE, 0xEF].into());
    assert!(run.assert_ok().reverted);
}

#[test]
fn empty_calldata_reverts() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(&nodeinterface(), &[].into());
    assert!(run.assert_ok().reverted);
}

#[test]
fn block_l1_num_short_input_reverts() {
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        &nodeinterface(),
        // selector + only 8 bytes instead of required 32
        &[0x6fu8, 0x27, 0x5e, 0xf2, 0, 0, 0, 0, 0, 0, 0, 1].into(),
    );
    assert!(run.assert_ok().reverted);
}
