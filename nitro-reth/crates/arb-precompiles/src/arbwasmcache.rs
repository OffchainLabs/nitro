use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{keccak256, Address, Log, B256, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{
    derive_subspace_key, map_slot, map_slot_b256, ARBOS_STATE_ADDRESS, CACHE_MANAGERS_KEY,
    CHAIN_OWNER_SUBSPACE, PROGRAMS_DATA_KEY, PROGRAMS_PARAMS_KEY, PROGRAMS_SUBSPACE,
    ROOT_STORAGE_KEY,
};

/// keccak256("UpdateProgramCache(address,bytes32,bool)")
fn update_program_cache_topic() -> B256 {
    keccak256("UpdateProgramCache(address,bytes32,bool)")
}

const ARBITRUM_START_TIME: u64 = 1_421_388_000;

fn hours_to_age(time: u64, hours_since_start: u32) -> u64 {
    let activated_at = ARBITRUM_START_TIME.saturating_add((hours_since_start as u64) * 3600);
    time.saturating_sub(activated_at)
}

/// ArbWasmCache precompile address (0x72).
pub const ARBWASMCACHE_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x72,
]);

// Function selectors (keccak256 of Solidity signatures).
const IS_CACHE_MANAGER: [u8; 4] = [0x85, 0xe2, 0xde, 0x85]; // isCacheManager(address)
const ALL_CACHE_MANAGERS: [u8; 4] = [0x0e, 0xc1, 0xd7, 0x73]; // allCacheManagers()
const CACHE_CODEHASH: [u8; 4] = [0x4c, 0xea, 0xc8, 0x17]; // cacheCodehash(bytes32)
const CACHE_PROGRAM: [u8; 4] = [0xe7, 0x3a, 0xc9, 0xf2]; // cacheProgram(address)
const EVICT_CODEHASH: [u8; 4] = [0xce, 0x97, 0x20, 0x13]; // evictCodehash(bytes32)
const CODEHASH_IS_CACHED: [u8; 4] = [0xa7, 0x2f, 0x17, 0x9b]; // codehashIsCached(bytes32)

const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;

const WARM_SLOAD_GAS: u64 = 100;
const COLD_ACCOUNT_ACCESS_GAS: u64 = 2600;
const SSTORE_SET_GAS: u64 = 20_000;
const SSTORE_RESET_GAS: u64 = 5_000;

/// LOG3 for UpdateProgramCache(address,bytes32,bool):
/// base 375 + 3 topics * 375 + 32 bytes data * 8.
const EMIT_UPDATE_PROGRAM_CACHE_GAS: u64 = 375 + 3 * 375 + 32 * 8;

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
        CACHE_CODEHASH => handle_cache_codehash(&mut input),
        CACHE_PROGRAM => handle_cache_program(&mut input),
        EVICT_CODEHASH => handle_evict_codehash(&mut input),
        IS_CACHE_MANAGER => handle_is_cache_manager(&mut input),
        ALL_CACHE_MANAGERS => handle_all_cache_managers(&mut input),
        CODEHASH_IS_CACHED => handle_codehash_is_cached(&mut input),
        _ => return crate::burn_all_revert(input.gas),
    };
    crate::gas_check(input.gas, result)
}

fn words_for_bytes(n: u64) -> u64 {
    n.div_ceil(32)
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
    // Gas: OpenArbosState(800) + sload(800) + argsCost(3) + resultCost(3)
    let args_cost = COPY_GAS * words_for_bytes(data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    Ok(PrecompileOutput::new(
        SLOAD_GAS + SLOAD_GAS + args_cost + result_cost,
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

    // Gas: OpenArbosState(800) + sloads * SLOAD(800) + argsCost + resultCost
    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(out.len() as u64);
    let total = SLOAD_GAS + sloads * SLOAD_GAS + args_cost + result_cost;
    Ok(PrecompileOutput::new(total.min(input.gas), out.into()))
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
    // Gas: OpenArbosState(800) + sload(800) + argsCost + resultCost
    let args_cost = COPY_GAS * words_for_bytes(data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    Ok(PrecompileOutput::new(
        SLOAD_GAS + SLOAD_GAS + args_cost + result_cost,
        result.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn address_to_b256(addr: Address) -> B256 {
    let mut bytes = [0u8; 32];
    bytes[12..32].copy_from_slice(addr.as_slice());
    B256::from(bytes)
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

/// Read `version` (bytes 0-1) and `expiry_days` (bytes 19-20) from slot 0
/// of the Programs.Params storage word.
fn read_program_params(input: &mut PrecompileInput<'_>) -> Result<(u16, u16), PrecompileError> {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let params_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_PARAMS_KEY);
    let slot = map_slot(params_key.as_slice(), 0);
    let word = sload_field(input, slot)?.to_be_bytes::<32>();
    let version = u16::from_be_bytes([word[0], word[1]]);
    let expiry_days = u16::from_be_bytes([word[19], word[20]]);
    Ok((version, expiry_days))
}

fn program_data_slot(codehash: B256) -> U256 {
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let data_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_DATA_KEY);
    map_slot_b256(data_key.as_slice(), &codehash)
}

/// Caller must be a cache manager OR chain owner. Returns `(has_access, gas)`:
/// `gas` is 1 SLOAD if the caller is a cache manager (short-circuit), else
/// 2 SLOADs (cache-managers probe then chain-owners probe).
fn caller_has_cache_access(
    input: &mut PrecompileInput<'_>,
    caller: Address,
) -> Result<(bool, u64), PrecompileError> {
    let cm_key = cache_managers_key();
    let cm_by_addr = derive_subspace_key(cm_key.as_slice(), BY_ADDRESS_KEY);
    let addr_hash = address_to_b256(caller);
    let cm_slot = map_slot_b256(cm_by_addr.as_slice(), &addr_hash);
    if sload_field(input, cm_slot)? != U256::ZERO {
        return Ok((true, SLOAD_GAS));
    }

    let owner_key = derive_subspace_key(ROOT_STORAGE_KEY, CHAIN_OWNER_SUBSPACE);
    let owner_by_addr = derive_subspace_key(owner_key.as_slice(), BY_ADDRESS_KEY);
    let owner_slot = map_slot_b256(owner_by_addr.as_slice(), &addr_hash);
    let is_owner = sload_field(input, owner_slot)? != U256::ZERO;
    Ok((is_owner, 2 * SLOAD_GAS))
}

/// `pre_set_gas` lets the caller include an extra charge that must be paid on
/// every exit path (e.g., the GetCodeHash access cost for `cacheProgram`).
fn set_program_cached(
    input: &mut PrecompileInput<'_>,
    codehash: B256,
    cache: bool,
    pre_set_gas: u64,
) -> PrecompileResult {
    let data_len = input.data.len();
    let caller = input.caller;
    let now: u64 = input
        .internals()
        .block_timestamp()
        .try_into()
        .unwrap_or(0u64);

    let args_cost = COPY_GAS * words_for_bytes(data_len.saturating_sub(4) as u64);
    let boilerplate_gas = args_cost + SLOAD_GAS + pre_set_gas;

    load_arbos(input)?;

    let (has_access, access_gas) = caller_has_cache_access(input, caller)?;
    if !has_access {
        return crate::burn_all_revert(input.gas);
    }

    let (params_version, expiry_days) = read_program_params(input)?;

    let prog_slot = program_data_slot(codehash);
    let mut prog_word = sload_field(input, prog_slot)?.to_be_bytes::<32>();
    let prog_version = u16::from_be_bytes([prog_word[0], prog_word[1]]);
    let prog_init_cost = u16::from_be_bytes([prog_word[2], prog_word[3]]);
    let activated_at_hours =
        ((prog_word[8] as u32) << 16) | ((prog_word[9] as u32) << 8) | prog_word[10] as u32;
    let age_seconds = hours_to_age(now, activated_at_hours);
    let expiry_seconds = (expiry_days as u64).saturating_mul(86_400);
    let expired = age_seconds > expiry_seconds;
    let already_cached = prog_word[14] != 0;

    // Matches the early-return point before any mutation.
    let after_get_program_gas = boilerplate_gas + access_gas + WARM_SLOAD_GAS + SLOAD_GAS;

    if cache && prog_version != params_version {
        let mut args = Vec::with_capacity(64);
        args.extend_from_slice(&crate::abi_word_u16(prog_version));
        args.extend_from_slice(&crate::abi_word_u16(params_version));
        return crate::sol_error_revert_with_args(
            program_needs_upgrade_selector(),
            &args,
            input.gas,
        );
    }
    if cache && expired {
        let args = crate::abi_word_u64(age_seconds);
        return crate::sol_error_revert_with_args(program_expired_selector(), &args, input.gas);
    }
    if already_cached == cache {
        return Ok(PrecompileOutput::new(
            after_get_program_gas.min(input.gas),
            Vec::new().into(),
        ));
    }

    prog_word[14] = if cache { 1 } else { 0 };
    let new_word = U256::from_be_bytes(prog_word);
    sstore_field(input, prog_slot, new_word)?;
    let sstore_gas = if new_word == U256::ZERO {
        SSTORE_RESET_GAS
    } else {
        SSTORE_SET_GAS
    };

    let topic1 = address_to_b256(caller);
    let mut event_data = Vec::with_capacity(32);
    event_data.extend_from_slice(&U256::from(cache as u64).to_be_bytes::<32>());
    input.internals_mut().log(Log::new_unchecked(
        ARBWASMCACHE_ADDRESS,
        vec![update_program_cache_topic(), topic1, codehash],
        event_data.into(),
    ));

    let gas_used = after_get_program_gas
        + EMIT_UPDATE_PROGRAM_CACHE_GAS
        + prog_init_cost as u64
        + SLOAD_GAS
        + sstore_gas;
    Ok(PrecompileOutput::new(
        gas_used.min(input.gas),
        Vec::new().into(),
    ))
}

fn program_needs_upgrade_selector() -> [u8; 4] {
    let h = keccak256(b"ProgramNeedsUpgrade(uint16,uint16)");
    [h[0], h[1], h[2], h[3]]
}

fn program_expired_selector() -> [u8; 4] {
    let h = keccak256(b"ProgramExpired(uint64)");
    [h[0], h[1], h[2], h[3]]
}

/// Deprecated: cacheCodehash(bytes32). Available before CacheProgram replaced it.
fn handle_cache_codehash(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("calldata too short for bytes32 arg"));
    }
    let mut bytes = [0u8; 32];
    bytes.copy_from_slice(&data[4..36]);
    let codehash = B256::from(bytes);
    set_program_cached(input, codehash, true, 0)
}

/// cacheProgram(address) reads the code hash from an account, which costs
/// ColdAccountAccessCostEIP2929 even when the slot is already warm.
fn handle_cache_program(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("calldata too short for address arg"));
    }
    let addr = Address::from_slice(&data[16..36]);
    let codehash = {
        let acct = input
            .internals_mut()
            .load_account(addr)
            .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;
        acct.data.info.code_hash
    };
    set_program_cached(input, codehash, true, COLD_ACCOUNT_ACCESS_GAS)
}

/// evictCodehash(bytes32): clears the cached flag.
fn handle_evict_codehash(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("calldata too short for bytes32 arg"));
    }
    let mut bytes = [0u8; 32];
    bytes.copy_from_slice(&data[4..36]);
    let codehash = B256::from(bytes);
    set_program_cached(input, codehash, false, 0)
}
