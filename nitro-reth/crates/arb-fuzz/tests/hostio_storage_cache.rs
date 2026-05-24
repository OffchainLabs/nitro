use alloy_primitives::{Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    guards::GuardedRun,
    scaffolding::{
        baseline_stylus_plus_helper, eoa_create_addr, selector4, signed, INVOKE_GAS_CAP,
    },
    shared_nodes::next_msg_idx,
};
use arb_test_harness::messaging::MessageBuilder;

fn stylus() -> Address {
    eoa_create_addr(0)
}

fn two_arg_b32(sig: &str, slot: B256, value: B256) -> Bytes {
    let mut out = Vec::with_capacity(68);
    out.extend_from_slice(&selector4(sig));
    out.extend_from_slice(slot.as_slice());
    out.extend_from_slice(value.as_slice());
    Bytes::from(out)
}

fn slot_n(n: u8) -> B256 {
    let mut b = [0u8; 32];
    b[31] = n;
    B256::from(b)
}

fn val_n(n: u8) -> B256 {
    let mut b = [0u8; 32];
    b[0] = n;
    b[31] = n;
    B256::from(b)
}

#[test]
#[ignore]
fn cache_and_flush_writes_slot() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(stylus()),
        two_arg_b32("cacheAndFlush(bytes32,bytes32)", slot_n(1), val_n(0x42)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("cache_and_flush_writes", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .expect_sentinel(stylus(), U256::from(1u64), val_n(0x42))
        .run();
}

#[test]
#[ignore]
fn cache_only_writes_on_tx_end() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(stylus()),
        two_arg_b32("cacheOnly(bytes32,bytes32)", slot_n(2), val_n(0x55)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("cache_only_writes_on_tx_end", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .expect_sentinel(stylus(), U256::from(2u64), val_n(0x55))
        .run();
}

#[test]
#[ignore]
fn cache_then_clear_does_not_write() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(stylus()),
        two_arg_b32("cacheThenClear(bytes32,bytes32)", slot_n(3), val_n(0x77)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("cache_then_clear_no_write", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .expect_sentinel(stylus(), U256::from(3u64), B256::ZERO)
        .run();
}

#[test]
#[ignore]
fn double_cache_same_slot_last_wins() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx1 = signed(
        3,
        Some(stylus()),
        two_arg_b32("cacheAndFlush(bytes32,bytes32)", slot_n(4), val_n(0xa1)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx1");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx1, idx));
    let tx2 = signed(
        4,
        Some(stylus()),
        two_arg_b32("cacheAndFlush(bytes32,bytes32)", slot_n(4), val_n(0xb2)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx2");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx2, idx));
    GuardedRun::new("double_cache_same_slot", steps)
        .expect_sentinel(stylus(), U256::from(4u64), val_n(0xb2))
        .run();
}

#[test]
#[ignore]
fn overwrite_after_flush_with_clear() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx1 = signed(
        3,
        Some(stylus()),
        two_arg_b32("cacheAndFlush(bytes32,bytes32)", slot_n(5), val_n(0xc3)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx1");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx1, idx));
    let tx2 = signed(
        4,
        Some(stylus()),
        two_arg_b32("cacheThenClear(bytes32,bytes32)", slot_n(5), val_n(0xd4)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx2");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx2, idx));
    GuardedRun::new("overwrite_after_flush_with_clear", steps)
        .expect_sentinel(stylus(), U256::from(5u64), val_n(0xc3))
        .run();
}

#[test]
#[ignore]
fn cache_zero_to_zero_no_change() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(stylus()),
        two_arg_b32("cacheAndFlush(bytes32,bytes32)", slot_n(6), B256::ZERO),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("cache_zero_to_zero", steps)
        .expect_last_tx_status(true)
        .expect_sentinel(stylus(), U256::from(6u64), B256::ZERO)
        .run();
}

#[test]
#[ignore]
fn cache_nonzero_then_zero_clears() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx1 = signed(
        3,
        Some(stylus()),
        two_arg_b32("cacheAndFlush(bytes32,bytes32)", slot_n(7), val_n(0xe5)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx1");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx1, idx));
    let tx2 = signed(
        4,
        Some(stylus()),
        two_arg_b32("cacheAndFlush(bytes32,bytes32)", slot_n(7), B256::ZERO),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx2");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx2, idx));
    GuardedRun::new("cache_nonzero_then_zero", steps)
        .expect_sentinel(stylus(), U256::from(7u64), B256::ZERO)
        .run();
}
