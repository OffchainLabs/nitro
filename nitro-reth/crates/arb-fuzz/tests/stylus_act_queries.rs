use alloy_primitives::{Address, Bytes, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    guards::GuardedRun,
    scaffolding::{
        baseline_stylus_plus_helper, eoa_create_addr, selector4, signed, ARBWASM_ADDR,
        INVOKE_GAS_CAP,
    },
    shared_nodes::next_msg_idx,
};
use arb_test_harness::messaging::MessageBuilder;

fn stylus() -> Address {
    eoa_create_addr(0)
}

fn one_arg_addr(sig: &str, who: Address) -> Bytes {
    let mut out = Vec::with_capacity(36);
    out.extend_from_slice(&selector4(sig));
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(who.as_slice());
    out.extend_from_slice(&pad);
    Bytes::from(out)
}

fn no_arg(sig: &str) -> Bytes {
    Bytes::from(selector4(sig).to_vec())
}

#[test]
#[ignore]
fn program_version_returns_for_activated_addr() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        one_arg_addr("programVersion(address)", stylus()),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("program_version_activated", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn program_version_reverts_for_unactivated() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        one_arg_addr("programVersion(address)", Address::repeat_byte(0xab)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("program_version_unactivated", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn program_init_gas_for_activated_addr() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        one_arg_addr("programInitGas(address)", stylus()),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("program_init_gas_activated", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn program_memory_footprint_for_activated_addr() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        one_arg_addr("programMemoryFootprint(address)", stylus()),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("program_memory_footprint_activated", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn program_time_left_for_activated_addr() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        one_arg_addr("programTimeLeft(address)", stylus()),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("program_time_left_activated", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn stylus_version_query() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        no_arg("stylusVersion()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("stylus_version_query", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn ink_price_query() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        no_arg("inkPrice()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("ink_price_query", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn max_stack_depth_query() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        no_arg("maxStackDepth()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("max_stack_depth_query", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn free_pages_query() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        no_arg("freePages()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("free_pages_query", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn page_gas_query() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        no_arg("pageGas()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("page_gas_query", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn page_ramp_query() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        no_arg("pageRamp()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("page_ramp_query", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn page_limit_query() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        no_arg("pageLimit()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("page_limit_query", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn min_init_gas_query() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        no_arg("minInitGas()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("min_init_gas_query", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn keepalive_days_query() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        no_arg("keepaliveDays()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("keepalive_days_query", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn expiry_days_query() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        no_arg("expiryDays()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("expiry_days_query", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn init_cost_scalar_query() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        no_arg("initCostScalar()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("init_cost_scalar_query", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn block_cache_size_query() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        no_arg("blockCacheSize()"),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("block_cache_size_query", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn activation_gas_for_activated_addr() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        one_arg_addr("activationGas(address)", stylus()),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("activation_gas_query", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn re_activate_already_active_program() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let mut data = Vec::with_capacity(36);
    data.extend_from_slice(&[0x58, 0xc7, 0x80, 0xc2]);
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(stylus().as_slice());
    data.extend_from_slice(&pad);
    let tx = signed(
        3,
        Some(ARBWASM_ADDR),
        Bytes::from(data),
        U256::from(10u128).pow(U256::from(15u64)),
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("reactivate_active_program", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}
