use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use alloy_sol_types::SolInterface;
use arb_context::ArbPrecompileCtx;
use arb_storage::ARBOS_STATE_ADDRESS;

use revm::precompile::{PrecompileId, PrecompileOutput, PrecompileResult};
use std::sync::Arc;

use crate::{interfaces::IArbAggregator, ArbPrecompileError};

/// ArbAggregator precompile address (0x6d).
pub const ARBAGGREGATOR_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x6d,
]);

/// Default batch poster address (the sequencer).
const BATCH_POSTER_ADDRESS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65, 0x71, 0x75, 0x65,
    0x6e, 0x63, 0x65, 0x72,
]);

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const SSTORE_ZERO_GAS: u64 = 5_000;
const COPY_GAS: u64 = 3;

pub fn create_arbaggregator_precompile(ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbaggregator"), move |input| {
        handler(input, &ctx)
    })
}

fn handler(mut input: PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    let mut gas_used = 0u64;
    let gas_limit = input.gas;
    crate::init_precompile_gas(&mut gas_used, input.data.len());

    let call = match IArbAggregator::ArbAggregatorCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbAggregator::ArbAggregatorCalls as Calls;
    let result = match call {
        Calls::getPreferredAggregator(_) => {
            let mut out = Vec::with_capacity(64);
            let mut addr_word = [0u8; 32];
            addr_word[12..32].copy_from_slice(BATCH_POSTER_ADDRESS.as_slice());
            out.extend_from_slice(&addr_word);
            out.extend_from_slice(&U256::from(1u64).to_be_bytes::<32>());
            crate::charge_precompile_gas(&mut gas_used, 2 * COPY_GAS);
            Ok(PrecompileOutput::new(gas_used.min(gas_limit), out.into()))
        }
        Calls::getDefaultAggregator(_) => {
            let mut out = [0u8; 32];
            out[12..32].copy_from_slice(BATCH_POSTER_ADDRESS.as_slice());
            Ok(PrecompileOutput::new(
                (SLOAD_GAS + COPY_GAS).min(gas_limit),
                out.to_vec().into(),
            ))
        }
        Calls::getTxBaseFee(_) => Ok(PrecompileOutput::new(
            (SLOAD_GAS + 6).min(gas_limit),
            U256::ZERO.to_be_bytes::<32>().to_vec().into(),
        )),
        Calls::setTxBaseFee(_) => Ok(PrecompileOutput::new(
            (SLOAD_GAS + 6).min(gas_limit),
            vec![].into(),
        )),
        Calls::getFeeCollector(c) => {
            handle_get_fee_collector(&mut input, &mut gas_used, c.batchPoster, ctx)
        }
        Calls::setFeeCollector(c) => handle_set_fee_collector(
            &mut input,
            &mut gas_used,
            c.batchPoster,
            c.newFeeCollector,
            ctx,
        ),
        Calls::getBatchPosters(_) => handle_get_batch_posters(&mut input, ctx),
        Calls::addBatchPoster(c) => {
            handle_add_batch_poster(&mut input, &mut gas_used, c.newBatchPoster, ctx)
        }
    };
    crate::gas_check(ctx, gas_limit, gas_used, result)
}

fn load_arbos(input: &mut PrecompileInput<'_>) -> Result<(), ArbPrecompileError> {
    input
        .internals_mut()
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;
    Ok(())
}

fn handle_get_fee_collector(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    poster: Address,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let bpt = arb_state.l1_pricing_state.batch_poster_table();

    let poster_state = match bpt.open_poster(internals, poster, false) {
        Ok(state) => state,
        Err(_) => return Err(ArbPrecompileError::empty_revert(*gas_used).into()),
    };
    let pay_to = poster_state
        .pay_to(internals)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, 2 * SLOAD_GAS + COPY_GAS);
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        U256::from_be_slice(pay_to.as_slice())
            .to_be_bytes::<32>()
            .to_vec()
            .into(),
    ))
}

/// Caller must be the batch poster, its current fee collector, or a chain owner.
fn handle_set_fee_collector(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    poster: Address,
    new_collector: Address,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let caller = input.caller;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let bpt = arb_state.l1_pricing_state.batch_poster_table();

    let poster_state = match bpt.open_poster(internals, poster, false) {
        Ok(state) => state,
        Err(_) => return Err(ArbPrecompileError::empty_revert(*gas_used).into()),
    };
    let old_collector = poster_state
        .pay_to(internals)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, 2 * SLOAD_GAS);

    if caller != poster && caller != old_collector {
        let is_owner = arb_state
            .chain_owners
            .is_member(internals, caller)
            .map_err(ArbPrecompileError::fatal)?;
        crate::charge_precompile_gas(gas_used, SLOAD_GAS);
        if !is_owner {
            return Err(ArbPrecompileError::empty_revert(*gas_used).into());
        }
    }

    poster_state
        .set_pay_to(internals, new_collector)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SSTORE_GAS);

    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        vec![].into(),
    ))
}

fn handle_get_batch_posters(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let bpt = arb_state.l1_pricing_state.batch_poster_table();

    const MAX_MEMBERS: u64 = 1024;
    let posters = bpt
        .all_posters_capped(internals, MAX_MEMBERS)
        .map_err(ArbPrecompileError::fatal)?;
    let count = posters.len() as u64;

    let mut out = Vec::with_capacity(64 + posters.len() * 32);
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(count).to_be_bytes::<32>());
    for addr in &posters {
        let mut word = [0u8; 32];
        word[12..32].copy_from_slice(addr.as_slice());
        out.extend_from_slice(&word);
    }

    let gas_cost = (2 + count) * SLOAD_GAS + (2 + count) * COPY_GAS;
    Ok(PrecompileOutput::new(gas_cost.min(gas_limit), out.into()))
}

/// Caller must be a chain owner.
fn handle_add_batch_poster(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    new_poster: Address,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let caller = input.caller;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;

    let is_owner = arb_state
        .chain_owners
        .is_member(internals, caller)
        .map_err(ArbPrecompileError::fatal)?;
    crate::charge_precompile_gas(gas_used, SLOAD_GAS);
    if !is_owner {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }

    let bpt = arb_state.l1_pricing_state.batch_poster_table();
    let already = bpt
        .contains_poster(internals, new_poster)
        .map_err(ArbPrecompileError::fatal)?;

    if already {
        return Ok(PrecompileOutput::new(
            (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
            vec![].into(),
        ));
    }

    bpt.add_poster(internals, new_poster, new_poster)
        .map_err(ArbPrecompileError::fatal)?;

    let gas_cost = 6 * SLOAD_GAS + SSTORE_ZERO_GAS + 4 * SSTORE_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(
        gas_cost.min(gas_limit),
        vec![].into(),
    ))
}
