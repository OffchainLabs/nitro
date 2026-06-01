use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{keccak256, Address, Log, B256, U256};
use alloy_sol_types::{SolError, SolEvent, SolInterface};
use arb_context::ArbPrecompileCtx;
use arb_storage::ARBOS_STATE_ADDRESS;
use arbos::retryables::{RetryableError, RETRYABLE_LIFETIME_SECONDS, RETRYABLE_REAP_PRICE};
use revm::precompile::{PrecompileId, PrecompileOutput, PrecompileResult};
use std::sync::Arc;

use crate::{interfaces::IArbRetryableTx, ArbPrecompileError};

/// ArbRetryableTx precompile address (0x6e).
pub const ARBRETRYABLETX_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x6e,
]);

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const SSTORE_ZERO_GAS: u64 = 5_000;
const SSTORE_RESET_GAS: u64 = 5_000;
const COPY_GAS: u64 = 3;
const TX_GAS: u64 = 21_000;
const LOG_GAS: u64 = 375;
const LOG_TOPIC_GAS: u64 = 375;
const LOG_DATA_GAS: u64 = 8;

/// ABI-encoded data size for RedeemScheduled: 4 non-indexed params × 32 bytes.
const REDEEM_SCHEDULED_DATA_BYTES: u64 = 128;

/// Gas cost for emitting the RedeemScheduled event (LOG4 with 128 data bytes).
const REDEEM_SCHEDULED_EVENT_COST: u64 =
    LOG_GAS + 4 * LOG_TOPIC_GAS + LOG_DATA_GAS * REDEEM_SCHEDULED_DATA_BYTES;

pub fn ticket_created_topic() -> B256 {
    IArbRetryableTx::TicketCreated::SIGNATURE_HASH
}

pub fn redeem_scheduled_topic() -> B256 {
    IArbRetryableTx::RedeemScheduled::SIGNATURE_HASH
}

pub fn lifetime_extended_topic() -> B256 {
    IArbRetryableTx::LifetimeExtended::SIGNATURE_HASH
}

pub fn canceled_topic() -> B256 {
    IArbRetryableTx::Canceled::SIGNATURE_HASH
}

pub fn create_arbretryabletx_precompile(ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbretryabletx"), move |input| {
        handler(input, &ctx)
    })
}

fn handler(mut input: PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    let mut gas_used = 0u64;
    let gas_limit = input.gas;
    crate::init_precompile_gas(&mut gas_used, ctx, input.data.len());

    let call = match IArbRetryableTx::ArbRetryableTxCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbRetryableTx::ArbRetryableTxCalls as Calls;
    let result = match call {
        Calls::getLifetime(_) => {
            let lifetime = U256::from(RETRYABLE_LIFETIME_SECONDS);
            crate::charge_computation(&mut gas_used, ctx, COPY_GAS);
            Ok(PrecompileOutput::new(
                (gas_used).min(gas_limit),
                lifetime.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        Calls::getCurrentRedeemer(_) => {
            let redeemer = ctx.tx_snapshot().redeemer_word();
            crate::charge_computation(&mut gas_used, ctx, COPY_GAS);
            Ok(PrecompileOutput::new(
                (gas_used).min(gas_limit),
                redeemer.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        Calls::submitRetryable(_) => {
            let data = IArbRetryableTx::NotCallable {}.abi_encode();
            return crate::sol_error_revert(&mut gas_used, ctx, data, gas_limit);
        }
        Calls::getTimeout(c) => handle_get_timeout(&mut input, &mut gas_used, c.ticketId, ctx),
        Calls::getBeneficiary(c) => {
            handle_get_beneficiary(&mut input, ctx, &mut gas_used, c.ticketId)
        }
        Calls::redeem(c) => handle_redeem(&mut input, ctx, &mut gas_used, c.ticketId),
        Calls::keepalive(c) => handle_keepalive(&mut input, ctx, &mut gas_used, c.ticketId),
        Calls::cancel(c) => handle_cancel(&mut input, ctx, &mut gas_used, c.ticketId),
    };
    crate::gas_check(ctx, gas_limit, gas_used, result)
}

fn load_arbos(input: &mut PrecompileInput<'_>) -> Result<(), ArbPrecompileError> {
    input
        .internals_mut()
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;
    Ok(())
}

fn current_timestamp(input: &PrecompileInput<'_>) -> u64 {
    input
        .internals()
        .block_timestamp()
        .try_into()
        .unwrap_or(u64::MAX)
}

/// Maps `RetryableError` from a precompile-layer lookup into the appropriate
/// revert/fatal outcome: storage faults are fatal, missing-ticket/auth/window
/// failures revert.
fn map_retryable_error(err: RetryableError, gas_used: u64) -> ArbPrecompileError {
    match err {
        RetryableError::Storage(_) => ArbPrecompileError::fatal(err),
        _ => ArbPrecompileError::empty_revert(gas_used),
    }
}

/// Pre-v3 burns the remaining gas; v3+ emits the `NoTicketWithIDError`
/// sol-error.
fn not_found_revert(
    ctx: &ArbPrecompileCtx,
    gas_used: &mut u64,
    gas_limit: u64,
) -> PrecompileResult {
    if ctx.block.arbos_version < arb_chainspec::arbos_version::ARBOS_VERSION_3 {
        return crate::burn_all_revert(gas_limit);
    }
    let data = IArbRetryableTx::NoTicketWithID {}.abi_encode();
    crate::sol_error_revert(gas_used, ctx, data, gas_limit)
}

fn handle_get_timeout(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ticket_id: B256,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let now = current_timestamp(input);
    load_arbos(input)?;

    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;

    let effective_timeout = match arb_state
        .retryable_state
        .get_timeout(internals, ticket_id, now)
    {
        Ok(t) => t,
        Err(RetryableError::NoTicketWithId) => {
            crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);
            let data = IArbRetryableTx::NoTicketWithID {}.abi_encode();
            return crate::sol_error_revert(gas_used, ctx, data, gas_limit);
        }
        Err(e) => return Err(map_retryable_error(e, *gas_used).into()),
    };

    crate::charge_storage_read(gas_used, ctx, 3 * SLOAD_GAS);
    crate::charge_computation(gas_used, ctx, COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        U256::from(effective_timeout)
            .to_be_bytes::<32>()
            .to_vec()
            .into(),
    ))
}

fn handle_get_beneficiary(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    gas_used: &mut u64,
    ticket_id: B256,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let now = current_timestamp(input);
    load_arbos(input)?;

    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;

    let beneficiary = match arb_state
        .retryable_state
        .get_beneficiary(internals, ticket_id, now)
    {
        Ok(addr) => addr,
        Err(RetryableError::NoTicketWithId) => {
            crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);
            return not_found_revert(ctx, gas_used, gas_limit);
        }
        Err(e) => return Err(map_retryable_error(e, *gas_used).into()),
    };

    crate::charge_storage_read(gas_used, ctx, 2 * SLOAD_GAS);
    crate::charge_computation(gas_used, ctx, COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        U256::from_be_slice(beneficiary.as_slice())
            .to_be_bytes::<32>()
            .to_vec()
            .into(),
    ))
}

fn handle_redeem(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    gas_used: &mut u64,
    ticket_id: B256,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let caller = input.caller;
    let now = current_timestamp(input);

    {
        let current_retryable = ctx.tx_snapshot().retryable_id;
        if !current_retryable.is_zero() && current_retryable == ticket_id {
            return Err(ArbPrecompileError::empty_revert(*gas_used).into());
        }
    }

    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let retryable_state = &arb_state.retryable_state;

    let opened = retryable_state
        .open_retryable(internals, ticket_id, now)
        .map_err(|e| map_retryable_error(e, *gas_used))?;
    crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);

    let calldata_raw_size = if let Some(ref ret) = opened {
        let size = ret
            .calldata_size(internals)
            .map_err(|e| map_retryable_error(e, *gas_used))?;
        crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);
        size
    } else {
        0
    };

    let calldata_words = calldata_raw_size.div_ceil(32);
    let write_bytes = if opened.is_some() {
        let nbytes = 6 * 32 + 32 + 32 * calldata_words;
        nbytes.div_ceil(32)
    } else {
        0
    };

    const PARAMS_SLOAD_GAS: u64 = 50;
    crate::charge_storage_read(gas_used, ctx, PARAMS_SLOAD_GAS.saturating_mul(write_bytes));

    let nonce = match retryable_state.increment_num_tries_for(internals, ticket_id, now) {
        Ok(n) => n,
        Err(RetryableError::NoTicketWithId) => {
            crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);
            return not_found_revert(ctx, gas_used, gas_limit);
        }
        Err(e) => return Err(map_retryable_error(e, *gas_used).into()),
    };
    crate::charge_storage_read(gas_used, ctx, 2 * SLOAD_GAS);
    crate::charge_storage_write(gas_used, ctx, SSTORE_GAS);

    let make_tx_reads = 5 + calldata_raw_size / 32;
    crate::charge_storage_read(gas_used, ctx, make_tx_reads * SLOAD_GAS);

    let mut hash_input = [0u8; 64];
    hash_input[..32].copy_from_slice(ticket_id.as_slice());
    hash_input[32..].copy_from_slice(&U256::from(nonce).to_be_bytes::<32>());
    let retry_tx_hash = keccak256(hash_input);

    let backlog_reservation = compute_backlog_update_cost(input, ctx, gas_used)?;

    let gas_used_so_far = *gas_used;
    let future_gas_costs = REDEEM_SCHEDULED_EVENT_COST + COPY_GAS + backlog_reservation;
    let gas_remaining = gas_limit.saturating_sub(gas_used_so_far);
    if gas_remaining < future_gas_costs + TX_GAS {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }
    let gas_to_donate = gas_remaining - future_gas_costs;

    let actual_backlog_cost = compute_actual_backlog_cost(input, ctx, gas_to_donate)?;

    let max_refund = U256::MAX;
    let submission_fee_refund = U256::ZERO;

    let topic0 = redeem_scheduled_topic();
    let topic1 = ticket_id;
    let topic2 = B256::from(retry_tx_hash);
    let mut seq_bytes = [0u8; 32];
    seq_bytes[24..32].copy_from_slice(&nonce.to_be_bytes());
    let topic3 = B256::from(seq_bytes);

    let mut event_data = Vec::with_capacity(128);
    event_data.extend_from_slice(&U256::from(gas_to_donate).to_be_bytes::<32>());
    event_data.extend_from_slice(&B256::left_padding_from(caller.as_slice()).0);
    event_data.extend_from_slice(&max_refund.to_be_bytes::<32>());
    event_data.extend_from_slice(&submission_fee_refund.to_be_bytes::<32>());

    input.internals_mut().log(Log::new_unchecked(
        ARBRETRYABLETX_ADDRESS,
        vec![topic0, topic1, topic2, topic3],
        event_data.into(),
    ));

    crate::charge_history_growth(gas_used, ctx, REDEEM_SCHEDULED_EVENT_COST);
    // `gas_to_donate` is forwarded to the scheduled retry tx and folds into
    // its execution gas; attribute as Computation at this precompile's
    // receipt to preserve the single-gas total.
    crate::charge_computation(gas_used, ctx, gas_to_donate);
    // `actual_backlog_cost` is the per-constraint SLOAD + SSTORE charge for
    // updating each constraint's backlog when the donation is applied.
    crate::charge_storage_write(gas_used, ctx, actual_backlog_cost);
    crate::charge_computation(gas_used, ctx, COPY_GAS);
    let _ = gas_used_so_far;

    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        retry_tx_hash.to_vec().into(),
    ))
}

fn handle_keepalive(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    gas_used: &mut u64,
    ticket_id: B256,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let now = current_timestamp(input);
    load_arbos(input)?;

    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let retryable_state = &arb_state.retryable_state;

    let calldata_size = retryable_state
        .calldata_size_for(internals, ticket_id, now)
        .map_err(|e| map_retryable_error(e, *gas_used))?;

    let window_limit = now + RETRYABLE_LIFETIME_SECONDS;
    let new_timeout = match retryable_state.keepalive(internals, ticket_id, now, window_limit, 0) {
        Ok(t) => t,
        Err(RetryableError::NoTicketWithId) => {
            crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);
            return not_found_revert(ctx, gas_used, gas_limit);
        }
        Err(RetryableError::TimeoutTooFarFuture) => {
            return Err(ArbPrecompileError::empty_revert(*gas_used).into());
        }
        Err(e) => return Err(map_retryable_error(e, *gas_used).into()),
    };

    let topic0 = lifetime_extended_topic();
    let mut event_data = Vec::with_capacity(32);
    event_data.extend_from_slice(&U256::from(new_timeout).to_be_bytes::<32>());
    input.internals_mut().log(Log::new_unchecked(
        ARBRETRYABLETX_ADDRESS,
        vec![topic0, ticket_id],
        event_data.into(),
    ));

    let calldata_words = calldata_size.div_ceil(32);
    let nbytes = 6 * 32 + 32 + 32 * calldata_words;
    let update_cost = nbytes.div_ceil(32) * (SSTORE_GAS / 100);
    let event_cost = LOG_GAS + 2 * LOG_TOPIC_GAS + LOG_DATA_GAS * 32;

    // Init already covered the framework SLOAD; body adds the remaining
    // 7 retryable SLOADs, 3 writes (timeout/ttl bumps), 2 result/COPY, the
    // calldata-bytes "phantom" update_cost (a stylus-cache style warm cost
    // — Read), the LifetimeExtended log, and the reap-price fee.
    crate::charge_storage_read(gas_used, ctx, 7 * SLOAD_GAS + update_cost);
    crate::charge_storage_write(gas_used, ctx, 3 * SSTORE_GAS);
    crate::charge_history_growth(gas_used, ctx, event_cost);
    crate::charge_computation(gas_used, ctx, 2 * COPY_GAS + RETRYABLE_REAP_PRICE);

    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        U256::from(new_timeout).to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_cancel(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    gas_used: &mut u64,
    ticket_id: B256,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let caller = input.caller;
    let now = current_timestamp(input);
    load_arbos(input)?;

    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let retryable_state = &arb_state.retryable_state;

    let calldata_size = match retryable_state.cancel(internals, ticket_id, caller, now) {
        Ok(size) => size,
        Err(RetryableError::NoTicketWithId) => {
            crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);
            return not_found_revert(ctx, gas_used, gas_limit);
        }
        Err(RetryableError::NotBeneficiary) => {
            crate::charge_storage_read(gas_used, ctx, 2 * SLOAD_GAS);
            return Err(ArbPrecompileError::empty_revert(*gas_used).into());
        }
        Err(e) => return Err(map_retryable_error(e, *gas_used).into()),
    };

    input.internals_mut().log(Log::new_unchecked(
        ARBRETRYABLETX_ADDRESS,
        vec![canceled_topic(), ticket_id],
        Default::default(),
    ));

    let calldata_words = calldata_size.div_ceil(32);
    let clear_bytes_cost = if calldata_size > 0 {
        (calldata_words + 1) * SSTORE_ZERO_GAS
    } else {
        0
    };
    let event_cost = LOG_GAS + 2 * LOG_TOPIC_GAS;

    // Init already covered the framework SLOAD; body adds 5 retryable
    // SLOADs (lookup + auth + state reads), 7 SSTORE-resets for clearing
    // the retryable record + the calldata-byte zeroing.
    crate::charge_storage_read(gas_used, ctx, 5 * SLOAD_GAS);
    crate::charge_storage_write(gas_used, ctx, 7 * SSTORE_ZERO_GAS + clear_bytes_cost);
    crate::charge_history_growth(gas_used, ctx, event_cost);
    crate::charge_computation(gas_used, ctx, COPY_GAS);

    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn compute_backlog_update_cost(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    gas_used: &mut u64,
) -> Result<u64, ArbPrecompileError> {
    use arb_chainspec::arbos_version as arb_ver;
    let arbos_version = ctx.block.arbos_version;
    if arbos_version >= arb_ver::ARBOS_VERSION_MULTI_GAS_CONSTRAINTS {
        return Ok(arbos::l2_pricing::MULTI_CONSTRAINT_STATIC_BACKLOG_UPDATE_COST);
    }

    let mut result = 0u64;
    if arbos_version >= arb_ver::ARBOS_VERSION_50 {
        result += SLOAD_GAS;
    }
    if arbos_version >= arb_ver::ARBOS_VERSION_MULTI_CONSTRAINT_FIX {
        let len = read_gas_constraints_length(input, ctx)?;
        crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);
        if len > 0 {
            result += SLOAD_GAS;
            result += len.saturating_mul(SLOAD_GAS + SSTORE_GAS);
            return Ok(result);
        }
    }
    result += SLOAD_GAS + SSTORE_GAS;
    Ok(result)
}

fn compute_actual_backlog_cost(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    gas_to_donate: u64,
) -> Result<u64, ArbPrecompileError> {
    use arb_chainspec::arbos_version as arb_ver;
    let arbos_version = ctx.block.arbos_version;
    if arbos_version >= arb_ver::ARBOS_VERSION_MULTI_GAS_CONSTRAINTS {
        return Ok(arbos::l2_pricing::MULTI_CONSTRAINT_STATIC_BACKLOG_UPDATE_COST);
    }
    if arbos_version >= arb_ver::ARBOS_VERSION_MULTI_CONSTRAINT_FIX {
        let internals = input.internals_mut();
        let arb_state = ctx
            .block
            .arbos_state(internals)
            .map_err(ArbPrecompileError::fatal)?;
        let len = arb_state
            .l2_pricing_state
            .gas_constraints_length(internals)
            .map_err(ArbPrecompileError::fatal)?;
        if len > 0 {
            let mut total = 2 * SLOAD_GAS;
            for i in 0..len {
                let constraint = arb_state.l2_pricing_state.open_gas_constraint_at(i);
                let backlog = constraint
                    .backlog(internals)
                    .map_err(ArbPrecompileError::fatal)?;
                total += constraint_actual_backlog_cost(backlog, gas_to_donate);
            }
            return Ok(total);
        }
    }
    Ok(legacy_actual_backlog_cost(
        ctx.block.current_gas_backlog(),
        gas_to_donate,
    ))
}

fn legacy_actual_backlog_cost(current_backlog: u64, gas_to_donate: u64) -> u64 {
    let new_backlog = current_backlog.saturating_sub(gas_to_donate);
    let write_cost = if new_backlog == 0 {
        SSTORE_RESET_GAS
    } else {
        SSTORE_GAS
    };
    SLOAD_GAS + write_cost
}

#[inline]
fn constraint_actual_backlog_cost(current_backlog: u64, gas_to_donate: u64) -> u64 {
    let new_backlog = current_backlog.saturating_sub(gas_to_donate);
    let write_cost = if new_backlog == 0 {
        SSTORE_RESET_GAS
    } else {
        SSTORE_GAS
    };
    SLOAD_GAS + write_cost
}

fn read_gas_constraints_length(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> Result<u64, ArbPrecompileError> {
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .l2_pricing_state
        .gas_constraints_length(internals)
        .map_err(ArbPrecompileError::fatal)
}

#[cfg(test)]
mod redeem_gas_tests {
    use super::*;

    #[test]
    fn drains_backlog_to_zero_uses_sstore_reset() {
        assert_eq!(
            legacy_actual_backlog_cost(100_000, 100_000),
            SLOAD_GAS + SSTORE_RESET_GAS,
        );
        assert_eq!(legacy_actual_backlog_cost(100_000, 100_000), 5_800);
    }

    #[test]
    fn drains_backlog_partially_uses_sstore_set() {
        assert_eq!(
            legacy_actual_backlog_cost(100_000, 99_000),
            SLOAD_GAS + SSTORE_GAS,
        );
        assert_eq!(legacy_actual_backlog_cost(100_000, 99_000), 20_800);
    }

    #[test]
    fn donate_exceeds_backlog_saturates_to_zero() {
        assert_eq!(
            legacy_actual_backlog_cost(50_000, 200_000),
            SLOAD_GAS + SSTORE_RESET_GAS,
        );
    }

    #[test]
    fn empty_backlog_zero_donate_still_writes_zero() {
        assert_eq!(
            legacy_actual_backlog_cost(0, 0),
            SLOAD_GAS + SSTORE_RESET_GAS,
        );
    }

    #[test]
    fn sepolia_block_100_435_687_diverges_by_15000_with_buggy_static_cost() {
        let buggy_static_cost = SLOAD_GAS + SSTORE_GAS;
        let fixed_drain_cost = legacy_actual_backlog_cost(100_000, 100_000);
        assert_eq!(buggy_static_cost - fixed_drain_cost, 15_000);
    }

    #[test]
    fn constraint_with_remaining_backlog_charges_sstore_set() {
        assert_eq!(
            constraint_actual_backlog_cost(10_000_000, 1_100_000),
            SLOAD_GAS + SSTORE_GAS,
        );
    }

    #[test]
    fn constraint_draining_to_zero_charges_sstore_reset() {
        assert_eq!(
            constraint_actual_backlog_cost(100_000, 100_000),
            SLOAD_GAS + SSTORE_RESET_GAS,
        );
    }

    #[test]
    fn six_non_draining_constraints_match_reservation() {
        let len = 6u64;
        let gas_to_donate = 1_100_000u64;
        let per_constraint_backlog = 10_000_000u64;
        let reservation = 2 * SLOAD_GAS + len * (SLOAD_GAS + SSTORE_GAS);
        let actual = 2 * SLOAD_GAS
            + (0..len)
                .map(|_| constraint_actual_backlog_cost(per_constraint_backlog, gas_to_donate))
                .sum::<u64>();
        assert_eq!(actual, reservation);
    }

    #[test]
    fn block_235_386_091_redeem_recovers_full_gas_limit() {
        let gas_limit = 1_200_000u64;
        let len = 6u64;
        let backlog_per_constraint = 10_000_000u64;
        let gas_used_so_far = 50_000u64;
        let reservation = 2 * SLOAD_GAS + len * (SLOAD_GAS + SSTORE_GAS);
        let future = REDEEM_SCHEDULED_EVENT_COST + COPY_GAS + reservation;
        let gas_to_donate = gas_limit - gas_used_so_far - future;

        let buggy_actual = 2 * SLOAD_GAS + len * (SLOAD_GAS + SSTORE_RESET_GAS);
        let buggy_total =
            gas_used_so_far + REDEEM_SCHEDULED_EVENT_COST + gas_to_donate + buggy_actual + COPY_GAS;
        assert_eq!(gas_limit - buggy_total, 90_000);

        let fixed_actual = 2 * SLOAD_GAS
            + (0..len)
                .map(|_| constraint_actual_backlog_cost(backlog_per_constraint, gas_to_donate))
                .sum::<u64>();
        let fixed_total =
            gas_used_so_far + REDEEM_SCHEDULED_EVENT_COST + gas_to_donate + fixed_actual + COPY_GAS;
        assert_eq!(fixed_total, gas_limit);
    }
}
