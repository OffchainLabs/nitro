use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use alloy_sol_types::SolInterface;
use revm::{
    context_interface::block::Block,
    precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult},
};

use crate::interfaces::IArbGasInfo;
use crate::storage_slot::{
    derive_subspace_key, gas_constraints_vec_key, map_slot, multi_gas_base_fees_subspace,
    multi_gas_constraints_vec_key, subspace_slot, vector_element_field, vector_element_key,
    vector_length_slot, ARBOS_STATE_ADDRESS, L1_PRICING_SUBSPACE, L2_PRICING_SUBSPACE,
    ROOT_STORAGE_KEY,
};

/// ArbGasInfo precompile address (0x6c).
pub const ARBGASINFO_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x6c,
]);

const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;

// L1 pricing field offsets (within L1 pricing subspace).
const L1_PAY_REWARDS_TO: u64 = 0;
const L1_INERTIA: u64 = 2;
const L1_PER_UNIT_REWARD: u64 = 3;
const L1_PRICE_PER_UNIT: u64 = 7;
const L1_LAST_SURPLUS: u64 = 8;
const L1_PER_BATCH_GAS_COST: u64 = 9;
const L1_AMORTIZED_COST_CAP_BIPS: u64 = 10;
const L1_EQUILIBRATION_UNITS: u64 = 1;
const L1_LAST_UPDATE_TIME: u64 = 4;
const L1_FUNDS_DUE_FOR_REWARDS: u64 = 5;
const L1_UNITS_SINCE: u64 = 6;
const L1_FEES_AVAILABLE: u64 = 11;

// L2 pricing field offsets (within L2 pricing subspace).
const L2_SPEED_LIMIT: u64 = 0;
const L2_PER_BLOCK_GAS_LIMIT: u64 = 1;
const L2_BASE_FEE: u64 = 2;
const L2_MIN_BASE_FEE: u64 = 3;
const L2_GAS_BACKLOG: u64 = 4;
const L2_PRICING_INERTIA: u64 = 5;
const L2_BACKLOG_TOLERANCE: u64 = 6;
const L2_PER_TX_GAS_LIMIT: u64 = 7;

const TX_DATA_NON_ZERO_GAS: u64 = 16;
const ASSUMED_SIMPLE_TX_SIZE: u64 = 140;
const STORAGE_WRITE_COST: u64 = 20_000;

/// Batch poster table subspace key within L1 pricing.
const BATCH_POSTER_TABLE_KEY: &[u8] = &[0];
/// TotalFundsDue offset within batch poster table subspace.
const TOTAL_FUNDS_DUE_OFFSET: u64 = 0;

/// L1 pricer funds pool address (0xa4b05...fffffffffffffffffffffffffffffffffff).
const L1_PRICER_FUNDS_POOL_ADDRESS: Address = Address::new([
    0xa4, 0xb0, 0x5f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
    0xff, 0xff, 0xff, 0xff,
]);

pub fn create_arbgasinfo_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbgasinfo"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    crate::init_precompile_gas(input.data.len());

    let call = match IArbGasInfo::ArbGasInfoCalls::abi_decode(input.data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbGasInfo::ArbGasInfoCalls as Calls;
    let result = match call {
        Calls::getL1BaseFeeEstimate(_) | Calls::getL1GasPriceEstimate(_) => {
            read_l1_field(&mut input, L1_PRICE_PER_UNIT)
        }
        Calls::getMinimumGasPrice(_) => read_l2_field(&mut input, L2_MIN_BASE_FEE),
        Calls::getPricesInWei(_) | Calls::getPricesInWeiWithAggregator(_) => {
            handle_prices_in_wei(&mut input)
        }
        Calls::getGasAccountingParams(_) => handle_gas_accounting_params(&mut input),
        Calls::getCurrentTxL1GasFees(_) => {
            let fee = U256::from(crate::get_current_tx_poster_fee());
            Ok(PrecompileOutput::new(
                (SLOAD_GAS + COPY_GAS).min(gas_limit),
                fee.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        Calls::getPricesInArbGas(_) | Calls::getPricesInArbGasWithAggregator(_) => {
            handle_prices_in_arbgas(&mut input)
        }
        Calls::getL1BaseFeeEstimateInertia(_) => read_l1_field(&mut input, L1_INERTIA),
        Calls::getGasBacklog(_) => read_l2_field(&mut input, L2_GAS_BACKLOG),
        Calls::getPricingInertia(_) => read_l2_field(&mut input, L2_PRICING_INERTIA),
        Calls::getGasBacklogTolerance(_) => read_l2_field(&mut input, L2_BACKLOG_TOLERANCE),
        Calls::getL1PricingSurplus(_) => handle_l1_pricing_surplus(&mut input),
        Calls::getPerBatchGasCharge(_) => read_l1_field(&mut input, L1_PER_BATCH_GAS_COST),
        Calls::getAmortizedCostCapBips(_) => read_l1_field(&mut input, L1_AMORTIZED_COST_CAP_BIPS),
        Calls::getL1FeesAvailable(_) => {
            if let Some(r) = crate::check_method_version(gas_limit, 10, 0) {
                return r;
            }
            read_l1_field(&mut input, L1_FEES_AVAILABLE)
        }
        Calls::getL1RewardRate(_) => {
            if let Some(r) = crate::check_method_version(gas_limit, 11, 0) {
                return r;
            }
            read_l1_field(&mut input, L1_PER_UNIT_REWARD)
        }
        Calls::getL1RewardRecipient(_) => {
            if let Some(r) = crate::check_method_version(gas_limit, 11, 0) {
                return r;
            }
            read_l1_field(&mut input, L1_PAY_REWARDS_TO)
        }
        Calls::getL1PricingEquilibrationUnits(_) => {
            if let Some(r) = crate::check_method_version(gas_limit, 20, 0) {
                return r;
            }
            read_l1_field(&mut input, L1_EQUILIBRATION_UNITS)
        }
        Calls::getLastL1PricingUpdateTime(_) => {
            if let Some(r) = crate::check_method_version(gas_limit, 20, 0) {
                return r;
            }
            read_l1_field(&mut input, L1_LAST_UPDATE_TIME)
        }
        Calls::getL1PricingFundsDueForRewards(_) => {
            if let Some(r) = crate::check_method_version(gas_limit, 20, 0) {
                return r;
            }
            read_l1_field(&mut input, L1_FUNDS_DUE_FOR_REWARDS)
        }
        Calls::getL1PricingUnitsSinceUpdate(_) => {
            if let Some(r) = crate::check_method_version(gas_limit, 20, 0) {
                return r;
            }
            read_l1_field(&mut input, L1_UNITS_SINCE)
        }
        Calls::getLastL1PricingSurplus(_) => {
            if let Some(r) = crate::check_method_version(gas_limit, 20, 0) {
                return r;
            }
            read_l1_field(&mut input, L1_LAST_SURPLUS)
        }
        Calls::getMaxBlockGasLimit(_) => {
            if let Some(r) = crate::check_method_version(gas_limit, 50, 0) {
                return r;
            }
            read_l2_field(&mut input, L2_PER_BLOCK_GAS_LIMIT)
        }
        Calls::getMaxTxGasLimit(_) => {
            if let Some(r) = crate::check_method_version(gas_limit, 50, 0) {
                return r;
            }
            read_l2_field(&mut input, L2_PER_TX_GAS_LIMIT)
        }
        Calls::getGasPricingConstraints(_) => {
            if let Some(r) = crate::check_method_version(gas_limit, 50, 0) {
                return r;
            }
            handle_gas_pricing_constraints(&mut input)
        }
        Calls::getMultiGasPricingConstraints(_) => {
            if let Some(r) = crate::check_method_version(gas_limit, 60, 0) {
                return r;
            }
            handle_multi_gas_pricing_constraints(&mut input)
        }
        Calls::getMultiGasBaseFee(_) => {
            if let Some(r) = crate::check_method_version(gas_limit, 60, 0) {
                return r;
            }
            handle_multi_gas_base_fee(&mut input)
        }
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
    crate::charge_precompile_gas(SLOAD_GAS);
    Ok(val.data)
}

fn read_l1_field(input: &mut PrecompileInput<'_>, offset: u64) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let field_slot = subspace_slot(L1_PRICING_SUBSPACE, offset);
    let value = sload_field(input, field_slot)?;
    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn read_l2_field(input: &mut PrecompileInput<'_>, offset: u64) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let field_slot = subspace_slot(L2_PRICING_SUBSPACE, offset);
    let value = sload_field(input, field_slot)?;
    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Compute L1 pricing surplus.
/// v10+: `L1FeesAvailable - (TotalFundsDue + FundsDueForRewards)` (signed).
/// pre-v10: `Balance(L1PricerFundsPool) - (TotalFundsDue + FundsDueForRewards)`.
fn handle_l1_pricing_surplus(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    let arbos_version = crate::get_arbos_version();

    load_arbos(input)?;

    // Read TotalFundsDue from batch poster table subspace.
    let l1_sub_key = derive_subspace_key(ROOT_STORAGE_KEY, L1_PRICING_SUBSPACE);
    let bpt_key = derive_subspace_key(l1_sub_key.as_slice(), BATCH_POSTER_TABLE_KEY);
    let total_funds_due_slot = map_slot(bpt_key.as_slice(), TOTAL_FUNDS_DUE_OFFSET);
    let total_funds_due = sload_field(input, total_funds_due_slot)?;

    // Read FundsDueForRewards from L1 pricing subspace.
    let fdr_slot = subspace_slot(L1_PRICING_SUBSPACE, L1_FUNDS_DUE_FOR_REWARDS);
    let funds_due_for_rewards = sload_field(input, fdr_slot)?;

    let need_funds = total_funds_due.saturating_add(funds_due_for_rewards);

    let have_funds = if arbos_version >= 10 {
        // v10+: read from stored L1FeesAvailable.
        let slot = subspace_slot(L1_PRICING_SUBSPACE, L1_FEES_AVAILABLE);
        sload_field(input, slot)?
    } else {
        // pre-v10: read actual balance of L1PricerFundsPool.
        let account = input
            .internals_mut()
            .load_account(L1_PRICER_FUNDS_POOL_ADDRESS)
            .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;
        account.data.info.balance
    };

    // Signed result: surplus can be negative.
    let surplus = if have_funds >= need_funds {
        have_funds - need_funds
    } else {
        // Two's complement encoding for negative value.
        let deficit = need_funds - have_funds;
        U256::ZERO.wrapping_sub(deficit)
    };

    let gas_cost = (4 * SLOAD_GAS + COPY_GAS).min(gas_limit);
    Ok(PrecompileOutput::new(
        gas_cost,
        surplus.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_prices_in_wei(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data_len = input.data.len();
    let gas_limit = input.gas;

    // Reth zeros BlockEnv basefee for eth_call without a gas price;
    // fall back to the L2PricingState slot (written at StartBlock) so
    // eth_call returns the current block's basefee.
    let block_basefee = U256::from(input.internals().block_env().basefee());
    load_arbos(input)?;

    let l1_price = sload_field(input, subspace_slot(L1_PRICING_SUBSPACE, L1_PRICE_PER_UNIT))?;
    let l2_min = sload_field(input, subspace_slot(L2_PRICING_SUBSPACE, L2_MIN_BASE_FEE))?;
    let l2_gas_price = if block_basefee.is_zero() {
        sload_field(input, subspace_slot(L2_PRICING_SUBSPACE, L2_BASE_FEE))?
    } else {
        block_basefee
    };

    let wei_for_l1_calldata = l1_price.saturating_mul(U256::from(TX_DATA_NON_ZERO_GAS));
    let per_l2_tx = wei_for_l1_calldata.saturating_mul(U256::from(ASSUMED_SIMPLE_TX_SIZE));
    let per_arbgas_base = l2_gas_price.min(l2_min);
    let per_arbgas_congestion = l2_gas_price.saturating_sub(per_arbgas_base);
    let per_arbgas_total = l2_gas_price;
    let wei_for_l2_storage = l2_gas_price.saturating_mul(U256::from(STORAGE_WRITE_COST));

    let mut out = Vec::with_capacity(192);
    out.extend_from_slice(&per_l2_tx.to_be_bytes::<32>());
    out.extend_from_slice(&wei_for_l1_calldata.to_be_bytes::<32>());
    out.extend_from_slice(&wei_for_l2_storage.to_be_bytes::<32>());
    out.extend_from_slice(&per_arbgas_base.to_be_bytes::<32>());
    out.extend_from_slice(&per_arbgas_congestion.to_be_bytes::<32>());
    out.extend_from_slice(&per_arbgas_total.to_be_bytes::<32>());

    // OpenArbosState SLOAD + 2 body SLOADs (L1_PRICE_PER_UNIT, L2_MIN_BASE_FEE)
    // + copy gas for args and 6-word return tuple. Total 2418 gas.
    let arg_words = (data_len as u64).saturating_sub(4).div_ceil(32);
    let gas_cost = (3 * SLOAD_GAS + (arg_words + 6) * COPY_GAS).min(gas_limit);
    Ok(PrecompileOutput::new(gas_cost, out.into()))
}

fn handle_gas_accounting_params(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let speed_limit = sload_field(input, subspace_slot(L2_PRICING_SUBSPACE, L2_SPEED_LIMIT))?;
    let gas_limit_val = sload_field(
        input,
        subspace_slot(L2_PRICING_SUBSPACE, L2_PER_BLOCK_GAS_LIMIT),
    )?;

    let mut out = Vec::with_capacity(96);
    out.extend_from_slice(&speed_limit.to_be_bytes::<32>());
    out.extend_from_slice(&gas_limit_val.to_be_bytes::<32>());
    out.extend_from_slice(&gas_limit_val.to_be_bytes::<32>());

    Ok(PrecompileOutput::new(
        (3 * SLOAD_GAS + 3 * COPY_GAS).min(gas_limit),
        out.into(),
    ))
}

fn handle_prices_in_arbgas(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data_len = input.data.len();
    let gas_limit = input.gas;

    let block_basefee = U256::from(input.internals().block_env().basefee());
    load_arbos(input)?;

    let l1_price = sload_field(input, subspace_slot(L1_PRICING_SUBSPACE, L1_PRICE_PER_UNIT))?;
    let l2_gas_price = if block_basefee.is_zero() {
        sload_field(input, subspace_slot(L2_PRICING_SUBSPACE, L2_BASE_FEE))?
    } else {
        block_basefee
    };

    let wei_for_l1_calldata = l1_price.saturating_mul(U256::from(TX_DATA_NON_ZERO_GAS));
    let wei_per_l2_tx = wei_for_l1_calldata.saturating_mul(U256::from(ASSUMED_SIMPLE_TX_SIZE));

    let (gas_for_l1_calldata, gas_per_l2_tx) = if l2_gas_price > U256::ZERO {
        (
            wei_for_l1_calldata / l2_gas_price,
            wei_per_l2_tx / l2_gas_price,
        )
    } else {
        (U256::ZERO, U256::ZERO)
    };

    let mut out = Vec::with_capacity(96);
    out.extend_from_slice(&gas_per_l2_tx.to_be_bytes::<32>());
    out.extend_from_slice(&gas_for_l1_calldata.to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(STORAGE_WRITE_COST).to_be_bytes::<32>());

    // OpenArbosState SLOAD + 1 body SLOAD (L1_PRICE_PER_UNIT) + copy gas.
    // l2GasPrice comes from evm.Context.BaseFee (free).
    let arg_words = (data_len as u64).saturating_sub(4).div_ceil(32);
    let gas_cost = (2 * SLOAD_GAS + (arg_words + 3) * COPY_GAS).min(gas_limit);
    Ok(PrecompileOutput::new(gas_cost, out.into()))
}

// ── Constraint getters (ArbOS v50+) ─────────────────────────────────

/// Constraint field offsets (matching gas_constraint.rs / multi_gas_constraint.rs).
const CONSTRAINT_TARGET: u64 = 0;
const CONSTRAINT_ADJ_WINDOW: u64 = 1;
const CONSTRAINT_BACKLOG: u64 = 2;
const MULTI_CONSTRAINT_WEIGHTED_BASE: u64 = 4;

const NUM_RESOURCE_KIND: u64 = 8;
/// Offset within MultiGasFees for current-block fees.
const CURRENT_BLOCK_FEES_OFFSET: u64 = NUM_RESOURCE_KIND;

/// Returns `[][3]uint64` — (target, adjustmentWindow, backlog) per constraint.
fn handle_gas_pricing_constraints(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let vec_key = gas_constraints_vec_key();
    let count = sload_field(input, vector_length_slot(&vec_key))?.saturating_to::<u64>();
    let mut sloads: u64 = 2; // 1 for OpenArbosState + 1 for vec length

    // ABI: offset to dynamic array, then length, then N×3 uint64 values.
    let mut out = Vec::with_capacity(64 + count as usize * 96);
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(count).to_be_bytes::<32>());

    for i in 0..count {
        let target = sload_field(input, vector_element_field(&vec_key, i, CONSTRAINT_TARGET))?;
        let window = sload_field(
            input,
            vector_element_field(&vec_key, i, CONSTRAINT_ADJ_WINDOW),
        )?;
        let backlog = sload_field(input, vector_element_field(&vec_key, i, CONSTRAINT_BACKLOG))?;

        out.extend_from_slice(&target.to_be_bytes::<32>());
        out.extend_from_slice(&window.to_be_bytes::<32>());
        out.extend_from_slice(&backlog.to_be_bytes::<32>());
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
fn handle_multi_gas_pricing_constraints(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let vec_key = multi_gas_constraints_vec_key();
    let count = sload_field(input, vector_length_slot(&vec_key))?.saturating_to::<u64>();
    let mut sloads: u64 = 2; // 1 for OpenArbosState + 1 for vec length

    // Collect per-constraint data before encoding, since we need to know sizes for offsets.
    struct ConstraintData {
        target: U256,
        window: U256,
        backlog: U256,
        resources: Vec<(u8, U256)>,
    }
    let mut constraints = Vec::with_capacity(count as usize);

    for i in 0..count {
        let target = sload_field(input, vector_element_field(&vec_key, i, CONSTRAINT_TARGET))?;
        let window = sload_field(
            input,
            vector_element_field(&vec_key, i, CONSTRAINT_ADJ_WINDOW),
        )?;
        let backlog = sload_field(input, vector_element_field(&vec_key, i, CONSTRAINT_BACKLOG))?;
        sloads += 3;

        let elem_key = vector_element_key(&vec_key, i);
        let mut resources = Vec::new();
        for kind in 0..NUM_RESOURCE_KIND {
            let w = sload_field(
                input,
                map_slot(elem_key.as_slice(), MULTI_CONSTRAINT_WEIGHTED_BASE + kind),
            )?;
            sloads += 1;
            if w > U256::ZERO {
                resources.push((kind as u8, w));
            }
        }
        constraints.push(ConstraintData {
            target,
            window,
            backlog,
            resources,
        });
    }

    // ABI-encode: dynamic array of dynamic tuples.
    let n = constraints.len();
    let mut out = Vec::new();

    // Top-level: offset to outer array.
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    // Array length.
    out.extend_from_slice(&U256::from(n).to_be_bytes::<32>());

    // Element tuple size = 4 × 32 (head) + 32 (resources length) + resources.len() × 64.
    let elem_sizes: Vec<usize> = constraints
        .iter()
        .map(|c| 4 * 32 + 32 + c.resources.len() * 64)
        .collect();

    // Write offsets (relative to start of offsets area).
    let mut running_offset = n * 32;
    for size in &elem_sizes {
        out.extend_from_slice(&U256::from(running_offset).to_be_bytes::<32>());
        running_offset += size;
    }

    // Write each element.
    for c in &constraints {
        let m = c.resources.len();
        // Tuple head: offset to Resources data = 4 × 32 = 128.
        out.extend_from_slice(&U256::from(4u64 * 32).to_be_bytes::<32>());
        // AdjustmentWindowSecs (uint32).
        out.extend_from_slice(&c.window.to_be_bytes::<32>());
        // TargetPerSec (uint64).
        out.extend_from_slice(&c.target.to_be_bytes::<32>());
        // Backlog (uint64).
        out.extend_from_slice(&c.backlog.to_be_bytes::<32>());
        // Resources array length.
        out.extend_from_slice(&U256::from(m).to_be_bytes::<32>());
        // Each WeightedResource (uint8 resource, uint64 weight).
        for &(kind, ref weight) in &c.resources {
            out.extend_from_slice(&U256::from(kind).to_be_bytes::<32>());
            out.extend_from_slice(&weight.to_be_bytes::<32>());
        }
    }

    let result_words = (out.len() as u64).div_ceil(32);
    Ok(PrecompileOutput::new(
        (sloads * SLOAD_GAS + result_words * COPY_GAS).min(gas_limit),
        out.into(),
    ))
}

/// Returns `uint256[]` — current-block base fee per resource kind.
fn handle_multi_gas_base_fee(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let fees_key = multi_gas_base_fees_subspace();

    let mut out = Vec::with_capacity(64 + NUM_RESOURCE_KIND as usize * 32);
    // ABI: offset, then length, then values.
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(NUM_RESOURCE_KIND).to_be_bytes::<32>());

    for kind in 0..NUM_RESOURCE_KIND {
        let slot = map_slot(fees_key.as_slice(), CURRENT_BLOCK_FEES_OFFSET + kind);
        let fee = sload_field(input, slot)?;
        out.extend_from_slice(&fee.to_be_bytes::<32>());
    }

    let result_words = (out.len() as u64).div_ceil(32);
    Ok(PrecompileOutput::new(
        ((1 + NUM_RESOURCE_KIND) * SLOAD_GAS + result_words * COPY_GAS).min(gas_limit),
        out.into(),
    ))
}
