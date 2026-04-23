use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use alloy_sol_types::SolInterface;
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::interfaces::IArbOwnerPublic;
use crate::storage_slot::{
    derive_subspace_key, map_slot, map_slot_b256, root_slot, subspace_slot, ARBOS_STATE_ADDRESS,
    CHAIN_OWNER_SUBSPACE, FEATURES_SUBSPACE, FILTERED_FUNDS_RECIPIENT_OFFSET, L1_PRICING_SUBSPACE,
    NATIVE_TOKEN_ENABLED_FROM_TIME_OFFSET, NATIVE_TOKEN_SUBSPACE, PROGRAMS_SUBSPACE,
    ROOT_STORAGE_KEY, TRANSACTION_FILTERER_SUBSPACE, TX_FILTERING_ENABLED_FROM_TIME_OFFSET,
};

/// ArbOwnerPublic precompile address (0x6b).
pub const ARBOWNERPUBLIC_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x6b,
]);

const INITIAL_MAX_FRAGMENT_COUNT: u8 = 2;
// ArbOS version where MaxFragmentCount was introduced.
const ARBOS_VERSION_STYLUS_CONTRACT_LIMIT: u64 = 60;
// ArbOS version where collectTips storage flag was introduced.
const ARBOS_VERSION_COLLECT_TIPS: u64 = 60;
// collectTipsOffset in arbosState (root field offset 11).
const COLLECT_TIPS_OFFSET: u64 = 11;

// ArbOS state offsets (from arbosState).
const NETWORK_FEE_ACCOUNT_OFFSET: u64 = 3;
const INFRA_FEE_ACCOUNT_OFFSET: u64 = 6;
const BROTLI_COMPRESSION_LEVEL_OFFSET: u64 = 7;
const UPGRADE_VERSION_OFFSET: u64 = 1;
const UPGRADE_TIMESTAMP_OFFSET: u64 = 2;

// L1 pricing field for gas floor per token.
const L1_GAS_FLOOR_PER_TOKEN: u64 = 12;

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const COPY_GAS: u64 = 3;

pub fn create_arbownerpublic_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbownerpublic"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    crate::init_precompile_gas(input.data.len());

    let call = match IArbOwnerPublic::ArbOwnerPublicCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbOwnerPublic::ArbOwnerPublicCalls as Calls;
    let result = match call {
        Calls::getNetworkFeeAccount(_) => read_state_field(&mut input, NETWORK_FEE_ACCOUNT_OFFSET),
        Calls::getInfraFeeAccount(_) => {
            if crate::get_arbos_version() < 6 {
                read_state_field(&mut input, NETWORK_FEE_ACCOUNT_OFFSET)
            } else {
                read_state_field(&mut input, INFRA_FEE_ACCOUNT_OFFSET)
            }
        }
        Calls::getBrotliCompressionLevel(_) => {
            read_state_field(&mut input, BROTLI_COMPRESSION_LEVEL_OFFSET)
        }
        Calls::getScheduledUpgrade(_) => handle_scheduled_upgrade(&mut input),
        Calls::isChainOwner(c) => handle_is_chain_owner(&mut input, c.addr),
        Calls::getAllChainOwners(_) => handle_get_all_members(&mut input),
        Calls::rectifyChainOwner(c) => handle_rectify_chain_owner(&mut input, c.ownerToRectify),
        Calls::isNativeTokenOwner(c) => {
            handle_is_set_member(&mut input, NATIVE_TOKEN_SUBSPACE, c.addr)
        }
        Calls::isTransactionFilterer(c) => {
            handle_is_set_member(&mut input, TRANSACTION_FILTERER_SUBSPACE, c.filterer)
        }
        Calls::getAllNativeTokenOwners(_) => {
            handle_get_all_set_members(&mut input, NATIVE_TOKEN_SUBSPACE)
        }
        Calls::getAllTransactionFilterers(_) => {
            handle_get_all_set_members(&mut input, TRANSACTION_FILTERER_SUBSPACE)
        }
        Calls::getNativeTokenManagementFrom(_) => {
            read_state_field(&mut input, NATIVE_TOKEN_ENABLED_FROM_TIME_OFFSET)
        }
        Calls::getTransactionFilteringFrom(_) => {
            read_state_field(&mut input, TX_FILTERING_ENABLED_FROM_TIME_OFFSET)
        }
        Calls::getFilteredFundsRecipient(_) => {
            read_state_field(&mut input, FILTERED_FUNDS_RECIPIENT_OFFSET)
        }
        Calls::isCalldataPriceIncreaseEnabled(_) => {
            load_arbos(&mut input)?;
            let features_key = derive_subspace_key(ROOT_STORAGE_KEY, FEATURES_SUBSPACE);
            let features_slot = map_slot(features_key.as_slice(), 0);
            let features = sload_field(&mut input, features_slot)?;
            let enabled = features & U256::from(1);
            Ok(PrecompileOutput::new(
                (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
                enabled.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        Calls::getParentGasFloorPerToken(_) => {
            load_arbos(&mut input)?;
            let field_slot = subspace_slot(L1_PRICING_SUBSPACE, L1_GAS_FLOOR_PER_TOKEN);
            let value = sload_field(&mut input, field_slot)?;
            Ok(PrecompileOutput::new(
                (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
                value.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        Calls::getMaxStylusContractFragments(_) => handle_max_stylus_fragments(&mut input),
        Calls::getCollectTips(_) => handle_get_collect_tips(&mut input),
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
    crate::charge_precompile_gas(SSTORE_GAS);
    Ok(())
}

fn read_state_field(input: &mut PrecompileInput<'_>, offset: u64) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let value = sload_field(input, root_slot(offset))?;
    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_scheduled_upgrade(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let version = sload_field(input, root_slot(UPGRADE_VERSION_OFFSET))?;
    let timestamp = sload_field(input, root_slot(UPGRADE_TIMESTAMP_OFFSET))?;

    let mut out = Vec::with_capacity(64);
    out.extend_from_slice(&version.to_be_bytes::<32>());
    out.extend_from_slice(&timestamp.to_be_bytes::<32>());

    // OAS(1) + version(1) + timestamp(1) = 3 sloads + resultCost = 2 words × 3 = 6.
    Ok(PrecompileOutput::new(
        (3 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
        out.into(),
    ))
}

fn handle_rectify_chain_owner(
    input: &mut PrecompileInput<'_>,
    addr: Address,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let set_key = derive_subspace_key(ROOT_STORAGE_KEY, CHAIN_OWNER_SUBSPACE);
    let by_address_key = derive_subspace_key(set_key.as_slice(), &[0]);
    let addr_hash = alloy_primitives::B256::left_padding_from(addr.as_slice());

    // IsMember check
    let member_slot = map_slot_b256(by_address_key.as_slice(), &addr_hash);
    let slot_val = sload_field(input, member_slot)?;
    if slot_val == U256::ZERO {
        return Err(PrecompileError::other("not an owner"));
    }

    // Check if mapping is already correct
    let slot_idx: u64 = slot_val
        .try_into()
        .map_err(|_| PrecompileError::other("invalid slot"))?;
    let at_slot_key = map_slot(set_key.as_slice(), slot_idx);
    let at_slot_val = sload_field(input, at_slot_key)?;
    let size_slot = map_slot(set_key.as_slice(), 0);
    let size: u64 = sload_field(input, size_slot)?
        .try_into()
        .map_err(|_| PrecompileError::other("invalid size"))?;

    // Compare: backingStorage[slot] should store the address as U256
    let addr_as_u256 = U256::from_be_slice(addr.as_slice());
    if at_slot_val == addr_as_u256 && slot_idx <= size {
        return Err(PrecompileError::other("already correctly mapped"));
    }

    // Clear byAddress mapping, then re-add
    sstore_field(input, member_slot, U256::ZERO)?;

    // Re-add using same logic as address_set_add in arbowner.rs
    let new_size = size + 1;
    let new_pos_slot = map_slot(set_key.as_slice(), new_size);
    sstore_field(input, new_pos_slot, addr_as_u256)?;
    sstore_field(input, member_slot, U256::from(new_size))?;
    sstore_field(input, size_slot, U256::from(new_size))?;

    // Emit ChainOwnerRectified(address) event
    let topic0 = alloy_primitives::keccak256("ChainOwnerRectified(address)");
    input
        .internals_mut()
        .log(alloy_primitives::Log::new_unchecked(
            ARBOWNERPUBLIC_ADDRESS,
            vec![topic0],
            addr_hash.0.to_vec().into(),
        ));

    const SSTORE_ZERO_GAS: u64 = 5_000;
    const RECTIFY_EVENT_GAS: u64 = 1_006; // LOG1 + 32 bytes data
    let gas_used =
        SLOAD_GAS + 7 * SLOAD_GAS + SSTORE_ZERO_GAS + 3 * SSTORE_GAS + RECTIFY_EVENT_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        Vec::new().into(),
    ))
}

fn handle_is_chain_owner(input: &mut PrecompileInput<'_>, addr: Address) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    // Chain owners AddressSet: byAddress sub-storage at key [0].
    let set_key = derive_subspace_key(ROOT_STORAGE_KEY, CHAIN_OWNER_SUBSPACE);
    let by_address_key = derive_subspace_key(set_key.as_slice(), &[0]);

    let addr_as_b256 = alloy_primitives::B256::left_padding_from(addr.as_slice());
    let member_slot = map_slot_b256(by_address_key.as_slice(), &addr_as_b256);

    let value = sload_field(input, member_slot)?;
    let is_owner = value != U256::ZERO;

    let result = if is_owner {
        U256::from(1u64)
    } else {
        U256::ZERO
    };

    // OAS(1) + IsMember(1) = 2 sloads + argsCost(3) + resultCost(3).
    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
        result.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_get_all_members(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    // AddressSet: size at offset 0, members at offsets 1..=size in backing storage.
    let set_key = derive_subspace_key(ROOT_STORAGE_KEY, CHAIN_OWNER_SUBSPACE);
    let size_slot = map_slot(set_key.as_slice(), 0);
    let size = sload_field(input, size_slot)?;
    let count: u64 = size.try_into().unwrap_or(0);

    // ABI: offset to dynamic array, array length, then elements.
    let max_members = count.min(256); // Safety cap
    let mut out = Vec::with_capacity(64 + max_members as usize * 32);
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(count).to_be_bytes::<32>());

    for i in 0..max_members {
        let member_slot = map_slot(set_key.as_slice(), i + 1);
        let addr_val = sload_field(input, member_slot)?;
        out.extend_from_slice(&addr_val.to_be_bytes::<32>());
    }

    // resultCost = (2 + N) words for dynamic array encoding.
    Ok(PrecompileOutput::new(
        ((2 + max_members) * SLOAD_GAS + (2 + max_members) * COPY_GAS).min(gas_limit),
        out.into(),
    ))
}

fn handle_is_set_member(
    input: &mut PrecompileInput<'_>,
    subspace: &[u8],
    addr: Address,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let set_key = derive_subspace_key(ROOT_STORAGE_KEY, subspace);
    let by_address_key = derive_subspace_key(set_key.as_slice(), &[0]);
    let addr_hash = alloy_primitives::B256::left_padding_from(addr.as_slice());
    let member_slot = map_slot_b256(by_address_key.as_slice(), &addr_hash);
    let value = sload_field(input, member_slot)?;
    let is_member = if value != U256::ZERO {
        U256::from(1u64)
    } else {
        U256::ZERO
    };

    // OAS(1) + IsMember(1) = 2 sloads + argsCost(3) + resultCost(3).
    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
        is_member.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_get_all_set_members(
    input: &mut PrecompileInput<'_>,
    subspace: &[u8],
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let set_key = derive_subspace_key(ROOT_STORAGE_KEY, subspace);
    let size_slot = map_slot(set_key.as_slice(), 0);
    let size = sload_field(input, size_slot)?;
    let count: u64 = size.try_into().unwrap_or(0);
    let max_members = count.min(65536);

    let mut out = Vec::with_capacity(64 + max_members as usize * 32);
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(count).to_be_bytes::<32>());

    for i in 0..max_members {
        let member_slot = map_slot(set_key.as_slice(), i + 1);
        let addr_val = sload_field(input, member_slot)?;
        out.extend_from_slice(&addr_val.to_be_bytes::<32>());
    }

    // resultCost = (2 + N) words for dynamic array encoding.
    Ok(PrecompileOutput::new(
        ((2 + max_members) * SLOAD_GAS + (2 + max_members) * COPY_GAS).min(gas_limit),
        out.into(),
    ))
}

fn handle_max_stylus_fragments(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    if crate::get_arbos_version() < ARBOS_VERSION_STYLUS_CONTRACT_LIMIT {
        return Ok(PrecompileOutput::new(
            (SLOAD_GAS + COPY_GAS).min(gas_limit),
            vec![0u8; 32].into(),
        ));
    }
    load_arbos(input)?;
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let params_key = derive_subspace_key(programs_key.as_slice(), &[0]);
    let params_slot = map_slot(params_key.as_slice(), 0);
    let val = sload_field(input, params_slot)?;
    let bytes = val.to_be_bytes::<32>();
    let mut count = bytes[29];
    if count == 0 {
        count = INITIAL_MAX_FRAGMENT_COUNT;
    }
    let mut out = [0u8; 32];
    out[31] = count;
    Ok(PrecompileOutput::new(
        (SLOAD_GAS + COPY_GAS).min(gas_limit),
        out.to_vec().into(),
    ))
}

fn handle_get_collect_tips(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    if crate::get_arbos_version() < ARBOS_VERSION_COLLECT_TIPS {
        return Ok(PrecompileOutput::new(
            COPY_GAS.min(gas_limit),
            vec![0u8; 32].into(),
        ));
    }
    load_arbos(input)?;
    let value = sload_field(input, root_slot(COLLECT_TIPS_OFFSET))?;
    let mut out = [0u8; 32];
    if !value.is_zero() {
        out[31] = 1;
    }
    Ok(PrecompileOutput::new(
        (SLOAD_GAS + COPY_GAS).min(gas_limit),
        out.to_vec().into(),
    ))
}
