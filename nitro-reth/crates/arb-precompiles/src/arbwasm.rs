use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, B256, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{
    derive_subspace_key, map_slot, map_slot_b256, ARBOS_STATE_ADDRESS, PROGRAMS_DATA_KEY,
    PROGRAMS_PARAMS_KEY, PROGRAMS_SUBSPACE, ROOT_STORAGE_KEY,
};

/// ArbWasm precompile address (0x71).
pub const ARBWASM_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x71,
]);

// Function selectors (keccak256 of Solidity signatures).
const STYLUS_VERSION: [u8; 4] = [0xa9, 0x96, 0xe0, 0xc2]; // stylusVersion()
const INK_PRICE: [u8; 4] = [0xd1, 0xc1, 0x7a, 0xbc]; // inkPrice()
const MAX_STACK_DEPTH: [u8; 4] = [0x8c, 0xcf, 0xaa, 0x70]; // maxStackDepth()
const FREE_PAGES: [u8; 4] = [0x44, 0x90, 0xc1, 0x9d]; // freePages()
const PAGE_GAS: [u8; 4] = [0x7a, 0xf4, 0xba, 0x49]; // pageGas()
const PAGE_RAMP: [u8; 4] = [0x11, 0xc8, 0x2a, 0xe8]; // pageRamp()
const PAGE_LIMIT: [u8; 4] = [0x97, 0x86, 0xf9, 0x6e]; // pageLimit()
const MIN_INIT_GAS: [u8; 4] = [0x99, 0xd0, 0xb3, 0x8d]; // minInitGas()
const INIT_COST_SCALAR: [u8; 4] = [0x5f, 0xc9, 0x4c, 0x0b]; // initCostScalar()
const EXPIRY_DAYS: [u8; 4] = [0x30, 0x9f, 0x65, 0x55]; // expiryDays()
const KEEPALIVE_DAYS: [u8; 4] = [0x0a, 0x93, 0x64, 0x55]; // keepaliveDays()
const BLOCK_CACHE_SIZE: [u8; 4] = [0x7a, 0xf6, 0xe8, 0x19]; // blockCacheSize()
const ACTIVATE_PROGRAM: [u8; 4] = [0x58, 0xc7, 0x80, 0xc2]; // activateProgram(address)
const CODEHASH_KEEPALIVE: [u8; 4] = [0xc6, 0x89, 0xba, 0xd5]; // codehashKeepalive(bytes32)
const CODEHASH_VERSION: [u8; 4] = [0xd7, 0x0c, 0x0c, 0xa7]; // codehashVersion(bytes32)
const CODEHASH_ASM_SIZE: [u8; 4] = [0x40, 0x89, 0x26, 0x7f]; // codehashAsmSize(bytes32)
const PROGRAM_VERSION: [u8; 4] = [0xcc, 0x8f, 0x4e, 0x88]; // programVersion(address)
const PROGRAM_INIT_GAS: [u8; 4] = [0x62, 0xb6, 0x88, 0xaa]; // programInitGas(address)
const PROGRAM_MEMORY_FOOTPRINT: [u8; 4] = [0xae, 0xf3, 0x6b, 0xe3]; // programMemoryFootprint(address)
const PROGRAM_TIME_LEFT: [u8; 4] = [0xc7, 0x75, 0xa6, 0x2a]; // programTimeLeft(address)

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const SSTORE_ZERO_GAS: u64 = 5_000;
const COPY_GAS: u64 = 3;

/// Initial page ramp constant (not stored in packed params).
const INITIAL_PAGE_RAMP: u64 = 620674314;

const MIN_INIT_GAS_UNITS: u64 = 128;
const MIN_CACHED_GAS_UNITS: u64 = 32;
const COST_SCALAR_PERCENT: u64 = 2;

fn program_not_activated_selector() -> [u8; 4] {
    let hash = alloy_primitives::keccak256(b"ProgramNotActivated()");
    [hash[0], hash[1], hash[2], hash[3]]
}

fn program_up_to_date_selector() -> [u8; 4] {
    let hash = alloy_primitives::keccak256(b"ProgramUpToDate()");
    [hash[0], hash[1], hash[2], hash[3]]
}

pub fn create_arbwasm_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbwasm"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    // ArbWasm requires ArbOS >= 30 (Stylus).
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

    // State-modifying methods handle their own gas to match Nitro's framework.
    match selector {
        ACTIVATE_PROGRAM => return handle_activate_program(input),
        CODEHASH_KEEPALIVE => return handle_codehash_keepalive(input),
        _ => {}
    }

    crate::init_precompile_gas(data.len());

    let result = match selector {
        STYLUS_VERSION => {
            let params = load_params_word(&mut input)?;
            let version = u16::from_be_bytes([params[0], params[1]]);
            ok_u256(SLOAD_GAS + COPY_GAS, U256::from(version))
        }
        INK_PRICE => {
            let params = load_params_word(&mut input)?;
            let ink_price = (params[2] as u32) << 16 | (params[3] as u32) << 8 | params[4] as u32;
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
            let codehash = extract_bytes32(input.data)?;
            let (params_word, program_word) = load_params_and_program(&mut input, codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let program = parse_program(&program_word, &params_word);
            if let Err(r) = validate_active_program(&program, params_version, input.gas) {
                return r;
            }
            ok_u256(2 * SLOAD_GAS + COPY_GAS, U256::from(program.version))
        }
        CODEHASH_ASM_SIZE => {
            let codehash = extract_bytes32(input.data)?;
            let (params_word, program_word) = load_params_and_program(&mut input, codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let program = parse_program(&program_word, &params_word);
            if let Err(r) = validate_active_program(&program, params_version, input.gas) {
                return r;
            }
            let asm_size = program.asm_estimate_kb.saturating_mul(1024);
            ok_u256(2 * SLOAD_GAS + COPY_GAS, U256::from(asm_size))
        }
        // Program queries by address (need to get codehash from account).
        PROGRAM_VERSION => {
            let address = extract_address(input.data)?;
            let codehash = get_account_codehash(&mut input, address)?;
            let (params_word, program_word) = load_params_and_program(&mut input, codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let program = parse_program(&program_word, &params_word);
            if let Err(r) = validate_active_program(&program, params_version, input.gas) {
                return r;
            }
            ok_u256(3 * SLOAD_GAS + COPY_GAS, U256::from(program.version))
        }
        PROGRAM_INIT_GAS => {
            let address = extract_address(input.data)?;
            let codehash = get_account_codehash(&mut input, address)?;
            let (params_word, program_word) = load_params_and_program(&mut input, codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let program = parse_program(&program_word, &params_word);
            if let Err(r) = validate_active_program(&program, params_version, input.gas) {
                return r;
            }

            let min_init = params_word[15] as u64;
            let min_cached = params_word[16] as u64;
            let init_cost_scalar = params_word[17] as u64;
            let cached_cost_scalar = params_word[18] as u64;

            let init_base = min_init.saturating_mul(MIN_INIT_GAS_UNITS);
            let init_dyno =
                (program.init_cost as u64).saturating_mul(init_cost_scalar * COST_SCALAR_PERCENT);
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
            let address = extract_address(input.data)?;
            let codehash = get_account_codehash(&mut input, address)?;
            let (params_word, program_word) = load_params_and_program(&mut input, codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let program = parse_program(&program_word, &params_word);
            if let Err(r) = validate_active_program(&program, params_version, input.gas) {
                return r;
            }
            ok_u256(3 * SLOAD_GAS + COPY_GAS, U256::from(program.footprint))
        }
        PROGRAM_TIME_LEFT => {
            let address = extract_address(input.data)?;
            let codehash = get_account_codehash(&mut input, address)?;
            let (params_word, program_word) = load_params_and_program(&mut input, codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let program = parse_program(&program_word, &params_word);
            if let Err(r) = validate_active_program(&program, params_version, input.gas) {
                return r;
            }

            let expiry_days = u16::from_be_bytes([params_word[19], params_word[20]]);
            let expiry_seconds = (expiry_days as u64) * 24 * 3600;
            let time_left = expiry_seconds.saturating_sub(program.age_seconds);
            ok_u256(3 * SLOAD_GAS + COPY_GAS, U256::from(time_left))
        }
        ACTIVATE_PROGRAM | CODEHASH_KEEPALIVE => unreachable!(),
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

/// Load the packed StylusParams word (slot 0) from storage.
fn load_params_word(input: &mut PrecompileInput<'_>) -> Result<[u8; 32], PrecompileError> {
    load_arbos(input)?;
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let params_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_PARAMS_KEY);
    let slot = map_slot(params_key.as_slice(), 0);
    let value = sload_field(input, slot)?;
    Ok(value.to_be_bytes::<32>())
}

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

    let data_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_DATA_KEY);
    let program_slot = map_slot_b256(data_key.as_slice(), &codehash);
    let program_value = sload_field(input, program_slot)?;

    Ok((
        params_value.to_be_bytes::<32>(),
        program_value.to_be_bytes::<32>(),
    ))
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
struct ProgramInfo {
    version: u16,
    init_cost: u16,
    cached_cost: u16,
    footprint: u16,
    asm_estimate_kb: u32,
    age_seconds: u64,
}

/// Arbitrum start time (epoch for encoding hours in program data).
/// Matches Nitro's ArbitrumStartTime constant in data_pricer.go.
const ARBITRUM_START_TIME: u64 = 1421388000;

fn parse_program(data: &[u8; 32], params_word: &[u8; 32]) -> ProgramInfo {
    let version = u16::from_be_bytes([data[0], data[1]]);
    let init_cost = u16::from_be_bytes([data[2], data[3]]);
    let cached_cost = u16::from_be_bytes([data[4], data[5]]);
    let footprint = u16::from_be_bytes([data[6], data[7]]);
    let activated_at = (data[8] as u32) << 16 | (data[9] as u32) << 8 | data[10] as u32;
    let asm_estimate_kb = (data[11] as u32) << 16 | (data[12] as u32) << 8 | data[13] as u32;

    let _ = params_word;
    let age_seconds = hours_to_age(block_timestamp(), activated_at);

    ProgramInfo {
        version,
        init_cost,
        cached_cost,
        footprint,
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
fn validate_active_program(
    program: &ProgramInfo,
    params_version: u16,
    gas_limit: u64,
) -> Result<(), PrecompileResult> {
    if program.version == 0 {
        return Err(crate::sol_error_revert(program_not_activated_selector(), gas_limit));
    }
    if program.version != params_version {
        return Err(crate::sol_error_revert(program_up_to_date_selector(), gas_limit));
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
    a.div_ceil(b)
}

fn hours_since_arbitrum(time: u64) -> u32 {
    let elapsed = time.saturating_sub(ARBITRUM_START_TIME);
    (elapsed / 3600) as u32
}

/// Approximates b * e^(x/b) where b = 10000 (basis points), using Horner's
/// method with accuracy=12. Matches Nitro's `arbmath.ApproxExpBasisPoints(x, 12)`.
fn approx_exp_basis_points(x: u64) -> u64 {
    let b = 10_000u64;
    let accuracy = 12u64;
    let mut res = b + x / accuracy;
    for i in (1..accuracy).rev() {
        res = b + res.saturating_mul(x) / (i * b);
    }
    res
}

fn handle_activate_program(mut input: PrecompileInput<'_>) -> PrecompileResult {
    const ACTIVATION_UPFRONT_GAS: u64 = 1_659_168;

    crate::reset_precompile_gas();
    let args_cost = COPY_GAS * (input.data.len() as u64).saturating_sub(4).div_ceil(32);
    crate::charge_precompile_gas(args_cost);
    crate::charge_precompile_gas(SLOAD_GAS); // OpenArbosState version read
    crate::charge_precompile_gas(ACTIVATION_UPFRONT_GAS);

    let program_address = extract_address(input.data)?;

    let code_hash = {
        let account = input
            .internals_mut()
            .load_account(program_address)
            .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;
        account.data.info.code_hash
    };

    let code_bytes = {
        let code_account = input
            .internals_mut()
            .load_account_code(program_address)
            .map_err(|e| PrecompileError::other(format!("load_account_code: {e:?}")))?;
        code_account
            .data
            .code()
            .map(|c| c.original_bytes())
            .unwrap_or_default()
            .to_vec()
    };

    if code_bytes.is_empty() || !arb_stylus::is_stylus_deployable(&code_bytes, crate::get_arbos_version()) {
        return Err(PrecompileError::other("ProgramNotWasm()"));
    }

    let wasm = arb_stylus::decompress_wasm(&code_bytes)
        .map_err(|e| PrecompileError::other(format!("ProgramNotWasm: {e}")))?;

    // Params: charge WarmStorageReadCost (100) like Nitro, then read free.
    load_arbos(&mut input)?;
    crate::charge_precompile_gas(100); // WarmStorageReadCostEIP2929
    let params_word = {
        let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
        let params_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_PARAMS_KEY);
        let slot = map_slot(params_key.as_slice(), 0);
        let val = input.internals_mut().sload(ARBOS_STATE_ADDRESS, slot)
            .map_err(|_| PrecompileError::other("sload failed"))?.data;
        val.to_be_bytes::<32>()
    };
    let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
    let page_limit = u16::from_be_bytes([params_word[13], params_word[14]]);

    let time = block_timestamp();
    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let data_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_DATA_KEY);
    let program_slot = map_slot_b256(data_key.as_slice(), &code_hash);

    let existing = sload_field(&mut input, program_slot)?;
    let existing_bytes = existing.to_be_bytes::<32>();
    let existing_version = u16::from_be_bytes([existing_bytes[0], existing_bytes[1]]);
    let was_cached = existing_bytes[14] != 0;

    if existing_version == params_version {
        let activated_at =
            (existing_bytes[8] as u32) << 16 | (existing_bytes[9] as u32) << 8 | existing_bytes[10] as u32;
        let age = hours_to_age(time, activated_at);
        let expiry_days = u16::from_be_bytes([params_word[19], params_word[20]]);
        if age <= (expiry_days as u64) * 86400 {
            return Err(PrecompileError::other("ProgramUpToDate()"));
        }
    }

    let gas_available = input.gas.saturating_sub(crate::get_precompile_gas());
    let mut gas_for_prover = gas_available;
    let diag_pre_prover = crate::get_precompile_gas();
    let diag_gas_to_prover = gas_for_prover;

    let info = match arb_stylus::activate_program(
        &wasm,
        code_hash.as_ref(),
        params_version,
        crate::get_arbos_version(),
        page_limit,
        false,
        &mut gas_for_prover,
    ) {
        Ok(info) => info,
        Err(e) => {
            crate::charge_precompile_gas(gas_available);
            return Err(PrecompileError::other(format!("{e}")));
        }
    };

    let prover_gas_used = gas_available.saturating_sub(gas_for_prover);
    crate::charge_precompile_gas(prover_gas_used);
    let wasm_hash = alloy_primitives::keccak256(&wasm);
    tracing::warn!(target: "stylus",
        input_gas = input.gas, pre_prover = diag_pre_prover, to_prover = diag_gas_to_prover,
        prover_used = prover_gas_used, after_prover = crate::get_precompile_gas(),
        wasm_len = wasm.len(), %wasm_hash, "activateProgram gas breakdown");

    // Store module hash
    let module_hashes_key = derive_subspace_key(programs_key.as_slice(), &[2]);
    let module_hash_slot = map_slot_b256(module_hashes_key.as_slice(), &code_hash);
    input
        .internals_mut()
        .sstore(ARBOS_STATE_ADDRESS, module_hash_slot, U256::from_be_bytes(info.module_hash.0))
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);
    let data_pricer_key = derive_subspace_key(programs_key.as_slice(), &[3]);
    let demand: u32 = sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 0))?.to::<u64>() as u32;
    let bps: u32 = sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 1))?.to::<u64>() as u32;
    let last_update: u64 = sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 2))?.to::<u64>();
    let min_price: u32 = sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 3))?.to::<u64>() as u32;
    let inertia: u32 = sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 4))?.to::<u64>() as u32;

    let passed = (time.saturating_sub(last_update)) as u32;
    let credit = bps.saturating_mul(passed);
    let new_demand = demand.saturating_sub(credit).saturating_add(info.asm_estimate);

    input.internals_mut()
        .sstore(ARBOS_STATE_ADDRESS, map_slot(data_pricer_key.as_slice(), 0), U256::from(new_demand))
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);
    input.internals_mut()
        .sstore(ARBOS_STATE_ADDRESS, map_slot(data_pricer_key.as_slice(), 2), U256::from(time))
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);

    let exponent = if inertia > 0 { 10_000u64 * (new_demand as u64) / (inertia as u64) } else { 0 };
    let multiplier = approx_exp_basis_points(exponent);
    let cost_per_byte = (min_price as u64).saturating_mul(multiplier) / 10_000;
    let data_fee = U256::from(cost_per_byte.saturating_mul(info.asm_estimate as u64));

    // Store program data
    let estimate_kb = div_ceil(info.asm_estimate as u64, 1024).min(0xFF_FFFF) as u32;
    let hours = hours_since_arbitrum(time);
    let mut pd = [0u8; 32];
    pd[0..2].copy_from_slice(&params_version.to_be_bytes());
    pd[2..4].copy_from_slice(&info.init_gas.to_be_bytes());
    pd[4..6].copy_from_slice(&info.cached_init_gas.to_be_bytes());
    pd[6..8].copy_from_slice(&info.footprint.to_be_bytes());
    pd[8] = (hours >> 16) as u8;
    pd[9] = (hours >> 8) as u8;
    pd[10] = hours as u8;
    pd[11] = (estimate_kb >> 16) as u8;
    pd[12] = (estimate_kb >> 8) as u8;
    pd[13] = estimate_kb as u8;
    pd[14] = was_cached as u8;

    input.internals_mut()
        .sstore(ARBOS_STATE_ADDRESS, program_slot, U256::from_be_bytes(pd))
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);

    // payActivationDataFee reads NetworkFeeAccount from storage (800 gas)
    crate::charge_precompile_gas(SLOAD_GAS);

    // Signal executor to handle the data fee payment
    crate::set_stylus_activation_request(Some(program_address));
    crate::set_stylus_activation_data_fee(data_fee);

    // Emit ProgramActivated(bytes32 codehash, bytes32 moduleHash, address program, uint256 dataFee, uint16 version)
    let event_topic = alloy_primitives::keccak256(
        b"ProgramActivated(bytes32,bytes32,address,uint256,uint16)",
    );
    let mut event_data = Vec::with_capacity(128);
    event_data.extend_from_slice(&info.module_hash.0);
    event_data.extend_from_slice(&[0u8; 12]);
    event_data.extend_from_slice(program_address.as_slice());
    event_data.extend_from_slice(&data_fee.to_be_bytes::<32>());
    let mut ver = [0u8; 32];
    ver[30..32].copy_from_slice(&params_version.to_be_bytes());
    event_data.extend_from_slice(&ver);
    // Event gas: LogGas(375) + (1 + indexed_count) * LogTopicGas(375) + LogDataGas(8) * data_bytes
    let event_gas = 375 + 2 * 375 + 8 * event_data.len() as u64;
    crate::charge_precompile_gas(event_gas);
    crate::emit_log(ARBWASM_ADDRESS, &[event_topic, code_hash], &event_data);

    // Return encoding gas: CopyGas * words(return_len)
    let return_data = {
        let mut output = Vec::with_capacity(64);
        let mut ver_out = [0u8; 32];
        ver_out[30..32].copy_from_slice(&params_version.to_be_bytes());
        output.extend_from_slice(&ver_out);
        output.extend_from_slice(&data_fee.to_be_bytes::<32>());
        output
    };
    let return_gas = COPY_GAS * (return_data.len() as u64).div_ceil(32);
    crate::charge_precompile_gas(return_gas);

    let gas_used = crate::get_precompile_gas();
    tracing::warn!(target: "stylus",
        total = gas_used, args = args_cost, prover = prover_gas_used,
        event = event_gas, ret = return_gas, "activateProgram total gas");
    Ok(PrecompileOutput::new(gas_used, return_data.into()))
}

fn handle_codehash_keepalive(mut input: PrecompileInput<'_>) -> PrecompileResult {
    crate::reset_precompile_gas();
    let args_cost = COPY_GAS * (input.data.len() as u64).saturating_sub(4).div_ceil(32);
    crate::charge_precompile_gas(args_cost);

    let codehash = extract_bytes32(input.data)?;

    load_arbos(&mut input)?;
    let params_word = load_params_word(&mut input)?;
    let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
    let keepalive_days = u16::from_be_bytes([params_word[21], params_word[22]]);
    let expiry_days = u16::from_be_bytes([params_word[19], params_word[20]]);
    let time = block_timestamp();

    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let data_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_DATA_KEY);
    let program_slot = map_slot_b256(data_key.as_slice(), &codehash);
    let program_bytes = sload_field(&mut input, program_slot)?.to_be_bytes::<32>();
    let program = parse_program(&program_bytes, &params_word);

    if program.version == 0 {
        return Err(PrecompileError::other("ProgramNotActivated()"));
    }
    let age = hours_to_age(time, program_bytes[8] as u32 * 65536 + program_bytes[9] as u32 * 256 + program_bytes[10] as u32);
    if age > (expiry_days as u64) * 86400 {
        return Err(PrecompileError::other("ProgramExpired()"));
    }
    if program.version != params_version {
        return Err(PrecompileError::other("ProgramNeedsUpgrade()"));
    }
    if age < (keepalive_days as u64) * 86400 {
        return Err(PrecompileError::other("ProgramKeepaliveTooSoon()"));
    }

    let asm_size = program.asm_estimate_kb * 1024;

    // Update data pricer
    let data_pricer_key = derive_subspace_key(programs_key.as_slice(), &[3]);
    let demand: u32 = sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 0))?.to::<u64>() as u32;
    let bps: u32 = sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 1))?.to::<u64>() as u32;
    let last_update: u64 = sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 2))?.to::<u64>();
    let min_price: u32 = sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 3))?.to::<u64>() as u32;
    let inertia: u32 = sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 4))?.to::<u64>() as u32;

    let passed = (time.saturating_sub(last_update)) as u32;
    let credit = bps.saturating_mul(passed);
    let new_demand = demand.saturating_sub(credit).saturating_add(asm_size);

    input.internals_mut()
        .sstore(ARBOS_STATE_ADDRESS, map_slot(data_pricer_key.as_slice(), 0), U256::from(new_demand))
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);
    input.internals_mut()
        .sstore(ARBOS_STATE_ADDRESS, map_slot(data_pricer_key.as_slice(), 2), U256::from(time))
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);

    let exponent = if inertia > 0 { 10_000u64 * (new_demand as u64) / (inertia as u64) } else { 0 };
    let multiplier = approx_exp_basis_points(exponent);
    let cost_per_byte = (min_price as u64).saturating_mul(multiplier) / 10_000;
    let data_fee = U256::from(cost_per_byte.saturating_mul(asm_size as u64));

    // Reset activatedAt
    let hours = hours_since_arbitrum(time);
    let mut pd = program_bytes;
    pd[8] = (hours >> 16) as u8;
    pd[9] = (hours >> 8) as u8;
    pd[10] = hours as u8;

    input.internals_mut()
        .sstore(ARBOS_STATE_ADDRESS, program_slot, U256::from_be_bytes(pd))
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);

    // payActivationDataFee reads NetworkFeeAccount from storage (800 gas)
    crate::charge_precompile_gas(SLOAD_GAS);

    // Signal executor to handle the data fee payment
    crate::set_stylus_keepalive_request(Some(codehash));
    crate::set_stylus_activation_data_fee(data_fee);

    // Emit ProgramLifetimeExtended(bytes32 codehash, uint256 dataFee)
    let event_topic = alloy_primitives::keccak256(
        b"ProgramLifetimeExtended(bytes32,uint256)",
    );
    let mut event_data = Vec::with_capacity(32);
    event_data.extend_from_slice(&data_fee.to_be_bytes::<32>());
    let event_gas = 375 + 2 * 375 + 8 * event_data.len() as u64;
    crate::charge_precompile_gas(event_gas);
    crate::emit_log(ARBWASM_ADDRESS, &[event_topic, codehash], &event_data);

    // No return value for keepalive
    let gas_used = crate::get_precompile_gas();
    Ok(PrecompileOutput::new(gas_used, Vec::new().into()))
}
