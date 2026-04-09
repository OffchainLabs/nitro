mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, B256, U256};
use arb_precompiles::{
    create_arbretryabletx_precompile,
    storage_slot::{
        current_redeemer_slot, current_retryable_slot, derive_subspace_key, map_slot,
        ARBOS_STATE_ADDRESS, RETRYABLES_SUBSPACE, ROOT_STORAGE_KEY,
    },
};
use common::{calldata, decode_address, decode_u256, PrecompileTest};

const ARBOS_V30: u64 = 30;
const RETRYABLE_LIFETIME: u64 = 7 * 24 * 60 * 60;

fn arbretryabletx() -> DynPrecompile {
    create_arbretryabletx_precompile()
}

fn ticket_storage_key(ticket_id: B256) -> B256 {
    let retryables_key = derive_subspace_key(ROOT_STORAGE_KEY, RETRYABLES_SUBSPACE);
    derive_subspace_key(retryables_key.as_slice(), ticket_id.as_slice())
}

#[test]
fn get_lifetime_returns_seven_days() {
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(&arbretryabletx(), &calldata("getLifetime()", &[]));
    assert_eq!(decode_u256(run.output()), U256::from(RETRYABLE_LIFETIME));
}

#[test]
fn submit_retryable_always_reverts_with_not_callable() {
    let payload = vec![0u8; 11 * 32 + 32];
    let mut data = vec![0xc9, 0xf9, 0x5d, 0x32];
    data.extend_from_slice(&payload);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(&arbretryabletx(), &data.into());
    let out = run.assert_ok();
    assert!(out.reverted, "SubmitRetryable must revert");
    let not_callable = alloy_primitives::keccak256(b"NotCallable()");
    assert_eq!(&out.bytes[..4], &not_callable[..4]);
}

#[test]
fn get_current_redeemer_returns_zero_outside_retry() {
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(&arbretryabletx(), &calldata("getCurrentRedeemer()", &[]));
    assert_eq!(decode_address(run.output()), Address::ZERO);
}

#[test]
fn get_current_redeemer_returns_value_set_by_executor() {
    let refund_to: Address = address!("00000000000000000000000000000000000000ee");
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            current_redeemer_slot(),
            U256::from_be_slice(refund_to.as_slice()),
        )
        .call(&arbretryabletx(), &calldata("getCurrentRedeemer()", &[]));
    assert_eq!(decode_address(run.output()), refund_to);
}

#[test]
fn get_timeout_unknown_ticket_reverts_with_no_ticket() {
    let ticket_id = B256::from([0x77; 32]);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(
            &arbretryabletx(),
            &calldata(
                "getTimeout(bytes32)",
                &[B256::from(ticket_id)],
            ),
        );
    let out = run.assert_ok();
    assert!(out.reverted);
    let no_ticket = alloy_primitives::keccak256(b"NoTicketWithID()");
    assert_eq!(&out.bytes[..4], &no_ticket[..4]);
}

#[test]
fn get_timeout_returns_effective_timeout_no_extension() {
    let ticket_id = B256::from([0x42; 32]);
    let stored_timeout: u64 = 1_800_000_000;
    let ticket_key = ticket_storage_key(ticket_id);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .block_timestamp(1_700_000_000)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            map_slot(ticket_key.as_slice(), 5),
            U256::from(stored_timeout),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            map_slot(ticket_key.as_slice(), 6),
            U256::ZERO,
        )
        .call(
            &arbretryabletx(),
            &calldata("getTimeout(bytes32)", &[ticket_id]),
        );
    assert_eq!(decode_u256(run.output()), U256::from(stored_timeout));
}

#[test]
fn get_timeout_includes_extra_lifetime_windows() {
    let ticket_id = B256::from([0x42; 32]);
    let stored_timeout: u64 = 1_800_000_000;
    let windows: u64 = 3;
    let ticket_key = ticket_storage_key(ticket_id);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .block_timestamp(1_700_000_000)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            map_slot(ticket_key.as_slice(), 5),
            U256::from(stored_timeout),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            map_slot(ticket_key.as_slice(), 6),
            U256::from(windows),
        )
        .call(
            &arbretryabletx(),
            &calldata("getTimeout(bytes32)", &[ticket_id]),
        );
    let expected = stored_timeout + windows * RETRYABLE_LIFETIME;
    assert_eq!(decode_u256(run.output()), U256::from(expected));
}

#[test]
fn get_beneficiary_unknown_ticket_reverts() {
    let ticket_id = B256::from([0x99; 32]);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(
            &arbretryabletx(),
            &calldata("getBeneficiary(bytes32)", &[ticket_id]),
        );
    let out = run.assert_ok();
    assert!(out.reverted);
    let no_ticket = alloy_primitives::keccak256(b"NoTicketWithID()");
    assert_eq!(&out.bytes[..4], &no_ticket[..4]);
}

#[test]
fn get_beneficiary_returns_stored_address() {
    let ticket_id = B256::from([0x10; 32]);
    let beneficiary: Address = address!("00000000000000000000000000000000000000bb");
    let ticket_key = ticket_storage_key(ticket_id);
    let now: u64 = 1_700_000_000;
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .block_timestamp(now)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            map_slot(ticket_key.as_slice(), 5),
            U256::from(now + RETRYABLE_LIFETIME),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            map_slot(ticket_key.as_slice(), 4),
            U256::from_be_slice(beneficiary.as_slice()),
        )
        .call(
            &arbretryabletx(),
            &calldata("getBeneficiary(bytes32)", &[ticket_id]),
        );
    assert_eq!(decode_address(run.output()), beneficiary);
}

#[test]
fn cancel_unknown_ticket_reverts() {
    let ticket_id = B256::from([0xaa; 32]);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(
            &arbretryabletx(),
            &calldata("cancel(bytes32)", &[ticket_id]),
        );
    let out = run.assert_ok();
    assert!(out.reverted);
}

#[test]
fn cancel_rejects_non_beneficiary_caller() {
    let ticket_id = B256::from([0x55; 32]);
    let beneficiary: Address = address!("00000000000000000000000000000000000000bb");
    let intruder: Address = address!("00000000000000000000000000000000000000cc");
    let now: u64 = 1_700_000_000;
    let ticket_key = ticket_storage_key(ticket_id);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .caller(intruder)
        .block_timestamp(now)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            map_slot(ticket_key.as_slice(), 5),
            U256::from(now + RETRYABLE_LIFETIME),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            map_slot(ticket_key.as_slice(), 4),
            U256::from_be_slice(beneficiary.as_slice()),
        )
        .call(
            &arbretryabletx(),
            &calldata("cancel(bytes32)", &[ticket_id]),
        );
    assert!(run.assert_ok().reverted);
}

#[test]
fn redeem_self_modifying_guard_rejects_current_retryable() {
    let ticket_id = B256::from([0x33; 32]);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            current_retryable_slot(),
            U256::from_be_bytes(ticket_id.0),
        )
        .call(
            &arbretryabletx(),
            &calldata("redeem(bytes32)", &[ticket_id]),
        );
    assert!(run.assert_ok().reverted);
}

#[test]
fn redeem_unknown_ticket_reverts_with_no_ticket() {
    let ticket_id = B256::from([0x44; 32]);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(
            &arbretryabletx(),
            &calldata("redeem(bytes32)", &[ticket_id]),
        );
    let out = run.assert_ok();
    assert!(out.reverted);
    let no_ticket = alloy_primitives::keccak256(b"NoTicketWithID()");
    assert_eq!(&out.bytes[..4], &no_ticket[..4]);
}
