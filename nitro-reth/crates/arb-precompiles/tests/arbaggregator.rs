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

fn arbaggregator(ctx: std::sync::Arc<arb_context::ArbPrecompileCtx>) -> DynPrecompile {
    create_arbaggregator_precompile(ctx)
}

const BATCH_POSTER: Address = address!("a4b000000000000000000073657175656e636572");

#[test]
fn get_preferred_aggregator_returns_address_then_bool() {
    let probe: Address = address!("00000000000000000000000000000000000000ee");
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        arbaggregator,
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
        .call(arbaggregator, &calldata("getDefaultAggregator()", &[]));
    assert_eq!(decode_address(run.output()), BATCH_POSTER);
}

#[test]
fn get_tx_base_fee_returns_zero() {
    let probe: Address = address!("00000000000000000000000000000000000000ee");
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        arbaggregator,
        &calldata("getTxBaseFee(address)", &[word_address(probe)]),
    );
    assert_eq!(decode_u256(run.output()), U256::ZERO);
}

const BATCH_POSTER_TABLE_KEY: &[u8] = &[0];
const POSTER_ADDRS_KEY: &[u8] = &[0];
const POSTER_INFO_KEY: &[u8] = &[1];
const PAY_TO_OFFSET: u64 = 1;

fn poster_info_pay_to_slot(poster: Address) -> U256 {
    let l1_pricing_key = derive_subspace_key(ROOT_STORAGE_KEY, L1_PRICING_SUBSPACE);
    let bpt_key = derive_subspace_key(l1_pricing_key.as_slice(), BATCH_POSTER_TABLE_KEY);
    let poster_info = derive_subspace_key(bpt_key.as_slice(), POSTER_INFO_KEY);
    let info_key = derive_subspace_key(poster_info.as_slice(), poster.as_slice());
    map_slot(info_key.as_slice(), PAY_TO_OFFSET)
}

fn poster_addrs_member_slot(poster: Address) -> U256 {
    let l1_pricing_key = derive_subspace_key(ROOT_STORAGE_KEY, L1_PRICING_SUBSPACE);
    let bpt_key = derive_subspace_key(l1_pricing_key.as_slice(), BATCH_POSTER_TABLE_KEY);
    let addrs_key = derive_subspace_key(bpt_key.as_slice(), POSTER_ADDRS_KEY);
    let by_addr_key = derive_subspace_key(addrs_key.as_slice(), &[0]);
    let poster_b256 = B256::left_padding_from(poster.as_slice());
    map_slot_b256(by_addr_key.as_slice(), &poster_b256)
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
            poster_addrs_member_slot(BATCH_POSTER),
            U256::from(1),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            poster_info_pay_to_slot(BATCH_POSTER),
            U256::from_be_slice(collector.as_slice()),
        )
        .call(
            arbaggregator,
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
            poster_addrs_member_slot(BATCH_POSTER),
            U256::from(1),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            poster_info_pay_to_slot(BATCH_POSTER),
            U256::from_be_slice(BATCH_POSTER.as_slice()),
        )
        .call(
            arbaggregator,
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
            poster_addrs_member_slot(BATCH_POSTER),
            U256::from(1),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            poster_info_pay_to_slot(BATCH_POSTER),
            U256::from_be_slice(current.as_slice()),
        )
        .call(
            arbaggregator,
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
            poster_addrs_member_slot(BATCH_POSTER),
            U256::from(1),
        )
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
            arbaggregator,
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
            arbaggregator,
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
        arbaggregator,
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
        arbaggregator,
        &calldata("getTxBaseFee(address)", &[word_address(probe)]),
    );
    assert_eq!(decode_u256(run2.output()), U256::ZERO);
}

// ── Per-selector gas-equality assertions ────────────────────────────────

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const SSTORE_ZERO_GAS: u64 = 5_000;
const COPY_GAS: u64 = 3;

#[test]
fn get_preferred_aggregator_charges_open_args_and_two_copy_words() {
    let probe: Address = address!("00000000000000000000000000000000000000ee");
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        arbaggregator,
        &calldata("getPreferredAggregator(address)", &[word_address(probe)]),
    );
    // init(800 + 1 arg word * 3) + 2 result words = 809.
    assert_eq!(run.gas_used(), SLOAD_GAS + COPY_GAS + 2 * COPY_GAS);
}

#[test]
fn get_default_aggregator_charges_one_sload_and_one_copy_word() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(arbaggregator, &calldata("getDefaultAggregator()", &[]));
    assert_eq!(run.gas_used(), SLOAD_GAS + COPY_GAS);
}

#[test]
fn get_tx_base_fee_charges_sload_plus_six() {
    let probe: Address = address!("00000000000000000000000000000000000000ee");
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        arbaggregator,
        &calldata("getTxBaseFee(address)", &[word_address(probe)]),
    );
    assert_eq!(run.gas_used(), SLOAD_GAS + 6);
}

#[test]
fn set_tx_base_fee_charges_sload_plus_six() {
    let probe: Address = address!("00000000000000000000000000000000000000ee");
    let run = PrecompileTest::new().arbos_version(30).arbos_state().call(
        arbaggregator,
        &calldata(
            "setTxBaseFee(address,uint256)",
            &[
                word_address(probe),
                B256::from(U256::from(0).to_be_bytes::<32>()),
            ],
        ),
    );
    assert_eq!(run.gas_used(), SLOAD_GAS + 6);
}

#[test]
fn get_fee_collector_charges_two_sloads_after_init_and_one_copy_word() {
    let collector: Address = address!("0000000000000000000000000000000000000bbb");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            poster_addrs_member_slot(BATCH_POSTER),
            U256::from(1),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            poster_info_pay_to_slot(BATCH_POSTER),
            U256::from_be_slice(collector.as_slice()),
        )
        .call(
            arbaggregator,
            &calldata("getFeeCollector(address)", &[word_address(BATCH_POSTER)]),
        );
    // init(800 + 1 arg word * 3) + 2 SLOADs + 1 result word = 2406.
    assert_eq!(run.gas_used(), 3 * SLOAD_GAS + 2 * COPY_GAS);
}

#[test]
fn set_fee_collector_caller_is_poster_charges_three_sloads_and_one_sstore() {
    let new_collector: Address = address!("0000000000000000000000000000000000000bbb");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .caller(BATCH_POSTER)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            poster_addrs_member_slot(BATCH_POSTER),
            U256::from(1),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            poster_info_pay_to_slot(BATCH_POSTER),
            U256::from_be_slice(BATCH_POSTER.as_slice()),
        )
        .call(
            arbaggregator,
            &calldata(
                "setFeeCollector(address,address)",
                &[word_address(BATCH_POSTER), word_address(new_collector)],
            ),
        );
    // init(800 + 2 arg words * 3) + 2 SLOADs + 1 SSTORE = 22_406.
    assert_eq!(run.gas_used(), 3 * SLOAD_GAS + 2 * COPY_GAS + SSTORE_GAS);
}

#[test]
fn set_fee_collector_caller_is_chain_owner_charges_extra_sload() {
    let owner: Address = address!("0000000000000000000000000000000000000aaa");
    let new_collector: Address = address!("0000000000000000000000000000000000000ddd");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .caller(owner)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            poster_addrs_member_slot(BATCH_POSTER),
            U256::from(1),
        )
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
            arbaggregator,
            &calldata(
                "setFeeCollector(address,address)",
                &[word_address(BATCH_POSTER), word_address(new_collector)],
            ),
        );
    // Owner path includes an extra is_chain_owner SLOAD: 23_206.
    assert_eq!(run.gas_used(), 4 * SLOAD_GAS + 2 * COPY_GAS + SSTORE_GAS);
}

#[test]
fn get_batch_posters_charges_two_sloads_and_two_copy_words_when_empty() {
    let run = PrecompileTest::new()
        .arbos_version(30)
        .arbos_state()
        .call(arbaggregator, &calldata("getBatchPosters()", &[]));
    // count=0: (2 + 0) * SLOAD + (2 + 0) * COPY = 1606.
    assert_eq!(run.gas_used(), 2 * SLOAD_GAS + 2 * COPY_GAS);
}

#[test]
fn add_batch_poster_already_exists_returns_two_sloads_and_one_copy_word() {
    let owner: Address = address!("0000000000000000000000000000000000000aaa");
    let existing: Address = address!("0000000000000000000000000000000000000eee");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .caller(owner)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            chain_owner_member_slot(owner),
            U256::from(1),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            poster_addrs_member_slot(existing),
            U256::from(1),
        )
        .call(
            arbaggregator,
            &calldata("addBatchPoster(address)", &[word_address(existing)]),
        );
    // No-op path: explicit return (2 * SLOAD + COPY) = 1603.
    assert_eq!(run.gas_used(), 2 * SLOAD_GAS + COPY_GAS);
}

#[test]
fn add_batch_poster_new_writes_charges_full_formula() {
    let owner: Address = address!("0000000000000000000000000000000000000aaa");
    let new_poster: Address = address!("0000000000000000000000000000000000000fff");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .caller(owner)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            chain_owner_member_slot(owner),
            U256::from(1),
        )
        .call(
            arbaggregator,
            &calldata("addBatchPoster(address)", &[word_address(new_poster)]),
        );
    // 6 SLOAD + 1 SSTORE_ZERO + 4 SSTORE + COPY = 4800 + 5000 + 80000 + 3 = 89803.
    assert_eq!(
        run.gas_used(),
        6 * SLOAD_GAS + SSTORE_ZERO_GAS + 4 * SSTORE_GAS + COPY_GAS,
    );
}

#[test]
fn add_batch_poster_non_owner_reverts_with_accumulated_gas() {
    let stranger: Address = address!("0000000000000000000000000000000000000bbb");
    let new_poster: Address = address!("0000000000000000000000000000000000000fff");
    let run = PrecompileTest::new()
        .arbos_version(30)
        .caller(stranger)
        .arbos_state()
        .call(
            arbaggregator,
            &calldata("addBatchPoster(address)", &[word_address(new_poster)]),
        );
    let out = run.assert_ok();
    assert!(out.reverted);
    // init(800 + 3) + is_chain_owner SLOAD(800) = 1603.
    assert_eq!(out.gas_used, 2 * SLOAD_GAS + COPY_GAS);
}
