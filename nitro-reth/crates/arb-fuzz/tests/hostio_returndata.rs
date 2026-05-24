use alloy_primitives::{keccak256, Address, Bytes, U256};
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

fn helper() -> Address {
    eoa_create_addr(2)
}

fn returndata_size_cd(target: Address, data: &[u8]) -> Bytes {
    let mut out = Vec::with_capacity(4 + 96 + data.len());
    out.extend_from_slice(&selector4("returndataSize(address,bytes)"));
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(target.as_slice());
    out.extend_from_slice(&pad);
    let mut off32 = [0u8; 32];
    off32[31] = 0x40;
    out.extend_from_slice(&off32);
    let mut len32 = [0u8; 32];
    len32[24..32].copy_from_slice(&(data.len() as u64).to_be_bytes());
    out.extend_from_slice(&len32);
    out.extend_from_slice(data);
    while out.len() % 32 != 0 {
        out.push(0);
    }
    Bytes::from(out)
}

fn returndata_slice_cd(target: Address, data: &[u8], offset: U256, size: U256) -> Bytes {
    let mut out = Vec::with_capacity(4 + 160 + data.len());
    out.extend_from_slice(&selector4("returndataSlice(address,bytes,uint256,uint256)"));
    let mut pad = [0u8; 32];
    pad[12..].copy_from_slice(target.as_slice());
    out.extend_from_slice(&pad);
    let mut off32 = [0u8; 32];
    off32[31] = 0x80;
    out.extend_from_slice(&off32);
    out.extend_from_slice(&offset.to_be_bytes::<32>());
    out.extend_from_slice(&size.to_be_bytes::<32>());
    let mut len32 = [0u8; 32];
    len32[24..32].copy_from_slice(&(data.len() as u64).to_be_bytes());
    out.extend_from_slice(&len32);
    out.extend_from_slice(data);
    while out.len() % 32 != 0 {
        out.push(0);
    }
    Bytes::from(out)
}

fn helper_return_n_zeros(n: u32) -> Vec<u8> {
    let len = n.to_be_bytes();
    vec![0x63, len[0], len[1], len[2], len[3], 0x60, 0x00, 0xf3]
}

fn helper_return_keccak_of_input() -> Vec<u8> {
    vec![
        0x36, 0x60, 0x00, 0x60, 0x00, 0x37, 0x36, 0x60, 0x00, 0x20, 0x60, 0x00, 0x52, 0x60, 0x20,
        0x60, 0x00, 0xf3,
    ]
}

#[test]
#[ignore]
fn return_data_size_zero() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&helper_return_n_zeros(0));
    let tx = signed(
        3,
        Some(stylus()),
        returndata_size_cd(helper(), &[]),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("return_data_size_zero", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn return_data_size_one_word() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&helper_return_n_zeros(32));
    let tx = signed(
        3,
        Some(stylus()),
        returndata_size_cd(helper(), &[]),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("return_data_size_one_word", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn return_data_size_64kb() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&helper_return_n_zeros(65_536));
    let tx = signed(
        3,
        Some(stylus()),
        returndata_size_cd(helper(), &[]),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("return_data_size_64kb", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn return_data_slice_within_bounds() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&helper_return_keccak_of_input());
    let tx = signed(
        3,
        Some(stylus()),
        returndata_slice_cd(
            helper(),
            &[0xde, 0xad, 0xbe, 0xef],
            U256::from(0u64),
            U256::from(32u64),
        ),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("return_data_slice_within", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn return_data_slice_partial_offset() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&helper_return_n_zeros(32));
    let tx = signed(
        3,
        Some(stylus()),
        returndata_slice_cd(helper(), &[], U256::from(16u64), U256::from(8u64)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("return_data_slice_partial", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn return_data_slice_offset_at_end() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&helper_return_n_zeros(32));
    let tx = signed(
        3,
        Some(stylus()),
        returndata_slice_cd(helper(), &[], U256::from(32u64), U256::from(0u64)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("return_data_slice_end", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn return_data_slice_size_zero() {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&helper_return_n_zeros(32));
    let tx = signed(
        3,
        Some(stylus()),
        returndata_slice_cd(helper(), &[], U256::from(8u64), U256::from(0u64)),
        U256::ZERO,
        INVOKE_GAS_CAP,
    )
    .build()
    .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    GuardedRun::new("return_data_slice_zero_size", steps)
        .expect_last_tx_status(true)
        .expect_last_tx_min_gas(25_000)
        .run();
}

#[test]
#[ignore]
fn _unused_keccak() {
    let _ = keccak256(b"keep import alive");
}
