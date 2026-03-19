use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{
    derive_subspace_key, map_slot, map_slot_b256, ADDRESS_TABLE_SUBSPACE, ARBOS_STATE_ADDRESS,
    ROOT_STORAGE_KEY,
};

/// ArbAddressTable precompile address (0x66).
pub const ARBADDRESSTABLE_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x66,
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
    let gas_limit = input.gas;
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    let result = match selector {
        SIZE => handle_size(&mut input),
        ADDRESS_EXISTS => handle_address_exists(&mut input),
        LOOKUP => handle_lookup(&mut input),
        LOOKUP_INDEX => handle_lookup_index(&mut input),
        REGISTER => handle_register(&mut input),
        COMPRESS => handle_compress(&mut input),
        DECOMPRESS => handle_decompress(&mut input),
        _ => Err(PrecompileError::other("unknown ArbAddressTable selector")),
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
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
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

    // OAS(1) + byAddress.Get(1) + argsCost(3) + resultCost(3).
    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
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
    // OAS(1) + byAddress.Get(1) + argsCost(3) + resultCost(3).
    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
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

    // OAS(1) + numItems(1) + backing(1) + argsCost(3) + resultCost(3).
    Ok(PrecompileOutput::new(
        (3 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
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
        // OAS(1) + byAddress.Get(1) + argsCost(3) + resultCost(3).
        return Ok(PrecompileOutput::new(
            (2 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
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

    // OAS(1) + byAddress.Get(1) + numItems.Get(1) + 3 sstores + argsCost(3) + resultCost(3).
    let gas_used = 3 * SLOAD_GAS + 3 * SSTORE_GAS + 2 * COPY_GAS;

    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        U256::from(index).to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Compress: if address is in table, RLP-encode its index; else RLP-encode the 20-byte address.
/// Returns ABI-encoded bytes.
fn handle_compress(input: &mut PrecompileInput<'_>) -> PrecompileResult {
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

    let rlp_bytes = if value != U256::ZERO {
        // Address exists — RLP-encode the 0-based index.
        let index = value.wrapping_sub(U256::from(1u64)).to::<u64>();
        rlp_encode_u64(index)
    } else {
        // Not in table — RLP-encode the raw 20-byte address.
        rlp_encode_bytes(addr.as_slice())
    };

    // ABI-encode as `bytes`: offset (32) + length (32) + padded data.
    let mut output = Vec::with_capacity(96);
    output.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    output.extend_from_slice(&U256::from(rlp_bytes.len() as u64).to_be_bytes::<32>());
    output.extend_from_slice(&rlp_bytes);
    // Pad to 32-byte boundary.
    let pad = (32 - rlp_bytes.len() % 32) % 32;
    output.extend(core::iter::repeat(0u8).take(pad));

    // OAS(1) + byAddress.Get(1) + argsCost(3) + resultCost(3 words for bytes encoding).
    let result_words = (output.len() as u64 + 31) / 32;
    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS + result_words * COPY_GAS).min(gas_limit),
        output.into(),
    ))
}

/// Decompress: RLP-decode from buf[offset..]. If 20 bytes, it's a raw address.
/// Otherwise decode as u64 index and look up in the table.
/// Returns ABI-encoded (address, uint256 bytesRead).
fn handle_decompress(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    // ABI: decompress(bytes buf, uint256 offset)
    // data[4..36] = offset to bytes data
    // data[36..68] = offset value
    if data.len() < 68 {
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;

    // Decode ABI bytes parameter.
    let bytes_offset = U256::from_be_slice(&data[4..36]).to::<usize>();
    let ioffset = U256::from_be_slice(&data[36..68]).to::<usize>();

    // bytes data starts at 4 + bytes_offset (skip selector).
    let bytes_start = 4 + bytes_offset;
    if data.len() < bytes_start + 32 {
        return Err(PrecompileError::other("invalid bytes offset"));
    }
    let bytes_len = U256::from_be_slice(&data[bytes_start..bytes_start + 32]).to::<usize>();
    let bytes_data_start = bytes_start + 32;
    if data.len() < bytes_data_start + bytes_len {
        return Err(PrecompileError::other("bytes data truncated"));
    }
    let buf = &data[bytes_data_start..bytes_data_start + bytes_len];

    if ioffset >= buf.len() {
        return Err(PrecompileError::other("offset out of bounds"));
    }
    let slice = &buf[ioffset..];

    load_arbos(input)?;

    // Try to RLP-decode as byte string first.
    let (decoded, bytes_read) =
        rlp_decode_bytes(slice).map_err(|_| PrecompileError::other("RLP decode failed"))?;

    let (addr, final_bytes_read) = if decoded.len() == 20 {
        // Raw 20-byte address.
        (Address::from_slice(&decoded), bytes_read)
    } else {
        // Re-decode as u64 index.
        let (index, idx_bytes_read) =
            rlp_decode_u64(slice).map_err(|_| PrecompileError::other("RLP decode index failed"))?;

        let table_key = derive_subspace_key(ROOT_STORAGE_KEY, ADDRESS_TABLE_SUBSPACE);

        // Bounds check: index < numItems.
        let num_items_slot = map_slot(table_key.as_slice(), 0);
        let num_items = sload_field(input, num_items_slot)?;
        if U256::from(index) >= num_items {
            return Err(PrecompileError::other(
                "index does not exist in AddressTable",
            ));
        }

        let entry_slot = map_slot(table_key.as_slice(), index + 1);
        let value = sload_field(input, entry_slot)?;

        // Extract 20-byte address from the 32-byte stored value.
        let value_bytes = value.to_be_bytes::<32>();
        let result_addr = Address::from_slice(&value_bytes[12..32]);
        (result_addr, idx_bytes_read)
    };

    // ABI-encode (address, uint256).
    let mut output = Vec::with_capacity(64);
    output.extend_from_slice(&alloy_primitives::B256::left_padding_from(addr.as_slice()).0);
    output.extend_from_slice(&U256::from(final_bytes_read as u64).to_be_bytes::<32>());

    // Variable body sloads + argsCost(dynamic) + resultCost = 2 words × 3 = 6.
    // Body: OAS(1) + 0 (raw addr) or OAS(1) + numItems(1) + backing(1) (index).
    let body_sloads: u64 = if decoded.len() == 20 { 1 } else { 3 };
    let arg_words = ((input.data.len() as u64).saturating_sub(4) + 31) / 32;
    Ok(PrecompileOutput::new(
        (body_sloads * SLOAD_GAS + (arg_words + 2) * COPY_GAS).min(gas_limit),
        output.into(),
    ))
}

// ── Minimal RLP helpers ─────────────────────────────────────────────

/// RLP-encode a u64 value.
fn rlp_encode_u64(val: u64) -> Vec<u8> {
    if val == 0 {
        return vec![0x80];
    }
    if val < 128 {
        return vec![val as u8];
    }
    let bytes = val.to_be_bytes();
    let start = bytes.iter().position(|&b| b != 0).unwrap_or(7);
    let len = 8 - start;
    let mut out = Vec::with_capacity(1 + len);
    out.push(0x80 + len as u8);
    out.extend_from_slice(&bytes[start..]);
    out
}

/// RLP-encode a byte slice as an RLP string.
fn rlp_encode_bytes(data: &[u8]) -> Vec<u8> {
    if data.len() == 1 && data[0] < 128 {
        return vec![data[0]];
    }
    if data.len() < 56 {
        let mut out = Vec::with_capacity(1 + data.len());
        out.push(0x80 + data.len() as u8);
        out.extend_from_slice(data);
        return out;
    }
    let len_bytes = {
        let l = data.len() as u64;
        let bytes = l.to_be_bytes();
        let start = bytes.iter().position(|&b| b != 0).unwrap_or(7);
        bytes[start..].to_vec()
    };
    let mut out = Vec::with_capacity(1 + len_bytes.len() + data.len());
    out.push(0xb7 + len_bytes.len() as u8);
    out.extend_from_slice(&len_bytes);
    out.extend_from_slice(data);
    out
}

/// RLP-decode a byte string from a slice, returning (decoded_bytes, total_bytes_consumed).
fn rlp_decode_bytes(data: &[u8]) -> Result<(Vec<u8>, usize), &'static str> {
    if data.is_empty() {
        return Err("empty input");
    }
    let prefix = data[0];
    if prefix < 0x80 {
        // Single byte.
        Ok((vec![prefix], 1))
    } else if prefix <= 0xb7 {
        // Short string (0-55 bytes).
        let len = (prefix - 0x80) as usize;
        if data.len() < 1 + len {
            return Err("truncated short string");
        }
        Ok((data[1..1 + len].to_vec(), 1 + len))
    } else if prefix <= 0xbf {
        // Long string.
        let len_of_len = (prefix - 0xb7) as usize;
        if data.len() < 1 + len_of_len {
            return Err("truncated long string length");
        }
        let mut len_bytes = [0u8; 8];
        len_bytes[8 - len_of_len..].copy_from_slice(&data[1..1 + len_of_len]);
        let len = u64::from_be_bytes(len_bytes) as usize;
        let total = 1 + len_of_len + len;
        if data.len() < total {
            return Err("truncated long string data");
        }
        Ok((data[1 + len_of_len..total].to_vec(), total))
    } else {
        Err("unexpected list prefix")
    }
}

/// RLP-decode a u64 from a slice.
fn rlp_decode_u64(data: &[u8]) -> Result<(u64, usize), &'static str> {
    if data.is_empty() {
        return Err("empty input");
    }
    let prefix = data[0];
    if prefix == 0x80 {
        // Zero.
        Ok((0, 1))
    } else if prefix < 0x80 {
        // Single byte value.
        Ok((prefix as u64, 1))
    } else if prefix <= 0x88 {
        // Short string encoding for integers (up to 8 bytes).
        let len = (prefix - 0x80) as usize;
        if len > 8 || data.len() < 1 + len {
            return Err("invalid u64 encoding");
        }
        let mut bytes = [0u8; 8];
        bytes[8 - len..].copy_from_slice(&data[1..1 + len]);
        Ok((u64::from_be_bytes(bytes), 1 + len))
    } else {
        Err("value too large for u64")
    }
}
