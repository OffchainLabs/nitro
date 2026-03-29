use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{
    derive_subspace_key, map_slot, map_slot_b256, root_slot, subspace_slot, ARBOS_STATE_ADDRESS,
    CHAIN_OWNER_SUBSPACE, FEATURES_SUBSPACE, FILTERED_FUNDS_RECIPIENT_OFFSET, L1_PRICING_SUBSPACE,
    NATIVE_TOKEN_ENABLED_FROM_TIME_OFFSET, NATIVE_TOKEN_SUBSPACE, ROOT_STORAGE_KEY,
    TRANSACTION_FILTERER_SUBSPACE, TX_FILTERING_ENABLED_FROM_TIME_OFFSET,
};

/// ArbOwnerPublic precompile address (0x6b).
pub const ARBOWNERPUBLIC_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x6b,
]);

// Function selectors.
const GET_NETWORK_FEE_ACCOUNT: [u8; 4] = [0x2d, 0x91, 0x25, 0xe9];
const GET_INFRA_FEE_ACCOUNT: [u8; 4] = [0xee, 0x95, 0xa8, 0x24];
const GET_BROTLI_COMPRESSION_LEVEL: [u8; 4] = [0x22, 0xd4, 0x99, 0xc7];
const GET_SCHEDULED_UPGRADE: [u8; 4] = [0x81, 0xef, 0x94, 0x4c];
const IS_CHAIN_OWNER: [u8; 4] = [0x26, 0xef, 0x7f, 0x68];
const GET_ALL_CHAIN_OWNERS: [u8; 4] = [0x51, 0x6b, 0x4e, 0x0f];
const RECTIFY_CHAIN_OWNER: [u8; 4] = [0x6f, 0xe8, 0x63, 0x73];
const IS_NATIVE_TOKEN_OWNER: [u8; 4] = [0xc6, 0x86, 0xf4, 0xdb];
const GET_ALL_NATIVE_TOKEN_OWNERS: [u8; 4] = [0x3f, 0x86, 0x01, 0xe4];
const GET_NATIVE_TOKEN_MANAGEMENT_FROM: [u8; 4] = [0x3f, 0xec, 0xba, 0xb0];
const GET_TRANSACTION_FILTERING_FROM: [u8; 4] = [0x7a, 0x86, 0xfe, 0x96];
const IS_TRANSACTION_FILTERER: [u8; 4] = [0xa5, 0x3f, 0xef, 0x64];
const GET_ALL_TRANSACTION_FILTERERS: [u8; 4] = [0x3d, 0xbb, 0x43, 0x98];
const GET_FILTERED_FUNDS_RECIPIENT: [u8; 4] = [0x8b, 0x00, 0x16, 0x72];
const IS_CALLDATA_PRICE_INCREASE_ENABLED: [u8; 4] = [0x2a, 0xa9, 0x55, 0x1e];
const GET_PARENT_GAS_FLOOR_PER_TOKEN: [u8; 4] = [0x49, 0xcc, 0xda, 0xff];
const GET_MAX_STYLUS_CONTRACT_FRAGMENTS: [u8; 4] = [0xe5, 0xa7, 0xf8, 0x93];

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
const EVENT_GAS: u64 = 3_000;
const COPY_GAS: u64 = 3;

pub fn create_arbownerpublic_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbownerpublic"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    let result = match selector {
        GET_NETWORK_FEE_ACCOUNT => read_state_field(&mut input, NETWORK_FEE_ACCOUNT_OFFSET),
        // GetInfraFeeAccount: ArbOS >= 5
        GET_INFRA_FEE_ACCOUNT => {
            if let Some(r) = crate::check_method_version(5, 0) {
                return r;
            }
            read_state_field(&mut input, INFRA_FEE_ACCOUNT_OFFSET)
        }
        // GetBrotliCompressionLevel: ArbOS >= 20
        GET_BROTLI_COMPRESSION_LEVEL => {
            if let Some(r) = crate::check_method_version(20, 0) {
                return r;
            }
            read_state_field(&mut input, BROTLI_COMPRESSION_LEVEL_OFFSET)
        }
        // GetScheduledUpgrade: ArbOS >= 20
        GET_SCHEDULED_UPGRADE => {
            if let Some(r) = crate::check_method_version(20, 0) {
                return r;
            }
            handle_scheduled_upgrade(&mut input)
        }
        IS_CHAIN_OWNER => handle_is_chain_owner(&mut input),
        GET_ALL_CHAIN_OWNERS => handle_get_all_members(&mut input),
        RECTIFY_CHAIN_OWNER => {
            if let Some(r) = crate::check_method_version(11, 0) {
                return r;
            }
            handle_rectify_chain_owner(&mut input)
        }
        // IsNativeTokenOwner: ArbOS >= 41
        IS_NATIVE_TOKEN_OWNER => {
            if let Some(r) = crate::check_method_version(41, 0) {
                return r;
            }
            handle_is_set_member(&mut input, NATIVE_TOKEN_SUBSPACE)
        }
        // IsTransactionFilterer: ArbOS >= 60 (TransactionFiltering)
        IS_TRANSACTION_FILTERER => {
            if let Some(r) = crate::check_method_version(60, 0) {
                return r;
            }
            handle_is_set_member(&mut input, TRANSACTION_FILTERER_SUBSPACE)
        }
        // GetAllNativeTokenOwners: ArbOS >= 41
        GET_ALL_NATIVE_TOKEN_OWNERS => {
            if let Some(r) = crate::check_method_version(41, 0) {
                return r;
            }
            handle_get_all_set_members(&mut input, NATIVE_TOKEN_SUBSPACE)
        }
        // GetAllTransactionFilterers: ArbOS >= 60 (TransactionFiltering)
        GET_ALL_TRANSACTION_FILTERERS => {
            if let Some(r) = crate::check_method_version(60, 0) {
                return r;
            }
            handle_get_all_set_members(&mut input, TRANSACTION_FILTERER_SUBSPACE)
        }
        // GetNativeTokenManagementFrom: ArbOS >= 50
        GET_NATIVE_TOKEN_MANAGEMENT_FROM => {
            if let Some(r) = crate::check_method_version(50, 0) {
                return r;
            }
            read_state_field(&mut input, NATIVE_TOKEN_ENABLED_FROM_TIME_OFFSET)
        }
        // GetTransactionFilteringFrom: ArbOS >= 60 (TransactionFiltering)
        GET_TRANSACTION_FILTERING_FROM => {
            if let Some(r) = crate::check_method_version(60, 0) {
                return r;
            }
            read_state_field(&mut input, TX_FILTERING_ENABLED_FROM_TIME_OFFSET)
        }
        // GetFilteredFundsRecipient: ArbOS >= 60 (TransactionFiltering)
        GET_FILTERED_FUNDS_RECIPIENT => {
            if let Some(r) = crate::check_method_version(60, 0) {
                return r;
            }
            read_state_field(&mut input, FILTERED_FUNDS_RECIPIENT_OFFSET)
        }
        // IsCalldataPriceIncreaseEnabled: ArbOS >= 40
        IS_CALLDATA_PRICE_INCREASE_ENABLED => {
            if let Some(r) = crate::check_method_version(40, 0) {
                return r;
            }
            let gas_limit = input.gas;
            load_arbos(&mut input)?;
            let features_key = derive_subspace_key(ROOT_STORAGE_KEY, FEATURES_SUBSPACE);
            let features_slot = map_slot(features_key.as_slice(), 0);
            let features = sload_field(&mut input, features_slot)?;
            let enabled = features & U256::from(1);
            let gas_cost = (2 * SLOAD_GAS + COPY_GAS).min(gas_limit);
            Ok(PrecompileOutput::new(
                gas_cost,
                enabled.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        // GetParentGasFloorPerToken: ArbOS >= 50
        GET_PARENT_GAS_FLOOR_PER_TOKEN => {
            if let Some(r) = crate::check_method_version(50, 0) {
                return r;
            }
            let gas_limit = input.gas;
            load_arbos(&mut input)?;
            let field_slot = subspace_slot(L1_PRICING_SUBSPACE, L1_GAS_FLOOR_PER_TOKEN);
            let value = sload_field(&mut input, field_slot)?;
            Ok(PrecompileOutput::new(
                (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
                value.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        // GetMaxStylusContractFragments: ArbOS >= 60 (StylusContractLimit)
        GET_MAX_STYLUS_CONTRACT_FRAGMENTS => {
            if let Some(r) = crate::check_method_version(60, 0) {
                return r;
            }
            // OAS(800) + Params() burn(100) + resultCost(3).
            let gas_cost = (SLOAD_GAS + 100 + COPY_GAS).min(input.gas);
            Ok(PrecompileOutput::new(gas_cost, vec![0u8; 32].into()))
        }
        _ => Err(PrecompileError::other("unknown ArbOwnerPublic selector")),
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

fn sstore_field(input: &mut PrecompileInput<'_>, slot: U256, value: U256) -> Result<(), PrecompileError> {
    input
        .internals_mut()
        .sstore(ARBOS_STATE_ADDRESS, slot, value)
        .map_err(|_| PrecompileError::other("sstore failed"))?;
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

fn handle_rectify_chain_owner(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
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
    input.internals_mut().log(alloy_primitives::Log::new_unchecked(
        ARBOWNERPUBLIC_ADDRESS,
        vec![topic0],
        addr_hash.0.to_vec().into(),
    ));

    const SSTORE_ZERO_GAS: u64 = 5_000;
    const RECTIFY_EVENT_GAS: u64 = 1_006; // LOG1 + 32 bytes data
    let gas_used = SLOAD_GAS + 7 * SLOAD_GAS + SSTORE_ZERO_GAS + 3 * SSTORE_GAS + RECTIFY_EVENT_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(gas_used.min(gas_limit), Vec::new().into()))
}

fn handle_is_chain_owner(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
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

fn handle_is_set_member(input: &mut PrecompileInput<'_>, subspace: &[u8]) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("input too short"));
    }
    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
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
