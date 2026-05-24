use alloy_primitives::{Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    guards::GuardedRun,
    scaffolding::{baseline_stylus_plus_helper, selector4, signed, INVOKE_GAS_CAP},
    shared_nodes::next_msg_idx,
};
use arb_test_harness::messaging::MessageBuilder;

const ARBFILTEREDTX: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x74,
]);

fn one_arg_b32(sig: &str, h: B256) -> Bytes {
    let mut out = Vec::with_capacity(36);
    out.extend_from_slice(&selector4(sig));
    out.extend_from_slice(h.as_slice());
    Bytes::from(out)
}

#[test]
#[ignore]
fn is_filtered_random_hash_returns_false() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBFILTEREDTX),
        one_arg_b32("isTransactionFiltered(bytes32)", B256::repeat_byte(0xab)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("is_filtered_random", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn is_filtered_zero_hash_returns_false() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBFILTEREDTX),
        one_arg_b32("isTransactionFiltered(bytes32)", B256::ZERO),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("is_filtered_zero", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn is_filtered_max_hash_returns_false() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBFILTEREDTX),
        one_arg_b32("isTransactionFiltered(bytes32)", B256::repeat_byte(0xff)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("is_filtered_max", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn add_filtered_from_non_filterer_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBFILTEREDTX),
        one_arg_b32("addFilteredTransaction(bytes32)", B256::repeat_byte(0x01)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("add_filtered_non_filterer", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn delete_filtered_from_non_filterer_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBFILTEREDTX),
        one_arg_b32(
            "deleteFilteredTransaction(bytes32)",
            B256::repeat_byte(0x02),
        ),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("delete_filtered_non_filterer", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn add_filtered_with_zero_hash_non_filterer_still_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBFILTEREDTX),
        one_arg_b32("addFilteredTransaction(bytes32)", B256::ZERO),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("add_filtered_zero", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn invalid_selector_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBFILTEREDTX),
        Bytes::from(vec![0xde, 0xad, 0xbe, 0xef]),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("invalid_selector_filtered_tx_manager", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn empty_calldata_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBFILTEREDTX),
        Bytes::new(),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("empty_cd_filtered_tx_manager", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}
