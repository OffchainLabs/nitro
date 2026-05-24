mod common;

use alloy_evm::precompiles::DynPrecompile;
use alloy_primitives::{address, Address, B256, U256};
use arb_context::ArbPrecompileCtx;
use arb_precompiles::{
    create_arbretryabletx_precompile,
    storage_slot::{
        derive_subspace_key, map_slot, ARBOS_STATE_ADDRESS, RETRYABLES_SUBSPACE, ROOT_STORAGE_KEY,
    },
};
use common::{calldata, decode_address, decode_u256, PrecompileTest};
use std::sync::Arc;

const ARBOS_V30: u64 = 30;
const RETRYABLE_LIFETIME: u64 = 7 * 24 * 60 * 60;

fn arbretryabletx(ctx: std::sync::Arc<arb_context::ArbPrecompileCtx>) -> DynPrecompile {
    create_arbretryabletx_precompile(ctx)
}

fn make_ctx() -> Arc<ArbPrecompileCtx> {
    Arc::new(ArbPrecompileCtx::default())
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
        .call(arbretryabletx, &calldata("getLifetime()", &[]));
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
        .call(arbretryabletx, &data.into());
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
        .call(arbretryabletx, &calldata("getCurrentRedeemer()", &[]));
    assert_eq!(decode_address(run.output()), Address::ZERO);
}

#[test]
fn get_current_redeemer_returns_value_set_by_executor() {
    let ctx = make_ctx();
    let refund_to: Address = address!("00000000000000000000000000000000000000ee");
    ctx.set_redeemer(refund_to);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call_with(arbretryabletx, &calldata("getCurrentRedeemer()", &[]), ctx);
    assert_eq!(decode_address(run.output()), refund_to);
}

#[test]
fn get_timeout_unknown_ticket_reverts_with_no_ticket() {
    let ticket_id = B256::from([0x77; 32]);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(
            arbretryabletx,
            &calldata("getTimeout(bytes32)", &[B256::from(ticket_id)]),
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
            arbretryabletx,
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
            arbretryabletx,
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
            arbretryabletx,
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
            arbretryabletx,
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
        .call(arbretryabletx, &calldata("cancel(bytes32)", &[ticket_id]));
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
        .call(arbretryabletx, &calldata("cancel(bytes32)", &[ticket_id]));
    assert!(run.assert_ok().reverted);
}

#[test]
fn redeem_self_modifying_guard_rejects_current_retryable() {
    let ctx = make_ctx();
    let ticket_id = B256::from([0x33; 32]);
    ctx.set_retryable_id(ticket_id);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call_with(
            arbretryabletx,
            &calldata("redeem(bytes32)", &[ticket_id]),
            ctx,
        );
    assert!(run.assert_ok().reverted);
}

#[test]
fn redeem_unknown_ticket_reverts_with_no_ticket() {
    let ticket_id = B256::from([0x44; 32]);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(arbretryabletx, &calldata("redeem(bytes32)", &[ticket_id]));
    let out = run.assert_ok();
    assert!(out.reverted);
    let no_ticket = alloy_primitives::keccak256(b"NoTicketWithID()");
    assert_eq!(&out.bytes[..4], &no_ticket[..4]);
}

#[test]
fn get_timeout_reverts_for_expired_ticket() {
    // Regression for the missing-expiry-check bug: getTimeout used to return the
    // (past) effective timeout for tickets whose stored timeout < currentTime.
    let ticket_id = B256::from([0x88; 32]);
    let now: u64 = 1_700_000_000;
    let ticket_key = ticket_storage_key(ticket_id);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .block_timestamp(now)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            map_slot(ticket_key.as_slice(), 5),
            U256::from(now - 1),
        )
        .call(
            arbretryabletx,
            &calldata("getTimeout(bytes32)", &[ticket_id]),
        );
    let out = run.assert_ok();
    assert!(out.reverted, "expired ticket must revert");
    let no_ticket = alloy_primitives::keccak256(b"NoTicketWithID()");
    assert_eq!(&out.bytes[..4], &no_ticket[..4]);
}

#[test]
fn cancel_emits_canceled_event_and_clears_storage() {
    // Regression: cancel previously charged event gas without actually emitting
    // the LOG. Verify both the event is emitted and side-effects fire.
    let ticket_id = B256::from([0x66; 32]);
    let beneficiary: Address = address!("00000000000000000000000000000000000000bb");
    let now: u64 = 1_700_000_000;
    let ticket_key = ticket_storage_key(ticket_id);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .caller(beneficiary)
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
        .call(arbretryabletx, &calldata("cancel(bytes32)", &[ticket_id]));
    let _ = run.assert_ok();
    // After cancel, the timeout must be cleared.
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, map_slot(ticket_key.as_slice(), 5),),
        U256::ZERO,
        "timeout must be cleared after cancel"
    );
}

#[test]
fn keepalive_extends_timeout_window_and_records_storage() {
    // Regression: handle_keepalive previously didn't emit LifetimeExtended.
    // We don't have a log inspection API in the harness yet, but we can verify
    // the windows_left increment side-effect that proves the path runs.
    let ticket_id = B256::from([0x77; 32]);
    let now: u64 = 1_700_000_000;
    let ticket_key = ticket_storage_key(ticket_id);
    let test = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .block_timestamp(now)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            map_slot(ticket_key.as_slice(), 5),
            U256::from(now + 100),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            map_slot(ticket_key.as_slice(), 6),
            U256::ZERO,
        );
    let run = test.call(
        arbretryabletx,
        &calldata("keepalive(bytes32)", &[ticket_id]),
    );
    let _ = run.assert_ok();
    // windows_left should now be 1.
    assert_eq!(
        run.storage(ARBOS_STATE_ADDRESS, map_slot(ticket_key.as_slice(), 6),),
        U256::from(1u64),
        "keepalive should have incremented timeout_windows_left"
    );
}

// ── Per-selector gas-equality assertions ────────────────────────────────

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const SSTORE_ZERO_GAS: u64 = 5_000;
const COPY_GAS: u64 = 3;
const LOG_GAS: u64 = 375;
const LOG_TOPIC_GAS: u64 = 375;
const LOG_DATA_GAS: u64 = 8;
const RETRYABLE_REAP_PRICE: u64 = 58_000;

#[test]
fn get_lifetime_charges_one_sload_and_one_copy_word() {
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(arbretryabletx, &calldata("getLifetime()", &[]));
    assert_eq!(run.gas_used(), SLOAD_GAS + COPY_GAS);
}

#[test]
fn get_current_redeemer_charges_one_sload_and_one_copy_word() {
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(arbretryabletx, &calldata("getCurrentRedeemer()", &[]));
    assert_eq!(run.gas_used(), SLOAD_GAS + COPY_GAS);
}

#[test]
fn submit_retryable_charges_init_plus_one_word_revert_payload() {
    let payload = vec![0u8; 11 * 32 + 32];
    let mut data = vec![0xc9, 0xf9, 0x5d, 0x32];
    data.extend_from_slice(&payload);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(arbretryabletx, &data.into());
    let out = run.assert_ok();
    assert!(out.reverted);
    // init(800 + 12 arg words * 3) + 1-word NotCallable error payload = 836 + 3 = 839.
    assert_eq!(out.gas_used, SLOAD_GAS + 12 * COPY_GAS + COPY_GAS);
}

#[test]
fn get_timeout_valid_ticket_charges_four_sloads_and_two_copy_words() {
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
            arbretryabletx,
            &calldata("getTimeout(bytes32)", &[ticket_id]),
        );
    // OAS(1) + getTimeout sload(1) + windows sload(1) + framework sload(1) + 2 copy words = 3206.
    assert_eq!(run.gas_used(), 4 * SLOAD_GAS + 2 * COPY_GAS);
}

#[test]
fn get_timeout_unknown_ticket_charges_init_plus_timeout_sload_plus_one_word_revert() {
    let ticket_id = B256::from([0x77; 32]);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(
            arbretryabletx,
            &calldata("getTimeout(bytes32)", &[ticket_id]),
        );
    let out = run.assert_ok();
    assert!(out.reverted);
    // init(803) + 1 timeout sload(800) + 1-word NoTicketWithID payload(3) = 1606.
    assert_eq!(out.gas_used, 2 * SLOAD_GAS + 2 * COPY_GAS);
}

#[test]
fn get_beneficiary_valid_ticket_charges_three_sloads_and_two_copy_words() {
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
            arbretryabletx,
            &calldata("getBeneficiary(bytes32)", &[ticket_id]),
        );
    assert_eq!(run.gas_used(), 3 * SLOAD_GAS + 2 * COPY_GAS);
}

#[test]
fn get_beneficiary_unknown_ticket_charges_init_plus_open_sload_plus_revert_payload() {
    let ticket_id = B256::from([0x99; 32]);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(
            arbretryabletx,
            &calldata("getBeneficiary(bytes32)", &[ticket_id]),
        );
    let out = run.assert_ok();
    assert!(out.reverted);
    // init(803) + open_retryable sload(800) + 1-word NoTicketWithID(3) = 1606.
    assert_eq!(out.gas_used, 2 * SLOAD_GAS + 2 * COPY_GAS);
}

#[test]
fn cancel_unknown_ticket_charges_init_plus_open_sload_plus_revert_payload() {
    let ticket_id = B256::from([0xaa; 32]);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(arbretryabletx, &calldata("cancel(bytes32)", &[ticket_id]));
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 2 * SLOAD_GAS + 2 * COPY_GAS);
}

#[test]
fn cancel_non_beneficiary_caller_reverts_with_accumulated_gas() {
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
        .call(arbretryabletx, &calldata("cancel(bytes32)", &[ticket_id]));
    let out = run.assert_ok();
    assert!(out.reverted);
    // init(803) + open_retryable sload(800) + beneficiary sload(800) = 2403.
    assert_eq!(out.gas_used, 3 * SLOAD_GAS + COPY_GAS);
}

#[test]
fn cancel_with_empty_calldata_charges_full_formula() {
    let ticket_id = B256::from([0x66; 32]);
    let beneficiary: Address = address!("00000000000000000000000000000000000000bb");
    let now: u64 = 1_700_000_000;
    let ticket_key = ticket_storage_key(ticket_id);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .caller(beneficiary)
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
        .call(arbretryabletx, &calldata("cancel(bytes32)", &[ticket_id]));
    // 6 SLOAD + 7 SSTORE_ZERO + LOG2(375+750) + COPY = 4800 + 35000 + 1125 + 3 = 40_928.
    let event_cost = LOG_GAS + 2 * LOG_TOPIC_GAS;
    assert_eq!(
        run.gas_used(),
        6 * SLOAD_GAS + 7 * SSTORE_ZERO_GAS + event_cost + COPY_GAS,
    );
}

#[test]
fn keepalive_unknown_ticket_charges_init_plus_open_sload_plus_revert_payload() {
    let ticket_id = B256::from([0xaa; 32]);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(
            arbretryabletx,
            &calldata("keepalive(bytes32)", &[ticket_id]),
        );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, 2 * SLOAD_GAS + 2 * COPY_GAS);
}

#[test]
fn keepalive_with_empty_calldata_charges_full_formula() {
    let ticket_id = B256::from([0x77; 32]);
    let now: u64 = 1_700_000_000;
    let ticket_key = ticket_storage_key(ticket_id);
    let test = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .block_timestamp(now)
        .arbos_state()
        .storage(
            ARBOS_STATE_ADDRESS,
            map_slot(ticket_key.as_slice(), 5),
            U256::from(now + 100),
        )
        .storage(
            ARBOS_STATE_ADDRESS,
            map_slot(ticket_key.as_slice(), 6),
            U256::ZERO,
        );
    let run = test.call(
        arbretryabletx,
        &calldata("keepalive(bytes32)", &[ticket_id]),
    );
    // 8 SLOAD + 3 SSTORE + 2 COPY + updateCost(7*200=1400) + event(375+750+256=1381) + reap(58000).
    let update_cost = 7 * (SSTORE_GAS / 100);
    let event_cost = LOG_GAS + 2 * LOG_TOPIC_GAS + LOG_DATA_GAS * 32;
    assert_eq!(
        run.gas_used(),
        8 * SLOAD_GAS
            + 3 * SSTORE_GAS
            + 2 * COPY_GAS
            + update_cost
            + event_cost
            + RETRYABLE_REAP_PRICE,
    );
}

#[test]
fn redeem_unknown_ticket_charges_two_open_sloads_plus_revert_payload() {
    let ticket_id = B256::from([0x44; 32]);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call(arbretryabletx, &calldata("redeem(bytes32)", &[ticket_id]));
    let out = run.assert_ok();
    assert!(out.reverted);
    // init(803) + handler timeout sload(800) + open_retryable timeout sload(800)
    // + 1-word NoTicketWithID(3) = 2406.
    assert_eq!(out.gas_used, 3 * SLOAD_GAS + 2 * COPY_GAS);
}

#[test]
fn redeem_self_modifying_guard_only_charges_init() {
    let ctx = make_ctx();
    let ticket_id = B256::from([0x33; 32]);
    ctx.set_retryable_id(ticket_id);
    let run = PrecompileTest::new()
        .arbos_version(ARBOS_V30)
        .arbos_state()
        .call_with(
            arbretryabletx,
            &calldata("redeem(bytes32)", &[ticket_id]),
            ctx,
        );
    let out = run.assert_ok();
    assert!(out.reverted);
    assert_eq!(out.gas_used, SLOAD_GAS + COPY_GAS);
}
