use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, Log, B256, U256};
use alloy_sol_types::{SolError, SolEvent, SolInterface};
use arb_context::ArbPrecompileCtx;
use arb_storage::ARBOS_STATE_ADDRESS;
use arbos::programs::{hours_since_arbitrum, hours_to_age, params::StylusParams, Program};
use revm::precompile::{PrecompileId, PrecompileOutput, PrecompileResult};
use std::sync::Arc;

use crate::{interfaces::IArbWasm, ArbPrecompileError};

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
const FRAMEWORK_GAS_PROGRAM_ADDR: u64 = COPY_GAS + 800;
const PROGRAM_LOOKUP_GAS: u64 =
    FRAMEWORK_GAS_PROGRAM_ADDR + WARM_SLOAD_GAS + STORAGE_CODE_HASH_COST + SLOAD_GAS;

const MIN_INIT_GAS_UNITS: u64 = 128;
const MIN_CACHED_GAS_UNITS: u64 = 32;
const COST_SCALAR_PERCENT: u64 = 2;

pub fn create_arbwasm_precompile(ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbwasm"), move |input| {
        handler(input, &ctx)
    })
}

fn handler(mut input: PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    if let Some(result) =
        crate::check_precompile_version(ctx, arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS)
    {
        return result;
    }

    let gas_limit = input.gas;

    let call = match IArbWasm::ArbWasmCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbWasm::ArbWasmCalls as Calls;
    match &call {
        Calls::activateProgram(c) => return handle_activate_program(input, ctx, c.program),
        Calls::codehashKeepalive(c) => return handle_codehash_keepalive(input, ctx, c.codehash),
        _ => {}
    }

    let mut gas_used = 0u64;
    crate::init_precompile_gas(&mut gas_used, input.data.len());

    let result = match call {
        Calls::stylusVersion(_) => {
            let params = load_params(&mut input, &mut gas_used, ctx)?;
            ok_u256(
                SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS,
                U256::from(params.version),
            )
        }
        Calls::inkPrice(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params(&mut input, &mut gas_used, ctx)?;
            ok_u256(METHOD_GAS, U256::from(params.ink_price))
        }
        Calls::maxStackDepth(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params(&mut input, &mut gas_used, ctx)?;
            ok_u256(METHOD_GAS, U256::from(params.max_stack_depth))
        }
        Calls::freePages(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params(&mut input, &mut gas_used, ctx)?;
            ok_u256(METHOD_GAS, U256::from(params.free_pages))
        }
        Calls::pageGas(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params(&mut input, &mut gas_used, ctx)?;
            ok_u256(METHOD_GAS, U256::from(params.page_gas))
        }
        Calls::pageRamp(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params(&mut input, &mut gas_used, ctx)?;
            ok_u256(METHOD_GAS, U256::from(params.page_ramp))
        }
        Calls::pageLimit(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params(&mut input, &mut gas_used, ctx)?;
            ok_u256(METHOD_GAS, U256::from(params.page_limit))
        }
        Calls::minInitGas(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + 2 * COPY_GAS;
            let params = load_params(&mut input, &mut gas_used, ctx)?;
            if ctx.block.arbos_version
                < arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS_CHARGING_FIXES
            {
                let pre_revert_gas = (SLOAD_GAS + WARM_SLOAD_GAS).min(input.gas);
                return Ok(PrecompileOutput::new_reverted(
                    pre_revert_gas,
                    Default::default(),
                ));
            }
            let init = (params.min_init_gas as u64).saturating_mul(MIN_INIT_GAS_UNITS);
            let cached = (params.min_cached_init_gas as u64).saturating_mul(MIN_CACHED_GAS_UNITS);
            ok_two_u256(METHOD_GAS, U256::from(init), U256::from(cached))
        }
        Calls::initCostScalar(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params(&mut input, &mut gas_used, ctx)?;
            let scalar = params.init_cost_scalar as u64;
            ok_u256(
                METHOD_GAS,
                U256::from(scalar.saturating_mul(COST_SCALAR_PERCENT)),
            )
        }
        Calls::expiryDays(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params(&mut input, &mut gas_used, ctx)?;
            ok_u256(METHOD_GAS, U256::from(params.expiry_days))
        }
        Calls::keepaliveDays(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params(&mut input, &mut gas_used, ctx)?;
            ok_u256(METHOD_GAS, U256::from(params.keepalive_days))
        }
        Calls::blockCacheSize(_) => {
            const METHOD_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + COPY_GAS;
            let params = load_params(&mut input, &mut gas_used, ctx)?;
            ok_u256(METHOD_GAS, U256::from(params.block_cache_size))
        }
        Calls::activationGas(_) => {
            if let Some(r) = crate::check_method_version(
                ctx,
                input.gas,
                arb_chainspec::arbos_version::ARBOS_VERSION_59,
                0,
            ) {
                return r;
            }
            load_arbos(&mut input)?;
            let internals = input.internals_mut();
            let arb_state = ctx
                .block
                .arbos_state(internals)
                .map_err(ArbPrecompileError::fatal)?;
            let gas = arb_state
                .programs
                .activation_gas(internals)
                .map_err(ArbPrecompileError::fatal)?;
            crate::charge_precompile_gas(&mut gas_used, SLOAD_GAS);
            ok_u256(SLOAD_GAS + SLOAD_GAS + COPY_GAS, U256::from(gas))
        }
        Calls::codehashVersion(c) => {
            const LOOKUP_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + SLOAD_GAS + COPY_GAS;
            const METHOD_GAS: u64 = LOOKUP_GAS + COPY_GAS;
            let (params, program) =
                load_params_and_program(&mut input, ctx, &mut gas_used, c.codehash)?;
            if let Err(r) = validate_active_program(
                &program,
                params.version,
                params.expiry_days,
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
            let (params, program) =
                load_params_and_program(&mut input, ctx, &mut gas_used, c.codehash)?;
            if let Err(r) = validate_active_program(
                &program,
                params.version,
                params.expiry_days,
                input.gas,
                LOOKUP_GAS,
            ) {
                return r;
            }
            ok_u256(METHOD_GAS, U256::from(program.asm_size()))
        }
        Calls::programVersion(c) => {
            const METHOD_GAS: u64 = PROGRAM_LOOKUP_GAS + COPY_GAS;
            let codehash = get_account_codehash(&mut input, c.program)?;
            let (params, program) =
                load_params_and_program(&mut input, ctx, &mut gas_used, codehash)?;
            if let Err(r) = validate_active_program(
                &program,
                params.version,
                params.expiry_days,
                input.gas,
                PROGRAM_LOOKUP_GAS,
            ) {
                return r;
            }
            ok_u256(METHOD_GAS, U256::from(program.version))
        }
        Calls::programInitGas(c) => {
            const METHOD_GAS: u64 = PROGRAM_LOOKUP_GAS + 2 * COPY_GAS;
            let codehash = get_account_codehash(&mut input, c.program)?;
            let (params, program) =
                load_params_and_program(&mut input, ctx, &mut gas_used, codehash)?;
            if let Err(r) = validate_active_program(
                &program,
                params.version,
                params.expiry_days,
                input.gas,
                PROGRAM_LOOKUP_GAS,
            ) {
                return r;
            }

            let mut init_gas = program.init_gas(&params);
            let cached_gas = program.cached_gas(&params);
            if params.version > 1 {
                init_gas = init_gas.saturating_add(cached_gas);
            }

            ok_two_u256(METHOD_GAS, U256::from(init_gas), U256::from(cached_gas))
        }
        Calls::programMemoryFootprint(c) => {
            const METHOD_GAS: u64 = PROGRAM_LOOKUP_GAS + COPY_GAS;
            let codehash = get_account_codehash(&mut input, c.program)?;
            let (params, program) =
                load_params_and_program(&mut input, ctx, &mut gas_used, codehash)?;
            if let Err(r) = validate_active_program(
                &program,
                params.version,
                params.expiry_days,
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
            let (params, program) =
                load_params_and_program(&mut input, ctx, &mut gas_used, codehash)?;
            if let Err(r) = validate_active_program(
                &program,
                params.version,
                params.expiry_days,
                input.gas,
                PROGRAM_LOOKUP_GAS,
            ) {
                return r;
            }

            let expiry_seconds = (params.expiry_days as u64) * 24 * 3600;
            let time_left = expiry_seconds.saturating_sub(program.age_seconds);
            ok_u256(METHOD_GAS, U256::from(time_left))
        }
        Calls::activateProgram(_) | Calls::codehashKeepalive(_) => unreachable!(),
    };
    crate::gas_check(ctx, input.gas, gas_used, result)
}

// ── Helpers ──────────────────────────────────────────────────────────

fn load_arbos(input: &mut PrecompileInput<'_>) -> Result<(), ArbPrecompileError> {
    input
        .internals_mut()
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;
    Ok(())
}

/// Load the active Stylus parameters and charge the params SLOAD (800 + warm).
fn load_params(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
) -> Result<StylusParams, ArbPrecompileError> {
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let params = arb_state
        .programs
        .params(internals)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SLOAD_GAS + WARM_SLOAD_GAS);
    Ok(params)
}

fn load_params_and_program(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    gas_used: &mut u64,
    codehash: B256,
) -> Result<(StylusParams, Program), ArbPrecompileError> {
    load_arbos(input)?;
    let time = ctx.block.block_timestamp;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let params = arb_state
        .programs
        .params(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let program = arb_state
        .programs
        .get_program(internals, codehash, time)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SLOAD_GAS + SLOAD_GAS);
    Ok((params, program))
}

/// Get the code hash for an account address.
fn get_account_codehash(
    input: &mut PrecompileInput<'_>,
    address: Address,
) -> Result<B256, ArbPrecompileError> {
    let account = input
        .internals_mut()
        .load_account(address)
        .map_err(ArbPrecompileError::fatal)?;
    Ok(account.data.info.code_hash)
}

/// Returns ProgramNotActivated, ProgramNeedsUpgrade(progV, paramsV),
/// or ProgramExpired(ageSeconds), in that order. `lookup_gas` is the
/// method's argsCost + state-access charges (everything except the
/// result-copy cost). The revert charges `lookup_gas + COPY_GAS * words`
/// where `words` is rounded up from the actual error payload length.
fn validate_active_program(
    program: &Program,
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

fn revert_sol_error(gas_used: &mut u64, payload: Vec<u8>, input_gas: u64) -> PrecompileResult {
    crate::charge_precompile_gas(gas_used, COPY_GAS * (payload.len() as u64).div_ceil(32));
    if *gas_used > input_gas {
        return Err(ArbPrecompileError::OutOfGas.into());
    }
    Ok(PrecompileOutput::new_reverted(*gas_used, payload.into()))
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

fn handle_activate_program(
    mut input: PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    program_address: Address,
) -> PrecompileResult {
    const ACTIVATION_UPFRONT_GAS: u64 = 1_659_168;

    let mut gas_used = 0u64;
    let args_cost = COPY_GAS * (input.data.len() as u64).saturating_sub(4).div_ceil(32);
    crate::charge_precompile_gas(&mut gas_used, args_cost);
    crate::charge_precompile_gas(&mut gas_used, SLOAD_GAS);

    if ctx.block.arbos_version >= arb_chainspec::arbos_version::ARBOS_VERSION_60 {
        load_arbos(&mut input)?;
        let internals = input.internals_mut();
        let arb_state = ctx
            .block
            .arbos_state(internals)
            .map_err(ArbPrecompileError::fatal)?;
        let _activation_gas = arb_state
            .programs
            .activation_gas(internals)
            .map_err(ArbPrecompileError::fatal)?;
        crate::charge_precompile_gas(&mut gas_used, SLOAD_GAS);
    }

    crate::charge_precompile_gas(&mut gas_used, ACTIVATION_UPFRONT_GAS);

    let code_hash = {
        let account = input
            .internals_mut()
            .load_account(program_address)
            .map_err(ArbPrecompileError::fatal)?;
        account.data.info.code_hash
    };

    let code_bytes = {
        let code_account = input
            .internals_mut()
            .load_account_code(program_address)
            .map_err(ArbPrecompileError::fatal)?;
        code_account
            .data
            .code()
            .map(|c| c.original_bytes())
            .unwrap_or_default()
            .to_vec()
    };

    load_arbos(&mut input)?;
    let time = ctx.block.block_timestamp;
    crate::charge_precompile_gas(&mut gas_used, WARM_SLOAD_GAS);
    let (params, existing_program) = {
        let internals = input.internals_mut();
        let arb_state = ctx
            .block
            .arbos_state(internals)
            .map_err(ArbPrecompileError::fatal)?;
        let params = arb_state
            .programs
            .params(internals)
            .map_err(ArbPrecompileError::fatal)?;
        let existing = arb_state
            .programs
            .get_program(internals, code_hash, time)
            .map_err(ArbPrecompileError::fatal)?;
        (params, existing)
    };
    crate::charge_precompile_gas(&mut gas_used, SLOAD_GAS);

    if code_bytes.is_empty() {
        return revert_sol_error(
            &mut gas_used,
            IArbWasm::ProgramNotWasm {}.abi_encode(),
            input.gas,
        );
    }
    if !arb_stylus::is_stylus_deployable(&code_bytes, ctx.block.arbos_version) {
        let arbos_v = ctx.block.arbos_version;
        if arbos_v < arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS_CONTRACT_LIMIT {
            return Ok(PrecompileOutput::new_reverted(
                gas_used.min(input.gas),
                Default::default(),
            ));
        }
        return revert_sol_error(
            &mut gas_used,
            IArbWasm::ProgramNotWasm {}.abi_encode(),
            input.gas,
        );
    }

    let wasm = match arb_stylus::decompress_wasm(&code_bytes) {
        Ok(w) => w,
        Err(_) => {
            return revert_sol_error(
                &mut gas_used,
                IArbWasm::ProgramNotWasm {}.abi_encode(),
                input.gas,
            )
        }
    };

    let was_cached = existing_program.cached;

    if existing_program.version == params.version {
        let age = existing_program.age_seconds;
        if age <= (params.expiry_days as u64) * 86400 {
            return revert_sol_error(
                &mut gas_used,
                IArbWasm::ProgramUpToDate {}.abi_encode(),
                input.gas,
            );
        }
    }

    let gas_available = input.gas.saturating_sub(gas_used);
    let mut gas_for_prover = gas_available;
    let diag_pre_prover = gas_used;
    let diag_gas_to_prover = gas_for_prover;

    let info = match arb_stylus::activate_program(
        &wasm,
        code_hash.as_ref(),
        params.version,
        ctx.block.arbos_version,
        params.page_limit,
        false,
        &mut gas_for_prover,
    ) {
        Ok(info) => info,
        Err(_) => {
            crate::charge_precompile_gas(&mut gas_used, gas_available);
            return Err(ArbPrecompileError::empty_revert(gas_used).into());
        }
    };

    let prover_gas_used = gas_available.saturating_sub(gas_for_prover);
    crate::charge_precompile_gas(&mut gas_used, prover_gas_used);
    let wasm_hash = alloy_primitives::keccak256(&wasm);
    tracing::warn!(target: "stylus",
        input_gas = input.gas, pre_prover = diag_pre_prover, to_prover = diag_gas_to_prover,
        prover_used = prover_gas_used, after_prover = gas_used,
        wasm_len = wasm.len(), %wasm_hash, "activateProgram gas breakdown");

    {
        let internals = input.internals_mut();
        let arb_state = ctx
            .block
            .arbos_state(internals)
            .map_err(ArbPrecompileError::fatal)?;
        arb_state
            .programs
            .set_module_hash(internals, code_hash, info.module_hash)
            .map_err(ArbPrecompileError::fatal)?;
    }
    crate::charge_precompile_gas(&mut gas_used, SSTORE_GAS);

    let data_fee = {
        let internals = input.internals_mut();
        let arb_state = ctx
            .block
            .arbos_state(internals)
            .map_err(ArbPrecompileError::fatal)?;
        arb_state
            .programs
            .data_pricer
            .update_model(internals, info.asm_estimate, time)
            .map_err(ArbPrecompileError::fatal)?
    };
    crate::charge_precompile_gas(&mut gas_used, 5 * SLOAD_GAS + 2 * SSTORE_GAS);

    let estimate_kb = div_ceil(info.asm_estimate as u64, 1024).min(0xFF_FFFF) as u32;
    let new_program = Program {
        version: params.version,
        init_cost: info.init_gas,
        cached_cost: info.cached_init_gas,
        footprint: info.footprint,
        asm_estimate_kb: estimate_kb,
        activated_at: hours_since_arbitrum(time),
        age_seconds: 0,
        cached: was_cached,
    };
    {
        let internals = input.internals_mut();
        let arb_state = ctx
            .block
            .arbos_state(internals)
            .map_err(ArbPrecompileError::fatal)?;
        arb_state
            .programs
            .set_program(internals, code_hash, new_program)
            .map_err(ArbPrecompileError::fatal)?;
    }
    crate::charge_precompile_gas(&mut gas_used, SSTORE_GAS);

    let stashed_outer_value = ctx.stylus_call_value();
    let inner_call_value = input.value;
    let effective_value = if inner_call_value > U256::ZERO {
        inner_call_value
    } else {
        stashed_outer_value
    };
    if effective_value < data_fee {
        return revert_sol_error(
            &mut gas_used,
            IArbWasm::ProgramInsufficientValue {
                have: effective_value,
                want: data_fee,
            }
            .abi_encode(),
            input.gas,
        );
    }

    if inner_call_value > U256::ZERO {
        let caller = input.caller;
        let network_addr = {
            let internals = input.internals_mut();
            let arb_state = ctx
                .block
                .arbos_state(internals)
                .map_err(ArbPrecompileError::fatal)?;
            arb_state
                .network_fee_account(internals)
                .map_err(ArbPrecompileError::fatal)?
        };
        crate::charge_precompile_gas(&mut gas_used, SLOAD_GAS);
        let _ = input
            .internals_mut()
            .transfer(ARBWASM_ADDRESS, network_addr, data_fee);
        let repay = inner_call_value.saturating_sub(data_fee);
        if repay > U256::ZERO {
            let _ = input
                .internals_mut()
                .transfer(ARBWASM_ADDRESS, caller, repay);
        }
        ctx.set_stylus_activation_addr(Some(program_address));
    } else {
        crate::charge_precompile_gas(&mut gas_used, SLOAD_GAS);
        ctx.set_stylus_activation_addr(Some(program_address));
        ctx.set_stylus_activation_data_fee(data_fee);
    }

    let event_topic = IArbWasm::ProgramActivated::SIGNATURE_HASH;
    let mut event_data = Vec::with_capacity(128);
    event_data.extend_from_slice(&info.module_hash.0);
    event_data.extend_from_slice(&[0u8; 12]);
    event_data.extend_from_slice(program_address.as_slice());
    event_data.extend_from_slice(&data_fee.to_be_bytes::<32>());
    let mut ver = [0u8; 32];
    ver[30..32].copy_from_slice(&params.version.to_be_bytes());
    event_data.extend_from_slice(&ver);
    let event_gas = 375 + 2 * 375 + 8 * event_data.len() as u64;
    crate::charge_precompile_gas(&mut gas_used, event_gas);
    input.internals_mut().log(Log::new_unchecked(
        ARBWASM_ADDRESS,
        vec![event_topic, code_hash],
        event_data.into(),
    ));

    let return_data = {
        let mut output = Vec::with_capacity(64);
        let mut ver_out = [0u8; 32];
        ver_out[30..32].copy_from_slice(&params.version.to_be_bytes());
        output.extend_from_slice(&ver_out);
        output.extend_from_slice(&data_fee.to_be_bytes::<32>());
        output
    };
    let return_gas = COPY_GAS * (return_data.len() as u64).div_ceil(32);
    crate::charge_precompile_gas(&mut gas_used, return_gas);

    tracing::warn!(target: "stylus",
        total = gas_used, args = args_cost, prover = prover_gas_used,
        event = event_gas, ret = return_gas, "activateProgram total gas");
    if gas_used > input.gas {
        return Err(ArbPrecompileError::OutOfGas.into());
    }
    Ok(PrecompileOutput::new(gas_used, return_data.into()))
}

fn handle_codehash_keepalive(
    mut input: PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    codehash: B256,
) -> PrecompileResult {
    let mut gas_used = 0u64;
    let args_cost = COPY_GAS * (input.data.len() as u64).saturating_sub(4).div_ceil(32);
    crate::charge_precompile_gas(&mut gas_used, args_cost);

    load_arbos(&mut input)?;
    let time = ctx.block.block_timestamp;
    let (params, mut program) = {
        let internals = input.internals_mut();
        let arb_state = ctx
            .block
            .arbos_state(internals)
            .map_err(ArbPrecompileError::fatal)?;
        let params = arb_state
            .programs
            .params(internals)
            .map_err(ArbPrecompileError::fatal)?;
        let program = arb_state
            .programs
            .get_program(internals, codehash, time)
            .map_err(ArbPrecompileError::fatal)?;
        (params, program)
    };
    crate::charge_precompile_gas(&mut gas_used, SLOAD_GAS + WARM_SLOAD_GAS + SLOAD_GAS);

    if program.version == 0 {
        return revert_sol_error(
            &mut gas_used,
            IArbWasm::ProgramNotActivated {}.abi_encode(),
            input.gas,
        );
    }
    let age = hours_to_age(time, program.activated_at);
    if age > (params.expiry_days as u64) * 86400 {
        return revert_sol_error(
            &mut gas_used,
            IArbWasm::ProgramExpired { ageInSeconds: age }.abi_encode(),
            input.gas,
        );
    }
    if program.version != params.version {
        return revert_sol_error(
            &mut gas_used,
            IArbWasm::ProgramNeedsUpgrade {
                version: program.version,
                stylusVersion: params.version,
            }
            .abi_encode(),
            input.gas,
        );
    }
    if age < (params.keepalive_days as u64) * 86400 {
        return revert_sol_error(
            &mut gas_used,
            IArbWasm::ProgramKeepaliveTooSoon { ageInSeconds: age }.abi_encode(),
            input.gas,
        );
    }

    let asm_size = program.asm_size();

    let data_fee = {
        let internals = input.internals_mut();
        let arb_state = ctx
            .block
            .arbos_state(internals)
            .map_err(ArbPrecompileError::fatal)?;
        arb_state
            .programs
            .data_pricer
            .update_model(internals, asm_size, time)
            .map_err(ArbPrecompileError::fatal)?
    };
    crate::charge_precompile_gas(&mut gas_used, 5 * SLOAD_GAS + 2 * SSTORE_GAS);

    program.activated_at = hours_since_arbitrum(time);
    program.age_seconds = 0;
    {
        let internals = input.internals_mut();
        let arb_state = ctx
            .block
            .arbos_state(internals)
            .map_err(ArbPrecompileError::fatal)?;
        arb_state
            .programs
            .set_program(internals, codehash, program)
            .map_err(ArbPrecompileError::fatal)?;
    }
    crate::charge_precompile_gas(&mut gas_used, SSTORE_GAS);

    let stashed_outer_value = ctx.stylus_call_value();
    let inner_call_value = input.value;
    let effective_value = if inner_call_value > U256::ZERO {
        inner_call_value
    } else {
        stashed_outer_value
    };
    if effective_value < data_fee {
        return revert_sol_error(
            &mut gas_used,
            IArbWasm::ProgramInsufficientValue {
                have: effective_value,
                want: data_fee,
            }
            .abi_encode(),
            input.gas,
        );
    }

    if inner_call_value > U256::ZERO {
        let caller = input.caller;
        let network_addr = {
            let internals = input.internals_mut();
            let arb_state = ctx
                .block
                .arbos_state(internals)
                .map_err(ArbPrecompileError::fatal)?;
            arb_state
                .network_fee_account(internals)
                .map_err(ArbPrecompileError::fatal)?
        };
        crate::charge_precompile_gas(&mut gas_used, SLOAD_GAS);
        let _ = input
            .internals_mut()
            .transfer(ARBWASM_ADDRESS, network_addr, data_fee);
        let repay = inner_call_value.saturating_sub(data_fee);
        if repay > U256::ZERO {
            let _ = input
                .internals_mut()
                .transfer(ARBWASM_ADDRESS, caller, repay);
        }
        ctx.set_stylus_keepalive_hash(Some(codehash));
    } else {
        crate::charge_precompile_gas(&mut gas_used, SLOAD_GAS);
        ctx.set_stylus_keepalive_hash(Some(codehash));
        ctx.set_stylus_activation_data_fee(data_fee);
    }

    let event_topic = IArbWasm::ProgramLifetimeExtended::SIGNATURE_HASH;
    let mut event_data = Vec::with_capacity(32);
    event_data.extend_from_slice(&data_fee.to_be_bytes::<32>());
    let event_gas = 375 + 2 * 375 + 8 * event_data.len() as u64;
    crate::charge_precompile_gas(&mut gas_used, event_gas);
    input.internals_mut().log(Log::new_unchecked(
        ARBWASM_ADDRESS,
        vec![event_topic, codehash],
        event_data.into(),
    ));

    if gas_used > input.gas {
        return Err(ArbPrecompileError::OutOfGas.into());
    }
    Ok(PrecompileOutput::new(gas_used, Vec::new().into()))
}

#[cfg(test)]
mod failure_gas_tests {
    use super::*;

    fn unactivated() -> Program {
        Program {
            version: 0,
            init_cost: 0,
            cached_cost: 0,
            footprint: 0,
            asm_estimate_kb: 0,
            activated_at: 0,
            age_seconds: 0,
            cached: false,
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
        const LOOKUP_GAS: u64 = SLOAD_GAS + WARM_SLOAD_GAS + SLOAD_GAS + COPY_GAS;
        assert_eq!(LOOKUP_GAS, 1703);
        assert_eq!(revert_gas(LOOKUP_GAS), 1706);
    }

    #[test]
    fn program_version_family_failure_charges_lookup_plus_error_word() {
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
