use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, B256, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{
    derive_subspace_key, map_slot_b256, ARBOS_STATE_ADDRESS, NATIVE_TOKEN_SUBSPACE,
    ROOT_STORAGE_KEY,
};

/// ArbNativeTokenManager precompile address (0x73).
pub const ARBNATIVETOKENMANAGER_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x73,
]);

// Function selectors.
const MINT_NATIVE_TOKEN: [u8; 4] = [0xf2, 0xe2, 0x34, 0x70];
const BURN_NATIVE_TOKEN: [u8; 4] = [0xa7, 0x54, 0x40, 0x2b];

const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;

/// Gas cost for mint/burn: WarmStorageReadCost + CallValueTransferGas.
const MINT_BURN_GAS: u64 = 100 + 9000;

pub fn create_arbnativetokenmanager_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbnativetokenmanager"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    match selector {
        MINT_NATIVE_TOKEN => handle_mint(&mut input),
        BURN_NATIVE_TOKEN => handle_burn(&mut input),
        _ => Err(PrecompileError::other("unknown selector")),
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

/// Mint native tokens to the caller's account.
fn handle_mint(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let amount = U256::from_be_slice(&data[4..36]);
    let caller = input.caller;
    load_arbos(input)?;

    if !is_native_token_owner(input, caller)? {
        return Err(PrecompileError::other(
            "caller is not a native token owner",
        ));
    }

    // Add balance to the caller.
    input
        .internals_mut()
        .balance_incr(caller, amount)
        .map_err(|e| PrecompileError::other(format!("balance_incr: {e:?}")))?;

    let gas_used = (SLOAD_GAS + MINT_BURN_GAS + COPY_GAS).min(gas_limit);
    Ok(PrecompileOutput::new(gas_used, vec![].into()))
}

/// Burn native tokens from the caller's account.
fn handle_burn(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let amount = U256::from_be_slice(&data[4..36]);
    let caller = input.caller;
    load_arbos(input)?;

    if !is_native_token_owner(input, caller)? {
        return Err(PrecompileError::other(
            "caller is not a native token owner",
        ));
    }

    // Check balance sufficiency.
    let acct = input
        .internals_mut()
        .load_account(caller)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;
    let current_balance = acct.data.info.balance;

    if current_balance < amount {
        return Err(PrecompileError::other("burn amount exceeds balance"));
    }

    // Set new balance.
    let new_balance = current_balance - amount;
    input
        .internals_mut()
        .set_balance(caller, new_balance)
        .map_err(|e| PrecompileError::other(format!("set_balance: {e:?}")))?;

    let gas_used = (SLOAD_GAS + MINT_BURN_GAS + COPY_GAS).min(gas_limit);
    Ok(PrecompileOutput::new(gas_used, vec![].into()))
}
