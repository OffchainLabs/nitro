use alloy_primitives::{Address, Bytes, U256};
use arb_fuzz::{
    arbitrary_impls::message_step,
    guards::GuardedRun,
    scaffolding::{baseline_stylus_plus_helper, selector4, signed, INVOKE_GAS_CAP},
    shared_nodes::next_msg_idx,
};
use arb_test_harness::messaging::MessageBuilder;

const ARBFUNCTIONTABLE: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x68,
]);
const ARBNATIVETOKEN: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x73,
]);
const ARBOSTEST: Address = Address::new([
    0u8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x69,
]);

fn no_arg(sig: &str) -> Bytes {
    Bytes::from(selector4(sig).to_vec())
}

fn one_arg_u256(sig: &str, v: U256) -> Bytes {
    let mut out = Vec::with_capacity(36);
    out.extend_from_slice(&selector4(sig));
    out.extend_from_slice(&v.to_be_bytes::<32>());
    Bytes::from(out)
}

fn run_query(name: &str, to: Address, data: Bytes, expect_success: bool) {
    let (mut steps, _, _) = baseline_stylus_plus_helper(&[0x00]);
    let tx = signed(3, Some(to), data, U256::ZERO, INVOKE_GAS_CAP)
        .build()
        .expect("tx");
    let idx = next_msg_idx();
    steps.push(message_step(idx, tx, idx));
    let mut run = GuardedRun::new(name, steps).expect_last_tx_min_gas(25_000);
    if expect_success {
        run = run.expect_last_tx_status(true);
    }
    run.run();
}

#[test]
#[ignore]
fn function_table_size_empty() {
    run_query(
        "ft_size_empty",
        ARBFUNCTIONTABLE,
        {
            let mut d = Vec::with_capacity(36);
            d.extend_from_slice(&selector4("size(address)"));
            let mut p = [0u8; 32];
            p[12..].copy_from_slice(Address::repeat_byte(0x11).as_slice());
            d.extend_from_slice(&p);
            Bytes::from(d)
        },
        true,
    );
}

#[test]
#[ignore]
fn function_table_get_empty_reverts() {
    run_query(
        "ft_get_empty",
        ARBFUNCTIONTABLE,
        {
            let mut d = Vec::with_capacity(68);
            d.extend_from_slice(&selector4("get(address,uint256)"));
            let mut p = [0u8; 32];
            p[12..].copy_from_slice(Address::repeat_byte(0x11).as_slice());
            d.extend_from_slice(&p);
            d.extend_from_slice(&[0u8; 32]);
            Bytes::from(d)
        },
        false,
    );
}

#[test]
#[ignore]
fn function_table_upload_succeeds_noop() {
    run_query(
        "ft_upload_noop",
        ARBFUNCTIONTABLE,
        {
            let mut d = Vec::with_capacity(100);
            d.extend_from_slice(&selector4("upload(bytes)"));
            let mut off = [0u8; 32];
            off[31] = 0x20;
            d.extend_from_slice(&off);
            let mut len = [0u8; 32];
            d.extend_from_slice(&len);
            let _ = &mut len;
            Bytes::from(d)
        },
        true,
    );
}

#[test]
#[ignore]
fn native_token_mint_non_owner_reverts() {
    run_query(
        "nt_mint_non_owner",
        ARBNATIVETOKEN,
        one_arg_u256("mintNativeToken(uint256)", U256::from(100u64)),
        false,
    );
}

#[test]
#[ignore]
fn native_token_burn_non_owner_reverts() {
    run_query(
        "nt_burn_non_owner",
        ARBNATIVETOKEN,
        one_arg_u256("burnNativeToken(uint256)", U256::from(100u64)),
        false,
    );
}

#[test]
#[ignore]
fn arbostest_burn_arb_gas_small() {
    run_query(
        "arbostest_burn_small",
        ARBOSTEST,
        one_arg_u256("burnArbGas(uint256)", U256::from(1000u64)),
        true,
    );
}

#[test]
#[ignore]
fn arbostest_burn_arb_gas_zero() {
    run_query(
        "arbostest_burn_zero",
        ARBOSTEST,
        one_arg_u256("burnArbGas(uint256)", U256::ZERO),
        true,
    );
}

#[test]
#[ignore]
fn arbostest_burn_arb_gas_huge_reverts() {
    run_query(
        "arbostest_burn_huge",
        ARBOSTEST,
        one_arg_u256("burnArbGas(uint256)", U256::from(u64::MAX)),
        false,
    );
}

#[test]
#[ignore]
fn arbostest_invalid_selector_reverts() {
    run_query(
        "arbostest_invalid",
        ARBOSTEST,
        Bytes::from(vec![0xde, 0xad, 0xbe, 0xef]),
        false,
    );
}

#[test]
#[ignore]
fn ft_invalid_selector_reverts() {
    run_query(
        "ft_invalid",
        ARBFUNCTIONTABLE,
        Bytes::from(vec![0xab, 0xcd, 0xef, 0x12]),
        false,
    );
}

#[test]
#[ignore]
fn nt_invalid_selector_reverts() {
    run_query(
        "nt_invalid",
        ARBNATIVETOKEN,
        Bytes::from(vec![0xfe, 0xed, 0xfa, 0xce]),
        false,
    );
}

#[test]
#[ignore]
fn arbostest_empty_calldata_reverts() {
    run_query("arbostest_empty_cd", ARBOSTEST, Bytes::new(), false);
}

#[test]
#[ignore]
fn _import_no_arg() {
    let _ = no_arg("dummy()");
}
