use alloy_primitives::{Address, Bytes, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    guards::GuardedRun,
    scaffolding::{
        baseline_stylus_plus_helper, eoa_create_addr, selector4, signed, INVOKE_GAS_CAP,
    },
    shared_nodes::next_msg_idx,
};
use arb_test_harness::messaging::MessageBuilder;

const ARBGASINFO: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x6c,
]);
const ARBOWNER: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x70,
]);
const ARBOWNERPUBLIC: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x6b,
]);

fn no_arg(sig: &str) -> Bytes {
    Bytes::from(selector4(sig).to_vec())
}

fn one_arg_addr(sig: &str, who: Address) -> Bytes {
    let mut out = Vec::with_capacity(36);
    out.extend_from_slice(&selector4(sig));
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(who.as_slice());
    out.extend_from_slice(&pad);
    Bytes::from(out)
}

#[test]
#[ignore]
fn get_multi_gas_base_fee_read() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBGASINFO),
        no_arg("getMultiGasBaseFee()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("get_multi_gas_base_fee", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn get_multi_gas_pricing_constraints_read() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBGASINFO),
        no_arg("getMultiGasPricingConstraints()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("get_multi_gas_constraints", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn set_multi_gas_constraints_non_owner_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let mut data = Vec::with_capacity(36);
    data.extend_from_slice(&selector4(
        "setMultiGasPricingConstraints(((uint8,uint64)[],uint32,uint64,uint64)[])",
    ));
    let mut off = [0u8; 32];
    off[31] = 0x20;
    data.extend_from_slice(&off);
    let mut len = [0u8; 32];
    data.extend_from_slice(&len);
    let _ = &mut len;

    let tx = signed(
        3,
        Some(ARBOWNER),
        Bytes::from(data),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("set_multi_gas_non_owner", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn set_collect_tips_non_owner_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let mut data = Vec::with_capacity(36);
    data.extend_from_slice(&selector4("setCollectTips(bool)"));
    let mut arg = [0u8; 32];
    arg[31] = 1;
    data.extend_from_slice(&arg);
    let tx = signed(
        3,
        Some(ARBOWNER),
        Bytes::from(data),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("set_collect_tips_non_owner", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn get_collect_tips_query() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBOWNERPUBLIC),
        no_arg("getCollectTips()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("get_collect_tips", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn add_transaction_filterer_non_owner_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBOWNER),
        one_arg_addr("addTransactionFilterer(address)", eoa_create_addr(0)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("add_filterer_non_owner", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn is_transaction_filterer_random_returns_false() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBOWNERPUBLIC),
        one_arg_addr("isTransactionFilterer(address)", Address::repeat_byte(0xfe)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("is_filterer_random", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn get_all_transaction_filterers_query() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBOWNERPUBLIC),
        no_arg("getAllTransactionFilterers()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("get_all_filterers", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn get_filtered_funds_recipient_query() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBOWNERPUBLIC),
        no_arg("getFilteredFundsRecipient()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("get_filtered_recipient", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}
