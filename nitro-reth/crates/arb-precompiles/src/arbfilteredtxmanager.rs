use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, Log, B256, U256};
use alloy_sol_types::{SolEvent, SolInterface};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::{
    interfaces::IArbFilteredTxManager,
    storage_slot::{
        derive_subspace_key, map_slot_b256, ARBOS_STATE_ADDRESS, FILTERED_TX_STATE_ADDRESS,
        ROOT_STORAGE_KEY, TRANSACTION_FILTERER_SUBSPACE,
    },
    ArbPrecompileError,
};

/// ArbFilteredTransactionsManager precompile address (0x74).
pub const ARBFILTEREDTXMANAGER_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x74,
]);

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const COPY_GAS: u64 = 3;

/// Sentinel value stored for filtered tx hashes.
const PRESENT_VALUE: U256 = U256::from_limbs([1, 0, 0, 0]);

pub fn create_arbfilteredtxmanager_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbfilteredtxmanager"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    if let Some(result) = crate::check_precompile_version(
        arb_chainspec::arbos_version::ARBOS_VERSION_TRANSACTION_FILTERING,
    ) {
        return result;
    }

    let gas_limit = input.gas;

    // Mimic the reference FreeAccessPrecompile wrapper: open ArbOS state and
    // check `filterers.IsMember(caller)` (2 SLOAD = 1600 gas total), without
    // charging argsCost. Then always run the inner method. The wrapper keeps
    // the inner's output and error, but overrides gas — 1600 for non-filterer
    // callers, 0 for filterers (free access).
    let mut wrapper_gas_used = 0u64;
    crate::charge_precompile_gas(&mut wrapper_gas_used, SLOAD_GAS);
    let caller = input.caller;
    load_accounts(&mut input)?;
    let is_filterer = is_transaction_filterer(&mut input, &mut wrapper_gas_used, caller)?;
    let wrapper_gas = wrapper_gas_used;

    let call =
        match IArbFilteredTxManager::ArbFilteredTransactionsManagerCalls::abi_decode(input.data) {
            Ok(c) => c,
            Err(_) => return crate::burn_all_revert(gas_limit),
        };

    let mut gas_used = 0u64;
    use IArbFilteredTxManager::ArbFilteredTransactionsManagerCalls as Calls;
    let inner_result = match call {
        Calls::addFilteredTransaction(c) => {
            handle_add_filtered_tx(&mut input, &mut gas_used, c.txHash)
        }
        Calls::deleteFilteredTransaction(c) => {
            handle_delete_filtered_tx(&mut input, &mut gas_used, c.txHash)
        }
        Calls::isTransactionFiltered(c) => {
            handle_is_tx_filtered(&mut input, &mut gas_used, c.txHash)
        }
    };

    // Wrapper overrides the inner's gas accounting: 0 for filterer, 1600 for
    // non-filterer. Inner's output and error are preserved.
    let final_gas = if is_filterer {
        0
    } else {
        wrapper_gas.min(gas_limit)
    };
    match inner_result {
        Ok(mut output) => {
            output.gas_used = final_gas;
            Ok(output)
        }
        Err(PrecompileError::Other(_)) => Ok(PrecompileOutput::new_reverted(
            final_gas,
            Default::default(),
        )),
        Err(e) => Err(e),
    }
}

// ── helpers ──────────────────────────────────────────────────────────

fn load_accounts(input: &mut PrecompileInput<'_>) -> Result<(), ArbPrecompileError> {
    input
        .internals_mut()
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;
    input
        .internals_mut()
        .load_account(FILTERED_TX_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;
    Ok(())
}

fn sload_arbos(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    slot: U256,
) -> Result<U256, ArbPrecompileError> {
    let val = input
        .internals_mut()
        .sload(ARBOS_STATE_ADDRESS, slot)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SLOAD_GAS);
    Ok(val.data)
}

fn sload_filtered(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    slot: U256,
) -> Result<U256, ArbPrecompileError> {
    let val = input
        .internals_mut()
        .sload(FILTERED_TX_STATE_ADDRESS, slot)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SLOAD_GAS);
    Ok(val.data)
}

fn sstore_filtered(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    slot: U256,
    value: U256,
) -> Result<(), ArbPrecompileError> {
    input
        .internals_mut()
        .sstore(FILTERED_TX_STATE_ADDRESS, slot, value)
        .map_err(ArbPrecompileError::fatal)?;
    let cost = if value.is_zero() { 5_000 } else { SSTORE_GAS };
    crate::charge_precompile_gas(gas_used, cost);
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
    gas_used: &mut u64,
    addr: Address,
) -> Result<bool, ArbPrecompileError> {
    // TransactionFilterers is at subspace [11] in ArbOS state.
    // byAddress sub-storage is at [0] within the address set.
    let filterer_key = derive_subspace_key(ROOT_STORAGE_KEY, TRANSACTION_FILTERER_SUBSPACE);
    let by_address_key = derive_subspace_key(filterer_key.as_slice(), &[0]);
    let addr_hash = B256::left_padding_from(addr.as_slice());
    let slot = map_slot_b256(by_address_key.as_slice(), &addr_hash);
    let val = sload_arbos(input, gas_used, slot)?;
    Ok(val != U256::ZERO)
}

fn handle_is_tx_filtered(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    tx_hash: B256,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_accounts(input)?;

    let slot = filtered_tx_slot(&tx_hash);
    let value = sload_filtered(input, gas_used, slot)?;
    let is_filtered = if value == PRESENT_VALUE {
        U256::from(1u64)
    } else {
        U256::ZERO
    };

    crate::charge_precompile_gas(gas_used, COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        is_filtered.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_add_filtered_tx(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    tx_hash: B256,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let caller = input.caller;
    load_accounts(input)?;

    if !is_transaction_filterer(input, gas_used, caller)? {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }

    let slot = filtered_tx_slot(&tx_hash);
    sstore_filtered(input, gas_used, slot, PRESENT_VALUE)?;

    input.internals_mut().log(Log::new_unchecked(
        ARBFILTEREDTXMANAGER_ADDRESS,
        vec![
            IArbFilteredTxManager::FilteredTransactionAdded::SIGNATURE_HASH,
            tx_hash,
        ],
        Default::default(),
    ));

    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        vec![].into(),
    ))
}

fn handle_delete_filtered_tx(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    tx_hash: B256,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let caller = input.caller;
    load_accounts(input)?;

    if !is_transaction_filterer(input, gas_used, caller)? {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }

    let slot = filtered_tx_slot(&tx_hash);
    sstore_filtered(input, gas_used, slot, U256::ZERO)?;

    input.internals_mut().log(Log::new_unchecked(
        ARBFILTEREDTXMANAGER_ADDRESS,
        vec![
            IArbFilteredTxManager::FilteredTransactionDeleted::SIGNATURE_HASH,
            tx_hash,
        ],
        Default::default(),
    ));

    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        vec![].into(),
    ))
}
