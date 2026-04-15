mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::U256;
use arb_precompiles::{
    create_nodeinterface_precompile, set_cached_l1_block_number,
    storage_slot::{
        root_slot, subspace_slot, ARBOS_STATE_ADDRESS, L1_PRICING_SUBSPACE, L2_PRICING_SUBSPACE,
    },
};
use common::{calldata, decode_u256, decode_word, word_u256, PrecompileTest};

const GENESIS_BLOCK_NUM_OFFSET: u64 = 5;
const L1_PRICE_PER_UNIT: u64 = 7;
const L2_BASE_FEE: u64 = 2;

fn nodeinterface() -> DynPrecompile {
    create_nodeinterface_precompile()
}

#[test]
fn nitro_genesis_block_returns_root_field() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            root_slot(GENESIS_BLOCK_NUM_OFFSET),
            U256::from(123_456_u64),
        )
        .call(&nodeinterface(), &calldata("nitroGenesisBlock()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(123_456_u64));
}

#[test]
fn block_l1_num_returns_cached_value() {
    set_cached_l1_block_number(99, 7_777_777);
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        &nodeinterface(),
        &calldata("blockL1Num(uint64)", &[word_u256(U256::from(99))]),
    );
    assert_eq!(decode_u256(run.output()), U256::from(7_777_777_u64));
}

#[test]
fn block_l1_num_returns_zero_for_unknown_l2_block() {
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
            &calldata("gasEstimateComponents(address,bool,bytes)", &[]),
        );
    let out = run.output();
    assert_eq!(decode_word(out, 0), common::word_u256(U256::ZERO));
    assert_eq!(decode_word(out, 2), common::word_u256(basefee));
    assert_eq!(decode_word(out, 3), common::word_u256(l1_price));
}

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
            &calldata("gasEstimateL1Component(address,bool,bytes)", &[]),
        );
    let out = run.output();
    assert_eq!(decode_word(out, 1), common::word_u256(basefee));
    assert_eq!(decode_word(out, 2), common::word_u256(l1_price));
}

#[test]
fn rpc_only_methods_return_revert() {
    for sig in [
        "l2BlockRangeForL1(uint64)",
        "constructOutboxProof(uint64,uint64)",
        "findBatchContainingBlock(uint64)",
        "getL1Confirmations(bytes32)",
        "legacyLookupMessageBatchProof(uint256,uint64)",
    ] {
        let run = PrecompileTest::new()
            .arbos_version(30)
            .arbos_state()
            .call(&nodeinterface(), &calldata(sig, &[word_u256(U256::ZERO)]));
        assert!(run.assert_ok().reverted, "{sig} must revert (RPC-only)",);
    }
}
