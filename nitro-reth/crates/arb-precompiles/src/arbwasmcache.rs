use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, B256, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{
    derive_subspace_key, map_slot, map_slot_b256, ARBOS_STATE_ADDRESS, CACHE_MANAGERS_KEY,
    PROGRAMS_DATA_KEY, PROGRAMS_SUBSPACE, ROOT_STORAGE_KEY,
};

/// ArbWasmCache precompile address (0x72).
pub const ARBWASMCACHE_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x72,
]);

// Function selectors.
const IS_CACHE_MANAGER: [u8; 4] = [0xf1, 0x37, 0xfc, 0xda];
const ALL_CACHE_MANAGERS: [u8; 4] = [0x35, 0x17, 0x3c, 0x26];
const CACHE_CODEHASH: [u8; 4] = [0x0e, 0xa0, 0x7a, 0x7a];
const CACHE_PROGRAM: [u8; 4] = [0xb6, 0xf4, 0xfb, 0x22];
const EVICT_CODEHASH: [u8; 4] = [0xd4, 0x56, 0xcd, 0x34];
const CODEHASH_IS_CACHED: [u8; 4] = [0x47, 0x97, 0x00, 0xf6];

const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;

/// AddressSet by_address sub-key.
const BY_ADDRESS_KEY: &[u8] = &[0];

pub fn create_arbwasmcache_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbwasmcache"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    // ArbWasmCache requires ArbOS >= 30 (Stylus).
    if let Some(result) =
        crate::check_precompile_version(arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS)
    {
        return result;
    }

    let data = input.data;
    if data.len() < 4 {
        return crate::burn_all_revert(input.gas);
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    crate::init_precompile_gas(data.len());

    let result = match selector {
        // CacheCodehash: available only on ArbOS 30, replaced by CacheProgram at 31.
        CACHE_CODEHASH => {
            if let Some(result) = crate::check_method_version(
                arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS,
                arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS,
            ) {
                return result;
            }
            let _ = &mut input;
            Err(PrecompileError::other("caller is not a cache manager"))
        }
        // CacheProgram: requires ArbOS >= 31 (StylusFixes).
        CACHE_PROGRAM => {
            if let Some(result) = crate::check_method_version(
                arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS_FIXES,
                0,
            ) {
                return result;
            }
            let _ = &mut input;
            Err(PrecompileError::other("caller is not a cache manager"))
        }
        IS_CACHE_MANAGER => handle_is_cache_manager(&mut input),
        ALL_CACHE_MANAGERS => handle_all_cache_managers(&mut input),
        CODEHASH_IS_CACHED => handle_codehash_is_cached(&mut input),
        EVICT_CODEHASH => {
            let _ = &mut input;
            Err(PrecompileError::other("caller is not a cache manager"))
        }
        _ => return crate::burn_all_revert(input.gas),
    };
    crate::gas_check(input.gas, result)
}

// ── Helpers ──────────────────────────────────────────────────────────

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

/// Compute the cache managers AddressSet storage key.
fn cache_managers_key() -> B256 {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    derive_subspace_key(programs_key.as_slice(), CACHE_MANAGERS_KEY)
}

/// Check if an address is a cache manager member.
fn handle_is_cache_manager(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("calldata too short for address arg"));
    }
    // Address is right-aligned in 32-byte word.
    let mut addr_bytes = [0u8; 20];
    addr_bytes.copy_from_slice(&data[16..36]);
    let addr = Address::from(addr_bytes);

    load_arbos(input)?;

    let cm_key = cache_managers_key();
    let by_addr_key = derive_subspace_key(cm_key.as_slice(), BY_ADDRESS_KEY);
    let addr_hash = address_to_b256(addr);
    let slot = map_slot_b256(by_addr_key.as_slice(), &addr_hash);
    let value = sload_field(input, slot)?;

    let is_member = value != U256::ZERO;
    let result = if is_member {
        U256::from(1u64)
    } else {
        U256::ZERO
    };
    Ok(PrecompileOutput::new(
        SLOAD_GAS + COPY_GAS,
        result.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Return all cache manager addresses.
fn handle_all_cache_managers(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    load_arbos(input)?;

    let cm_key = cache_managers_key();
    let size_slot = map_slot(cm_key.as_slice(), 0);
    let size = sload_field(input, size_slot)?.saturating_to::<u64>();
    let mut sloads: u64 = 1;

    // Cap to prevent excessive reads.
    let count = size.min(256);

    // ABI: offset to dynamic array, then length, then elements.
    let mut out = Vec::with_capacity(64 + count as usize * 32);
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(count).to_be_bytes::<32>());

    for i in 1..=count {
        let addr_slot = map_slot(cm_key.as_slice(), i);
        let addr_value = sload_field(input, addr_slot)?;
        out.extend_from_slice(&addr_value.to_be_bytes::<32>());
        sloads += 1;
    }

    Ok(PrecompileOutput::new(
        (sloads * SLOAD_GAS + COPY_GAS).min(input.gas),
        out.into(),
    ))
}

/// Check if a program codehash is cached.
fn handle_codehash_is_cached(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("calldata too short for bytes32 arg"));
    }
    let mut bytes = [0u8; 32];
    bytes.copy_from_slice(&data[4..36]);
    let codehash = B256::from(bytes);

    load_arbos(input)?;

    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let data_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_DATA_KEY);
    let program_slot = map_slot_b256(data_key.as_slice(), &codehash);
    let program_word = sload_field(input, program_slot)?;

    // Byte 14 of the program word is the cached flag.
    let word_bytes = program_word.to_be_bytes::<32>();
    let is_cached = word_bytes[14] != 0;

    let result = if is_cached {
        U256::from(1u64)
    } else {
        U256::ZERO
    };
    Ok(PrecompileOutput::new(
        SLOAD_GAS + COPY_GAS,
        result.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn address_to_b256(addr: Address) -> B256 {
    let mut bytes = [0u8; 32];
    bytes[12..32].copy_from_slice(addr.as_slice());
    B256::from(bytes)
}
