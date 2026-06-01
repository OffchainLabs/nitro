use alloy_primitives::{Address, Bytes, B256, U256};
use arb_fuzz::{
    arbitrary_impls::{interop::interop_signing_key, message_step},
    guards::GuardedRun,
    scaffolding::{fund_interop_eoa, FUZZ_L1_BASE_FEE, INVOKE_GAS_CAP, SEQUENCER_ALIAS},
    shared_nodes::{next_msg_idx, FUZZ_L2_CHAIN_ID},
};
use arb_test_harness::messaging::{
    signed_tx::{L2TxKind, SignedL2TxBuilder},
    MessageBuilder,
};

fn make_tx(kind: L2TxKind, nonce: u64) -> SignedL2TxBuilder {
    SignedL2TxBuilder {
        chain_id: FUZZ_L2_CHAIN_ID,
        nonce,
        to: Some(Address::repeat_byte(0x99)),
        value: U256::from(1u64),
        data: Bytes::new(),
        gas_limit: INVOKE_GAS_CAP,
        gas_price: 2_000_000_000,
        max_fee_per_gas: 2_000_000_000,
        max_priority_fee_per_gas: 0,
        access_list: Vec::new(),
        authorization_list: Vec::new(),
        kind,
        signing_key: interop_signing_key(),
        l1_block_number: 2,
        timestamp: 1_700_000_000,
        request_id: None,
        sender: SEQUENCER_ALIAS,
        base_fee_l1: FUZZ_L1_BASE_FEE,
    }
}

#[test]
#[ignore]
fn legacy_tx_transfer_to_eoa() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let tx = make_tx(L2TxKind::Legacy, 0).build().expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("txtype_legacy_xfer", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(21_000)
        .run();
}

#[test]
#[ignore]
fn eip2930_tx_transfer_no_access_list() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let tx = make_tx(L2TxKind::Eip2930, 0).build().expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("txtype_2930_empty_al", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(21_000)
        .run();
}

#[test]
#[ignore]
fn eip2930_tx_with_one_entry_access_list() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let mut tx = make_tx(L2TxKind::Eip2930, 0);
    tx.access_list = vec![(
        Address::repeat_byte(0x55),
        vec![B256::from([7u8; 32]), B256::from([8u8; 32])],
    )];
    let built = tx.build().expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, built, idx));
    GuardedRun::new("txtype_2930_with_al", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(21_000)
        .run();
}

#[test]
#[ignore]
fn eip2930_tx_with_large_access_list() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let mut tx = make_tx(L2TxKind::Eip2930, 0);
    let mut al = Vec::new();
    for i in 0..10 {
        let mut addr_bytes = [0u8; 20];
        addr_bytes[19] = i;
        let slots: Vec<B256> = (0..5)
            .map(|j| {
                let mut s = [0u8; 32];
                s[31] = j;
                B256::from(s)
            })
            .collect();
        al.push((Address::from(addr_bytes), slots));
    }
    tx.access_list = al;
    let built = tx.build().expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, built, idx));
    GuardedRun::new("txtype_2930_large_al", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(21_000)
        .run();
}

#[test]
#[ignore]
fn eip1559_tx_transfer() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let tx = make_tx(L2TxKind::Eip1559, 0).build().expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("txtype_1559_xfer", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(21_000)
        .run();
}

#[test]
#[ignore]
fn eip1559_tx_with_access_list() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let mut tx = make_tx(L2TxKind::Eip1559, 0);
    tx.access_list = vec![(Address::repeat_byte(0x33), vec![B256::ZERO])];
    let built = tx.build().expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, built, idx));
    GuardedRun::new("txtype_1559_with_al", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(21_000)
        .run();
}

#[test]
#[ignore]
fn legacy_tx_with_high_value_insufficient_balance_reverts_or_skipped() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let mut tx = make_tx(L2TxKind::Legacy, 0);
    tx.value = U256::from(u128::MAX);
    let built = tx.build().expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, built, idx));
    GuardedRun::new("txtype_legacy_too_big_value", steps)
        .allow_skipped()
        .expect_last_tx_min_gas(0)
        .run();
}

#[test]
#[ignore]
fn eip1559_create_contract() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let mut tx = make_tx(L2TxKind::Eip1559, 0);
    tx.to = None;
    tx.data = Bytes::from(vec![0x60, 0x00, 0x60, 0x00, 0xf3]);
    tx.value = U256::ZERO;
    let built = tx.build().expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, built, idx));
    GuardedRun::new("txtype_1559_create", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(53_000)
        .run();
}

#[test]
#[ignore]
fn eip2930_create_contract() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let mut tx = make_tx(L2TxKind::Eip2930, 0);
    tx.to = None;
    tx.data = Bytes::from(vec![0x60, 0x00, 0x60, 0x00, 0xf3]);
    tx.value = U256::ZERO;
    let built = tx.build().expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, built, idx));
    GuardedRun::new("txtype_2930_create", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(53_000)
        .run();
}

#[test]
#[ignore]
fn legacy_create_contract() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let mut tx = make_tx(L2TxKind::Legacy, 0);
    tx.to = None;
    tx.data = Bytes::from(vec![0x60, 0x00, 0x60, 0x00, 0xf3]);
    tx.value = U256::ZERO;
    let built = tx.build().expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, built, idx));
    GuardedRun::new("txtype_legacy_create", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(53_000)
        .run();
}
