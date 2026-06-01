use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, Log, B256, U256};
use alloy_sol_types::{SolError, SolEvent, SolInterface};
use arb_context::ArbPrecompileCtx;
use arb_storage::ARBOS_STATE_ADDRESS;
use arbos::programs::{params::StylusParams, Program};
use revm::precompile::{PrecompileId, PrecompileOutput, PrecompileResult};
use std::sync::Arc;

use crate::{
    interfaces::{IArbWasm, IArbWasmCache},
    ArbPrecompileError,
};

/// ArbWasmCache precompile address (0x72).
pub const ARBWASMCACHE_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x72,
]);

const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;

const WARM_SLOAD_GAS: u64 = 100;
const COLD_ACCOUNT_ACCESS_GAS: u64 = 2600;
const SSTORE_SET_GAS: u64 = 20_000;
const SSTORE_RESET_GAS: u64 = 5_000;

/// LOG3 for UpdateProgramCache(address,bytes32,bool):
/// base 375 + 3 topics * 375 + 32 bytes data * 8.
const EMIT_UPDATE_PROGRAM_CACHE_GAS: u64 = 375 + 3 * 375 + 32 * 8;

pub fn create_arbwasmcache_precompile(ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbwasmcache"), move |input| {
        handler(input, &ctx)
    })
}

fn handler(mut input: PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    if let Some(result) =
        crate::check_precompile_version(ctx, arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS)
    {
        return result;
    }

    let mut gas_used = 0u64;
    let gas_limit = input.gas;
    crate::init_precompile_gas(&mut gas_used, ctx, input.data.len());

    let call = match IArbWasmCache::ArbWasmCacheCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbWasmCache::ArbWasmCacheCalls;
    let result = match call {
        ArbWasmCacheCalls::cacheCodehash(c) => {
            handle_cache_codehash(&mut input, ctx, &mut gas_used, c.codehash)
        }
        ArbWasmCacheCalls::cacheProgram(c) => {
            handle_cache_program(&mut input, ctx, &mut gas_used, c.addr)
        }
        ArbWasmCacheCalls::evictCodehash(c) => {
            handle_evict_codehash(&mut input, ctx, &mut gas_used, c.codehash)
        }
        ArbWasmCacheCalls::isCacheManager(c) => {
            handle_is_cache_manager(&mut input, &mut gas_used, c.manager, ctx)
        }
        ArbWasmCacheCalls::allCacheManagers(_) => {
            handle_all_cache_managers(&mut input, &mut gas_used, ctx)
        }
        ArbWasmCacheCalls::codehashIsCached(c) => {
            handle_codehash_is_cached(&mut input, &mut gas_used, ctx, c.codehash)
        }
    };
    crate::gas_check(ctx, gas_limit, gas_used, result)
}

fn words_for_bytes(n: u64) -> u64 {
    n.div_ceil(32)
}

// ── Helpers ──────────────────────────────────────────────────────────

fn load_arbos(input: &mut PrecompileInput<'_>) -> Result<(), ArbPrecompileError> {
    input
        .internals_mut()
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;
    Ok(())
}

fn handle_is_cache_manager(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    addr: Address,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let is_member = arb_state
        .programs
        .cache_managers
        .is_member(internals, addr)
        .map_err(ArbPrecompileError::fatal)?;

    let result = if is_member {
        U256::from(1u64)
    } else {
        U256::ZERO
    };
    crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);
    crate::charge_computation(gas_used, ctx, COPY_GAS * words_for_bytes(32));
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        result.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Return all cache manager addresses.
fn handle_all_cache_managers(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let members = arb_state
        .programs
        .cache_managers
        .all_members(internals, 256)
        .map_err(ArbPrecompileError::fatal)?;
    let count = members.len() as u64;
    let sloads: u64 = 1 + count;

    let mut out = Vec::with_capacity(64 + members.len() * 32);
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(count).to_be_bytes::<32>());
    for member in &members {
        let mut word = [0u8; 32];
        word[12..32].copy_from_slice(member.as_slice());
        out.extend_from_slice(&word);
    }

    crate::charge_storage_read(gas_used, ctx, sloads * SLOAD_GAS);
    crate::charge_computation(gas_used, ctx, COPY_GAS * words_for_bytes(out.len() as u64));
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        out.into(),
    ))
}

fn handle_codehash_is_cached(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
    codehash: B256,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let time = ctx.block.block_timestamp;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let program = arb_state
        .programs
        .get_program(internals, codehash, time)
        .map_err(ArbPrecompileError::fatal)?;

    let result = if program.cached {
        U256::from(1u64)
    } else {
        U256::ZERO
    };
    crate::charge_storage_read(gas_used, ctx, SLOAD_GAS);
    crate::charge_computation(gas_used, ctx, COPY_GAS * words_for_bytes(32));
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        result.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Caller must be a cache manager OR chain owner. Returns `(has_access, gas)`:
/// `gas` is 1 SLOAD if the caller is a cache manager (short-circuit), else
/// 2 SLOADs (cache-managers probe then chain-owners probe).
fn caller_has_cache_access(
    input: &mut PrecompileInput<'_>,
    caller: Address,
    ctx: &ArbPrecompileCtx,
) -> Result<(bool, u64), ArbPrecompileError> {
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    if arb_state
        .programs
        .cache_managers
        .is_member(internals, caller)
        .map_err(ArbPrecompileError::fatal)?
    {
        return Ok((true, SLOAD_GAS));
    }
    let is_owner = arb_state
        .chain_owners
        .is_member(internals, caller)
        .map_err(ArbPrecompileError::fatal)?;
    Ok((is_owner, 2 * SLOAD_GAS))
}

fn read_params_and_program(
    input: &mut PrecompileInput<'_>,
    codehash: B256,
    time: u64,
    ctx: &ArbPrecompileCtx,
) -> Result<(StylusParams, Program), ArbPrecompileError> {
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
    Ok((params, program))
}

/// `pre_set_gas` lets the caller include an extra cold-account-access charge
/// that must be paid on every exit path (e.g., `cacheProgram`'s GetCodeHash).
fn set_program_cached(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    gas_used: &mut u64,
    codehash: B256,
    cache: bool,
    pre_set_gas: u64,
) -> PrecompileResult {
    let caller = input.caller;
    let now = ctx.block.block_timestamp;
    let gas_limit = input.gas;

    // `pre_set_gas` is `ColdAccountAccessCostEIP2929` (an account read) for
    // `cacheProgram`'s GetCodeHash, and zero otherwise — both are read-class.
    crate::charge_storage_read(gas_used, ctx, pre_set_gas);

    load_arbos(input)?;

    let (has_access, access_gas) = caller_has_cache_access(input, caller, ctx)?;
    crate::charge_storage_read(gas_used, ctx, access_gas);
    if !has_access {
        return crate::burn_all_revert(gas_limit);
    }

    let (params, mut program) = read_params_and_program(input, codehash, now, ctx)?;
    // `programs.params` (one SLOAD) + `get_program` (one SLOAD).
    crate::charge_storage_read(gas_used, ctx, SLOAD_GAS + SLOAD_GAS);
    let already_cached = program.cached;
    let expiry_seconds = (params.expiry_days as u64).saturating_mul(86_400);
    let expired = program.age_seconds > expiry_seconds;

    if cache && program.version != params.version {
        let data = IArbWasm::ProgramNeedsUpgrade {
            version: program.version,
            stylusVersion: params.version,
        }
        .abi_encode();
        return crate::sol_error_revert(gas_used, ctx, data, gas_limit);
    }
    if cache && expired {
        let data = IArbWasm::ProgramExpired {
            ageInSeconds: program.age_seconds,
        }
        .abi_encode();
        return crate::sol_error_revert(gas_used, ctx, data, gas_limit);
    }
    if already_cached == cache {
        // Original formula's pre-set-write tail: WARM_SLOAD + SLOAD, both Read.
        crate::charge_storage_read(gas_used, ctx, WARM_SLOAD_GAS + SLOAD_GAS);
        return Ok(PrecompileOutput::new(
            (*gas_used).min(gas_limit),
            Vec::new().into(),
        ));
    }

    program.cached = cache;
    let prog_init_cost = program.init_cost;
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
    let stored = program.to_storage();
    let stored_u = U256::from_be_bytes(stored.0);
    let sstore_gas = if stored_u == U256::ZERO {
        SSTORE_RESET_GAS
    } else {
        SSTORE_SET_GAS
    };

    let topic1 = address_to_b256(caller);
    let event_data = U256::from(cache as u64).to_be_bytes::<32>().to_vec();
    input.internals_mut().log(Log::new_unchecked(
        ARBWASMCACHE_ADDRESS,
        vec![
            IArbWasmCache::UpdateProgramCache::SIGNATURE_HASH,
            topic1,
            codehash,
        ],
        event_data.into(),
    ));

    // Tail: WARM_SLOAD + SLOAD (reads), prog_init_cost (compute warm-up), emit
    // log (HistoryGrowth), final SLOAD + SSTORE for the program write.
    crate::charge_storage_read(gas_used, ctx, WARM_SLOAD_GAS + SLOAD_GAS + SLOAD_GAS);
    crate::charge_computation(gas_used, ctx, prog_init_cost as u64);
    crate::charge_history_growth(gas_used, ctx, EMIT_UPDATE_PROGRAM_CACHE_GAS);
    crate::charge_storage_write(gas_used, ctx, sstore_gas);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        Vec::new().into(),
    ))
}

fn address_to_b256(addr: Address) -> B256 {
    let mut bytes = [0u8; 32];
    bytes[12..32].copy_from_slice(addr.as_slice());
    B256::from(bytes)
}

fn handle_cache_codehash(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    gas_used: &mut u64,
    codehash: B256,
) -> PrecompileResult {
    if let Some(r) = crate::check_method_version(
        ctx,
        input.gas,
        arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS,
        arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS,
    ) {
        return r;
    }
    set_program_cached(input, ctx, gas_used, codehash, true, 0)
}

/// `cacheProgram` reads the code hash from an account, which costs
/// `ColdAccountAccessCostEIP2929` even when the slot is already warm.
fn handle_cache_program(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    gas_used: &mut u64,
    addr: Address,
) -> PrecompileResult {
    if let Some(r) = crate::check_method_version(
        ctx,
        input.gas,
        arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS_FIXES,
        0,
    ) {
        return r;
    }
    let codehash = crate::without_access_list_effect(input.internals_mut(), |internals| {
        internals
            .load_account(addr)
            .map(|acct| acct.data.info.code_hash)
            .map_err(ArbPrecompileError::fatal)
    })?;
    set_program_cached(
        input,
        ctx,
        gas_used,
        codehash,
        true,
        COLD_ACCOUNT_ACCESS_GAS,
    )
}

fn handle_evict_codehash(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    gas_used: &mut u64,
    codehash: B256,
) -> PrecompileResult {
    set_program_cached(input, ctx, gas_used, codehash, false, 0)
}
