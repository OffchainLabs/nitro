use alloy_primitives::{Bytes, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    guards::GuardedRun,
    scaffolding::{fund_interop_eoa, signed, INVOKE_GAS_CAP},
    shared_nodes::next_msg_idx,
};
use arb_test_harness::messaging::MessageBuilder;

#[test]
#[ignore]
fn floor_catches_early_revert() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let bad_calldata: &[u8] = &[0xde, 0xad, 0xbe, 0xef];
    let tx = signed(
        0,
        Some(alloy_primitives::Address::repeat_byte(0x42)),
        Bytes::from(bad_calldata.to_vec()),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));

    let result = std::panic::catch_unwind(|| {
        GuardedRun::new("floor_catches_early_revert", steps)
            .expect_last_tx_min_gas(10_000_000)
            .run();
    });
    assert!(result.is_err(), "guarded run should have panicked on gas floor violation");
}

#[test]
#[ignore]
fn passing_scenario_satisfies_guards() {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let tx = signed(
        0,
        Some(alloy_primitives::Address::repeat_byte(0x99)),
        Bytes::new(),
        U256::from(1u64),
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));

    GuardedRun::new("passing_scenario_satisfies_guards", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_log_count(0)
        .run();
}
