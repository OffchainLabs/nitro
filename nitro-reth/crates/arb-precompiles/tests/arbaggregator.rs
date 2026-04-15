mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, B256, U256};
use arb_precompiles::{
    create_arbaggregator_precompile,
    storage_slot::{
        derive_subspace_key, map_slot, map_slot_b256, ARBOS_STATE_ADDRESS, CHAIN_OWNER_SUBSPACE,
        L1_PRICING_SUBSPACE, ROOT_STORAGE_KEY,
    },
};
use common::{calldata, decode_address, decode_u256, decode_word, word_address, PrecompileTest};

fn arbaggregator() -> DynPrecompile {
    create_arbaggregator_precompile()
}

const BATCH_POSTER: Address = address!("a4b000000000000000000073657175656e636572");

#[test]
fn get_preferred_aggregator_returns_address_then_bool() {
    let probe: Address = address!("00000000000000000000000000000000000000ee");
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
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
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        &arbaggregator(),
        &calldata("getTxBaseFee(address)", &[word_address(probe)]),
    );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

// ── Nitro TestFeeCollector / TestTxBaseFee ports ─────────────────────

const BATCH_POSTER_TABLE_KEY: &[u8] = &[0];
const POSTER_INFO_KEY: &[u8] = &[1];
const PAY_TO_OFFSET: u64 = 1;

fn poster_info_pay_to_slot(poster: Address) -> U256 {
    let l1_pricing_key = derive_subspace_key(ROOT_STORAGE_KEY, L1_PRICING_SUBSPACE);
    let bpt_key = derive_subspace_key(l1_pricing_key.as_slice(), BATCH_POSTER_TABLE_KEY);
    let poster_info = derive_subspace_key(bpt_key.as_slice(), POSTER_INFO_KEY);
    let info_key = derive_subspace_key(poster_info.as_slice(), poster.as_slice());
    map_slot(info_key.as_slice(), PAY_TO_OFFSET)
}

fn chain_owner_member_slot(addr: Address) -> U256 {
    let owner_key = derive_subspace_key(ROOT_STORAGE_KEY, CHAIN_OWNER_SUBSPACE);
    let by_addr_key = derive_subspace_key(owner_key.as_slice(), &[0]);
    let mut padded = [0u8; 32];
    padded[12..32].copy_from_slice(addr.as_slice());
    map_slot_b256(by_addr_key.as_slice(), &B256::from(padded))
}

#[test]
fn get_fee_collector_returns_stored_pay_to() {
    let collector: Address = address!("0000000000000000000000000000000000000bbb");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            poster_info_pay_to_slot(BATCH_POSTER),
            U256::from_be_slice(collector.as_slice()),
        )
        .call(
            &arbaggregator(),
            &calldata("getFeeCollector(address)", &[word_address(BATCH_POSTER)]),
        );
    assert_eq!(decode_address(run.output()), collector);
}

#[test]
fn set_fee_collector_succeeds_when_caller_is_batch_poster() {
    let new_collector: Address = address!("0000000000000000000000000000000000000bbb");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .caller(BATCH_POSTER)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            poster_info_pay_to_slot(BATCH_POSTER),
            U256::from_be_slice(BATCH_POSTER.as_slice()),
        )
        .call(
            &arbaggregator(),
            &calldata(
                "setFeeCollector(address,address)",
                &[word_address(BATCH_POSTER), word_address(new_collector)],
            ),
        );
    let _ = run.assert_ok();
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, poster_info_pay_to_slot(BATCH_POSTER)),
        U256::from_be_slice(new_collector.as_slice())
    );
}

#[test]
fn set_fee_collector_succeeds_when_caller_is_current_collector() {
    // The currently-configured fee collector can replace itself, even if it is
    // not the batch poster.
    let current: Address = address!("0000000000000000000000000000000000000ccc");
    let new_collector: Address = address!("0000000000000000000000000000000000000ddd");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .caller(current)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            poster_info_pay_to_slot(BATCH_POSTER),
            U256::from_be_slice(current.as_slice()),
        )
        .call(
            &arbaggregator(),
            &calldata(
                "setFeeCollector(address,address)",
                &[word_address(BATCH_POSTER), word_address(new_collector)],
            ),
        );
    let _ = run.assert_ok();
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, poster_info_pay_to_slot(BATCH_POSTER)),
        U256::from_be_slice(new_collector.as_slice())
    );
}

#[test]
fn set_fee_collector_succeeds_when_caller_is_chain_owner() {
    let owner: Address = address!("0000000000000000000000000000000000000aaa");
    let new_collector: Address = address!("0000000000000000000000000000000000000ddd");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .caller(owner)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            poster_info_pay_to_slot(BATCH_POSTER),
            U256::from_be_slice(BATCH_POSTER.as_slice()),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            chain_owner_member_slot(owner),
            U256::from(1),
        )
        .call(
            &arbaggregator(),
            &calldata(
                "setFeeCollector(address,address)",
                &[word_address(BATCH_POSTER), word_address(new_collector)],
            ),
        );
    let _ = run.assert_ok();
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, poster_info_pay_to_slot(BATCH_POSTER)),
        U256::from_be_slice(new_collector.as_slice())
    );
}

#[test]
fn set_fee_collector_rejects_unauthorised_caller() {
    let stranger: Address = address!("0000000000000000000000000000000000000eee");
    let new_collector: Address = address!("0000000000000000000000000000000000000ddd");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .caller(stranger)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            poster_info_pay_to_slot(BATCH_POSTER),
            U256::from_be_slice(BATCH_POSTER.as_slice()),
        )
        .call(
            &arbaggregator(),
            &calldata(
                "setFeeCollector(address,address)",
                &[word_address(BATCH_POSTER), word_address(new_collector)],
            ),
        );
    let out = run.assert_ok();
    assert!(out.reverted, "stranger setFeeCollector must revert");
}

#[test]
fn set_tx_base_fee_is_a_noop_returning_no_data() {
    let probe: Address = address!("00000000000000000000000000000000000000ee");
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        &arbaggregator(),
        &calldata(
            "setTxBaseFee(address,uint256)",
            &[
                word_address(probe),
                B256::from(U256::from(973).to_be_bytes::<32>()),
            ],
        ),
    );
    let out = run.assert_ok();
    assert!(out.bytes.is_empty(), "setTxBaseFee returns no data");
    // Verify a follow-up getter still returns 0.
    let run2 = PrecompileTest::new().arbos_version(30).arbos_state().call(
        &arbaggregator(),
        &calldata("getTxBaseFee(address)", &[word_address(probe)]),
    );
    assert_eq!(decode_u256(run2.output()), U256::ZERO);
}
