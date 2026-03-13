use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{keccak256, Address, B256, Log, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{
    current_redeemer_slot, current_retryable_slot, derive_subspace_key, map_slot,
    ARBOS_STATE_ADDRESS, RETRYABLES_SUBSPACE, ROOT_STORAGE_KEY,
};

/// ArbRetryableTx precompile address (0x6e).
pub const ARBRETRYABLETX_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x6e,
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

pub fn create_arbretryabletx_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbretryabletx"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];
    let gas_limit = input.gas;

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
            // Not callable — exists only for ABI/explorer purposes.
            Err(PrecompileError::other("not callable"))
        }
        GET_TIMEOUT => handle_get_timeout(&mut input),
        GET_BENEFICIARY => handle_get_beneficiary(&mut input),
        REDEEM => handle_redeem(&mut input),
        KEEPALIVE => handle_keepalive(&mut input),
        CANCEL => handle_cancel(&mut input),
        _ => Err(PrecompileError::other(
            "unknown ArbRetryableTx selector",
        )),
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
) -> Result<B256, PrecompileError> {
    let ticket_key = ticket_storage_key(ticket_id);
    let timeout_slot = map_slot(ticket_key.as_slice(), TIMEOUT_OFFSET);
    let timeout = sload_field(input, timeout_slot)?;
    let timeout_u64: u64 = timeout
        .try_into()
        .map_err(|_| PrecompileError::other("invalid timeout value"))?;

    if timeout_u64 == 0 {
        return Err(PrecompileError::other("retryable ticket not found"));
    }
    if timeout_u64 < current_timestamp {
        return Err(PrecompileError::other("retryable ticket expired"));
    }

    Ok(ticket_key)
}

/// GetTimeout returns the effective timeout for a retryable ticket.
/// Effective timeout = stored_timeout + timeout_windows_left * RETRYABLE_LIFETIME.
fn handle_get_timeout(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let ticket_id = B256::from_slice(&data[4..36]);

    load_arbos(input)?;

    let ticket_key = ticket_storage_key(ticket_id);

    // Read raw timeout.
    let timeout_slot = map_slot(ticket_key.as_slice(), TIMEOUT_OFFSET);
    let timeout = sload_field(input, timeout_slot)?;
    let timeout_u64: u64 = timeout.try_into().unwrap_or(0);

    if timeout_u64 == 0 {
        return Err(PrecompileError::other("retryable ticket not found"));
    }

    // Read timeout_windows_left for effective timeout calculation.
    let windows_slot = map_slot(ticket_key.as_slice(), TIMEOUT_WINDOWS_LEFT_OFFSET);
    let windows = sload_field(input, windows_slot)?;
    let windows_u64: u64 = windows.try_into().unwrap_or(0);

    let effective_timeout = timeout_u64 + windows_u64 * RETRYABLE_LIFETIME_SECONDS;

    // OAS(1) + OpenRetryable timeout(1) + CalculateTimeout timeout+windows(2) + argsCost(3) + resultCost(3).
    Ok(PrecompileOutput::new(
        (4 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
        U256::from(effective_timeout).to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Derive the timeout queue storage key.
fn timeout_queue_key() -> B256 {
    let retryables_key = derive_subspace_key(ROOT_STORAGE_KEY, RETRYABLES_SUBSPACE);
    derive_subspace_key(retryables_key.as_slice(), TIMEOUT_QUEUE_KEY)
}

/// Queue Put: reads nextPutOffset (slot 0), writes the value at that offset, increments nextPutOffset.
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
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let ticket_id = B256::from_slice(&data[4..36]);
    let current_timestamp: u64 = input
        .internals()
        .block_timestamp()
        .try_into()
        .unwrap_or(u64::MAX);

    load_arbos(input)?;

    let ticket_key = open_retryable(input, ticket_id, current_timestamp)?;

    // Read beneficiary (stored as address in 32 bytes, right-aligned).
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
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let ticket_id = B256::from_slice(&data[4..36]);
    let caller = input.caller;
    let current_timestamp: u64 = input
        .internals()
        .block_timestamp()
        .try_into()
        .unwrap_or(u64::MAX);

    let internals = input.internals_mut();

    // Load the ArbOS state account.
    internals
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;

    // Guard: a retryable cannot redeem itself during its own retry execution.
    let current_retryable = internals
        .sload(ARBOS_STATE_ADDRESS, current_retryable_slot())
        .map_err(|_| PrecompileError::other("sload failed"))?
        .data;
    if !current_retryable.is_zero()
        && B256::from(current_retryable.to_be_bytes::<32>()) == ticket_id
    {
        return Err(PrecompileError::other(
            "retryable cannot redeem itself",
        ));
    }

    // Compute RetryableSizeBytes and charge gas for reading retryable data.
    // Nitro charges SloadGas (800) per 32-byte word of the retryable's total
    // stored size: 6 fixed fields + calldata length word + calldata content.
    let ticket_key_pre = ticket_storage_key(ticket_id);
    let retryable_size_gas = {
        // Read timeout to verify retryable exists (for RetryableSizeBytes).
        let timeout_slot = map_slot(ticket_key_pre.as_slice(), TIMEOUT_OFFSET);
        let timeout_check = internals
            .sload(ARBOS_STATE_ADDRESS, timeout_slot)
            .map_err(|_| PrecompileError::other("sload failed"))?
            .data;
        let timeout_check_u64: u64 = timeout_check
            .try_into()
            .map_err(|_| PrecompileError::other("invalid timeout value"))?;
        if timeout_check_u64 == 0 || timeout_check_u64 < current_timestamp {
            return Err(PrecompileError::other("retryable ticket not found or expired"));
        }

        // Read calldata size.
        let calldata_sub = derive_subspace_key(ticket_key_pre.as_slice(), &[1]);
        let calldata_size_slot = map_slot(calldata_sub.as_slice(), 0);
        let calldata_size = internals
            .sload(ARBOS_STATE_ADDRESS, calldata_size_slot)
            .map_err(|_| PrecompileError::other("sload failed"))?
            .data;
        let calldata_size_u64: u64 = calldata_size.try_into().unwrap_or(0);
        let calldata_words = (calldata_size_u64 + 31) / 32;
        // nbytes = 6*32 (fixed fields) + 32 (length word) + 32*calldata_words
        let nbytes = 6 * 32 + 32 + 32 * calldata_words;
        let write_bytes = (nbytes + 31) / 32;
        // SloadGas (800) per word, matching Nitro's c.Burn(StorageAccess, SloadGas*writeBytes)
        800u64.saturating_mul(write_bytes)
    };

    // Open the retryable (re-read timeout after size computation).
    let ticket_key = {
        let tk = ticket_key_pre;
        let timeout_slot = map_slot(tk.as_slice(), TIMEOUT_OFFSET);
        let timeout = internals
            .sload(ARBOS_STATE_ADDRESS, timeout_slot)
            .map_err(|_| PrecompileError::other("sload failed"))?
            .data;
        let timeout_u64: u64 = timeout
            .try_into()
            .map_err(|_| PrecompileError::other("invalid timeout value"))?;
        if timeout_u64 == 0 {
            return Err(PrecompileError::other("retryable ticket not found"));
        }
        if timeout_u64 < current_timestamp {
            return Err(PrecompileError::other("retryable ticket expired"));
        }
        tk
    };

    // Read and increment numTries.
    let num_tries_slot = map_slot(ticket_key.as_slice(), NUM_TRIES_OFFSET);
    let num_tries = internals
        .sload(ARBOS_STATE_ADDRESS, num_tries_slot)
        .map_err(|_| PrecompileError::other("sload failed"))?
        .data;
    let nonce: u64 = num_tries
        .try_into()
        .map_err(|_| PrecompileError::other("invalid numTries"))?;
    internals
        .sstore(ARBOS_STATE_ADDRESS, num_tries_slot, U256::from(nonce + 1))
        .map_err(|_| PrecompileError::other("sstore failed"))?;

    // Compute deterministic retry tx hash: keccak256(ticket_id || nonce).
    let mut hash_input = [0u8; 64];
    hash_input[..32].copy_from_slice(ticket_id.as_slice());
    hash_input[32..].copy_from_slice(&U256::from(nonce).to_be_bytes::<32>());
    let retry_tx_hash = keccak256(&hash_input);

    // Gas consumed so far, matching Nitro's gas accounting:
    // - retryable_size_gas: explicit c.Burn(SloadGas * writeBytes) = 800 * writeBytes
    // - currentRetryable: 1 sload (ArbOS scratch slot check)
    // - timeout: 1 sload (OpenRetryable)
    // - numTries: 1 sload + 1 sstore (read + increment)
    // Note: RetryableSizeBytes reads (timeout + calldata_size) go through
    // SystemBurner in Nitro = no gas cost. OpenRetryable re-read also free.
    let gas_used_so_far = 3 * SLOAD_GAS + SSTORE_GAS + retryable_size_gas;

    // Compute backlog update cost dynamically. In Nitro, ShrinkBacklog
    // runs inside the precompile with the burner charging StorageReadCost
    // (800) for the read + writeCost for the write. writeCost depends on
    // the new value: StorageClearCost (5000) if zero, StorageWriteCost
    // (20000) if non-zero. The donated gas shrinks the backlog, so if
    // the backlog will reach zero, the write is cheaper.
    let backlog_update_cost = {
        let backlog_slot = alloy_primitives::uint!(0xe54de2a4cdacc0a0059d2b6e16348103df8c4aff409c31e40ec73d11926c8204_U256);
        let current_backlog = internals
            .sload(ARBOS_STATE_ADDRESS, backlog_slot)
            .map(|r| r.data.to::<u64>())
            .unwrap_or(0);
        // gas_to_donate is approximately gas_remaining - future_costs.
        // If backlog <= approximate_donated: new backlog = 0, write = 5000
        // Otherwise: write = 20000. Use conservative estimate.
        let write_cost = if current_backlog == 0 {
            SSTORE_ZERO_GAS // 5000
        } else {
            SSTORE_GAS // 20000
        };
        SLOAD_GAS + write_cost // read + write
    };

    // Calculate gas to donate to the retry tx.
    // Reserve gas for: event emission + copy (return result) + backlog update.
    let future_gas_costs = REDEEM_SCHEDULED_EVENT_COST + COPY_GAS + backlog_update_cost;
    let gas_remaining = gas_limit.saturating_sub(gas_used_so_far);
    if gas_remaining < future_gas_costs + TX_GAS {
        return Err(PrecompileError::other(
            "not enough gas to run redeem attempt",
        ));
    }
    let gas_to_donate = gas_remaining - future_gas_costs;

    // Manual redeem: maxRefund = 2^256 - 1, submissionFeeRefund = 0.
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

    internals.log(Log::new_unchecked(
        ARBRETRYABLETX_ADDRESS,
        vec![topic0, topic1, topic2, topic3],
        event_data.into(),
    ));

    // Return only the precompile's own gas (NOT gas_to_donate).
    // In Nitro, gas_to_donate stays as gasLeft in the precompile return.
    // The EVM sees: gas_consumed = gasSupplied - gasLeft = own_gas.
    // The donated gas is then consumed by the block executor when it
    // processes the RedeemScheduled event and schedules the retry tx.
    let own_gas = gas_used_so_far + REDEEM_SCHEDULED_EVENT_COST + COPY_GAS;

    Ok(PrecompileOutput::new(
        own_gas.min(gas_limit),
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
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let ticket_id = B256::from_slice(&data[4..36]);
    let current_timestamp: u64 = input
        .internals()
        .block_timestamp()
        .try_into()
        .unwrap_or(u64::MAX);

    load_arbos(input)?;

    // Open the retryable (verifies exists and not expired).
    let ticket_key = open_retryable(input, ticket_id, current_timestamp)?;

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

    // 8 SLOADs + 3 SSTOREs + argsCost(3) + updateCost + event(1381)
    // + RetryableReapPrice(58000) + resultCost(3).
    // updateCost = WordsForBytes(nbytes) * SstoreSetGas/100, where
    // nbytes = 6*32 + 32 + 32*WordsForBytes(calldataSize).
    let calldata_words = (calldata_size_u64 + 31) / 32;
    let nbytes = 6 * 32 + 32 + 32 * calldata_words;
    let update_cost = ((nbytes + 31) / 32) * (SSTORE_GAS / 100);
    let event_cost = LOG_GAS + 2 * LOG_TOPIC_GAS + LOG_DATA_GAS * 32;
    let gas_used = 8 * SLOAD_GAS + 3 * SSTORE_GAS + 2 * COPY_GAS
        + update_cost + event_cost + RETRYABLE_REAP_PRICE;

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
        return Err(PrecompileError::other("input too short"));
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

    // Open the retryable (verifies exists and not expired).
    let ticket_key = open_retryable(input, ticket_id, current_timestamp)?;

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
    let calldata_words = (calldata_size_u64 + 31) / 32;
    if calldata_size_u64 > 0 {
        for i in 0..calldata_words {
            let word_slot = map_slot(calldata_sub.as_slice(), 1 + i);
            sstore_field(input, word_slot, U256::ZERO)?;
        }
        sstore_field(input, calldata_size_slot, U256::ZERO)?;
    }

    // 6 SLOADs + 7 × ClearByUint64(5000) + ClearBytes(variable)
    // + Canceled event (LOG2: 375+2*375=1125) + argsCost(3).
    // DeleteRetryable SLOADs: timeout(1) + beneficiary(1) + ClearBytes size(1) = 3
    // Total SLOADs: OAS(1) + OpenRetryable(1) + beneficiary(1) + DeleteRetryable(3) = 6
    let clear_bytes_cost = if calldata_size_u64 > 0 {
        (calldata_words + 1) * SSTORE_ZERO_GAS
    } else {
        0
    };
    let event_cost = LOG_GAS + 2 * LOG_TOPIC_GAS;
    let gas_used = 6 * SLOAD_GAS + 7 * SSTORE_ZERO_GAS + clear_bytes_cost
        + event_cost + COPY_GAS;

    Ok(PrecompileOutput::new(gas_used.min(gas_limit), Vec::new().into()))
}
