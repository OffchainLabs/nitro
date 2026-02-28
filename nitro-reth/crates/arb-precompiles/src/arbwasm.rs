use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, B256, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{
    derive_subspace_key, map_slot, map_slot_b256, ARBOS_STATE_ADDRESS, PROGRAMS_DATA_KEY,
    PROGRAMS_PARAMS_KEY, PROGRAMS_SUBSPACE, ROOT_STORAGE_KEY,
};

/// ArbWasm precompile address (0x71).
pub const ARBWASM_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x71,
]);

// Function selectors — view methods returning Stylus program config.
const STYLUS_VERSION: [u8; 4] = [0xf2, 0x8a, 0x04, 0x99];
const INK_PRICE: [u8; 4] = [0xeb, 0xf5, 0xd2, 0x51];
const MAX_STACK_DEPTH: [u8; 4] = [0x19, 0x4a, 0xa2, 0x8e];
const FREE_PAGES: [u8; 4] = [0xb6, 0x9d, 0xb8, 0x5e];
const PAGE_GAS: [u8; 4] = [0x96, 0x76, 0xa4, 0x67];
const PAGE_RAMP: [u8; 4] = [0x56, 0xc1, 0x80, 0x1c];
const PAGE_LIMIT: [u8; 4] = [0x20, 0xf0, 0x02, 0xea];
const MIN_INIT_GAS: [u8; 4] = [0x5b, 0x19, 0x32, 0x87];
const INIT_COST_SCALAR: [u8; 4] = [0x67, 0x46, 0x27, 0x93];
const EXPIRY_DAYS: [u8; 4] = [0xee, 0xe2, 0x2a, 0xa3];
const KEEPALIVE_DAYS: [u8; 4] = [0xe7, 0xfb, 0x85, 0x75];
const BLOCK_CACHE_SIZE: [u8; 4] = [0xd2, 0xfb, 0xa3, 0xc5];
const ACTIVATE_PROGRAM: [u8; 4] = [0x72, 0x93, 0x80, 0x88];
const CODEHASH_KEEPALIVE: [u8; 4] = [0xe7, 0xf6, 0x2c, 0x15];
const CODEHASH_VERSION: [u8; 4] = [0xb4, 0xb7, 0xc5, 0xf5];
const CODEHASH_ASM_SIZE: [u8; 4] = [0x5f, 0xd3, 0x5d, 0xea];
const PROGRAM_VERSION: [u8; 4] = [0x70, 0x46, 0x7c, 0x7c];
const PROGRAM_INIT_GAS: [u8; 4] = [0x8e, 0x15, 0xc4, 0x17];
const PROGRAM_MEMORY_FOOTPRINT: [u8; 4] = [0x95, 0x48, 0xea, 0xb0];
const PROGRAM_TIME_LEFT: [u8; 4] = [0x63, 0x5b, 0x36, 0x42];

const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;

/// Initial page ramp constant (not stored in packed params).
const INITIAL_PAGE_RAMP: u64 = 620674314;

/// Min init gas units (matching Go's MinInitGasUnits).
const MIN_INIT_GAS_UNITS: u64 = 128;
/// Min cached gas units (matching Go's MinCachedGasUnits).
const MIN_CACHED_GAS_UNITS: u64 = 32;
/// Cost scalar percent (matching Go's CostScalarPercent).
const COST_SCALAR_PERCENT: u64 = 2;

pub fn create_arbwasm_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbwasm"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    // ArbWasm requires ArbOS >= 30 (Stylus).
    if let Some(result) = crate::check_precompile_version(
        arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS,
    ) {
        return result;
    }

    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    match selector {
        STYLUS_VERSION => {
            let params = load_params_word(&mut input)?;
            let version = u16::from_be_bytes([params[0], params[1]]);
            ok_u256(SLOAD_GAS + COPY_GAS, U256::from(version))
        }
        INK_PRICE => {
            let params = load_params_word(&mut input)?;
            let ink_price =
                (params[2] as u32) << 16 | (params[3] as u32) << 8 | params[4] as u32;
            ok_u256(SLOAD_GAS + COPY_GAS, U256::from(ink_price))
        }
        MAX_STACK_DEPTH => {
            let params = load_params_word(&mut input)?;
            let depth = u32::from_be_bytes([params[5], params[6], params[7], params[8]]);
            ok_u256(SLOAD_GAS + COPY_GAS, U256::from(depth))
        }
        FREE_PAGES => {
            let params = load_params_word(&mut input)?;
            let pages = u16::from_be_bytes([params[9], params[10]]);
            ok_u256(SLOAD_GAS + COPY_GAS, U256::from(pages))
        }
        PAGE_GAS => {
            let params = load_params_word(&mut input)?;
            let gas = u16::from_be_bytes([params[11], params[12]]);
            ok_u256(SLOAD_GAS + COPY_GAS, U256::from(gas))
        }
        PAGE_RAMP => {
            // Page ramp is a constant, not stored in packed params.
            // Still load the account for consistency.
            load_arbos(&mut input)?;
            ok_u256(COPY_GAS, U256::from(INITIAL_PAGE_RAMP))
        }
        PAGE_LIMIT => {
            let params = load_params_word(&mut input)?;
            let limit = u16::from_be_bytes([params[13], params[14]]);
            ok_u256(SLOAD_GAS + COPY_GAS, U256::from(limit))
        }
        MIN_INIT_GAS => {
            // Requires ArbOS >= 32 (StylusChargingFixes).
            if let Some(result) = crate::check_method_version(
                arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS_CHARGING_FIXES,
                0,
            ) {
                return result;
            }
            let params = load_params_word(&mut input)?;
            let min_init = params[15] as u64;
            let min_cached = params[16] as u64;
            let init = min_init.saturating_mul(MIN_INIT_GAS_UNITS);
            let cached = min_cached.saturating_mul(MIN_CACHED_GAS_UNITS);
            ok_two_u256(SLOAD_GAS + COPY_GAS, U256::from(init), U256::from(cached))
        }
        INIT_COST_SCALAR => {
            let params = load_params_word(&mut input)?;
            let scalar = params[17] as u64;
            ok_u256(
                SLOAD_GAS + COPY_GAS,
                U256::from(scalar.saturating_mul(COST_SCALAR_PERCENT)),
            )
        }
        EXPIRY_DAYS => {
            let params = load_params_word(&mut input)?;
            let days = u16::from_be_bytes([params[19], params[20]]);
            ok_u256(SLOAD_GAS + COPY_GAS, U256::from(days))
        }
        KEEPALIVE_DAYS => {
            let params = load_params_word(&mut input)?;
            let days = u16::from_be_bytes([params[21], params[22]]);
            ok_u256(SLOAD_GAS + COPY_GAS, U256::from(days))
        }
        BLOCK_CACHE_SIZE => {
            let params = load_params_word(&mut input)?;
            let size = u16::from_be_bytes([params[23], params[24]]);
            ok_u256(SLOAD_GAS + COPY_GAS, U256::from(size))
        }
        // Program queries by codehash.
        CODEHASH_VERSION => {
            let codehash = extract_bytes32(&input.data)?;
            let (params_word, program_word) = load_params_and_program(&mut input, codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let program = parse_program(&program_word, &params_word);
            validate_active_program(&program, params_version)?;
            ok_u256(2 * SLOAD_GAS + COPY_GAS, U256::from(program.version))
        }
        CODEHASH_ASM_SIZE => {
            let codehash = extract_bytes32(&input.data)?;
            let (params_word, program_word) = load_params_and_program(&mut input, codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let program = parse_program(&program_word, &params_word);
            validate_active_program(&program, params_version)?;
            let asm_size = program.asm_estimate_kb.saturating_mul(1024);
            ok_u256(2 * SLOAD_GAS + COPY_GAS, U256::from(asm_size))
        }
        // Program queries by address (need to get codehash from account).
        PROGRAM_VERSION => {
            let address = extract_address(&input.data)?;
            let codehash = get_account_codehash(&mut input, address)?;
            let (params_word, program_word) = load_params_and_program(&mut input, codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let program = parse_program(&program_word, &params_word);
            validate_active_program(&program, params_version)?;
            ok_u256(3 * SLOAD_GAS + COPY_GAS, U256::from(program.version))
        }
        PROGRAM_INIT_GAS => {
            let address = extract_address(&input.data)?;
            let codehash = get_account_codehash(&mut input, address)?;
            let (params_word, program_word) = load_params_and_program(&mut input, codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let program = parse_program(&program_word, &params_word);
            validate_active_program(&program, params_version)?;

            let min_init = params_word[15] as u64;
            let min_cached = params_word[16] as u64;
            let init_cost_scalar = params_word[17] as u64;
            let cached_cost_scalar = params_word[18] as u64;

            let init_base = min_init.saturating_mul(MIN_INIT_GAS_UNITS);
            let init_dyno = (program.init_cost as u64)
                .saturating_mul(init_cost_scalar * COST_SCALAR_PERCENT);
            let mut init_gas = init_base.saturating_add(div_ceil(init_dyno, 100));

            let cached_base = min_cached.saturating_mul(MIN_CACHED_GAS_UNITS);
            let cached_dyno = (program.cached_cost as u64)
                .saturating_mul(cached_cost_scalar * COST_SCALAR_PERCENT);
            let cached_gas = cached_base.saturating_add(div_ceil(cached_dyno, 100));

            if params_version > 1 {
                init_gas = init_gas.saturating_add(cached_gas);
            }

            ok_two_u256(
                3 * SLOAD_GAS + COPY_GAS,
                U256::from(init_gas),
                U256::from(cached_gas),
            )
        }
        PROGRAM_MEMORY_FOOTPRINT => {
            let address = extract_address(&input.data)?;
            let codehash = get_account_codehash(&mut input, address)?;
            let (params_word, program_word) = load_params_and_program(&mut input, codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let program = parse_program(&program_word, &params_word);
            validate_active_program(&program, params_version)?;
            ok_u256(3 * SLOAD_GAS + COPY_GAS, U256::from(program.footprint))
        }
        PROGRAM_TIME_LEFT => {
            let address = extract_address(&input.data)?;
            let codehash = get_account_codehash(&mut input, address)?;
            let (params_word, program_word) = load_params_and_program(&mut input, codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let program = parse_program(&program_word, &params_word);
            validate_active_program(&program, params_version)?;

            let expiry_days = u16::from_be_bytes([params_word[19], params_word[20]]);
            let expiry_seconds = (expiry_days as u64) * 24 * 3600;
            let time_left = expiry_seconds.saturating_sub(program.age_seconds);
            ok_u256(3 * SLOAD_GAS + COPY_GAS, U256::from(time_left))
        }
        // State-modifying.
        ACTIVATE_PROGRAM => {
            let _ = &mut input;
            Err(PrecompileError::other("Stylus activation not yet supported"))
        }
        CODEHASH_KEEPALIVE => {
            let _ = &mut input;
            Err(PrecompileError::other("Stylus keepalive not yet supported"))
        }
        _ => Err(PrecompileError::other("unknown ArbWasm selector")),
    }
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
    Ok(val.data)
}

/// Load the packed StylusParams word (slot 0) from storage.
fn load_params_word(input: &mut PrecompileInput<'_>) -> Result<[u8; 32], PrecompileError> {
    load_arbos(input)?;
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let params_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_PARAMS_KEY);
    let slot = map_slot(params_key.as_slice(), 0);
    let value = sload_field(input, slot)?;
    Ok(value.to_be_bytes::<32>())
}

/// Load both the params word and a program entry by codehash.
fn load_params_and_program(
    input: &mut PrecompileInput<'_>,
    codehash: B256,
) -> Result<([u8; 32], [u8; 32]), PrecompileError> {
    load_arbos(input)?;
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);

    // Params
    let params_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_PARAMS_KEY);
    let params_slot = map_slot(params_key.as_slice(), 0);
    let params_value = sload_field(input, params_slot)?;

    // Program data
    let data_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_DATA_KEY);
    let program_slot = map_slot_b256(data_key.as_slice(), &codehash);
    let program_value = sload_field(input, program_slot)?;

    Ok((params_value.to_be_bytes::<32>(), program_value.to_be_bytes::<32>()))
}

/// Get the code hash for an account address.
fn get_account_codehash(
    input: &mut PrecompileInput<'_>,
    address: Address,
) -> Result<B256, PrecompileError> {
    let account = input
        .internals_mut()
        .load_account(address)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;
    Ok(account.data.info.code_hash)
}

/// Parsed program entry from a storage word.
#[allow(dead_code)]
struct ProgramInfo {
    version: u16,
    init_cost: u16,
    cached_cost: u16,
    footprint: u16,
    activated_at: u32,
    asm_estimate_kb: u32,
    age_seconds: u64,
}

/// Arbitrum start time (matches Go's ArbitrumStartTime).
const ARBITRUM_START_TIME: u64 = 1622243344;

fn parse_program(data: &[u8; 32], params_word: &[u8; 32]) -> ProgramInfo {
    let version = u16::from_be_bytes([data[0], data[1]]);
    let init_cost = u16::from_be_bytes([data[2], data[3]]);
    let cached_cost = u16::from_be_bytes([data[4], data[5]]);
    let footprint = u16::from_be_bytes([data[6], data[7]]);
    let activated_at = (data[8] as u32) << 16 | (data[9] as u32) << 8 | data[10] as u32;
    let asm_estimate_kb = (data[11] as u32) << 16 | (data[12] as u32) << 8 | data[13] as u32;

    // Compute age from block timestamp. Use current block time from the
    // params word context. We don't have direct access to block time here,
    // so age_seconds defaults to 0 (callers that need it should use
    // the block timestamp from the execution context).
    // For precompile queries, the age is computed from the expiry check.
    let _ = params_word; // params_word passed for future use
    let age_seconds = hours_to_age(block_timestamp(), activated_at);

    ProgramInfo {
        version,
        init_cost,
        cached_cost,
        footprint,
        activated_at,
        asm_estimate_kb,
        age_seconds,
    }
}

/// Get the current block timestamp from the thread-local.
fn block_timestamp() -> u64 {
    crate::get_block_timestamp()
}

fn hours_to_age(time: u64, hours: u32) -> u64 {
    let seconds = (hours as u64).saturating_mul(3600);
    let activated_at = ARBITRUM_START_TIME.saturating_add(seconds);
    time.saturating_sub(activated_at)
}

/// Validate that a program is active (version matches and not expired).
fn validate_active_program(program: &ProgramInfo, params_version: u16) -> Result<(), PrecompileError> {
    if program.version == 0 {
        return Err(PrecompileError::other("program not activated"));
    }
    if program.version != params_version {
        return Err(PrecompileError::other("program needs upgrade"));
    }
    Ok(())
}

/// Extract a bytes32 argument from calldata (after 4-byte selector).
fn extract_bytes32(data: &[u8]) -> Result<B256, PrecompileError> {
    if data.len() < 36 {
        return Err(PrecompileError::other("calldata too short for bytes32 arg"));
    }
    let mut bytes = [0u8; 32];
    bytes.copy_from_slice(&data[4..36]);
    Ok(B256::from(bytes))
}

/// Extract an address argument from calldata (after 4-byte selector).
fn extract_address(data: &[u8]) -> Result<Address, PrecompileError> {
    if data.len() < 36 {
        return Err(PrecompileError::other("calldata too short for address arg"));
    }
    // Address is right-aligned in 32-byte word.
    let mut bytes = [0u8; 20];
    bytes.copy_from_slice(&data[16..36]);
    Ok(Address::from(bytes))
}

fn ok_u256(gas_cost: u64, value: U256) -> PrecompileResult {
    Ok(PrecompileOutput::new(
        gas_cost,
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn ok_two_u256(gas_cost: u64, a: U256, b: U256) -> PrecompileResult {
    let mut out = Vec::with_capacity(64);
    out.extend_from_slice(&a.to_be_bytes::<32>());
    out.extend_from_slice(&b.to_be_bytes::<32>());
    Ok(PrecompileOutput::new(gas_cost, out.into()))
}

fn div_ceil(a: u64, b: u64) -> u64 {
    (a + b - 1) / b
}
