use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, Log, B256, U256};
use alloy_sol_types::{SolEvent, SolInterface};
use arb_context::ArbPrecompileCtx;
use arb_storage::{ARBOS_STATE_ADDRESS, FILTERED_TX_STATE_ADDRESS};

use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};
use std::sync::Arc;

use crate::{interfaces::IArbFilteredTxManager, ArbPrecompileError};

/// ArbFilteredTransactionsManager precompile address (0x74).
pub const ARBFILTEREDTXMANAGER_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x74,
]);

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const COPY_GAS: u64 = 3;

pub fn create_arbfilteredtxmanager_precompile(ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbfilteredtxmanager"), move |input| {
        handler(input, &ctx)
    })
}

fn handler(mut input: PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    if let Some(result) = crate::check_precompile_version(
        ctx,
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
    let is_filterer = {
        let internals = input.internals_mut();
        let arb_state = ctx
            .block
            .arbos_state(internals)
            .map_err(ArbPrecompileError::fatal)?;
        let res = arb_state
            .transaction_filterers
            .is_member(internals, caller)
            .map_err(ArbPrecompileError::fatal)?;
        crate::charge_precompile_gas(&mut wrapper_gas_used, SLOAD_GAS);
        res
    };
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
            handle_add_filtered_tx(&mut input, &mut gas_used, c.txHash, ctx)
        }
        Calls::deleteFilteredTransaction(c) => {
            handle_delete_filtered_tx(&mut input, &mut gas_used, c.txHash, ctx)
        }
        Calls::isTransactionFiltered(c) => {
            handle_is_tx_filtered(&mut input, &mut gas_used, c.txHash, ctx)
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

/// Check if caller is a transaction filterer via the TransactionFilterers address set.
fn is_transaction_filterer(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    addr: Address,
    ctx: &ArbPrecompileCtx,
) -> Result<bool, ArbPrecompileError> {
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let is_member = arb_state
        .transaction_filterers
        .is_member(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SLOAD_GAS);
    Ok(is_member)
}

fn handle_is_tx_filtered(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    tx_hash: B256,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_accounts(input)?;

    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let is_filtered_bool = arb_state
        .filtered_transactions
        .is_filtered(internals, tx_hash)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SLOAD_GAS);

    let is_filtered = if is_filtered_bool {
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
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let caller = input.caller;
    load_accounts(input)?;

    if !is_transaction_filterer(input, gas_used, caller, ctx)? {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }

    {
        let internals = input.internals_mut();
        let arb_state = ctx
            .block
            .arbos_state(internals)
            .map_err(ArbPrecompileError::fatal)?;
        arb_state
            .filtered_transactions
            .set(internals, tx_hash, true)
            .map_err(ArbPrecompileError::fatal)?;
        crate::charge_precompile_gas(gas_used, SSTORE_GAS);
    }

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
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let caller = input.caller;
    load_accounts(input)?;

    if !is_transaction_filterer(input, gas_used, caller, ctx)? {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }

    {
        let internals = input.internals_mut();
        let arb_state = ctx
            .block
            .arbos_state(internals)
            .map_err(ArbPrecompileError::fatal)?;
        arb_state
            .filtered_transactions
            .set(internals, tx_hash, false)
            .map_err(ArbPrecompileError::fatal)?;
        crate::charge_precompile_gas(gas_used, 5_000);
    }

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
