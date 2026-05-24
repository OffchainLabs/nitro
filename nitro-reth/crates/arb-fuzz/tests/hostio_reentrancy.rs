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

fn forward_call(target: Address, inner: &[u8]) -> Bytes {
    wrap_two_arg("forward(address,bytes)", target, inner)
}

fn helper_pings_back_stylus() -> Vec<u8> {
    let inner = selector4("callCount()").to_vec();
    let mut wrap = Vec::with_capacity(4 + 96);
    wrap.extend_from_slice(&selector4("forward(address,bytes)"));
    let stylus_addr = stylus();
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(stylus_addr.as_slice());
    wrap.extend_from_slice(&pad);
    let mut off32 = [0u8; 32];
    off32[31] = 0x40;
    wrap.extend_from_slice(&off32);
    let mut len32 = [0u8; 32];
    len32[24..32].copy_from_slice(&(inner.len() as u64).to_be_bytes());
    wrap.extend_from_slice(&len32);
    wrap.extend_from_slice(&inner);
    while wrap.len() % 32 != 0 {
        wrap.push(0);
    }
    let cd_len = wrap.len();

    let mut runtime = Vec::with_capacity(cd_len + 100);
    for (i, byte) in wrap.iter().enumerate() {
        runtime.extend_from_slice(&[0x60, *byte, 0x60, i as u8, 0x53]);
    }
    runtime.extend_from_slice(&[
        0x60, 0x00, 0x60, 0x00, 0x60, 0x20, 0x60, cd_len as u8, 0x60, 0x00,
    ]);
    runtime.push(0x73);
    runtime.extend_from_slice(stylus_addr.as_slice());
    runtime.extend_from_slice(&[0x5a, 0xf1, 0x60, 0x00, 0x60, 0x00, 0xf3]);
    runtime
}

fn helper_three_hop_back_to_stylus() -> Vec<u8> {
    let mut runtime = Vec::with_capacity(40);
    let stylus_addr = stylus();
    let inner = selector4("callCount()").to_vec();
    for (i, b) in inner.iter().enumerate() {
        runtime.extend_from_slice(&[0x60, *b, 0x60, i as u8, 0x53]);
    }
    runtime.extend_from_slice(&[
        0x60, 0x00, 0x60, 0x00, 0x60, 0x20, 0x60, 0x04, 0x60, 0x00,
    ]);
    runtime.push(0x73);
    runtime.extend_from_slice(stylus_addr.as_slice());
    runtime.extend_from_slice(&[0x5a, 0xf1, 0x60, 0x00, 0x60, 0x00, 0xf3]);
    runtime
}

#[test]
#[ignore]
fn stylus_calls_helper_calls_stylus_forward() {
    let helper_runtime = helper_pings_back_stylus();
    let (mut steps, _, _) = baseline_stylus_plus_helper(&helper_runtime);
    let cdata = forward_call(helper(), &[]);
    let tx = signed(3, Some(stylus()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("stylus_helper_stylus_forward", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn stylus_calls_helper_calls_stylus_callcount() {
    let helper_runtime = helper_three_hop_back_to_stylus();
    let (mut steps, _, _) = baseline_stylus_plus_helper(&helper_runtime);
    let cdata = forward_call(helper(), &[]);
    let tx = signed(3, Some(stylus()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("stylus_helper_callcount", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn stylus_self_call_via_forward() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let inner_call_count = selector4("callCount()").to_vec();
    let cdata = forward_call(stylus(), &inner_call_count);
    let tx = signed(3, Some(stylus()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("stylus_self_call_forward", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn stylus_self_call_via_forward_into_forward() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let inner = forward_call(stylus(), &selector4("callCount()"));
    let cdata = forward_call(stylus(), inner.as_ref());
    let tx = signed(3, Some(stylus()), cdata, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("stylus_double_self_forward", steps)
        .expect_last_tx_min_gas(25_000)
        .run();
}
