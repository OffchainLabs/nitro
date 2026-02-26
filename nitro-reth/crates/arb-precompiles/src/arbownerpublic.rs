use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{compute_storage_slot, ARBOS_STATE_ADDRESS};

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

// ArbOS state offsets (from arbosState).
const NETWORK_FEE_ACCOUNT_OFFSET: u64 = 3;
const INFRA_FEE_ACCOUNT_OFFSET: u64 = 6;
const BROTLI_COMPRESSION_LEVEL_OFFSET: u64 = 7;
const UPGRADE_VERSION_OFFSET: u64 = 1;
const UPGRADE_TIMESTAMP_OFFSET: u64 = 2;

// Chain owners are stored in an AddressSet at state offset 5.
const CHAIN_OWNERS_OFFSET: u64 = 5;

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
