use alloy_primitives::{Address, Bytes, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    guards::GuardedRun,
    scaffolding::{
        baseline_stylus_plus_helper, eoa_create_addr, selector4, signed, wrap_two_arg,
        INVOKE_GAS_CAP,
    },
    shared_nodes::next_msg_idx,
};
use arb_test_harness::messaging::MessageBuilder;

fn stylus() -> Address {
    eoa_create_addr(0)
}

fn helper() -> Address {
    eoa_create_addr(2)
}

fn one_arg_addr(sig: &str, who: Address) -> Bytes {
    let mut out = Vec::with_capacity(36);
    out.extend_from_slice(&selector4(sig));
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(who.as_slice());
    out.extend_from_slice(&pad);
    Bytes::from(out)
}

fn forward_to_stylus(sig: &str, who: Address) -> Bytes {
    wrap_two_arg(
        "forward(address,bytes)",
        stylus(),
        one_arg_addr(sig, who).as_ref(),
    )
}

#[test]
#[ignore]
fn balance_of_self() {
    let (mut steps, _stylus, _helper) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(stylus()),
        one_arg_addr("probeBalance(address)", stylus()),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("balance_of_self", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn balance_of_eoa_cold() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(stylus()),
        one_arg_addr("probeBalance(address)", Address::repeat_byte(0xaa)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("balance_of_eoa_cold", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn balance_of_precompile() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(stylus()),
        one_arg_addr("probeBalance(address)", Address::with_last_byte(0x04)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("balance_of_precompile", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn code_size_of_helper() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x60, 0x00, 0x60, 0x00, 0xf3]);
    let tx = signed(
        3,
        Some(stylus()),
        one_arg_addr("probeCodeSize(address)", helper()),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("code_size_of_helper", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn code_size_of_eoa_zero() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(stylus()),
        one_arg_addr("probeCodeSize(address)", Address::repeat_byte(0xbb)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("code_size_of_eoa_zero", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn code_hash_of_helper() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x60, 0x00, 0x60, 0x00, 0xf3]);
    let tx = signed(
        3,
        Some(stylus()),
        one_arg_addr("probeCodeHash(address)", helper()),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("code_hash_of_helper", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn code_hash_of_empty_account() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(stylus()),
        one_arg_addr("probeCodeHash(address)", Address::repeat_byte(0xcc)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("code_hash_of_empty_account", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn full_code_of_helper() {
    let runtime: &[u8] = &[0x60, 0xaa, 0x60, 0xbb, 0x60, 0xcc, 0x60, 0x00, 0xf3];
    let (mut steps, _, _) = baseline_stylus_plus_helper(runtime);
    let tx = signed(
        3,
        Some(stylus()),
        one_arg_addr("probeCode(address)", helper()),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("full_code_of_helper", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn code_of_stylus_contract_returns_prefix() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(stylus()),
        one_arg_addr("probeCode(address)", stylus()),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("code_of_stylus_returns_prefix", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn balance_of_self_via_forward_chain() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(stylus()),
        forward_to_stylus("probeBalance(address)", stylus()),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("balance_self_via_forward", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}
