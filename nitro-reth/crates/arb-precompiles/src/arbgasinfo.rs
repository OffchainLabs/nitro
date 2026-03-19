use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

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

// Function selectors (keccak256 of Solidity signatures, first 4 bytes).
const GET_L1_BASEFEE_ESTIMATE: [u8; 4] = [0xf5, 0xd6, 0xde, 0xd7]; // getL1BaseFeeEstimate()
const GET_L1_GAS_PRICE_ESTIMATE: [u8; 4] = [0x05, 0x5f, 0x36, 0x2f]; // getL1GasPriceEstimate()
const GET_MINIMUM_GAS_PRICE: [u8; 4] = [0xf9, 0x18, 0x37, 0x9a]; // getMinimumGasPrice()
const GET_PRICES_IN_WEI: [u8; 4] = [0x41, 0xb2, 0x47, 0xa8]; // getPricesInWei()
const GET_GAS_ACCOUNTING_PARAMS: [u8; 4] = [0x61, 0x2a, 0xf1, 0x78]; // getGasAccountingParams()
const GET_CURRENT_TX_L1_FEES: [u8; 4] = [0xc6, 0xf7, 0xde, 0x0e]; // getCurrentTxL1GasFees()
const GET_PRICES_IN_ARBGAS: [u8; 4] = [0x02, 0x19, 0x9f, 0x34]; // getPricesInArbGas()
const GET_L1_BASEFEE_ESTIMATE_INERTIA: [u8; 4] = [0x29, 0xeb, 0x31, 0xee]; // getL1BaseFeeEstimateInertia()
const GET_GAS_BACKLOG: [u8; 4] = [0x1d, 0x5b, 0x5c, 0x20]; // getGasBacklog()
const GET_PRICING_INERTIA: [u8; 4] = [0x3d, 0xfb, 0x45, 0xb9]; // getPricingInertia()
const GET_GAS_BACKLOG_TOLERANCE: [u8; 4] = [0x25, 0x75, 0x4f, 0x91]; // getGasBacklogTolerance()
const GET_L1_PRICING_SURPLUS: [u8; 4] = [0x52, 0x0a, 0xcd, 0xd7]; // getL1PricingSurplus()
const GET_PER_BATCH_GAS_CHARGE: [u8; 4] = [0x6e, 0xcc, 0xa4, 0x5a]; // getPerBatchGasCharge()
const GET_AMORTIZED_COST_CAP_BIPS: [u8; 4] = [0x7a, 0x7d, 0x6b, 0xeb]; // getAmortizedCostCapBips()
const GET_L1_FEES_AVAILABLE: [u8; 4] = [0x5b, 0x39, 0xd2, 0x3c]; // getL1FeesAvailable()
const GET_L1_REWARD_RATE: [u8; 4] = [0x8a, 0x5b, 0x1d, 0x28]; // getL1RewardRate()
const GET_L1_REWARD_RECIPIENT: [u8; 4] = [0x9e, 0x6d, 0x7e, 0x31]; // getL1RewardRecipient()
const GET_L1_PRICING_EQUILIBRATION_UNITS: [u8; 4] = [0xad, 0x26, 0xce, 0x90]; // getL1PricingEquilibrationUnits()
const GET_LAST_L1_PRICING_UPDATE_TIME: [u8; 4] = [0x13, 0x8b, 0x47, 0xb4]; // getLastL1PricingUpdateTime()
const GET_L1_PRICING_FUNDS_DUE_FOR_REWARDS: [u8; 4] = [0x96, 0x3d, 0x60, 0x02]; // getL1PricingFundsDueForRewards()
const GET_L1_PRICING_UNITS_SINCE_UPDATE: [u8; 4] = [0xef, 0xf0, 0x13, 0x06]; // getL1PricingUnitsSinceUpdate()
const GET_LAST_L1_PRICING_SURPLUS: [u8; 4] = [0x29, 0x87, 0xd0, 0x27]; // getLastL1PricingSurplus()
const GET_MAX_BLOCK_GAS_LIMIT: [u8; 4] = [0x03, 0x71, 0xfd, 0xb4]; // getMaxBlockGasLimit()
const GET_MAX_TX_GAS_LIMIT: [u8; 4] = [0xaa, 0xe1, 0xcd, 0x4c]; // getMaxTxGasLimit()
const GET_GAS_PRICING_CONSTRAINTS: [u8; 4] = [0x23, 0x20, 0x27, 0xd1]; // getGasPricingConstraints()
const GET_MULTI_GAS_PRICING_CONSTRAINTS: [u8; 4] = [0xbb, 0xfc, 0x0a, 0x72]; // getMultiGasPricingConstraints()
const GET_MULTI_GAS_BASE_FEE: [u8; 4] = [0xc0, 0xe1, 0x0b, 0xbb]; // getMultiGasBaseFee()
                                                                  // Legacy selectors for WithAggregator variants (aggregator param is ignored post-v4).
const GET_PRICES_IN_WEI_WITH_AGG: [u8; 4] = [0xba, 0x9c, 0x91, 0x6e]; // getPricesInWeiWithAggregator(address)
const GET_PRICES_IN_ARBGAS_WITH_AGG: [u8; 4] = [0x7a, 0x1e, 0xa7, 0x32]; // getPricesInArbGasWithAggregator(address)

// Gas costs.
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
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    let result = match selector {
        GET_L1_BASEFEE_ESTIMATE | GET_L1_GAS_PRICE_ESTIMATE => {
            read_l1_field(&mut input, L1_PRICE_PER_UNIT)
        }
        GET_MINIMUM_GAS_PRICE => read_l2_field(&mut input, L2_MIN_BASE_FEE),
        GET_PRICES_IN_WEI | GET_PRICES_IN_WEI_WITH_AGG => handle_prices_in_wei(&mut input),
        GET_GAS_ACCOUNTING_PARAMS => handle_gas_accounting_params(&mut input),
        GET_CURRENT_TX_L1_FEES => {
            // Nitro reads c.txProcessor.PosterFee (in-memory, no storage access).
            // We use a thread-local set by the executor to avoid extra sloads
            // that would charge EVM gas and cause gasUsed to differ.
            let gas_limit = input.gas;
            let fee_wei = crate::get_current_tx_poster_fee();
            let fee = U256::from(fee_wei);
            Ok(PrecompileOutput::new(
                (SLOAD_GAS + COPY_GAS).min(gas_limit),
                fee.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        GET_PRICES_IN_ARBGAS | GET_PRICES_IN_ARBGAS_WITH_AGG => handle_prices_in_arbgas(&mut input),
        GET_L1_BASEFEE_ESTIMATE_INERTIA => read_l1_field(&mut input, L1_INERTIA),
        GET_GAS_BACKLOG => read_l2_field(&mut input, L2_GAS_BACKLOG),
        GET_PRICING_INERTIA => read_l2_field(&mut input, L2_PRICING_INERTIA),
        GET_GAS_BACKLOG_TOLERANCE => read_l2_field(&mut input, L2_BACKLOG_TOLERANCE),
        GET_L1_PRICING_SURPLUS => handle_l1_pricing_surplus(&mut input),
        GET_PER_BATCH_GAS_CHARGE => read_l1_field(&mut input, L1_PER_BATCH_GAS_COST),
        GET_AMORTIZED_COST_CAP_BIPS => read_l1_field(&mut input, L1_AMORTIZED_COST_CAP_BIPS),
        // GetL1FeesAvailable: ArbOS >= 10
        GET_L1_FEES_AVAILABLE => {
            if let Some(r) = crate::check_method_version(10, 0) {
                return r;
            }
            read_l1_field(&mut input, L1_FEES_AVAILABLE)
        }
        // GetL1RewardRate: ArbOS >= 11
        GET_L1_REWARD_RATE => {
            if let Some(r) = crate::check_method_version(11, 0) {
                return r;
            }
            read_l1_field(&mut input, L1_PER_UNIT_REWARD)
        }
        // GetL1RewardRecipient: ArbOS >= 11
        GET_L1_REWARD_RECIPIENT => {
            if let Some(r) = crate::check_method_version(11, 0) {
                return r;
            }
            read_l1_field(&mut input, L1_PAY_REWARDS_TO)
        }
        // GetL1PricingEquilibrationUnits: ArbOS >= 20
        GET_L1_PRICING_EQUILIBRATION_UNITS => {
            if let Some(r) = crate::check_method_version(20, 0) {
                return r;
            }
            read_l1_field(&mut input, L1_EQUILIBRATION_UNITS)
        }
        // GetLastL1PricingUpdateTime: ArbOS >= 20
        GET_LAST_L1_PRICING_UPDATE_TIME => {
            if let Some(r) = crate::check_method_version(20, 0) {
                return r;
            }
            read_l1_field(&mut input, L1_LAST_UPDATE_TIME)
        }
        // GetL1PricingFundsDueForRewards: ArbOS >= 20
        GET_L1_PRICING_FUNDS_DUE_FOR_REWARDS => {
            if let Some(r) = crate::check_method_version(20, 0) {
                return r;
            }
            read_l1_field(&mut input, L1_FUNDS_DUE_FOR_REWARDS)
        }
        // GetL1PricingUnitsSinceUpdate: ArbOS >= 20
        GET_L1_PRICING_UNITS_SINCE_UPDATE => {
            if let Some(r) = crate::check_method_version(20, 0) {
                return r;
            }
            read_l1_field(&mut input, L1_UNITS_SINCE)
        }
        // GetLastL1PricingSurplus: ArbOS >= 20
        GET_LAST_L1_PRICING_SURPLUS => {
            if let Some(r) = crate::check_method_version(20, 0) {
                return r;
            }
            read_l1_field(&mut input, L1_LAST_SURPLUS)
        }
        // GetMaxBlockGasLimit: ArbOS >= 50
        GET_MAX_BLOCK_GAS_LIMIT => {
            if let Some(r) = crate::check_method_version(50, 0) {
                return r;
            }
            read_l2_field(&mut input, L2_PER_BLOCK_GAS_LIMIT)
        }
        // GetMaxTxGasLimit: ArbOS >= 50
        GET_MAX_TX_GAS_LIMIT => {
            if let Some(r) = crate::check_method_version(50, 0) {
                return r;
            }
            read_l2_field(&mut input, L2_PER_TX_GAS_LIMIT)
        }
        // GetGasPricingConstraints: ArbOS >= 50
        GET_GAS_PRICING_CONSTRAINTS => {
            if let Some(r) = crate::check_method_version(50, 0) {
                return r;
            }
            handle_gas_pricing_constraints(&mut input)
        }
        // GetMultiGasPricingConstraints: ArbOS >= 60
        GET_MULTI_GAS_PRICING_CONSTRAINTS => {
            if let Some(r) = crate::check_method_version(60, 0) {
                return r;
            }
            handle_multi_gas_pricing_constraints(&mut input)
        }
        // GetMultiGasBaseFee: ArbOS >= 60
        GET_MULTI_GAS_BASE_FEE => {
            if let Some(r) = crate::check_method_version(60, 0) {
                return r;
            }
            handle_multi_gas_base_fee(&mut input)
        }
        _ => Err(PrecompileError::other("unknown selector")),
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
    load_arbos(input)?;

    let l1_price = sload_field(input, subspace_slot(L1_PRICING_SUBSPACE, L1_PRICE_PER_UNIT))?;
    let l2_base = sload_field(input, subspace_slot(L2_PRICING_SUBSPACE, L2_BASE_FEE))?;
    let l2_min = sload_field(input, subspace_slot(L2_PRICING_SUBSPACE, L2_MIN_BASE_FEE))?;

    let wei_for_l1_calldata = l1_price.saturating_mul(U256::from(TX_DATA_NON_ZERO_GAS));
    let per_l2_tx = wei_for_l1_calldata.saturating_mul(U256::from(ASSUMED_SIMPLE_TX_SIZE));
    let per_arbgas_base = l2_base.min(l2_min);
    let per_arbgas_congestion = l2_base.saturating_sub(per_arbgas_base);
    let per_arbgas_total = l2_base;
    let wei_for_l2_storage = l2_base.saturating_mul(U256::from(STORAGE_WRITE_COST));

    let mut out = Vec::with_capacity(192);
    out.extend_from_slice(&per_l2_tx.to_be_bytes::<32>());
    out.extend_from_slice(&wei_for_l1_calldata.to_be_bytes::<32>());
    out.extend_from_slice(&wei_for_l2_storage.to_be_bytes::<32>());
    out.extend_from_slice(&per_arbgas_base.to_be_bytes::<32>());
    out.extend_from_slice(&per_arbgas_congestion.to_be_bytes::<32>());
    out.extend_from_slice(&per_arbgas_total.to_be_bytes::<32>());

    // baseFee is read from evm.Context (free), only 2 body SLOADs (PricePerUnit, MinBaseFee).
    let arg_words = ((data_len as u64).saturating_sub(4) + 31) / 32;
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
    load_arbos(input)?;

    let l1_price = sload_field(input, subspace_slot(L1_PRICING_SUBSPACE, L1_PRICE_PER_UNIT))?;
    let l2_base = sload_field(input, subspace_slot(L2_PRICING_SUBSPACE, L2_BASE_FEE))?;

    let wei_for_l1_calldata = l1_price.saturating_mul(U256::from(TX_DATA_NON_ZERO_GAS));
    let wei_per_l2_tx = wei_for_l1_calldata.saturating_mul(U256::from(ASSUMED_SIMPLE_TX_SIZE));

    let (gas_for_l1_calldata, gas_per_l2_tx) = if l2_base > U256::ZERO {
        (wei_for_l1_calldata / l2_base, wei_per_l2_tx / l2_base)
    } else {
        (U256::ZERO, U256::ZERO)
    };

    let mut out = Vec::with_capacity(96);
    out.extend_from_slice(&gas_per_l2_tx.to_be_bytes::<32>());
    out.extend_from_slice(&gas_for_l1_calldata.to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(STORAGE_WRITE_COST).to_be_bytes::<32>());

    // baseFee is read from evm.Context (free), only 1 body SLOAD (PricePerUnit).
    let arg_words = ((data_len as u64).saturating_sub(4) + 31) / 32;
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

    let result_words = (out.len() as u64 + 31) / 32;
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

    let result_words = (out.len() as u64 + 31) / 32;
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

    let result_words = (out.len() as u64 + 31) / 32;
    Ok(PrecompileOutput::new(
        ((1 + NUM_RESOURCE_KIND) * SLOAD_GAS + result_words * COPY_GAS).min(gas_limit),
        out.into(),
    ))
}
