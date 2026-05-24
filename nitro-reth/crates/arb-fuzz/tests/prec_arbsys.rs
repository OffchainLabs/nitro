use alloy_primitives::{Address, Bytes, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    guards::GuardedRun,
    scaffolding::{
        baseline_stylus_plus_helper, eoa_create_addr, selector4, signed, ARBSYS_ADDR,
        INVOKE_GAS_CAP,
    },
    shared_nodes::next_msg_idx,
};
use arb_test_harness::messaging::MessageBuilder;

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

fn one_arg_u256(sig: &str, v: U256) -> Bytes {
    let mut out = Vec::with_capacity(36);
    out.extend_from_slice(&selector4(sig));
    out.extend_from_slice(&v.to_be_bytes::<32>());
    Bytes::from(out)
}

#[test]
#[ignore]
fn arbsys_block_number() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBSYS_ADDR),
        no_arg("arbBlockNumber()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("arbsys_block_number", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn arbsys_chain_id() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBSYS_ADDR),
        no_arg("arbChainID()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("arbsys_chain_id", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn arbsys_arbos_version() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBSYS_ADDR),
        no_arg("arbOSVersion()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("arbsys_arbos_version", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn arbsys_storage_gas_available() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBSYS_ADDR),
        no_arg("getStorageGasAvailable()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("arbsys_storage_gas", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn arbsys_is_top_level_call_true() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBSYS_ADDR),
        no_arg("isTopLevelCall()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("arbsys_top_level", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn arbsys_was_aliased() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBSYS_ADDR),
        no_arg("wasMyCallersAddressAliased()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("arbsys_was_aliased", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn arbsys_caller_without_alias() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBSYS_ADDR),
        no_arg("myCallersAddressWithoutAliasing()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("arbsys_caller_no_alias", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn arbsys_send_merkle_tree_state() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBSYS_ADDR),
        no_arg("sendMerkleTreeState()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("arbsys_merkle_state", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn arbsys_arb_block_hash_recent() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBSYS_ADDR),
        one_arg_u256("arbBlockHash(uint256)", U256::from(1u64)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("arbsys_block_hash_recent", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn arbsys_arb_block_hash_future_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBSYS_ADDR),
        one_arg_u256("arbBlockHash(uint256)", U256::from(u64::MAX)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("arbsys_block_hash_future", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn arbsys_map_l1_sender_contract_to_l2_alias() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let mut data = Vec::with_capacity(68);
    data.extend_from_slice(&selector4(
        "mapL1SenderContractAddressToL2Alias(address,address)",
    ));
    let mut pad1 = [0u8; 32];
    pad1[12..].copy_from_slice(Address::repeat_byte(0x11).as_slice());
    data.extend_from_slice(&pad1);
    let mut pad2 = [0u8; 32];
    pad2[12..].copy_from_slice(Address::repeat_byte(0x22).as_slice());
    data.extend_from_slice(&pad2);
    let tx = signed(
        3,
        Some(ARBSYS_ADDR),
        Bytes::from(data),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("arbsys_map_alias", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn arbsys_withdraw_eth_zero_value() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBSYS_ADDR),
        one_arg_addr("withdrawEth(address)", eoa_create_addr(99)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("arbsys_withdraw_zero", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn arbsys_send_tx_to_l1_zero_value() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let mut data = Vec::with_capacity(100);
    data.extend_from_slice(&selector4("sendTxToL1(address,bytes)"));
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(eoa_create_addr(100).as_slice());
    data.extend_from_slice(&pad);
    let mut off32 = [0u8; 32];
    off32[31] = 0x40;
    data.extend_from_slice(&off32);
    let mut len32 = [0u8; 32];
    data.extend_from_slice(&len32);
    let _ = &mut len32;
    let tx = signed(
        3,
        Some(ARBSYS_ADDR),
        Bytes::from(data),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("arbsys_send_to_l1", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn arbsys_invalid_selector_reverts() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBSYS_ADDR),
        Bytes::from(vec![0xab, 0xab, 0xab, 0xab]),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("arbsys_invalid_selector", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}
