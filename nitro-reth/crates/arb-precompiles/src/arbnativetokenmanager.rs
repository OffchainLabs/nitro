use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, Log, B256, U256};
use alloy_sol_types::{SolEvent, SolInterface};
use arb_context::ArbPrecompileCtx;
use arb_storage::ARBOS_STATE_ADDRESS;

use revm::precompile::{PrecompileId, PrecompileOutput, PrecompileResult};
use std::sync::Arc;

use crate::{interfaces::IArbNativeTokenManager, ArbPrecompileError};

/// ArbNativeTokenManager precompile address (0x73).
pub const ARBNATIVETOKENMANAGER_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x73,
]);

const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;

/// Gas cost for mint/burn: WarmStorageReadCost + CallValueTransferGas.
const MINT_BURN_GAS: u64 = 100 + 9000;

/// LOG2 with one 32-byte data word: base + 2 topics + data.
const EVENT_GAS: u64 = 375 + 2 * 375 + 8 * 32;

pub fn create_arbnativetokenmanager_precompile(ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(
        PrecompileId::custom("arbnativetokenmanager"),
        move |input| handler(input, &ctx),
    )
}

fn handler(mut input: PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    if let Some(result) =
        crate::check_precompile_version(ctx, arb_chainspec::arbos_version::ARBOS_VERSION_41)
    {
        return result;
    }

    let mut gas_used = 0u64;
    let gas_limit = input.gas;
    crate::init_precompile_gas(&mut gas_used, input.data.len());

    let call = match IArbNativeTokenManager::ArbNativeTokenManagerCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbNativeTokenManager::ArbNativeTokenManagerCalls;
    let result = match call {
        ArbNativeTokenManagerCalls::mintNativeToken(c) => handle_mint(&mut input, c.amount, ctx),
        ArbNativeTokenManagerCalls::burnNativeToken(c) => {
            handle_burn(&mut input, gas_used, c.amount, ctx)
        }
    };
    crate::gas_check(ctx, gas_limit, gas_used, result)
}

// ── helpers ──────────────────────────────────────────────────────────

fn load_arbos(input: &mut PrecompileInput<'_>) -> Result<(), ArbPrecompileError> {
    input
        .internals_mut()
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;
    Ok(())
}

/// Check if caller is a native token owner via the NativeTokenOwners address set.
fn is_native_token_owner(
    input: &mut PrecompileInput<'_>,
    addr: Address,
    ctx: &ArbPrecompileCtx,
) -> Result<bool, ArbPrecompileError> {
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    arb_state
        .native_token_owners
        .is_member(internals, addr)
        .map_err(ArbPrecompileError::fatal)
}

fn handle_mint(
    input: &mut PrecompileInput<'_>,
    amount: U256,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let caller = input.caller;
    load_arbos(input)?;

    if !is_native_token_owner(input, caller, ctx)? {
        // Burn-out on unauthorized: consume all gas, not a soft revert.
        return crate::burn_all_revert(gas_limit);
    }

    input
        .internals_mut()
        .balance_incr(caller, amount)
        .map_err(ArbPrecompileError::fatal)?;

    let topic1 = B256::left_padding_from(caller.as_slice());
    let event_data = amount.to_be_bytes::<32>().to_vec();
    input.internals_mut().log(Log::new_unchecked(
        ARBNATIVETOKENMANAGER_ADDRESS,
        vec![
            IArbNativeTokenManager::NativeTokenMinted::SIGNATURE_HASH,
            topic1,
        ],
        event_data.into(),
    ));

    let gas_cost = (SLOAD_GAS + SLOAD_GAS + MINT_BURN_GAS + EVENT_GAS + COPY_GAS).min(gas_limit);
    Ok(PrecompileOutput::new(gas_cost, vec![].into()))
}

fn handle_burn(
    input: &mut PrecompileInput<'_>,
    gas_used: u64,
    amount: U256,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let caller = input.caller;
    load_arbos(input)?;

    if !is_native_token_owner(input, caller, ctx)? {
        // Burn-out on unauthorized: consume all gas.
        return crate::burn_all_revert(gas_limit);
    }

    let acct = input
        .internals_mut()
        .load_account(caller)
        .map_err(ArbPrecompileError::fatal)?;
    let current_balance = acct.data.info.balance;

    if current_balance < amount {
        return Err(ArbPrecompileError::empty_revert(gas_used).into());
    }

    let new_balance = current_balance - amount;
    input
        .internals_mut()
        .set_balance(caller, new_balance)
        .map_err(ArbPrecompileError::fatal)?;

    let topic1 = B256::left_padding_from(caller.as_slice());
    let event_data = amount.to_be_bytes::<32>().to_vec();
    input.internals_mut().log(Log::new_unchecked(
        ARBNATIVETOKENMANAGER_ADDRESS,
        vec![
            IArbNativeTokenManager::NativeTokenBurned::SIGNATURE_HASH,
            topic1,
        ],
        event_data.into(),
    ));

    let gas_cost = (SLOAD_GAS + SLOAD_GAS + MINT_BURN_GAS + EVENT_GAS + COPY_GAS).min(gas_limit);
    Ok(PrecompileOutput::new(gas_cost, vec![].into()))
}
