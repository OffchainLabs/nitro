use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, Bytes, U256};
use alloy_sol_types::SolInterface;
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::{
    interfaces::IArbAddressTable,
    storage_slot::{
        derive_subspace_key, map_slot, map_slot_b256, ADDRESS_TABLE_SUBSPACE, ARBOS_STATE_ADDRESS,
        ROOT_STORAGE_KEY,
    },
};

/// ArbAddressTable precompile address (0x66).
pub const ARBADDRESSTABLE_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x66,
]);

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const COPY_GAS: u64 = 3;

pub fn create_arbaddresstable_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbaddresstable"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    crate::init_precompile_gas(input.data.len());

    let call = match IArbAddressTable::ArbAddressTableCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbAddressTable::ArbAddressTableCalls as Calls;
    let result = match call {
        Calls::size(_) => handle_size(&mut input),
        Calls::addressExists(c) => handle_address_exists(&mut input, c.addr),
        Calls::lookup(c) => handle_lookup(&mut input, c.addr),
        Calls::lookupIndex(c) => handle_lookup_index(&mut input, c.index),
        Calls::register(c) => handle_register(&mut input, c.addr),
        Calls::compress(c) => handle_compress(&mut input, c.addr),
        Calls::decompress(c) => handle_decompress(&mut input, &c.buf, c.offset),
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

fn handle_address_exists(input: &mut PrecompileInput<'_>, addr: Address) -> PrecompileResult {
    let gas_limit = input.gas;
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

fn handle_lookup(input: &mut PrecompileInput<'_>, addr: Address) -> PrecompileResult {
    let gas_limit = input.gas;
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

/// Reverse entries are stored at offset (index + 1) in the table's backing storage.
fn handle_lookup_index(input: &mut PrecompileInput<'_>, index_u256: U256) -> PrecompileResult {
    let gas_limit = input.gas;
    let index: u64 = index_u256
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

/// If already registered, returns the existing index; otherwise registers and
/// returns the new 0-based index.
fn handle_register(input: &mut PrecompileInput<'_>, addr: Address) -> PrecompileResult {
    let gas_limit = input.gas;
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

fn handle_compress(input: &mut PrecompileInput<'_>, addr: Address) -> PrecompileResult {
    let gas_limit = input.gas;
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
    output.extend(std::iter::repeat_n(0u8, pad));

    // OAS(1) + byAddress.Get(1) + argsCost(3) + resultCost(3 words for bytes encoding).
    let result_words = (output.len() as u64).div_ceil(32);
    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS + result_words * COPY_GAS).min(gas_limit),
        output.into(),
    ))
}

/// If the RLP payload is 20 bytes it's a raw address; otherwise it's a u64
/// index looked up in the table. Returns ABI-encoded (address, uint256 bytesRead).
fn handle_decompress(
    input: &mut PrecompileInput<'_>,
    buf: &Bytes,
    offset: U256,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let data_len = input.data.len();
    let ioffset: usize = offset
        .try_into()
        .map_err(|_| PrecompileError::other("offset too large"))?;

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

    // Body: OAS(1) + 0 (raw addr) or OAS(1) + numItems(1) + backing(1) (index).
    let body_sloads: u64 = if decoded.len() == 20 { 1 } else { 3 };
    let arg_words = (data_len as u64).saturating_sub(4).div_ceil(32);
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
