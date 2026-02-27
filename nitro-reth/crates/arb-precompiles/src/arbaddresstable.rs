use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{
    derive_subspace_key, map_slot, map_slot_b256, ARBOS_STATE_ADDRESS, ADDRESS_TABLE_SUBSPACE,
    ROOT_STORAGE_KEY,
};

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

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
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
        REGISTER => handle_register(&mut input),
        COMPRESS | DECOMPRESS => {
            // Compress/Decompress involve RLP encoding/decoding.
            Err(PrecompileError::other(
                "address table compress/decompress not yet supported",
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

fn sstore_field(
    input: &mut PrecompileInput<'_>,
    slot: U256,
    value: U256,
) -> Result<(), PrecompileError> {
    input
        .internals_mut()
        .sstore(ARBOS_STATE_ADDRESS, slot, value)
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    Ok(())
}

/// AddressTable numItems is stored at offset 0 in the table's subspace storage.
fn handle_size(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let table_key = derive_subspace_key(ROOT_STORAGE_KEY, ADDRESS_TABLE_SUBSPACE);
    let size_slot = map_slot(table_key.as_slice(), 0);
    let size = sload_field(input, size_slot)?;

    Ok(PrecompileOutput::new(
        (SLOAD_GAS + COPY_GAS).min(gas_limit),
        size.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Check if an address exists in the table by looking up the byAddress sub-storage.
fn handle_address_exists(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    load_arbos(input)?;

    // byAddress = OpenSubStorage([]byte{}) — sub-storage with empty key.
    let table_key = derive_subspace_key(ROOT_STORAGE_KEY, ADDRESS_TABLE_SUBSPACE);
    let by_address_key = derive_subspace_key(table_key.as_slice(), &[]);

    let addr_as_b256 = alloy_primitives::B256::left_padding_from(addr.as_slice());
    let member_slot = map_slot_b256(by_address_key.as_slice(), &addr_as_b256);

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

    let table_key = derive_subspace_key(ROOT_STORAGE_KEY, ADDRESS_TABLE_SUBSPACE);
    let by_address_key = derive_subspace_key(table_key.as_slice(), &[]);

    let addr_as_b256 = alloy_primitives::B256::left_padding_from(addr.as_slice());
    let member_slot = map_slot_b256(by_address_key.as_slice(), &addr_as_b256);

    let value = sload_field(input, member_slot)?;
    if value == U256::ZERO {
        return Err(PrecompileError::other(
            "address does not exist in AddressTable",
        ));
    }

    // Stored value is the 1-based index, so subtract 1.
    let index = value.wrapping_sub(U256::from(1u64));
    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        index.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Lookup an address by index in the table.
/// Reverse entries are stored at offset (index + 1) in the table's backing storage.
fn handle_lookup_index(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let index: u64 = U256::from_be_slice(&data[4..36])
        .try_into()
        .map_err(|_| PrecompileError::other("index too large"))?;
    load_arbos(input)?;

    let table_key = derive_subspace_key(ROOT_STORAGE_KEY, ADDRESS_TABLE_SUBSPACE);
    // Reverse lookup is at offset (index + 1) — 1-indexed.
    let entry_slot = map_slot(table_key.as_slice(), index + 1);
    let value = sload_field(input, entry_slot)?;

    if value == U256::ZERO {
        return Err(PrecompileError::other(
            "index does not exist in AddressTable",
        ));
    }

    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Register an address in the table. If it already exists, returns its index.
/// Otherwise, adds it and returns the new 0-based index.
///
/// Storage layout:
/// - numItems at offset 0 in table subspace
/// - byAddress: sub-storage with empty key, maps addr_hash → 1-based index
/// - backing: maps (index + 1) → addr_hash (reverse lookup)
fn handle_register(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let addr = Address::from_slice(&data[16..36]);
    load_arbos(input)?;

    let table_key = derive_subspace_key(ROOT_STORAGE_KEY, ADDRESS_TABLE_SUBSPACE);
    let by_address_key = derive_subspace_key(table_key.as_slice(), &[]);
    let addr_as_b256 = alloy_primitives::B256::left_padding_from(addr.as_slice());

    // Check if address already exists in byAddress mapping.
    let member_slot = map_slot_b256(by_address_key.as_slice(), &addr_as_b256);
    let existing = sload_field(input, member_slot)?;

    if existing != U256::ZERO {
        // Already registered — return 0-based index.
        let index = existing.wrapping_sub(U256::from(1u64));
        return Ok(PrecompileOutput::new(
            (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
            index.to_be_bytes::<32>().to_vec().into(),
        ));
    }

    // Not yet registered — add it.
    // Read numItems and increment it.
    let num_items_slot = map_slot(table_key.as_slice(), 0);
    let num_items = sload_field(input, num_items_slot)?;
    let num_items_u64: u64 = num_items
        .try_into()
        .map_err(|_| PrecompileError::other("invalid numItems"))?;
    let new_num_items = num_items_u64 + 1;
    sstore_field(input, num_items_slot, U256::from(new_num_items))?;

    // Store reverse mapping: backingStorage[newNumItems] = addr_hash.
    let reverse_slot = map_slot(table_key.as_slice(), new_num_items);
    sstore_field(input, reverse_slot, U256::from_be_bytes(addr_as_b256.0))?;

    // Store byAddress mapping: byAddress[addr_hash] = newNumItems (1-based).
    sstore_field(input, member_slot, U256::from(new_num_items))?;

    // Return 0-based index.
    let index = new_num_items - 1;

    // Gas: 2 sloads (byAddress lookup + numItems) + 3 sstores (numItems, reverse, byAddress)
    let gas_used = 2 * SLOAD_GAS + 3 * SSTORE_GAS + COPY_GAS;

    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        U256::from(index).to_be_bytes::<32>().to_vec().into(),
    ))
}
