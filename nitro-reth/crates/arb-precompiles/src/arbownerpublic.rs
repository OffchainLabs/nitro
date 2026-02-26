use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{
    compute_storage_slot, ARBOS_STATE_ADDRESS, L1_PRICING_SPACE,
};

/// ArbOwnerPublic precompile address (0x6b).
pub const ARBOWNERPUBLIC_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x6b,
]);

// Function selectors.
const GET_NETWORK_FEE_ACCOUNT: [u8; 4] = [0x3e, 0x7a, 0x47, 0xb1];
const GET_INFRA_FEE_ACCOUNT: [u8; 4] = [0x74, 0x33, 0x16, 0x04];
const GET_BROTLI_COMPRESSION_LEVEL: [u8; 4] = [0xb1, 0x9e, 0x6b, 0xef];
const GET_SCHEDULED_UPGRADE: [u8; 4] = [0xed, 0x23, 0xfa, 0x57];
const IS_CHAIN_OWNER: [u8; 4] = [0x26, 0xef, 0x69, 0x9d];
const GET_ALL_CHAIN_OWNERS: [u8; 4] = [0x51, 0x6b, 0xaf, 0x03];
const RECTIFY_CHAIN_OWNER: [u8; 4] = [0x18, 0x3b, 0xe5, 0xf2];
const IS_NATIVE_TOKEN_OWNER: [u8; 4] = [0x40, 0xb6, 0x62, 0x08];
const GET_ALL_NATIVE_TOKEN_OWNERS: [u8; 4] = [0xf5, 0xc8, 0x16, 0x7a];
const GET_NATIVE_TOKEN_MANAGEMENT_FROM: [u8; 4] = [0xaa, 0x57, 0x87, 0x88];
const GET_TRANSACTION_FILTERING_FROM: [u8; 4] = [0x7a, 0x86, 0xfe, 0x96];
const IS_TRANSACTION_FILTERER: [u8; 4] = [0xa5, 0x3f, 0xef, 0x64];
const GET_ALL_TRANSACTION_FILTERERS: [u8; 4] = [0x3d, 0xbb, 0x43, 0x98];
const GET_FILTERED_FUNDS_RECIPIENT: [u8; 4] = [0x8b, 0x00, 0x16, 0x72];
const IS_CALLDATA_PRICE_INCREASE_ENABLED: [u8; 4] = [0x7f, 0xe5, 0x5a, 0x2f];
const GET_PARENT_GAS_FLOOR_PER_TOKEN: [u8; 4] = [0xee, 0x36, 0x03, 0x8e];
const GET_MAX_STYLUS_CONTRACT_FRAGMENTS: [u8; 4] = [0xea, 0x25, 0x8c, 0x64];

// ArbOS state offsets (from arbosState).
const NETWORK_FEE_ACCOUNT_OFFSET: u64 = 3;
const INFRA_FEE_ACCOUNT_OFFSET: u64 = 6;
const BROTLI_COMPRESSION_LEVEL_OFFSET: u64 = 7;
const UPGRADE_VERSION_OFFSET: u64 = 1;
const UPGRADE_TIMESTAMP_OFFSET: u64 = 2;

// Chain owners are stored in an AddressSet at state offset 5.
const CHAIN_OWNERS_OFFSET: u64 = 5;
// L1 pricing field for gas floor per token
const L1_GAS_FLOOR_PER_TOKEN: u64 = 12;

const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;

pub fn create_arbownerpublic_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbownerpublic"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    match selector {
        GET_NETWORK_FEE_ACCOUNT => read_state_field(&mut input, NETWORK_FEE_ACCOUNT_OFFSET),
        GET_INFRA_FEE_ACCOUNT => read_state_field(&mut input, INFRA_FEE_ACCOUNT_OFFSET),
        GET_BROTLI_COMPRESSION_LEVEL => {
            read_state_field(&mut input, BROTLI_COMPRESSION_LEVEL_OFFSET)
        }
        GET_SCHEDULED_UPGRADE => handle_scheduled_upgrade(&mut input),
        IS_CHAIN_OWNER => handle_is_chain_owner(&mut input),
        GET_ALL_CHAIN_OWNERS => handle_get_all_members(&mut input, CHAIN_OWNERS_OFFSET),
        RECTIFY_CHAIN_OWNER => {
            // Rectify is a no-op if the address is already an owner.
            let gas_cost = COPY_GAS.min(input.gas);
            Ok(PrecompileOutput::new(gas_cost, Vec::new().into()))
        }
        IS_NATIVE_TOKEN_OWNER | IS_TRANSACTION_FILTERER => {
            // Check membership — return false for now (no address set reading).
            let gas_cost = COPY_GAS.min(input.gas);
            Ok(PrecompileOutput::new(gas_cost, vec![0u8; 32].into()))
        }
        GET_ALL_NATIVE_TOKEN_OWNERS | GET_ALL_TRANSACTION_FILTERERS => {
            handle_empty_address_array(&mut input)
        }
        GET_NATIVE_TOKEN_MANAGEMENT_FROM | GET_TRANSACTION_FILTERING_FROM => {
            // Return zero timestamp (feature not enabled).
            let gas_cost = COPY_GAS.min(input.gas);
            Ok(PrecompileOutput::new(gas_cost, vec![0u8; 32].into()))
        }
        GET_FILTERED_FUNDS_RECIPIENT => {
            let gas_cost = COPY_GAS.min(input.gas);
            Ok(PrecompileOutput::new(gas_cost, vec![0u8; 32].into()))
        }
        IS_CALLDATA_PRICE_INCREASE_ENABLED => {
            // Return true (enabled by default on recent ArbOS versions).
            let gas_cost = COPY_GAS.min(input.gas);
            Ok(PrecompileOutput::new(
                gas_cost,
                U256::from(1u64).to_be_bytes::<32>().to_vec().into(),
            ))
        }
        GET_PARENT_GAS_FLOOR_PER_TOKEN => {
            let gas_limit = input.gas;
            load_arbos(&mut input)?;
            let l1_slot = compute_storage_slot(&[], L1_PRICING_SPACE);
            let field_slot = compute_storage_slot(&[l1_slot], L1_GAS_FLOOR_PER_TOKEN);
            let value = sload_field(&mut input, field_slot)?;
            Ok(PrecompileOutput::new(
                (SLOAD_GAS + COPY_GAS).min(gas_limit),
                value.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        GET_MAX_STYLUS_CONTRACT_FRAGMENTS => {
            // Return 0 (Stylus config not yet stored in state).
            let gas_cost = COPY_GAS.min(input.gas);
            Ok(PrecompileOutput::new(gas_cost, vec![0u8; 32].into()))
        }
        _ => Err(PrecompileError::other(
            "unknown ArbOwnerPublic selector",
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

fn read_state_field(input: &mut PrecompileInput<'_>, offset: u64) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    // Root-level ArbOS state fields are at slot = offset directly.
    let value = sload_field(input, U256::from(offset))?;
    Ok(PrecompileOutput::new(
        (SLOAD_GAS + COPY_GAS).min(gas_limit),
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_scheduled_upgrade(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let version = sload_field(input, U256::from(UPGRADE_VERSION_OFFSET))?;
    let timestamp = sload_field(input, U256::from(UPGRADE_TIMESTAMP_OFFSET))?;

    let mut out = Vec::with_capacity(64);
    out.extend_from_slice(&version.to_be_bytes::<32>());
    out.extend_from_slice(&timestamp.to_be_bytes::<32>());

    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        out.into(),
    ))
}

fn handle_is_chain_owner(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    load_arbos(input)?;

    // Chain owners are in an AddressSet at offset CHAIN_OWNERS_OFFSET.
    // AddressSet stores size at slot 0, and has a by_address sub-storage
    // that maps keccak256(address) to a non-zero value if the address is a member.
    let owners_slot = compute_storage_slot(&[], CHAIN_OWNERS_OFFSET);
    let by_address_slot = compute_storage_slot(&[owners_slot], 0);

    // The by_address mapping key is keccak256(address padded to 32 bytes).
    let mut addr_padded = [0u8; 32];
    addr_padded[12..32].copy_from_slice(addr.as_slice());
    let addr_key = U256::from_be_bytes(alloy_primitives::keccak256(&addr_padded).0);
    let member_slot = by_address_slot.wrapping_add(addr_key);

    let value = sload_field(input, member_slot)?;
    let is_owner = value != U256::ZERO;

    let result = if is_owner { U256::from(1u64) } else { U256::ZERO };

    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        result.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_get_all_members(input: &mut PrecompileInput<'_>, set_offset: u64) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    // AddressSet: slot 0 = size, sub-storage at offset 1 = by_address, offset 2 = members list.
    let set_slot = compute_storage_slot(&[], set_offset);
    let size = sload_field(input, set_slot)?;
    let count: u64 = size.try_into().unwrap_or(0);

    // Members are stored starting at the list sub-storage.
    let list_slot = compute_storage_slot(&[set_slot], 2);

    // ABI: offset to dynamic array, array length, then elements
    let mut out = Vec::with_capacity(64 + count as usize * 32);
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(count).to_be_bytes::<32>());

    let max_members = count.min(256); // Safety cap
    for i in 0..max_members {
        let member_slot = list_slot.wrapping_add(U256::from(i));
        let addr_val = sload_field(input, member_slot)?;
        out.extend_from_slice(&addr_val.to_be_bytes::<32>());
    }

    Ok(PrecompileOutput::new(
        ((1 + max_members) * SLOAD_GAS + COPY_GAS).min(gas_limit),
        out.into(),
    ))
}

fn handle_empty_address_array(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_cost = COPY_GAS.min(input.gas);
    let mut out = Vec::with_capacity(64);
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::ZERO.to_be_bytes::<32>());
    Ok(PrecompileOutput::new(gas_cost, out.into()))
}
