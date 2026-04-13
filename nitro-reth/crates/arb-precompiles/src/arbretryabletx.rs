use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{keccak256, Address, Log, B256, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{
    current_redeemer_slot, current_retryable_slot, derive_subspace_key, map_slot,
    ARBOS_STATE_ADDRESS, RETRYABLES_SUBSPACE, ROOT_STORAGE_KEY,
};

/// ArbRetryableTx precompile address (0x6e).
pub const ARBRETRYABLETX_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x6e,
]);

// Function selectors.
const REDEEM: [u8; 4] = [0xed, 0xa1, 0x12, 0x2c];
const GET_LIFETIME: [u8; 4] = [0x81, 0xe6, 0xe0, 0x83];
const GET_TIMEOUT: [u8; 4] = [0x9f, 0x10, 0x25, 0xc6];
const KEEPALIVE: [u8; 4] = [0xf0, 0xb2, 0x1a, 0x41];
const GET_BENEFICIARY: [u8; 4] = [0xba, 0x20, 0xdd, 0xa4];
const CANCEL: [u8; 4] = [0xc4, 0xd2, 0x52, 0xf5];
const GET_CURRENT_REDEEMER: [u8; 4] = [0xde, 0x4b, 0xa2, 0xb3];
const SUBMIT_RETRYABLE: [u8; 4] = [0xc9, 0xf9, 0x5d, 0x32];

/// Default retryable lifetime: 7 days in seconds.
const RETRYABLE_LIFETIME_SECONDS: u64 = 7 * 24 * 60 * 60;
const RETRYABLE_REAP_PRICE: u64 = 58_000;

// Retryable ticket storage field offsets (within the ticket's sub-storage).
const NUM_TRIES_OFFSET: u64 = 0;
const FROM_OFFSET: u64 = 1;
const TO_OFFSET: u64 = 2;
const CALLVALUE_OFFSET: u64 = 3;
const BENEFICIARY_OFFSET: u64 = 4;
const TIMEOUT_OFFSET: u64 = 5;
const TIMEOUT_WINDOWS_LEFT_OFFSET: u64 = 6;

/// Timeout queue subspace key within the retryables storage.
const TIMEOUT_QUEUE_KEY: &[u8] = &[0];

/// SolError selectors (keccak256 of error signature, first 4 bytes).
fn no_ticket_with_id_selector() -> [u8; 4] {
    let hash = keccak256(b"NoTicketWithID()");
    [hash[0], hash[1], hash[2], hash[3]]
}

fn not_callable_selector() -> [u8; 4] {
    let hash = keccak256(b"NotCallable()");
    [hash[0], hash[1], hash[2], hash[3]]
}

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const SSTORE_ZERO_GAS: u64 = 5_000;
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

/// Backlog update cost: read + write. Write cost depends on whether
/// the new value is zero (StorageClearCost=5000) or non-zero (StorageWriteCost=20000).
/// This is computed dynamically in handle_redeem based on current backlog.
///
/// TicketCreated event topic0.
/// keccak256("TicketCreated(bytes32)")
pub fn ticket_created_topic() -> B256 {
    keccak256("TicketCreated(bytes32)")
}

/// RedeemScheduled event topic0.
/// keccak256("RedeemScheduled(bytes32,bytes32,uint64,uint64,address,uint256,uint256)")
pub fn redeem_scheduled_topic() -> B256 {
    keccak256("RedeemScheduled(bytes32,bytes32,uint64,uint64,address,uint256,uint256)")
}

/// LifetimeExtended event topic0. keccak256("LifetimeExtended(bytes32,uint256)")
pub fn lifetime_extended_topic() -> B256 {
    keccak256("LifetimeExtended(bytes32,uint256)")
}

/// Canceled event topic0. keccak256("Canceled(bytes32)")
pub fn canceled_topic() -> B256 {
    keccak256("Canceled(bytes32)")
}

pub fn create_arbretryabletx_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbretryabletx"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 {
        return crate::burn_all_revert(input.gas);
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];
    let gas_limit = input.gas;

    crate::init_precompile_gas(data.len());

    let result = match selector {
        GET_LIFETIME => {
            let lifetime = U256::from(RETRYABLE_LIFETIME_SECONDS);
            Ok(PrecompileOutput::new(
                (SLOAD_GAS + COPY_GAS).min(gas_limit),
                lifetime.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        GET_CURRENT_REDEEMER => {
            // Read the current redeemer from scratch storage slot.
            // The executor writes refund_to here before retry tx execution.
            let internals = input.internals_mut();
            internals
                .load_account(ARBOS_STATE_ADDRESS)
                .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;
            let redeemer = internals
                .sload(ARBOS_STATE_ADDRESS, current_redeemer_slot())
                .map_err(|_| PrecompileError::other("sload failed"))?
                .data;
            Ok(PrecompileOutput::new(
                (SLOAD_GAS + COPY_GAS).min(gas_limit),
                redeemer.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        SUBMIT_RETRYABLE => {
            return crate::sol_error_revert(not_callable_selector(), gas_limit);
        }
        GET_TIMEOUT => handle_get_timeout(&mut input),
        GET_BENEFICIARY => handle_get_beneficiary(&mut input),
        REDEEM => handle_redeem(&mut input),
        KEEPALIVE => handle_keepalive(&mut input),
        CANCEL => handle_cancel(&mut input),
        _ => return crate::burn_all_revert(gas_limit),
    };
    crate::gas_check(gas_limit, result)
}

// ── helpers ──────────────────────────────────────────────────────────

fn load_arbos(input: &mut PrecompileInput<'_>) -> Result<(), PrecompileError> {
    input
        .internals_mut()
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;
    Ok(())
}

fn sload_field(input: &mut PrecompileInput<'_>, slot: U256) -> Result<U256, PrecompileError> {
    let val = input
        .internals_mut()
        .sload(ARBOS_STATE_ADDRESS, slot)
        .map_err(|_| PrecompileError::other("sload failed"))?;
    crate::charge_precompile_gas(SLOAD_GAS);
    Ok(val.data)
}

fn sstore_field(
    input: &mut PrecompileInput<'_>,
    slot: U256,
    value: U256,
) -> Result<(), PrecompileError> {
    input
        .internals_mut()
        .sstore(ARBOS_STATE_ADDRESS, slot, value)
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    Ok(())
}

/// Derive the storage key for a specific retryable ticket.
fn ticket_storage_key(ticket_id: B256) -> B256 {
    let retryables_key = derive_subspace_key(ROOT_STORAGE_KEY, RETRYABLES_SUBSPACE);
    derive_subspace_key(retryables_key.as_slice(), ticket_id.as_slice())
}

/// Open a retryable ticket by verifying it exists (timeout > 0) and hasn't expired.
/// Returns the ticket's storage key.
fn open_retryable(
    input: &mut PrecompileInput<'_>,
    ticket_id: B256,
    current_timestamp: u64,
) -> Result<Option<B256>, PrecompileError> {
    let ticket_key = ticket_storage_key(ticket_id);
    let timeout_slot = map_slot(ticket_key.as_slice(), TIMEOUT_OFFSET);
    let timeout = sload_field(input, timeout_slot)?;
    let timeout_u64: u64 = timeout.try_into().unwrap_or(0);

    if timeout_u64 == 0 || timeout_u64 < current_timestamp {
        return Ok(None);
    }

    Ok(Some(ticket_key))
}

/// GetTimeout returns the effective timeout for a retryable ticket.
/// Effective timeout = stored_timeout + timeout_windows_left * RETRYABLE_LIFETIME.
fn handle_get_timeout(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }

    let gas_limit = input.gas;
    let ticket_id = B256::from_slice(&data[4..36]);
    let current_timestamp: u64 = input
        .internals()
        .block_timestamp()
        .try_into()
        .unwrap_or(u64::MAX);

    load_arbos(input)?;

    let ticket_key = ticket_storage_key(ticket_id);

    let timeout_slot = map_slot(ticket_key.as_slice(), TIMEOUT_OFFSET);
    let timeout = sload_field(input, timeout_slot)?;
    let timeout_u64: u64 = timeout.try_into().unwrap_or(0);

    // Match Nitro: OpenRetryable returns nil if timeout == 0 OR timeout < currentTime.
    if timeout_u64 == 0 || timeout_u64 < current_timestamp {
        return crate::sol_error_revert(no_ticket_with_id_selector(), gas_limit);
    }

    // Read timeout_windows_left for effective timeout calculation.
    let windows_slot = map_slot(ticket_key.as_slice(), TIMEOUT_WINDOWS_LEFT_OFFSET);
    let windows = sload_field(input, windows_slot)?;
    let windows_u64: u64 = windows.try_into().unwrap_or(0);

    let effective_timeout = timeout_u64 + windows_u64 * RETRYABLE_LIFETIME_SECONDS;

    // OAS(1) + OpenRetryable timeout(1) + CalculateTimeout timeout+windows(2) + argsCost(3) +
    // resultCost(3).
    Ok(PrecompileOutput::new(
        (4 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
        U256::from(effective_timeout)
            .to_be_bytes::<32>()
            .to_vec()
            .into(),
    ))
}

/// Derive the timeout queue storage key.
fn timeout_queue_key() -> B256 {
    let retryables_key = derive_subspace_key(ROOT_STORAGE_KEY, RETRYABLES_SUBSPACE);
    derive_subspace_key(retryables_key.as_slice(), TIMEOUT_QUEUE_KEY)
}

/// Queue Put: reads nextPutOffset (slot 0), writes the value at that offset, increments
/// nextPutOffset.
fn queue_put(input: &mut PrecompileInput<'_>, value: B256) -> Result<(), PrecompileError> {
    let queue_key = timeout_queue_key();

    // nextPutOffset is at offset 0 within the queue sub-storage.
    let put_offset_slot = map_slot(queue_key.as_slice(), 0);
    let put_offset = sload_field(input, put_offset_slot)?;
    let put_offset_u64: u64 = put_offset
        .try_into()
        .map_err(|_| PrecompileError::other("invalid queue put offset"))?;

    // Store the value at map_slot_b256(queue_key, value_as_key) using the offset as key.
    let item_slot = map_slot(queue_key.as_slice(), put_offset_u64);
    sstore_field(input, item_slot, U256::from_be_bytes(value.0))?;

    // Increment nextPutOffset.
    sstore_field(input, put_offset_slot, U256::from(put_offset_u64 + 1))?;

    Ok(())
}

/// GetBeneficiary returns the beneficiary address for a retryable ticket.
fn handle_get_beneficiary(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }

    let gas_limit = input.gas;
    let ticket_id = B256::from_slice(&data[4..36]);
    let current_timestamp: u64 = input
        .internals()
        .block_timestamp()
        .try_into()
        .unwrap_or(u64::MAX);

    load_arbos(input)?;

    let ticket_key = match open_retryable(input, ticket_id, current_timestamp)? {
        Some(k) => k,
        None => return crate::sol_error_revert(no_ticket_with_id_selector(), gas_limit),
    };

    let beneficiary_slot = map_slot(ticket_key.as_slice(), BENEFICIARY_OFFSET);
    let beneficiary = sload_field(input, beneficiary_slot)?;

    // OAS(1) + OpenRetryable timeout(1) + beneficiary(1) + argsCost(3) + resultCost(3).
    Ok(PrecompileOutput::new(
        (3 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
        beneficiary.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Redeem validates the retryable, increments numTries, donates remaining gas
/// to the retry tx, and emits a RedeemScheduled event. The block executor
/// discovers the event in the execution logs and schedules the retry tx.
fn handle_redeem(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }

    let gas_limit = input.gas;
    let ticket_id = B256::from_slice(&data[4..36]);
    let caller = input.caller;
    let current_timestamp: u64 = input
        .internals()
        .block_timestamp()
        .try_into()
        .unwrap_or(u64::MAX);

    // Guard: cannot redeem itself during its own retry execution.
    {
        let internals = input.internals_mut();
        internals
            .load_account(ARBOS_STATE_ADDRESS)
            .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;
        let current_retryable = internals
            .sload(ARBOS_STATE_ADDRESS, current_retryable_slot())
            .map_err(|_| PrecompileError::other("sload failed"))?
            .data;
        if !current_retryable.is_zero()
            && B256::from(current_retryable.to_be_bytes::<32>()) == ticket_id
        {
            return Err(PrecompileError::other("retryable cannot redeem itself"));
        }
    }

    // RetryableSizeBytes → OpenRetryable reads timeout (1 sload).
    let ticket_key_pre = ticket_storage_key(ticket_id);
    let timeout_slot = map_slot(ticket_key_pre.as_slice(), TIMEOUT_OFFSET);
    let timeout_val = sload_field(input, timeout_slot)?;
    let timeout_u64: u64 = timeout_val.try_into().unwrap_or(0);

    let (_calldata_words, write_bytes, calldata_raw_size) = if timeout_u64 == 0
        || timeout_u64 < current_timestamp
    {
        (0u64, 0u64, 0u64)
    } else {
        let calldata_sub = derive_subspace_key(ticket_key_pre.as_slice(), &[1]);
        let calldata_size_slot = map_slot(calldata_sub.as_slice(), 0);
        let calldata_size = sload_field(input, calldata_size_slot)?;
        let calldata_size_u64: u64 = calldata_size.try_into().unwrap_or(0);
        let cw = calldata_size_u64.div_ceil(32);
        let nbytes = 6 * 32 + 32 + 32 * cw;
        let wb = nbytes.div_ceil(32);
        (cw, wb, calldata_size_u64)
    };

    const PARAMS_SLOAD_GAS: u64 = 50;
    let retryable_size_gas = PARAMS_SLOAD_GAS.saturating_mul(write_bytes);
    crate::charge_precompile_gas(retryable_size_gas);

    // OpenRetryable reads timeout again (second sload, matching Nitro's flow).
    let timeout_val2 = sload_field(input, timeout_slot)?;
    let timeout_u64_2: u64 = timeout_val2.try_into().unwrap_or(0);
    if timeout_u64_2 == 0 || timeout_u64_2 < current_timestamp {
        return crate::sol_error_revert(no_ticket_with_id_selector(), gas_limit);
    }

    let num_tries_slot = map_slot(ticket_key_pre.as_slice(), NUM_TRIES_OFFSET);
    let num_tries = sload_field(input, num_tries_slot)?;
    crate::charge_precompile_gas(SSTORE_GAS);
    let nonce: u64 = num_tries.try_into().unwrap_or(0);
    let internals = input.internals_mut();
    internals
        .sstore(ARBOS_STATE_ADDRESS, num_tries_slot, U256::from(nonce + 1))
        .map_err(|_| PrecompileError::other("sstore failed"))?;

    // MakeTx reads: from + to + callvalue + GetBytes(size + floor(len/32) loop + trailing)
    let make_tx_reads = 5 + calldata_raw_size / 32;
    crate::charge_precompile_gas(make_tx_reads * SLOAD_GAS);

    // Compute deterministic retry tx hash: keccak256(ticket_id || nonce).
    let mut hash_input = [0u8; 64];
    hash_input[..32].copy_from_slice(ticket_id.as_slice());
    hash_input[32..].copy_from_slice(&U256::from(nonce).to_be_bytes::<32>());
    let retry_tx_hash = keccak256(hash_input);

    let gas_used_so_far = crate::get_precompile_gas();

    let backlog_reservation = SLOAD_GAS + SSTORE_GAS;
    let future_gas_costs = REDEEM_SCHEDULED_EVENT_COST + COPY_GAS + backlog_reservation;
    let gas_remaining = gas_limit.saturating_sub(gas_used_so_far);
    if gas_remaining < future_gas_costs + TX_GAS {
        return Err(PrecompileError::other(
            "not enough gas to run redeem attempt",
        ));
    }
    let gas_to_donate = gas_remaining - future_gas_costs;

    let actual_backlog_cost = {
        let current_backlog = crate::get_current_gas_backlog();
        let new_backlog = current_backlog.saturating_sub(gas_to_donate);
        let write_cost = if new_backlog == 0 {
            SSTORE_ZERO_GAS
        } else {
            SSTORE_GAS
        };
        SLOAD_GAS + write_cost
    };

    let max_refund = U256::MAX;
    let submission_fee_refund = U256::ZERO;

    // Emit RedeemScheduled event.
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

    let internals = input.internals_mut();
    internals.log(Log::new_unchecked(
        ARBRETRYABLETX_ADDRESS,
        vec![topic0, topic1, topic2, topic3],
        event_data.into(),
    ));

    // Total gas = pre-donate charges + event + donated gas + actual backlog cost + resultCost
    let total_gas = gas_used_so_far
        + REDEEM_SCHEDULED_EVENT_COST
        + gas_to_donate
        + actual_backlog_cost
        + COPY_GAS;

    Ok(PrecompileOutput::new(
        total_gas.min(gas_limit),
        retry_tx_hash.to_vec().into(),
    ))
}

/// Keepalive adds one lifetime period to the ticket's expiry.
///
/// Opens the retryable, verifies effective timeout isn't too far in the future,
/// adds a duplicate entry to the timeout queue, and increments timeout_windows_left.
fn handle_keepalive(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }

    let gas_limit = input.gas;
    let ticket_id = B256::from_slice(&data[4..36]);
    let current_timestamp: u64 = input
        .internals()
        .block_timestamp()
        .try_into()
        .unwrap_or(u64::MAX);

    load_arbos(input)?;

    let ticket_key = match open_retryable(input, ticket_id, current_timestamp)? {
        Some(k) => k,
        None => return crate::sol_error_revert(no_ticket_with_id_selector(), gas_limit),
    };

    // Read calldata size for updateCost computation (RetryableSizeBytes).
    let calldata_sub = derive_subspace_key(ticket_key.as_slice(), &[1]);
    let calldata_size_slot = map_slot(calldata_sub.as_slice(), 0);
    let calldata_size = sload_field(input, calldata_size_slot)?;
    let calldata_size_u64: u64 = calldata_size.try_into().unwrap_or(0);

    // Read timeout and timeout_windows_left to compute effective timeout.
    let timeout_slot = map_slot(ticket_key.as_slice(), TIMEOUT_OFFSET);
    let timeout = sload_field(input, timeout_slot)?;
    let timeout_u64: u64 = timeout
        .try_into()
        .map_err(|_| PrecompileError::other("invalid timeout"))?;

    let windows_slot = map_slot(ticket_key.as_slice(), TIMEOUT_WINDOWS_LEFT_OFFSET);
    let windows = sload_field(input, windows_slot)?;
    let windows_u64: u64 = windows
        .try_into()
        .map_err(|_| PrecompileError::other("invalid windows"))?;

    let effective_timeout = timeout_u64 + windows_u64 * RETRYABLE_LIFETIME_SECONDS;

    // The window limit is current_time + one lifetime.
    let window_limit = current_timestamp + RETRYABLE_LIFETIME_SECONDS;
    if effective_timeout > window_limit {
        return Err(PrecompileError::other("timeout too far into the future"));
    }

    // Put the ticket into the timeout queue (duplicate entry for the new window).
    queue_put(input, ticket_id)?;

    // Increment timeout_windows_left.
    let new_windows = windows_u64 + 1;
    sstore_field(input, windows_slot, U256::from(new_windows))?;

    let new_timeout = effective_timeout + RETRYABLE_LIFETIME_SECONDS;

    // Emit LifetimeExtended(bytes32 indexed ticketId, uint256 newTimeout).
    let topic0 = lifetime_extended_topic();
    let mut event_data = Vec::with_capacity(32);
    event_data.extend_from_slice(&U256::from(new_timeout).to_be_bytes::<32>());
    input.internals_mut().log(Log::new_unchecked(
        ARBRETRYABLETX_ADDRESS,
        vec![topic0, ticket_id],
        event_data.into(),
    ));

    // 8 SLOADs + 3 SSTOREs + argsCost(3) + updateCost + event(1381)
    // + RetryableReapPrice(58000) + resultCost(3).
    // updateCost = WordsForBytes(nbytes) * SstoreSetGas/100, where
    // nbytes = 6*32 + 32 + 32*WordsForBytes(calldataSize).
    let calldata_words = calldata_size_u64.div_ceil(32);
    let nbytes = 6 * 32 + 32 + 32 * calldata_words;
    let update_cost = nbytes.div_ceil(32) * (SSTORE_GAS / 100);
    let event_cost = LOG_GAS + 2 * LOG_TOPIC_GAS + LOG_DATA_GAS * 32;
    let gas_used = 8 * SLOAD_GAS
        + 3 * SSTORE_GAS
        + 2 * COPY_GAS
        + update_cost
        + event_cost
        + RETRYABLE_REAP_PRICE;

    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        U256::from(new_timeout).to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Cancel the ticket and refund its callvalue to its beneficiary.
///
/// Verifies the caller is the beneficiary, then clears all storage fields.
/// Balance transfer (escrow → beneficiary) is handled by the executor.
fn handle_cancel(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }

    let gas_limit = input.gas;
    let ticket_id = B256::from_slice(&data[4..36]);
    let caller = input.caller;
    let current_timestamp: u64 = input
        .internals()
        .block_timestamp()
        .try_into()
        .unwrap_or(u64::MAX);

    load_arbos(input)?;

    let ticket_key = match open_retryable(input, ticket_id, current_timestamp)? {
        Some(k) => k,
        None => return crate::sol_error_revert(no_ticket_with_id_selector(), gas_limit),
    };

    // Read beneficiary and verify caller is the beneficiary.
    let beneficiary_slot = map_slot(ticket_key.as_slice(), BENEFICIARY_OFFSET);
    let beneficiary = sload_field(input, beneficiary_slot)?;

    // The caller address is left-padded with zeros in 20 bytes.
    let caller_u256 = U256::from_be_slice(caller.as_slice());
    if caller_u256 != beneficiary {
        return Err(PrecompileError::other(
            "only the beneficiary may cancel a retryable",
        ));
    }

    // Clear all storage fields for this retryable ticket (DeleteRetryable).
    let offsets = [
        NUM_TRIES_OFFSET,
        FROM_OFFSET,
        TO_OFFSET,
        CALLVALUE_OFFSET,
        BENEFICIARY_OFFSET,
        TIMEOUT_OFFSET,
        TIMEOUT_WINDOWS_LEFT_OFFSET,
    ];
    for offset in offsets {
        let slot = map_slot(ticket_key.as_slice(), offset);
        sstore_field(input, slot, U256::ZERO)?;
    }

    // Clear calldata bytes (ClearBytes on calldata sub-storage).
    let calldata_sub = derive_subspace_key(ticket_key.as_slice(), &[1]);
    let calldata_size_slot = map_slot(calldata_sub.as_slice(), 0);
    let calldata_size = sload_field(input, calldata_size_slot)?;
    let calldata_size_u64: u64 = calldata_size.try_into().unwrap_or(0);
    let calldata_words = calldata_size_u64.div_ceil(32);
    if calldata_size_u64 > 0 {
        for i in 0..calldata_words {
            let word_slot = map_slot(calldata_sub.as_slice(), 1 + i);
            sstore_field(input, word_slot, U256::ZERO)?;
        }
        sstore_field(input, calldata_size_slot, U256::ZERO)?;
    }

    // Emit Canceled(bytes32 indexed ticketId).
    input.internals_mut().log(Log::new_unchecked(
        ARBRETRYABLETX_ADDRESS,
        vec![canceled_topic(), ticket_id],
        Default::default(),
    ));

    // 6 SLOADs + 7 × ClearByUint64(5000) + ClearBytes(variable)
    // + Canceled event (LOG2: 375+2*375=1125) + argsCost(3).
    let clear_bytes_cost = if calldata_size_u64 > 0 {
        (calldata_words + 1) * SSTORE_ZERO_GAS
    } else {
        0
    };
    let event_cost = LOG_GAS + 2 * LOG_TOPIC_GAS;
    let gas_used = 6 * SLOAD_GAS + 7 * SSTORE_ZERO_GAS + clear_bytes_cost + event_cost + COPY_GAS;

    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        Vec::new().into(),
    ))
}
