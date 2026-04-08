use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, B256, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{
    derive_subspace_key, map_slot_b256, ARBOS_STATE_ADDRESS, FILTERED_TX_STATE_ADDRESS,
    ROOT_STORAGE_KEY, TRANSACTION_FILTERER_SUBSPACE,
};

/// ArbFilteredTransactionsManager precompile address (0x74).
pub const ARBFILTEREDTXMANAGER_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x74,
]);

// Function selectors.
const ADD_FILTERED_TX: [u8; 4] = [0xbf, 0xc1, 0xd5, 0x0e];
const DELETE_FILTERED_TX: [u8; 4] = [0x0b, 0x23, 0x48, 0x5a];
const IS_TX_FILTERED: [u8; 4] = [0x37, 0x94, 0x6f, 0x6a];

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const COPY_GAS: u64 = 3;

/// Sentinel value stored for filtered tx hashes.
const PRESENT_VALUE: U256 = U256::from_limbs([1, 0, 0, 0]);

pub fn create_arbfilteredtxmanager_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbfilteredtxmanager"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    // ArbFilteredTransactionsManager requires ArbOS >= 60 (TransactionFiltering).
    if let Some(result) = crate::check_precompile_version(
        arb_chainspec::arbos_version::ARBOS_VERSION_TRANSACTION_FILTERING,
    ) {
        return result;
    }

    let data = input.data;
    if data.len() < 4 {
        return crate::burn_all_revert(input.gas);
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    crate::init_precompile_gas(data.len());

    let result = match selector {
        ADD_FILTERED_TX => handle_add_filtered_tx(&mut input),
        DELETE_FILTERED_TX => handle_delete_filtered_tx(&mut input),
        IS_TX_FILTERED => handle_is_tx_filtered(&mut input),
        _ => return crate::burn_all_revert(input.gas),
    };
    crate::gas_check(input.gas, result)
}

// ── helpers ──────────────────────────────────────────────────────────

fn load_accounts(input: &mut PrecompileInput<'_>) -> Result<(), PrecompileError> {
    input
        .internals_mut()
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;
    input
        .internals_mut()
        .load_account(FILTERED_TX_STATE_ADDRESS)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;
    Ok(())
}

fn sload_arbos(input: &mut PrecompileInput<'_>, slot: U256) -> Result<U256, PrecompileError> {
    let val = input
        .internals_mut()
        .sload(ARBOS_STATE_ADDRESS, slot)
        .map_err(|_| PrecompileError::other("sload failed"))?;
    Ok(val.data)
}

fn sload_filtered(input: &mut PrecompileInput<'_>, slot: U256) -> Result<U256, PrecompileError> {
    let val = input
        .internals_mut()
        .sload(FILTERED_TX_STATE_ADDRESS, slot)
        .map_err(|_| PrecompileError::other("sload failed"))?;
    Ok(val.data)
}

fn sstore_filtered(
    input: &mut PrecompileInput<'_>,
    slot: U256,
    value: U256,
) -> Result<(), PrecompileError> {
    input
        .internals_mut()
        .sstore(FILTERED_TX_STATE_ADDRESS, slot, value)
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    Ok(())
}

/// Compute the storage slot for a tx hash in the filtered transactions account.
/// The filtered tx storage uses an empty storageKey, so: map_slot_b256(&[], &tx_hash).
fn filtered_tx_slot(tx_hash: &B256) -> U256 {
    map_slot_b256(&[], tx_hash)
}

/// Check if caller is a transaction filterer via the TransactionFilterers address set.
fn is_transaction_filterer(
    input: &mut PrecompileInput<'_>,
    addr: Address,
) -> Result<bool, PrecompileError> {
    // TransactionFilterers is at subspace [11] in ArbOS state.
    // byAddress sub-storage is at [0] within the address set.
    let filterer_key = derive_subspace_key(ROOT_STORAGE_KEY, TRANSACTION_FILTERER_SUBSPACE);
    let by_address_key = derive_subspace_key(filterer_key.as_slice(), &[0]);
    let addr_hash = B256::left_padding_from(addr.as_slice());
    let slot = map_slot_b256(by_address_key.as_slice(), &addr_hash);
    let val = sload_arbos(input, slot)?;
    Ok(val != U256::ZERO)
}

/// Check if a transaction hash is in the filtered transactions list.
fn handle_is_tx_filtered(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }

    let gas_limit = input.gas;
    let tx_hash = B256::from_slice(&data[4..36]);
    load_accounts(input)?;

    let slot = filtered_tx_slot(&tx_hash);
    let value = sload_filtered(input, slot)?;
    let is_filtered = if value == PRESENT_VALUE {
        U256::from(1u64)
    } else {
        U256::ZERO
    };

    Ok(PrecompileOutput::new(
        (SLOAD_GAS + COPY_GAS).min(gas_limit),
        is_filtered.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Add a transaction hash to the filtered transactions list.
fn handle_add_filtered_tx(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }

    let gas_limit = input.gas;
    let tx_hash = B256::from_slice(&data[4..36]);
    let caller = input.caller;
    load_accounts(input)?;

    if !is_transaction_filterer(input, caller)? {
        return Err(PrecompileError::other(
            "caller is not a transaction filterer",
        ));
    }

    let slot = filtered_tx_slot(&tx_hash);
    sstore_filtered(input, slot, PRESENT_VALUE)?;

    let gas_used = 2 * SLOAD_GAS + SSTORE_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        vec![].into(),
    ))
}

/// Delete a transaction hash from the filtered transactions list.
fn handle_delete_filtered_tx(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return crate::burn_all_revert(input.gas);
    }

    let gas_limit = input.gas;
    let tx_hash = B256::from_slice(&data[4..36]);
    let caller = input.caller;
    load_accounts(input)?;

    if !is_transaction_filterer(input, caller)? {
        return Err(PrecompileError::other(
            "caller is not a transaction filterer",
        ));
    }

    let slot = filtered_tx_slot(&tx_hash);
    sstore_filtered(input, slot, U256::ZERO)?;

    let gas_used = 2 * SLOAD_GAS + SSTORE_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        vec![].into(),
    ))
}
