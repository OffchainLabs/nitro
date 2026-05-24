use alloy_primitives::{Address, Bytes, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    guards::GuardedRun,
    scaffolding::{fund_interop_eoa, signed, INVOKE_GAS_CAP},
    shared_nodes::next_msg_idx,
};
use arb_test_harness::messaging::MessageBuilder;

fn run_call(name: &str, to: Address, data: Bytes, expect_success: bool, min_gas: u64) {
    let mut steps = Vec::new();
    fund_interop_eoa(&mut steps);
    let tx = signed(0, Some(to), data, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    let mut g = GuardedRun::new(name, steps).expect_last_tx_min_gas(min_gas);
    if expect_success {
        g = g.expect_last_tx_status(true);
    }
    g.run();
}

#[test]
#[ignore]
fn ecrecover_valid_inputs() {
    let mut data = vec![0u8; 128];
    data[31] = 0x01;
    data[63] = 27;
    run_call(
        "ec_valid",
        Address::with_last_byte(0x01),
        Bytes::from(data),
        true,
        25_000,
    );
}

#[test]
#[ignore]
fn ecrecover_zero_input() {
    let data = vec![0u8; 128];
    run_call(
        "ec_zero",
        Address::with_last_byte(0x01),
        Bytes::from(data),
        true,
        25_000,
    );
}

#[test]
#[ignore]
fn sha256_empty() {
    run_call(
        "sha256_empty",
        Address::with_last_byte(0x02),
        Bytes::new(),
        true,
        25_000,
    );
}

#[test]
#[ignore]
fn sha256_one_kb() {
    run_call(
        "sha256_1kb",
        Address::with_last_byte(0x02),
        Bytes::from(vec![0x77u8; 1024]),
        true,
        25_000,
    );
}

#[test]
#[ignore]
fn ripemd160_eoa() {
    run_call(
        "ripemd",
        Address::with_last_byte(0x03),
        Bytes::from(vec![0xab; 32]),
        true,
        25_000,
    );
}

#[test]
#[ignore]
fn identity_passes_through() {
    run_call(
        "identity",
        Address::with_last_byte(0x04),
        Bytes::from(vec![0xde, 0xad, 0xbe, 0xef]),
        true,
        25_000,
    );
}

#[test]
#[ignore]
fn modexp_small() {
    let mut data = Vec::with_capacity(96 + 3);
    let mut base_len = [0u8; 32];
    base_len[31] = 1;
    let mut exp_len = [0u8; 32];
    exp_len[31] = 1;
    let mut mod_len = [0u8; 32];
    mod_len[31] = 1;
    data.extend_from_slice(&base_len);
    data.extend_from_slice(&exp_len);
    data.extend_from_slice(&mod_len);
    data.push(0x03);
    data.push(0x02);
    data.push(0x05);
    run_call(
        "modexp_small",
        Address::with_last_byte(0x05),
        Bytes::from(data),
        true,
        25_000,
    );
}

#[test]
#[ignore]
fn modexp_zero_modulus_returns_zero() {
    let mut data = Vec::with_capacity(99);
    let mut len = [0u8; 32];
    len[31] = 1;
    data.extend_from_slice(&len);
    data.extend_from_slice(&len);
    data.extend_from_slice(&len);
    data.push(0x05);
    data.push(0x02);
    data.push(0x00);
    run_call(
        "modexp_zero_mod",
        Address::with_last_byte(0x05),
        Bytes::from(data),
        true,
        25_000,
    );
}

#[test]
#[ignore]
fn ec_add_zero_points() {
    let data = vec![0u8; 128];
    run_call(
        "ecadd_zero",
        Address::with_last_byte(0x06),
        Bytes::from(data),
        true,
        25_000,
    );
}

#[test]
#[ignore]
fn ec_mul_zero_point() {
    let data = vec![0u8; 96];
    run_call(
        "ecmul_zero",
        Address::with_last_byte(0x07),
        Bytes::from(data),
        true,
        25_000,
    );
}

#[test]
#[ignore]
fn ec_pairing_empty() {
    run_call(
        "ecpair_empty",
        Address::with_last_byte(0x08),
        Bytes::new(),
        true,
        25_000,
    );
}

#[test]
#[ignore]
fn blake2f_invalid_input_reverts() {
    let data = vec![0u8; 100];
    run_call(
        "blake2_invalid",
        Address::with_last_byte(0x09),
        Bytes::from(data),
        false,
        21_000,
    );
}
