use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, B256, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{
    derive_subspace_key, map_slot, ARBOS_STATE_ADDRESS, RETRYABLES_SUBSPACE, ROOT_STORAGE_KEY,
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

// Retryable ticket storage field offsets (within the ticket's sub-storage).
const TIMEOUT_OFFSET: u64 = 5;
const TIMEOUT_WINDOWS_LEFT_OFFSET: u64 = 6;
const BENEFICIARY_OFFSET: u64 = 4;

const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;

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

    match selector {
        GET_LIFETIME => {
            let lifetime = U256::from(RETRYABLE_LIFETIME_SECONDS);
            Ok(PrecompileOutput::new(
                COPY_GAS.min(gas_limit),
                lifetime.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        GET_CURRENT_REDEEMER => {
            // Returns zero address when not in a retryable redeem context.
            Ok(PrecompileOutput::new(
                COPY_GAS.min(gas_limit),
                U256::ZERO.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        SUBMIT_RETRYABLE => {
            // Not callable — exists only for ABI/explorer purposes.
            Err(PrecompileError::other("not callable"))
        }
        GET_TIMEOUT => handle_get_timeout(&mut input),
        GET_BENEFICIARY => handle_get_beneficiary(&mut input),
        REDEEM => {
            // Redeem schedules a retry tx — handled by the block executor, not here.
            Err(PrecompileError::other(
                "redeem is handled by the block executor",
            ))
        }
        KEEPALIVE => {
            // Keepalive involves timeout queue writes — not yet implemented in precompile.
            Err(PrecompileError::other("keepalive not yet supported"))
        }
        CANCEL => {
            // Cancel involves balance transfers and storage clearing.
            Err(PrecompileError::other("cancel not yet supported"))
        }
        _ => Err(PrecompileError::other(
            "unknown ArbRetryableTx selector",
        )),
    }
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

    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        U256::from(effective_timeout).to_be_bytes::<32>().to_vec().into(),
    ))
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

    Ok(PrecompileOutput::new(
        (3 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        beneficiary.to_be_bytes::<32>().to_vec().into(),
    ))
}
