use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, Log, B256, U256};
use alloy_sol_types::{SolEvent, SolInterface};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::interfaces::IArbNativeTokenManager;
use crate::storage_slot::{
    derive_subspace_key, map_slot_b256, ARBOS_STATE_ADDRESS, NATIVE_TOKEN_SUBSPACE,
    ROOT_STORAGE_KEY,
};

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

pub fn create_arbnativetokenmanager_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbnativetokenmanager"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    if let Some(result) =
        crate::check_precompile_version(arb_chainspec::arbos_version::ARBOS_VERSION_41)
    {
        return result;
    }

    let gas_limit = input.gas;
    crate::init_precompile_gas(input.data.len());

    let call = match IArbNativeTokenManager::ArbNativeTokenManagerCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbNativeTokenManager::ArbNativeTokenManagerCalls;
    let result = match call {
        ArbNativeTokenManagerCalls::mintNativeToken(c) => handle_mint(&mut input, c.amount),
        ArbNativeTokenManagerCalls::burnNativeToken(c) => handle_burn(&mut input, c.amount),
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

fn sload_arbos(input: &mut PrecompileInput<'_>, slot: U256) -> Result<U256, PrecompileError> {
    let val = input
        .internals_mut()
        .sload(ARBOS_STATE_ADDRESS, slot)
        .map_err(|_| PrecompileError::other("sload failed"))?;
    Ok(val.data)
}

/// Check if caller is a native token owner via the NativeTokenOwners address set.
fn is_native_token_owner(
    input: &mut PrecompileInput<'_>,
    addr: Address,
) -> Result<bool, PrecompileError> {
    // NativeTokenOwners is at subspace [10] in ArbOS state.
    // byAddress sub-storage is at [0] within the address set.
    let owner_key = derive_subspace_key(ROOT_STORAGE_KEY, NATIVE_TOKEN_SUBSPACE);
    let by_address_key = derive_subspace_key(owner_key.as_slice(), &[0]);
    let addr_hash = B256::left_padding_from(addr.as_slice());
    let slot = map_slot_b256(by_address_key.as_slice(), &addr_hash);
    let val = sload_arbos(input, slot)?;
    Ok(val != U256::ZERO)
}

fn handle_mint(input: &mut PrecompileInput<'_>, amount: U256) -> PrecompileResult {
    let gas_limit = input.gas;
    let caller = input.caller;
    load_arbos(input)?;

    if !is_native_token_owner(input, caller)? {
        return Err(PrecompileError::other("caller is not a native token owner"));
    }

    input
        .internals_mut()
        .balance_incr(caller, amount)
        .map_err(|e| PrecompileError::other(format!("balance_incr: {e:?}")))?;

    let topic1 = B256::left_padding_from(caller.as_slice());
    let event_data = amount.to_be_bytes::<32>().to_vec();
    input.internals_mut().log(Log::new_unchecked(
        ARBNATIVETOKENMANAGER_ADDRESS,
        vec![IArbNativeTokenManager::NativeTokenMinted::SIGNATURE_HASH, topic1],
        event_data.into(),
    ));

    let gas_used = (SLOAD_GAS + SLOAD_GAS + MINT_BURN_GAS + EVENT_GAS + COPY_GAS).min(gas_limit);
    Ok(PrecompileOutput::new(gas_used, vec![].into()))
}

fn handle_burn(input: &mut PrecompileInput<'_>, amount: U256) -> PrecompileResult {
    let gas_limit = input.gas;
    let caller = input.caller;
    load_arbos(input)?;

    if !is_native_token_owner(input, caller)? {
        return Err(PrecompileError::other("caller is not a native token owner"));
    }

    let acct = input
        .internals_mut()
        .load_account(caller)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;
    let current_balance = acct.data.info.balance;

    if current_balance < amount {
        return Err(PrecompileError::other("burn amount exceeds balance"));
    }

    let new_balance = current_balance - amount;
    input
        .internals_mut()
        .set_balance(caller, new_balance)
        .map_err(|e| PrecompileError::other(format!("set_balance: {e:?}")))?;

    let topic1 = B256::left_padding_from(caller.as_slice());
    let event_data = amount.to_be_bytes::<32>().to_vec();
    input.internals_mut().log(Log::new_unchecked(
        ARBNATIVETOKENMANAGER_ADDRESS,
        vec![IArbNativeTokenManager::NativeTokenBurned::SIGNATURE_HASH, topic1],
        event_data.into(),
    ));

    let gas_used = (SLOAD_GAS + SLOAD_GAS + MINT_BURN_GAS + EVENT_GAS + COPY_GAS).min(gas_limit);
    Ok(PrecompileOutput::new(gas_used, vec![].into()))
}
