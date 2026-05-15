use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, Log, B256, U256};
use alloy_sol_types::{SolError, SolEvent, SolInterface};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::{
    interfaces::IArbWasm,
    storage_slot::{
        derive_subspace_key, map_slot, map_slot_b256, root_slot, ARBOS_STATE_ADDRESS,
        NETWORK_FEE_ACCOUNT_OFFSET, PROGRAMS_DATA_KEY, PROGRAMS_PARAMS_KEY, PROGRAMS_SUBSPACE,
        ROOT_STORAGE_KEY,
    },
};

/// ArbWasm precompile address (0x71).
pub const ARBWASM_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x71,
]);

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const COPY_GAS: u64 = 3;
const WARM_SLOAD_GAS: u64 = 100;
const STORAGE_CODE_HASH_COST: u64 = 2_600;
/// Framework cost: argsCost (CopyGas * 1 word for 32-byte address) + OpenArbosState (800).
const FRAMEWORK_GAS_PROGRAM_ADDR: u64 = COPY_GAS + 800;
/// Total gas for ArbWasm methods that look up program metadata by address: framework
/// + Params (warm) + GetCodeHash (cold account access) + getProgram SLOAD. Mirrors
/// Nitro's `con.getCodeHash` + `c.State.Programs().Params/Get` charge sequence.
const PROGRAM_LOOKUP_GAS: u64 =
    FRAMEWORK_GAS_PROGRAM_ADDR + WARM_SLOAD_GAS + STORAGE_CODE_HASH_COST + SLOAD_GAS;

/// Initial page ramp constant (not stored in packed params).
const INITIAL_PAGE_RAMP: u64 = 620674314;

const MIN_INIT_GAS_UNITS: u64 = 128;
const MIN_CACHED_GAS_UNITS: u64 = 32;
const COST_SCALAR_PERCENT: u64 = 2;

pub fn create_arbwasm_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbwasm"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    if let Some(result) =
        crate::check_precompile_version(arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS)
    {
        return result;
    }

    let gas_limit = input.gas;

    let call = match IArbWasm::ArbWasmCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbWasm::ArbWasmCalls as Calls;
    // State-modifying methods own their gas accounting.
    match &call {
        Calls::activateProgram(c) => return handle_activate_program(input, c.program),
        Calls::codehashKeepalive(c) => return handle_codehash_keepalive(input, c.codehash),
        _ => {}
    }

    crate::init_precompile_gas(input.data.len());

    let result = match call {
        Calls::stylusVersion(_) => {
            // Open(800) + Params warm(100) + result(3) = 903 — matches Nitro's
            // `c.State.Programs().Params()` charge sequence.
            let params = load_params_word(&mut input)?;
            let version = u16::from_be_bytes([params[0], params[1]]);
            ok_u256(SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS, U256::from(version))
        }
        Calls::inkPrice(_) => {
            // Open(800) + Params warm(100) + result(3) = 903.
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params_word(&mut input)?;
            let ink_price = (params[2] as u32) << 16 | (params[3] as u32) << 8 | params[4] as u32;
            ok_u256(METHOD_GAS, U256::from(ink_price))
        }
        Calls::maxStackDepth(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params_word(&mut input)?;
            let depth = u32::from_be_bytes([params[5], params[6], params[7], params[8]]);
            ok_u256(METHOD_GAS, U256::from(depth))
        }
        Calls::freePages(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params_word(&mut input)?;
            let pages = u16::from_be_bytes([params[9], params[10]]);
            ok_u256(METHOD_GAS, U256::from(pages))
        }
        Calls::pageGas(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params_word(&mut input)?;
            let gas = u16::from_be_bytes([params[11], params[12]]);
            ok_u256(METHOD_GAS, U256::from(gas))
        }
        Calls::pageRamp(_) => {
            // Nitro reads Params() — same Open(800) + warm(100) + result(3).
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            load_arbos(&mut input)?;
            ok_u256(METHOD_GAS, U256::from(INITIAL_PAGE_RAMP))
        }
        Calls::pageLimit(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params_word(&mut input)?;
            let limit = u16::from_be_bytes([params[13], params[14]]);
            ok_u256(METHOD_GAS, U256::from(limit))
        }
        Calls::minInitGas(_) => {
            if let Some(result) = crate::check_method_version(
                input.gas,
                arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS_CHARGING_FIXES,
                0,
            ) {
                return result;
            }
            // Returns (uint64, uint64) → 2 result words → 2*COPY_GAS.
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + 2 * COPY_GAS;
            let params = load_params_word(&mut input)?;
            let min_init = params[15] as u64;
            let min_cached = params[16] as u64;
            let init = min_init.saturating_mul(MIN_INIT_GAS_UNITS);
            let cached = min_cached.saturating_mul(MIN_CACHED_GAS_UNITS);
            ok_two_u256(METHOD_GAS, U256::from(init), U256::from(cached))
        }
        Calls::initCostScalar(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params_word(&mut input)?;
            let scalar = params[17] as u64;
            ok_u256(
                METHOD_GAS,
                U256::from(scalar.saturating_mul(COST_SCALAR_PERCENT)),
            )
        }
        Calls::expiryDays(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params_word(&mut input)?;
            let days = u16::from_be_bytes([params[19], params[20]]);
            ok_u256(METHOD_GAS, U256::from(days))
        }
        Calls::keepaliveDays(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params_word(&mut input)?;
            let days = u16::from_be_bytes([params[21], params[22]]);
            ok_u256(METHOD_GAS, U256::from(days))
        }
        Calls::blockCacheSize(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params_word(&mut input)?;
            let size = u16::from_be_bytes([params[23], params[24]]);
            ok_u256(METHOD_GAS, U256::from(size))
        }
        Calls::activationGas(_) => {
            // Nitro registers `ActivationGas` only at ArbOS >= 60 (see
            // precompile.go: methodsByName["ActivationGas"].arbosVersion =
            // ArbosVersion_StylusActivationGas). Calling it pre-v60 hits the
            // unregistered-method path and burns the full gas budget.
            if let Some(r) = crate::check_method_version(
                input.gas,
                arb_chainspec::arbos_version::ARBOS_VERSION_60,
                0,
            ) {
                return r;
            }
            let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
            let activation_key = derive_subspace_key(programs_key.as_slice(), &[5]);
            let slot = map_slot(activation_key.as_slice(), 0);
            let gas = sload_field(&mut input, slot)?;
            // Open(800) + ActivationGas SLOAD(800) + result(3) = 1603.
            ok_u256(SLOAD_GAS + SLOAD_GAS + COPY_GAS, gas)
        }
        Calls::codehashVersion(c) => {
            // argsCost(3) + Open(800) + Params warm(100) + getProgram SLOAD(800) = 1703 (lookup);
            // success adds result(3) → 1706. Revert sizes resultCost from the error payload.
            const LOOKUP_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + SLOAD_GAS + COPY_GAS;
            const METHOD_GAS: u64 = LOOKUP_GAS + COPY_GAS;
            let (params_word, program_word) = load_params_and_program(&mut input, c.codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let expiry_days = params_expiry_days(&params_word);
            let program = parse_program(&program_word, &params_word);
            if let Err(r) = validate_active_program(
                &program,
                params_version,
                expiry_days,
                input.gas,
                LOOKUP_GAS,
            ) {
                return r;
            }
            ok_u256(METHOD_GAS, U256::from(program.version))
        }
        Calls::codehashAsmSize(c) => {
            const LOOKUP_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + SLOAD_GAS + COPY_GAS;
            const METHOD_GAS: u64 = LOOKUP_GAS + COPY_GAS;
            let (params_word, program_word) = load_params_and_program(&mut input, c.codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let expiry_days = params_expiry_days(&params_word);
            let program = parse_program(&program_word, &params_word);
            if let Err(r) = validate_active_program(
                &program,
                params_version,
                expiry_days,
                input.gas,
                LOOKUP_GAS,
            ) {
                return r;
            }
            let asm_size = program.asm_estimate_kb.saturating_mul(1024);
            ok_u256(METHOD_GAS, U256::from(asm_size))
        }
        Calls::programVersion(c) => {
            const METHOD_GAS: u64 = PROGRAM_LOOKUP_GAS + COPY_GAS;
            let codehash = get_account_codehash(&mut input, c.program)?;
            let (params_word, program_word) = load_params_and_program(&mut input, codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let expiry_days = params_expiry_days(&params_word);
            let program = parse_program(&program_word, &params_word);
            if let Err(r) = validate_active_program(
                &program,
                params_version,
                expiry_days,
                input.gas,
                PROGRAM_LOOKUP_GAS,
            ) {
                return r;
            }
            ok_u256(METHOD_GAS, U256::from(program.version))
        }
        Calls::programInitGas(c) => {
            // Returns (uint64, uint64) → 64-byte output → 2 words of result_cost.
            const METHOD_GAS: u64 = PROGRAM_LOOKUP_GAS + 2 * COPY_GAS;
            let codehash = get_account_codehash(&mut input, c.program)?;
            let (params_word, program_word) = load_params_and_program(&mut input, codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let expiry_days = params_expiry_days(&params_word);
            let program = parse_program(&program_word, &params_word);
            if let Err(r) = validate_active_program(
                &program,
                params_version,
                expiry_days,
                input.gas,
                PROGRAM_LOOKUP_GAS,
            ) {
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

            ok_two_u256(METHOD_GAS, U256::from(init_gas), U256::from(cached_gas))
        }
        Calls::programMemoryFootprint(c) => {
            const METHOD_GAS: u64 = PROGRAM_LOOKUP_GAS + COPY_GAS;
            let codehash = get_account_codehash(&mut input, c.program)?;
            let (params_word, program_word) = load_params_and_program(&mut input, codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let expiry_days = params_expiry_days(&params_word);
            let program = parse_program(&program_word, &params_word);
            if let Err(r) = validate_active_program(
                &program,
                params_version,
                expiry_days,
                input.gas,
                PROGRAM_LOOKUP_GAS,
            ) {
                return r;
            }
            ok_u256(METHOD_GAS, U256::from(program.footprint))
        }
        Calls::programTimeLeft(c) => {
            const METHOD_GAS: u64 = PROGRAM_LOOKUP_GAS + COPY_GAS;
            let codehash = get_account_codehash(&mut input, c.program)?;
            let (params_word, program_word) = load_params_and_program(&mut input, codehash)?;
            let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
            let expiry_days = params_expiry_days(&params_word);
            let program = parse_program(&program_word, &params_word);
            if let Err(r) = validate_active_program(
                &program,
                params_version,
                expiry_days,
                input.gas,
                PROGRAM_LOOKUP_GAS,
            ) {
                return r;
            }

            let expiry_seconds = (expiry_days as u64) * 24 * 3600;
            let time_left = expiry_seconds.saturating_sub(program.age_seconds);
            ok_u256(METHOD_GAS, U256::from(time_left))
        }
        Calls::activateProgram(_) | Calls::codehashKeepalive(_) => unreachable!(),
    };
    crate::gas_check(input.gas, result)
}

// ── Helpers ──────────────────────────────────────────────────────────

fn params_expiry_days(params_word: &[u8; 32]) -> u16 {
    u16::from_be_bytes([params_word[19], params_word[20]])
}

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

/// Returns ProgramNotActivated, ProgramNeedsUpgrade(progV, paramsV),
/// or ProgramExpired(ageSeconds), in that order. `lookup_gas` is the
/// method's argsCost + state-access charges (everything except the
/// result-copy cost). The revert charges `lookup_gas + COPY_GAS * words`
/// where `words` is rounded up from the actual error payload length —
/// matching Nitro's precompile framework which sizes resultCost from
/// `len(solErr.data)` rather than the success-path output.
fn validate_active_program(
    program: &ProgramInfo,
    params_version: u16,
    expiry_days: u16,
    gas_limit: u64,
    lookup_gas: u64,
) -> Result<(), PrecompileResult> {
    if program.version == 0 {
        let data = IArbWasm::ProgramNotActivated {}.abi_encode();
        return Err(revert_with_payload(data, lookup_gas, gas_limit));
    }
    if program.version != params_version {
        let data = IArbWasm::ProgramNeedsUpgrade {
            version: program.version,
            stylusVersion: params_version,
        }
        .abi_encode();
        return Err(revert_with_payload(data, lookup_gas, gas_limit));
    }
    let expiry_seconds = (expiry_days as u64).saturating_mul(86_400);
    if program.age_seconds > expiry_seconds {
        let data = IArbWasm::ProgramExpired {
            ageInSeconds: program.age_seconds,
        }
        .abi_encode();
        return Err(revert_with_payload(data, lookup_gas, gas_limit));
    }
    Ok(())
}

/// Revert with `lookup_gas + CopyGas * ceil(payload_len / 32)` charged.
fn revert_with_payload(payload: Vec<u8>, lookup_gas: u64, gas_limit: u64) -> PrecompileResult {
    let result_cost = COPY_GAS.saturating_mul((payload.len() as u64).div_ceil(32));
    let gas_used = lookup_gas.saturating_add(result_cost);
    Ok(PrecompileOutput::new_reverted(
        gas_used.min(gas_limit),
        payload.into(),
    ))
}

fn revert_sol_error(payload: Vec<u8>) -> PrecompileResult {
    crate::charge_precompile_gas(COPY_GAS * (payload.len() as u64).div_ceil(32));
    Ok(PrecompileOutput::new_reverted(
        crate::get_precompile_gas(),
        payload.into(),
    ))
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

/// Approximates `b * e^(x/b)` where `b = 10_000` (basis points), via Horner's
/// method with accuracy=12.
fn approx_exp_basis_points(x: u64) -> u64 {
    let b = 10_000u64;
    let accuracy = 12u64;
    let mut res = b + x / accuracy;
    for i in (1..accuracy).rev() {
        res = b + res.saturating_mul(x) / (i * b);
    }
    res
}

fn handle_activate_program(
    mut input: PrecompileInput<'_>,
    program_address: Address,
) -> PrecompileResult {
    const ACTIVATION_UPFRONT_GAS: u64 = 1_659_168;

    crate::reset_precompile_gas();
    let args_cost = COPY_GAS * (input.data.len() as u64).saturating_sub(4).div_ceil(32);
    crate::charge_precompile_gas(args_cost);
    crate::charge_precompile_gas(SLOAD_GAS); // OpenArbosState version read

    // Nitro's `programs.ActivationGas()` SLOADs the activationGas slot (800) at
    // ArbOS >= 60; pre-v60 it short-circuits with no SLOAD. Mirror that exactly
    // so live Sepolia activations at v60+ match canon. The Nitro Docker
    // reference image (v3.10.0-rc.2) predates this feature, so the matrix
    // differential test will report a baseline -800 against that older image
    // at v60 — an artifact of the Docker, not an arbreth bug.
    if crate::get_arbos_version() >= arb_chainspec::arbos_version::ARBOS_VERSION_60 {
        let programs_key_for_act = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
        let activation_gas_key = derive_subspace_key(programs_key_for_act.as_slice(), &[5]);
        let activation_gas_slot = map_slot(activation_gas_key.as_slice(), 0);
        let _activation_gas = sload_field(&mut input, activation_gas_slot)?;
    }

    crate::charge_precompile_gas(ACTIVATION_UPFRONT_GAS);

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

    // Charge Params warm read (100) and existing-program SLOAD (800) *before*
    // the prefix check so revert and success paths both account for them.
    // Mirrors the order in Nitro's `programs.ActivateProgram`: `Params()` then
    // `programExists()` (SLOAD of `programs[codeHash]`) run before `getWasm`,
    // which is where a non-Stylus prefix is rejected.
    load_arbos(&mut input)?;
    crate::charge_precompile_gas(100); // WarmStorageReadCostEIP2929 (Params)
    let params_word = {
        let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
        let params_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_PARAMS_KEY);
        let slot = map_slot(params_key.as_slice(), 0);
        let val = input
            .internals_mut()
            .sload(ARBOS_STATE_ADDRESS, slot)
            .map_err(|_| PrecompileError::other("sload failed"))?
            .data;
        val.to_be_bytes::<32>()
    };
    let params_version = u16::from_be_bytes([params_word[0], params_word[1]]);
    let page_limit = u16::from_be_bytes([params_word[13], params_word[14]]);

    let programs_key = derive_subspace_key(ROOT_STORAGE_KEY, PROGRAMS_SUBSPACE);
    let data_key = derive_subspace_key(programs_key.as_slice(), PROGRAMS_DATA_KEY);
    let program_slot = map_slot_b256(data_key.as_slice(), &code_hash);
    let existing = sload_field(&mut input, program_slot)?;

    // Nitro distinguishes two failure modes for invalid Stylus bytecode at v40
    // and below. Empty bytecode returns the `ProgramNotWasm()` Solidity error
    // (3 gas result-copy cost). A non-empty but non-classic-prefix bytecode
    // returns a non-Solidity error (`errors.New("specified bytecode is not a
    // Stylus program")`) which the framework reverts WITHOUT charging
    // result-copy cost. See `arbos/programs/programs.go::getWasmFromContractCode`
    // line 420-421 ("Old arbOS behavior - this is not a solidity error").
    if code_bytes.is_empty() {
        return revert_sol_error(IArbWasm::ProgramNotWasm {}.abi_encode());
    }
    if !arb_stylus::is_stylus_deployable(&code_bytes, crate::get_arbos_version()) {
        let arbos_v = crate::get_arbos_version();
        if arbos_v < arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS_CONTRACT_LIMIT {
            // Old ArbOS behavior: revert with empty payload, no result-copy cost.
            return Ok(PrecompileOutput::new_reverted(
                crate::get_precompile_gas().min(input.gas),
                Default::default(),
            ));
        }
        return revert_sol_error(IArbWasm::ProgramNotWasm {}.abi_encode());
    }

    let wasm = match arb_stylus::decompress_wasm(&code_bytes) {
        Ok(w) => w,
        Err(_) => return revert_sol_error(IArbWasm::ProgramNotWasm {}.abi_encode()),
    };

    let time = block_timestamp();
    let existing_bytes = existing.to_be_bytes::<32>();
    let existing_version = u16::from_be_bytes([existing_bytes[0], existing_bytes[1]]);
    let was_cached = existing_bytes[14] != 0;

    if existing_version == params_version {
        let activated_at = (existing_bytes[8] as u32) << 16
            | (existing_bytes[9] as u32) << 8
            | existing_bytes[10] as u32;
        let age = hours_to_age(time, activated_at);
        let expiry_days = u16::from_be_bytes([params_word[19], params_word[20]]);
        if age <= (expiry_days as u64) * 86400 {
            return revert_sol_error(IArbWasm::ProgramUpToDate {}.abi_encode());
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
        .sstore(
            ARBOS_STATE_ADDRESS,
            module_hash_slot,
            U256::from_be_bytes(info.module_hash.0),
        )
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);
    let data_pricer_key = derive_subspace_key(programs_key.as_slice(), &[3]);
    let demand: u32 =
        sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 0))?.to::<u64>() as u32;
    let bps: u32 =
        sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 1))?.to::<u64>() as u32;
    let last_update: u64 =
        sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 2))?.to::<u64>();
    let min_price: u32 =
        sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 3))?.to::<u64>() as u32;
    let inertia: u32 =
        sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 4))?.to::<u64>() as u32;

    let passed = (time.saturating_sub(last_update)) as u32;
    let credit = bps.saturating_mul(passed);
    let new_demand = demand
        .saturating_sub(credit)
        .saturating_add(info.asm_estimate);

    input
        .internals_mut()
        .sstore(
            ARBOS_STATE_ADDRESS,
            map_slot(data_pricer_key.as_slice(), 0),
            U256::from(new_demand),
        )
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);
    input
        .internals_mut()
        .sstore(
            ARBOS_STATE_ADDRESS,
            map_slot(data_pricer_key.as_slice(), 2),
            U256::from(time),
        )
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);

    let exponent = if inertia > 0 {
        10_000u64 * (new_demand as u64) / (inertia as u64)
    } else {
        0
    };
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

    input
        .internals_mut()
        .sstore(ARBOS_STATE_ADDRESS, program_slot, U256::from_be_bytes(pd))
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);

    // The activation `value` is the value of the call frame entering 0x71,
    // which differs by call site:
    //   * Outer EOA → 0x71: `build.rs` zeros `tx_env.value` (so revm won't
    //     transfer ETH to the precompile) and stashes the original tx value
    //     in `STYLUS_CALL_VALUE`. `input.value` is therefore 0 here; the
    //     stashed value drives the check, and the post-commit hook burns
    //     the data fee from the sender.
    //   * Inner CONTRACT → 0x71 (e.g. a Solidity factory calling
    //     `ArbWasm.activateProgram{value: budget}(...)`): revm has already
    //     transferred `budget` from the caller to 0x71 by the time the
    //     precompile runs. `input.value == budget`; the stashed value is 0.
    //     We mirror Nitro's `payActivationDataFee` here by forwarding the
    //     data fee to NetworkFeeAccount and refunding the rest to the
    //     immediate caller, so the post-commit hook is skipped.
    let stashed_outer_value = crate::get_stylus_call_value();
    let inner_call_value = input.value;
    let effective_value = if inner_call_value > U256::ZERO {
        inner_call_value
    } else {
        stashed_outer_value
    };
    if effective_value < data_fee {
        return revert_sol_error(
            IArbWasm::ProgramInsufficientValue {
                have: effective_value,
                want: data_fee,
            }
            .abi_encode(),
        );
    }

    if inner_call_value > U256::ZERO {
        let caller = input.caller;
        let net_acct_word = sload_field(&mut input, root_slot(NETWORK_FEE_ACCOUNT_OFFSET))?;
        let network_addr = Address::from_word(B256::from(net_acct_word.to_be_bytes::<32>()));
        let _ = input
            .internals_mut()
            .transfer(ARBWASM_ADDRESS, network_addr, data_fee);
        let repay = inner_call_value.saturating_sub(data_fee);
        if repay > U256::ZERO {
            let _ = input
                .internals_mut()
                .transfer(ARBWASM_ADDRESS, caller, repay);
        }
        crate::set_stylus_activation_request(Some(program_address));
    } else {
        crate::charge_precompile_gas(SLOAD_GAS);
        crate::set_stylus_activation_request(Some(program_address));
        crate::set_stylus_activation_data_fee(data_fee);
    }

    let event_topic = IArbWasm::ProgramActivated::SIGNATURE_HASH;
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
    // Insert the log directly into the journal at the call-frame's
    // position so it interleaves correctly with the caller's own LOG
    // opcodes. Buffering it for a post-tx flush would append it after
    // every inline log emitted by the calling contract.
    input.internals_mut().log(Log::new_unchecked(
        ARBWASM_ADDRESS,
        vec![event_topic, code_hash],
        event_data.into(),
    ));

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

fn handle_codehash_keepalive(mut input: PrecompileInput<'_>, codehash: B256) -> PrecompileResult {
    crate::reset_precompile_gas();
    let args_cost = COPY_GAS * (input.data.len() as u64).saturating_sub(4).div_ceil(32);
    crate::charge_precompile_gas(args_cost);

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
        return revert_sol_error(IArbWasm::ProgramNotActivated {}.abi_encode());
    }
    let age = hours_to_age(
        time,
        program_bytes[8] as u32 * 65536 + program_bytes[9] as u32 * 256 + program_bytes[10] as u32,
    );
    if age > (expiry_days as u64) * 86400 {
        return revert_sol_error(
            IArbWasm::ProgramExpired { ageInSeconds: age }.abi_encode(),
        );
    }
    if program.version != params_version {
        return revert_sol_error(
            IArbWasm::ProgramNeedsUpgrade {
                version: program.version,
                stylusVersion: params_version,
            }
            .abi_encode(),
        );
    }
    if age < (keepalive_days as u64) * 86400 {
        return revert_sol_error(
            IArbWasm::ProgramKeepaliveTooSoon { ageInSeconds: age }.abi_encode(),
        );
    }

    let asm_size = program.asm_estimate_kb * 1024;

    // Update data pricer
    let data_pricer_key = derive_subspace_key(programs_key.as_slice(), &[3]);
    let demand: u32 =
        sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 0))?.to::<u64>() as u32;
    let bps: u32 =
        sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 1))?.to::<u64>() as u32;
    let last_update: u64 =
        sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 2))?.to::<u64>();
    let min_price: u32 =
        sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 3))?.to::<u64>() as u32;
    let inertia: u32 =
        sload_field(&mut input, map_slot(data_pricer_key.as_slice(), 4))?.to::<u64>() as u32;

    let passed = (time.saturating_sub(last_update)) as u32;
    let credit = bps.saturating_mul(passed);
    let new_demand = demand.saturating_sub(credit).saturating_add(asm_size);

    input
        .internals_mut()
        .sstore(
            ARBOS_STATE_ADDRESS,
            map_slot(data_pricer_key.as_slice(), 0),
            U256::from(new_demand),
        )
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);
    input
        .internals_mut()
        .sstore(
            ARBOS_STATE_ADDRESS,
            map_slot(data_pricer_key.as_slice(), 2),
            U256::from(time),
        )
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);

    let exponent = if inertia > 0 {
        10_000u64 * (new_demand as u64) / (inertia as u64)
    } else {
        0
    };
    let multiplier = approx_exp_basis_points(exponent);
    let cost_per_byte = (min_price as u64).saturating_mul(multiplier) / 10_000;
    let data_fee = U256::from(cost_per_byte.saturating_mul(asm_size as u64));

    // Reset activatedAt
    let hours = hours_since_arbitrum(time);
    let mut pd = program_bytes;
    pd[8] = (hours >> 16) as u8;
    pd[9] = (hours >> 8) as u8;
    pd[10] = hours as u8;

    input
        .internals_mut()
        .sstore(ARBOS_STATE_ADDRESS, program_slot, U256::from_be_bytes(pd))
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);

    // See `handle_activate_program` for the call-frame-value rationale.
    let stashed_outer_value = crate::get_stylus_call_value();
    let inner_call_value = input.value;
    let effective_value = if inner_call_value > U256::ZERO {
        inner_call_value
    } else {
        stashed_outer_value
    };
    if effective_value < data_fee {
        return revert_sol_error(
            IArbWasm::ProgramInsufficientValue {
                have: effective_value,
                want: data_fee,
            }
            .abi_encode(),
        );
    }

    if inner_call_value > U256::ZERO {
        let caller = input.caller;
        let net_acct_word = sload_field(&mut input, root_slot(NETWORK_FEE_ACCOUNT_OFFSET))?;
        let network_addr = Address::from_word(B256::from(net_acct_word.to_be_bytes::<32>()));
        let _ = input
            .internals_mut()
            .transfer(ARBWASM_ADDRESS, network_addr, data_fee);
        let repay = inner_call_value.saturating_sub(data_fee);
        if repay > U256::ZERO {
            let _ = input
                .internals_mut()
                .transfer(ARBWASM_ADDRESS, caller, repay);
        }
        crate::set_stylus_keepalive_request(Some(codehash));
    } else {
        crate::charge_precompile_gas(SLOAD_GAS);
        crate::set_stylus_keepalive_request(Some(codehash));
        crate::set_stylus_activation_data_fee(data_fee);
    }

    let event_topic = IArbWasm::ProgramLifetimeExtended::SIGNATURE_HASH;
    let mut event_data = Vec::with_capacity(32);
    event_data.extend_from_slice(&data_fee.to_be_bytes::<32>());
    let event_gas = 375 + 2 * 375 + 8 * event_data.len() as u64;
    crate::charge_precompile_gas(event_gas);
    input.internals_mut().log(Log::new_unchecked(
        ARBWASM_ADDRESS,
        vec![event_topic, codehash],
        event_data.into(),
    ));

    // No return value for keepalive
    let gas_used = crate::get_precompile_gas();
    Ok(PrecompileOutput::new(gas_used, Vec::new().into()))
}

#[cfg(test)]
mod failure_gas_tests {
    use super::*;

    fn unactivated() -> ProgramInfo {
        ProgramInfo {
            version: 0,
            init_cost: 0,
            cached_cost: 0,
            footprint: 0,
            asm_estimate_kb: 0,
            age_seconds: 0,
        }
    }

    fn revert_gas(lookup_gas: u64) -> u64 {
        let r = validate_active_program(&unactivated(), 1, 365, 1_000_000, lookup_gas)
            .expect_err("unactivated program should revert");
        let out = r.expect("revert wraps an Ok(PrecompileOutput)");
        out.gas_used
    }

    #[test]
    fn codehash_asm_size_failure_charges_lookup_plus_error_word() {
        // lookup = argsCost(3) + Open(800) + Params warm(100) + getProgram SLOAD(800) = 1703;
        // ProgramNotActivated() = 4 bytes = 1 word → +3.
        const LOOKUP_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + SLOAD_GAS + COPY_GAS;
        assert_eq!(LOOKUP_GAS, 1703);
        assert_eq!(revert_gas(LOOKUP_GAS), 1706);
    }

    #[test]
    fn program_version_family_failure_charges_lookup_plus_error_word() {
        // lookup = framework(803) + Params warm(100) + GetCodeHash(2600) + getProgram(800) = 4303;
        // ProgramNotActivated() = 4 bytes = 1 word → +3.
        assert_eq!(PROGRAM_LOOKUP_GAS, 4303);
        assert_eq!(revert_gas(PROGRAM_LOOKUP_GAS), 4306);
    }

    #[test]
    fn revert_is_capped_at_gas_limit() {
        let r = validate_active_program(&unactivated(), 1, 365, 500, 1703)
            .expect_err("unactivated program should revert");
        let out = r.expect("revert wraps an Ok(PrecompileOutput)");
        assert_eq!(out.gas_used, 500);
    }
}
