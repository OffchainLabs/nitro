use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{compute_storage_slot, ARBOS_STATE_ADDRESS};

/// ArbAddressTable precompile address (0x66).
pub const ARBADDRESSTABLE_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x66,
]);

// Function selectors.
const ADDRESS_EXISTS: [u8; 4] = [0xa5, 0x02, 0x52, 0x22];
const COMPRESS: [u8; 4] = [0xf6, 0xa4, 0x55, 0xa2];
const DECOMPRESS: [u8; 4] = [0x31, 0x86, 0x2a, 0xda];
const LOOKUP: [u8; 4] = [0xd4, 0xb6, 0xb5, 0xda];
const LOOKUP_INDEX: [u8; 4] = [0x8a, 0x18, 0x67, 0x88];
const REGISTER: [u8; 4] = [0x44, 0x20, 0xe4, 0x86];
const SIZE: [u8; 4] = [0x94, 0x9d, 0x22, 0x5d];

/// ArbOS state offset for the address table.
const ADDRESS_TABLE_OFFSET: u64 = 4;

const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;

pub fn create_arbaddresstable_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbaddresstable"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    match selector {
        SIZE => handle_size(&mut input),
        ADDRESS_EXISTS => handle_address_exists(&mut input),
        LOOKUP => handle_lookup(&mut input),
        LOOKUP_INDEX => handle_lookup_index(&mut input),
        REGISTER | COMPRESS | DECOMPRESS => {
            // These methods require write access or complex serialization.
            Err(PrecompileError::other(
                "address table write/compress operations not yet supported",
            ))
        }
        _ => Err(PrecompileError::other(
            "unknown ArbAddressTable selector",
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

/// AddressTable size is stored at the root of the address table storage space.
/// In Go: `addressTable.size()` reads from `storageBackedUint64` at offset 0 in the
/// address table's storage slot.
fn handle_size(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let table_slot = compute_storage_slot(&[], ADDRESS_TABLE_OFFSET);
    // The "numItems" field is at sub-offset 0 within the address table.
    let size_slot = compute_storage_slot(&[table_slot], 0);
    let size = sload_field(input, size_slot)?;

    Ok(PrecompileOutput::new(
        (SLOAD_GAS + COPY_GAS).min(gas_limit),
        size.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Check if an address exists in the table by looking up the by-address mapping.
fn handle_address_exists(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    load_arbos(input)?;

    let table_slot = compute_storage_slot(&[], ADDRESS_TABLE_OFFSET);
    // byAddress mapping is at sub-offset 1 within the address table.
    let by_address_slot = compute_storage_slot(&[table_slot], 1);

    let mut addr_padded = [0u8; 32];
    addr_padded[12..32].copy_from_slice(addr.as_slice());
    let addr_key = U256::from_be_bytes(alloy_primitives::keccak256(&addr_padded).0);
    let member_slot = by_address_slot.wrapping_add(addr_key);

    let value = sload_field(input, member_slot)?;
    let exists = if value != U256::ZERO {
        U256::from(1u64)
    } else {
        U256::ZERO
    };

    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        exists.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Lookup the index of an address in the table.
fn handle_lookup(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    load_arbos(input)?;

    let table_slot = compute_storage_slot(&[], ADDRESS_TABLE_OFFSET);
    let by_address_slot = compute_storage_slot(&[table_slot], 1);

    let mut addr_padded = [0u8; 32];
    addr_padded[12..32].copy_from_slice(addr.as_slice());
    let addr_key = U256::from_be_bytes(alloy_primitives::keccak256(&addr_padded).0);
    let member_slot = by_address_slot.wrapping_add(addr_key);

    let value = sload_field(input, member_slot)?;
    if value == U256::ZERO {
        return Err(PrecompileError::other(
            "address does not exist in AddressTable",
        ));
    }

    // The stored value is the 1-based index, so subtract 1.
    let index = value.wrapping_sub(U256::from(1u64));
    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        index.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Lookup an address by index in the table.
fn handle_lookup_index(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let index = U256::from_be_slice(&data[4..36]);
    load_arbos(input)?;

    let table_slot = compute_storage_slot(&[], ADDRESS_TABLE_OFFSET);
    // byIndex mapping is at sub-offset 2 within the address table.
    let by_index_slot = compute_storage_slot(&[table_slot], 2);

    let entry_slot = by_index_slot.wrapping_add(index);
    let value = sload_field(input, entry_slot)?;

    if value == U256::ZERO {
        return Err(PrecompileError::other(
            "index does not exist in AddressTable",
        ));
    }

    // The stored value is the address (stored as a U256, right-aligned).
    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}
