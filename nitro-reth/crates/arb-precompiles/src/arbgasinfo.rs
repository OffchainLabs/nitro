use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use alloy_sol_types::SolInterface;
use arb_context::ArbPrecompileCtx;
use arb_storage::ARBOS_STATE_ADDRESS;

use revm::{
    context_interface::block::Block,
    precompile::{PrecompileId, PrecompileOutput, PrecompileResult},
};
use std::sync::Arc;

use crate::{interfaces::IArbGasInfo, ArbPrecompileError};

/// ArbGasInfo precompile address (0x6c).
pub const ARBGASINFO_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x6c,
]);

const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;

const TX_DATA_NON_ZERO_GAS: u64 = 16;
const ASSUMED_SIMPLE_TX_SIZE: u64 = 140;
const STORAGE_WRITE_COST: u64 = 20_000;

/// L1 pricer funds pool address.
const L1_PRICER_FUNDS_POOL_ADDRESS: Address = Address::new([
    0xa4, 0xb0, 0x5f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    0xff, 0xff, 0xff, 0xff,
]);

pub fn create_arbgasinfo_precompile(ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbgasinfo"), move |input| {
        handler(input, &ctx)
    })
}

fn handler(mut input: PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    let mut gas_used = 0u64;
    let gas_limit = input.gas;
    crate::init_precompile_gas(&mut gas_used, input.data.len());

    let call = match IArbGasInfo::ArbGasInfoCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbGasInfo::ArbGasInfoCalls as Calls;
    let result = match call {
        Calls::getL1BaseFeeEstimate(_) | Calls::getL1GasPriceEstimate(_) => {
            read_l1_price_per_unit(&mut input, ctx)
        }
        Calls::getMinimumGasPrice(_) => read_l2_min_base_fee(&mut input, ctx),
        Calls::getPricesInWei(_) | Calls::getPricesInWeiWithAggregator(_) => {
            handle_prices_in_wei(&mut input, ctx)
        }
        Calls::getGasAccountingParams(_) => handle_gas_accounting_params(&mut input, ctx),
        Calls::getCurrentTxL1GasFees(_) => {
            let fee = U256::from(ctx.tx_snapshot().poster_fee);
            Ok(PrecompileOutput::new(
                (SLOAD_GAS + COPY_GAS).min(gas_limit),
                fee.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        Calls::getPricesInArbGas(_) | Calls::getPricesInArbGasWithAggregator(_) => {
            handle_prices_in_arbgas(&mut input, ctx)
        }
        Calls::getL1BaseFeeEstimateInertia(_) => read_l1_inertia(&mut input, ctx),
        Calls::getGasBacklog(_) => read_l2_gas_backlog(&mut input, ctx),
        Calls::getPricingInertia(_) => read_l2_pricing_inertia(&mut input, ctx),
        Calls::getGasBacklogTolerance(_) => read_l2_backlog_tolerance(&mut input, ctx),
        Calls::getL1PricingSurplus(_) => handle_l1_pricing_surplus(&mut input, ctx),
        Calls::getPerBatchGasCharge(_) => read_l1_per_batch_gas_cost(&mut input, ctx),
        Calls::getAmortizedCostCapBips(_) => read_l1_amortized_cost_cap_bips(&mut input, ctx),
        Calls::getL1FeesAvailable(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 10, 0) {
                return r;
            }
            read_l1_fees_available(&mut input, ctx)
        }
        Calls::getL1RewardRate(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 11, 0) {
                return r;
            }
            read_l1_per_unit_reward(&mut input, ctx)
        }
        Calls::getL1RewardRecipient(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 11, 0) {
                return r;
            }
            read_l1_pay_rewards_to(&mut input, ctx)
        }
        Calls::getL1PricingEquilibrationUnits(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 20, 0) {
                return r;
            }
            read_l1_equilibration_units(&mut input, ctx)
        }
        Calls::getLastL1PricingUpdateTime(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 20, 0) {
                return r;
            }
            read_l1_last_update_time(&mut input, ctx)
        }
        Calls::getL1PricingFundsDueForRewards(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 20, 0) {
                return r;
            }
            read_l1_funds_due_for_rewards(&mut input, ctx)
        }
        Calls::getL1PricingUnitsSinceUpdate(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 20, 0) {
                return r;
            }
            read_l1_units_since_update(&mut input, ctx)
        }
        Calls::getLastL1PricingSurplus(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 20, 0) {
                return r;
            }
            read_l1_last_surplus(&mut input, ctx)
        }
        Calls::getMaxBlockGasLimit(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 50, 0) {
                return r;
            }
            read_l2_per_block_gas_limit(&mut input, ctx)
        }
        Calls::getMaxTxGasLimit(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 50, 0) {
                return r;
            }
            read_l2_per_tx_gas_limit(&mut input, ctx)
        }
        Calls::getGasPricingConstraints(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 50, 0) {
                return r;
            }
            handle_gas_pricing_constraints(&mut input, ctx)
        }
        Calls::getMultiGasPricingConstraints(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 60, 0) {
                return r;
            }
            handle_multi_gas_pricing_constraints(&mut input, ctx)
        }
        Calls::getMultiGasBaseFee(_) => {
            if let Some(r) = crate::check_method_version(ctx, gas_limit, 60, 0) {
                return r;
            }
            handle_multi_gas_base_fee(&mut input, ctx)
        }
    };
    crate::gas_check(ctx, gas_limit, gas_used, result)
}

// ── helpers ──────────────────────────────────────────────────────────

fn load_arbos(input: &mut PrecompileInput<'_>) -> Result<(), ArbPrecompileError> {
    input
        .internals_mut()
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;
    Ok(())
}

fn field_read_output(gas_limit: u64, value: U256) -> PrecompileResult {
    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}

// ── L1 pricing field readers ────────────────────────────────────────

fn read_l1_price_per_unit(
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
    let value = arb_state
        .l1_pricing_state
        .price_per_unit(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, value)
}

fn read_l1_inertia(input: &mut PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let value = arb_state
        .l1_pricing_state
        .inertia(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from(value))
}

fn read_l1_per_unit_reward(
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
    let value = arb_state
        .l1_pricing_state
        .per_unit_reward(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from(value))
}

fn read_l1_pay_rewards_to(
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
    let addr = arb_state
        .l1_pricing_state
        .pay_rewards_to(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from_be_slice(addr.as_slice()))
}

fn read_l1_last_surplus(
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
    let (magnitude, negative) = arb_state
        .l1_pricing_state
        .last_surplus(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let raw = if negative {
        U256::ZERO.wrapping_sub(magnitude)
    } else {
        magnitude
    };
    field_read_output(gas_limit, raw)
}

fn read_l1_per_batch_gas_cost(
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
    let value = arb_state
        .l1_pricing_state
        .per_batch_gas_cost(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from(value as u64))
}

fn read_l1_amortized_cost_cap_bips(
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
    let value = arb_state
        .l1_pricing_state
        .amortized_cost_cap_bips(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from(value))
}

fn read_l1_equilibration_units(
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
    let value = arb_state
        .l1_pricing_state
        .equilibration_units(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, value)
}

fn read_l1_last_update_time(
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
    let value = arb_state
        .l1_pricing_state
        .last_update_time(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from(value))
}

fn read_l1_funds_due_for_rewards(
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
    let value = arb_state
        .l1_pricing_state
        .funds_due_for_rewards(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, value)
}

fn read_l1_units_since_update(
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
    let value = arb_state
        .l1_pricing_state
        .units_since_update(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from(value))
}

fn read_l1_fees_available(
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
    let value = arb_state
        .l1_pricing_state
        .l1_fees_available(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, value)
}

// ── L2 pricing field readers ────────────────────────────────────────

fn read_l2_min_base_fee(
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
    let value = arb_state
        .l2_pricing_state
        .min_base_fee_wei(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, value)
}

fn read_l2_gas_backlog(
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
    let value = arb_state
        .l2_pricing_state
        .gas_backlog(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from(value))
}

fn read_l2_pricing_inertia(
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
    let value = arb_state
        .l2_pricing_state
        .pricing_inertia(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from(value))
}

fn read_l2_backlog_tolerance(
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
    let value = arb_state
        .l2_pricing_state
        .backlog_tolerance(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from(value))
}

fn read_l2_per_block_gas_limit(
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
    let value = arb_state
        .l2_pricing_state
        .per_block_gas_limit(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from(value))
}

fn read_l2_per_tx_gas_limit(
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
    let value = arb_state
        .l2_pricing_state
        .per_tx_gas_limit(internals)
        .map_err(ArbPrecompileError::fatal)?;
    field_read_output(gas_limit, U256::from(value))
}

// ── Compound handlers ───────────────────────────────────────────────

/// Compute L1 pricing surplus.
/// v10+: `L1FeesAvailable - (TotalFundsDue + FundsDueForRewards)` (signed).
/// pre-v10: `Balance(L1PricerFundsPool) - (TotalFundsDue + FundsDueForRewards)`.
fn handle_l1_pricing_surplus(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let arbos_version = ctx.block.arbos_version;
    load_arbos(input)?;

    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;

    let bpt = arb_state.l1_pricing_state.batch_poster_table();
    let total_funds_due = bpt
        .total_funds_due(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let funds_due_for_rewards = arb_state
        .l1_pricing_state
        .funds_due_for_rewards(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let need_funds = total_funds_due.saturating_add(funds_due_for_rewards);

    let have_funds = if arbos_version >= 10 {
        arb_state
            .l1_pricing_state
            .l1_fees_available(internals)
            .map_err(ArbPrecompileError::fatal)?
    } else {
        let account = internals
            .load_account(L1_PRICER_FUNDS_POOL_ADDRESS)
            .map_err(ArbPrecompileError::fatal)?;
        account.data.info.balance
    };

    let surplus = if have_funds >= need_funds {
        have_funds - need_funds
    } else {
        let deficit = need_funds - have_funds;
        U256::ZERO.wrapping_sub(deficit)
    };

    // Pre-v10: 3 SLOAD (OAS + TotalFundsDue + FundsDueForRewards). v10+: 4 SLOAD (adds
    // L1FeesAvailable).
    let sloads = if arbos_version >= 10 { 4 } else { 3 };
    let gas_cost = (sloads * SLOAD_GAS + COPY_GAS).min(gas_limit);
    Ok(PrecompileOutput::new(
        gas_cost,
        surplus.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_prices_in_wei(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let data_len = input.data.len();
    let gas_limit = input.gas;
    let arbos_version = ctx.block.arbos_version;

    // Reth zeros BlockEnv basefee for eth_call without a gas price;
    // fall back to the L2PricingState slot (written at StartBlock) so
    // eth_call returns the current block's basefee.
    let block_basefee = U256::from(input.internals().block_env().basefee());
    load_arbos(input)?;

    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;

    let l1_price = arb_state
        .l1_pricing_state
        .price_per_unit(internals)
        .map_err(ArbPrecompileError::fatal)?;

    // Pre-v4: no MinBaseFeeWei read; perArbGasBase = l2GasPrice, congestion = 0.
    let read_min_base = arbos_version >= arb_chainspec::arbos_version::ARBOS_VERSION_4;
    let l2_min = if read_min_base {
        arb_state
            .l2_pricing_state
            .min_base_fee_wei(internals)
            .map_err(ArbPrecompileError::fatal)?
    } else {
        U256::ZERO
    };
    let l2_gas_price = if block_basefee.is_zero() {
        arb_state
            .l2_pricing_state
            .base_fee_wei(internals)
            .map_err(ArbPrecompileError::fatal)?
    } else {
        block_basefee
    };

    let wei_for_l1_calldata = l1_price.saturating_mul(U256::from(TX_DATA_NON_ZERO_GAS));
    let per_l2_tx = wei_for_l1_calldata.saturating_mul(U256::from(ASSUMED_SIMPLE_TX_SIZE));
    let (per_arbgas_base, per_arbgas_congestion) = if read_min_base {
        let base = l2_gas_price.min(l2_min);
        (base, l2_gas_price.saturating_sub(base))
    } else {
        (l2_gas_price, U256::ZERO)
    };
    let per_arbgas_total = l2_gas_price;
    let wei_for_l2_storage = l2_gas_price.saturating_mul(U256::from(STORAGE_WRITE_COST));

    let mut out = Vec::with_capacity(192);
    out.extend_from_slice(&per_l2_tx.to_be_bytes::<32>());
    out.extend_from_slice(&wei_for_l1_calldata.to_be_bytes::<32>());
    out.extend_from_slice(&wei_for_l2_storage.to_be_bytes::<32>());
    out.extend_from_slice(&per_arbgas_base.to_be_bytes::<32>());
    out.extend_from_slice(&per_arbgas_congestion.to_be_bytes::<32>());
    out.extend_from_slice(&per_arbgas_total.to_be_bytes::<32>());

    // OpenArbosState SLOAD + body SLOADs (1 pre-v4, 2 v4+) + copy gas.
    let arg_words = (data_len as u64).saturating_sub(4).div_ceil(32);
    let sloads = if read_min_base { 3 } else { 2 };
    let gas_cost = (sloads * SLOAD_GAS + (arg_words + 6) * COPY_GAS).min(gas_limit);
    Ok(PrecompileOutput::new(gas_cost, out.into()))
}

fn handle_gas_accounting_params(
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
    let speed_limit = arb_state
        .l2_pricing_state
        .speed_limit_per_second(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let gas_limit_val = arb_state
        .l2_pricing_state
        .per_block_gas_limit(internals)
        .map_err(ArbPrecompileError::fatal)?;

    let speed_word = U256::from(speed_limit);
    let limit_word = U256::from(gas_limit_val);
    let mut out = Vec::with_capacity(96);
    out.extend_from_slice(&speed_word.to_be_bytes::<32>());
    out.extend_from_slice(&limit_word.to_be_bytes::<32>());
    out.extend_from_slice(&limit_word.to_be_bytes::<32>());

    Ok(PrecompileOutput::new(
        (3 * SLOAD_GAS + 3 * COPY_GAS).min(gas_limit),
        out.into(),
    ))
}

fn handle_prices_in_arbgas(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let data_len = input.data.len();
    let gas_limit = input.gas;

    let block_basefee = U256::from(input.internals().block_env().basefee());
    load_arbos(input)?;

    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let l1_price = arb_state
        .l1_pricing_state
        .price_per_unit(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let l2_gas_price = if block_basefee.is_zero() {
        arb_state
            .l2_pricing_state
            .base_fee_wei(internals)
            .map_err(ArbPrecompileError::fatal)?
    } else {
        block_basefee
    };

    let arbos_version = ctx.block.arbos_version;
    let wei_for_l1_calldata = l1_price.saturating_mul(U256::from(TX_DATA_NON_ZERO_GAS));

    let gas_for_l1_calldata = if l2_gas_price > U256::ZERO {
        wei_for_l1_calldata / l2_gas_price
    } else {
        U256::ZERO
    };
    // Pre-v4: gasPerL2Tx = AssumedSimpleTxSize (constant).
    // v4+: gasPerL2Tx = wei_per_l2_tx / l2_gas_price.
    let gas_per_l2_tx = if arbos_version >= arb_chainspec::arbos_version::ARBOS_VERSION_4 {
        let wei_per_l2_tx = wei_for_l1_calldata.saturating_mul(U256::from(ASSUMED_SIMPLE_TX_SIZE));
        if l2_gas_price > U256::ZERO {
            wei_per_l2_tx / l2_gas_price
        } else {
            U256::ZERO
        }
    } else {
        U256::from(ASSUMED_SIMPLE_TX_SIZE)
    };

    let mut out = Vec::with_capacity(96);
    out.extend_from_slice(&gas_per_l2_tx.to_be_bytes::<32>());
    out.extend_from_slice(&gas_for_l1_calldata.to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(STORAGE_WRITE_COST).to_be_bytes::<32>());

    // OpenArbosState SLOAD + 1 body SLOAD (L1 price per unit) + copy gas.
    // l2GasPrice comes from evm.Context.BaseFee (free).
    let arg_words = (data_len as u64).saturating_sub(4).div_ceil(32);
    let gas_cost = (2 * SLOAD_GAS + (arg_words + 3) * COPY_GAS).min(gas_limit);
    Ok(PrecompileOutput::new(gas_cost, out.into()))
}

// ── Constraint getters (ArbOS v50+) ─────────────────────────────────

/// Index of `ResourceKindSingleDim` — special-cased to fall back to the
/// global L2 base fee in `getMultiGasBaseFee`.
const RESOURCE_KIND_SINGLE_DIM: u64 = 6;

/// Returns `[][3]uint64` — (target, adjustmentWindow, backlog) per constraint.
fn handle_gas_pricing_constraints(
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

    let count = arb_state
        .l2_pricing_state
        .gas_constraints_length(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let mut sloads: u64 = 2; // OAS + vec length

    // ABI: offset to dynamic array, then length, then N×3 uint64 values.
    let mut out = Vec::with_capacity(64 + count as usize * 96);
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(count).to_be_bytes::<32>());

    for i in 0..count {
        let constraint = arb_state.l2_pricing_state.open_gas_constraint_at(i);
        let target = constraint
            .target(internals)
            .map_err(ArbPrecompileError::fatal)?;
        let window = constraint
            .adjustment_window(internals)
            .map_err(ArbPrecompileError::fatal)?;
        let backlog = constraint
            .backlog(internals)
            .map_err(ArbPrecompileError::fatal)?;

        out.extend_from_slice(&U256::from(target).to_be_bytes::<32>());
        out.extend_from_slice(&U256::from(window).to_be_bytes::<32>());
        out.extend_from_slice(&U256::from(backlog).to_be_bytes::<32>());
        sloads += 3;
    }

    let result_words = (out.len() as u64).div_ceil(32);
    Ok(PrecompileOutput::new(
        (sloads * SLOAD_GAS + result_words * COPY_GAS).min(gas_limit),
        out.into(),
    ))
}

/// Returns `[]MultiGasConstraint` ABI-encoded.
///
/// MultiGasConstraint = (WeightedResource[] resources, uint32 adjustmentWindowSecs,
///                        uint64 targetPerSec, uint64 backlog)
/// WeightedResource   = (uint8 resource, uint64 weight)
fn handle_multi_gas_pricing_constraints(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    use arb_primitives::multigas::ResourceKind;
    let gas_limit = input.gas;
    load_arbos(input)?;

    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;

    let count = arb_state
        .l2_pricing_state
        .multi_gas_constraints_length(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let mut sloads: u64 = 2; // OAS + vec length

    struct ConstraintData {
        target: u64,
        window: u32,
        backlog: u64,
        resources: Vec<(u8, u64)>,
    }
    let mut constraints = Vec::with_capacity(count as usize);

    for i in 0..count {
        let constraint = arb_state.l2_pricing_state.open_multi_gas_constraint_at(i);
        let target = constraint
            .target(internals)
            .map_err(ArbPrecompileError::fatal)?;
        let window = constraint
            .adjustment_window(internals)
            .map_err(ArbPrecompileError::fatal)?;
        let backlog = constraint
            .backlog(internals)
            .map_err(ArbPrecompileError::fatal)?;
        sloads += 3;

        let mut resources = Vec::new();
        for kind in ResourceKind::ALL {
            let weight = constraint
                .resource_weight(internals, kind)
                .map_err(ArbPrecompileError::fatal)?;
            sloads += 1;
            if weight > 0 {
                resources.push((kind as u8, weight));
            }
        }
        constraints.push(ConstraintData {
            target,
            window,
            backlog,
            resources,
        });
    }

    let n = constraints.len();
    let mut out = Vec::new();
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(n).to_be_bytes::<32>());

    let elem_sizes: Vec<usize> = constraints
        .iter()
        .map(|c| 4 * 32 + 32 + c.resources.len() * 64)
        .collect();

    let mut running_offset = n * 32;
    for size in &elem_sizes {
        out.extend_from_slice(&U256::from(running_offset).to_be_bytes::<32>());
        running_offset += size;
    }

    for c in &constraints {
        let m = c.resources.len();
        out.extend_from_slice(&U256::from(4u64 * 32).to_be_bytes::<32>());
        out.extend_from_slice(&U256::from(c.window).to_be_bytes::<32>());
        out.extend_from_slice(&U256::from(c.target).to_be_bytes::<32>());
        out.extend_from_slice(&U256::from(c.backlog).to_be_bytes::<32>());
        out.extend_from_slice(&U256::from(m).to_be_bytes::<32>());
        for &(kind, weight) in &c.resources {
            out.extend_from_slice(&U256::from(kind).to_be_bytes::<32>());
            out.extend_from_slice(&U256::from(weight).to_be_bytes::<32>());
        }
    }

    let result_words = (out.len() as u64).div_ceil(32);
    Ok(PrecompileOutput::new(
        (sloads * SLOAD_GAS + result_words * COPY_GAS).min(gas_limit),
        out.into(),
    ))
}

/// Returns `uint256[]` — current-block base fee per resource kind. Reads BaseFeeWei,
/// then per-kind fees; for `ResourceKindSingleDim` and any zero per-kind fee, falls
/// back to BaseFeeWei.
fn handle_multi_gas_base_fee(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    use arb_primitives::multigas::{ResourceKind, NUM_RESOURCE_KIND};
    let gas_limit = input.gas;
    load_arbos(input)?;

    let internals = input.internals_mut();
    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;

    let base_fee_wei = arb_state
        .l2_pricing_state
        .base_fee_wei(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let multi_gas_fees = arb_state.l2_pricing_state.multi_gas_fees();

    let mut out = Vec::with_capacity(64 + NUM_RESOURCE_KIND * 32);
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(NUM_RESOURCE_KIND).to_be_bytes::<32>());

    for kind in ResourceKind::ALL {
        let raw = multi_gas_fees
            .get_current_block_fee(internals, kind)
            .map_err(ArbPrecompileError::fatal)?;
        let fee = if kind as u64 == RESOURCE_KIND_SINGLE_DIM || raw == U256::ZERO {
            base_fee_wei
        } else {
            raw
        };
        out.extend_from_slice(&fee.to_be_bytes::<32>());
    }

    let result_words = (out.len() as u64).div_ceil(32);
    Ok(PrecompileOutput::new(
        ((2 + NUM_RESOURCE_KIND as u64) * SLOAD_GAS + result_words * COPY_GAS).min(gas_limit),
        out.into(),
    ))
}
