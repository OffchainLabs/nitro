//! Per-selector gas pins for ArbRetryableTx (0x6e).

mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::B256;
use arb_precompiles::create_arbretryabletx_precompile;
use common::{calldata, word_u256, PrecompileTest};

const ARBOS_V30: u64 = 30;

fn arbretryabletx(ctx: std::sync::Arc<arb_context::ArbPrecompileCtx>) -> DynPrecompile {
    create_arbretryabletx_precompile(ctx)
}

fn fixture() -> PrecompileTest {
    PrecompileTest::new().arbos_version(ARBOS_V30).arbos_state()
}

#[test]
fn get_lifetime_v30_gas_pin() {
    let run = fixture().call(arbretryabletx, &calldata("getLifetime()", &[]));
    assert_eq!(run.gas_used(), 803);
}

#[test]
fn get_current_redeemer_v30_gas_pin() {
    let run = fixture().call(arbretryabletx, &calldata("getCurrentRedeemer()", &[]));
    assert_eq!(run.gas_used(), 803);
}

#[test]
fn submit_retryable_v30_revert_gas_pin() {
    let payload = vec![0u8; 11 * 32 + 32];
    let mut data = vec![0xc9, 0xf9, 0x5d, 0x32];
    data.extend_from_slice(&payload);
    let run = fixture().call(arbretryabletx, &data.into());
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 839);
}

#[test]
fn get_timeout_unknown_v30_revert_gas_pin() {
    let ticket_id = B256::repeat_byte(0x42);
    let run = fixture().call(
        arbretryabletx,
        &calldata("getTimeout(bytes32)", &[word_u256(ticket_id.into())]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 1606);
}

#[test]
fn get_beneficiary_unknown_v30_revert_gas_pin() {
    let ticket_id = B256::repeat_byte(0x43);
    let run = fixture().call(
        arbretryabletx,
        &calldata("getBeneficiary(bytes32)", &[word_u256(ticket_id.into())]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 1606);
}

#[test]
fn redeem_unknown_v30_revert_gas_pin() {
    let ticket_id = B256::repeat_byte(0x44);
    let run = fixture().call(
        arbretryabletx,
        &calldata("redeem(bytes32)", &[word_u256(ticket_id.into())]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 2406);
}

#[test]
fn keepalive_unknown_v30_revert_gas_pin() {
    let ticket_id = B256::repeat_byte(0x45);
    let run = fixture().call(
        arbretryabletx,
        &calldata("keepalive(bytes32)", &[word_u256(ticket_id.into())]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 1606);
}

#[test]
fn cancel_unknown_v30_revert_gas_pin() {
    let ticket_id = B256::repeat_byte(0x46);
    let run = fixture().call(
        arbretryabletx,
        &calldata("cancel(bytes32)", &[word_u256(ticket_id.into())]),
    );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 1606);
}
