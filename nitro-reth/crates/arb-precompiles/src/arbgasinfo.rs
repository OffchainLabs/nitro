use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{
    compute_storage_slot, ARBOS_STATE_ADDRESS, L1_PRICING_SPACE, L2_PRICING_SPACE,
};

/// ArbGasInfo precompile address (0x6c).
pub const ARBGASINFO_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x6c,
]);

// Function selectors.
const GET_L1_BASEFEE_ESTIMATE: [u8; 4] = [0xf5, 0xd6, 0xde, 0xd7];
const GET_L1_GAS_PRICE_ESTIMATE: [u8; 4] = [0x05, 0x5f, 0x36, 0x2f];
const GET_MINIMUM_GAS_PRICE: [u8; 4] = [0xf9, 0x18, 0x37, 0x9a];
const GET_PRICES_IN_WEI: [u8; 4] = [0x41, 0xb2, 0x47, 0xa8];
const GET_GAS_ACCOUNTING_PARAMS: [u8; 4] = [0x61, 0x2a, 0xf1, 0x78];
const GET_CURRENT_TX_L1_FEES: [u8; 4] = [0xc6, 0xf7, 0xde, 0x0e];
const GET_PRICES_IN_ARBGAS: [u8; 4] = [0x02, 0x19, 0x9f, 0x34];
const GET_L1_BASEFEE_ESTIMATE_INERTIA: [u8; 4] = [0x29, 0xeb, 0x31, 0xee];
const GET_GAS_BACKLOG: [u8; 4] = [0x1d, 0x5b, 0x5c, 0x20];
const GET_PRICING_INERTIA: [u8; 4] = [0x3d, 0xfb, 0x45, 0xb9];
const GET_GAS_BACKLOG_TOLERANCE: [u8; 4] = [0x25, 0x75, 0x4f, 0x91];
const GET_L1_PRICING_SURPLUS: [u8; 4] = [0x52, 0x0a, 0xcd, 0xd7];
const GET_PER_BATCH_GAS_CHARGE: [u8; 4] = [0x6e, 0xcc, 0xa4, 0x5a];
const GET_AMORTIZED_COST_CAP_BIPS: [u8; 4] = [0x7a, 0x7d, 0x6b, 0xeb];
const GET_L1_FEES_AVAILABLE: [u8; 4] = [0x5b, 0x39, 0xd2, 0x3c];
const GET_L1_REWARD_RATE: [u8; 4] = [0x8a, 0x5b, 0x1d, 0x28];
const GET_L1_REWARD_RECIPIENT: [u8; 4] = [0x9e, 0x6d, 0x7d, 0xe5];

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
const L1_FEES_AVAILABLE: u64 = 11;

// L2 pricing field offsets (within L2 pricing subspace).
const L2_SPEED_LIMIT: u64 = 0;
const L2_PER_BLOCK_GAS_LIMIT: u64 = 1;
const L2_BASE_FEE: u64 = 2;
const L2_MIN_BASE_FEE: u64 = 3;
const L2_GAS_BACKLOG: u64 = 4;
const L2_PRICING_INERTIA: u64 = 5;
const L2_BACKLOG_TOLERANCE: u64 = 6;

const TX_DATA_NON_ZERO_GAS: u64 = 16;
const ASSUMED_SIMPLE_TX_SIZE: u64 = 140;
const STORAGE_WRITE_COST: u64 = 20_000;

pub fn create_arbgasinfo_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbgasinfo"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    match selector {
        GET_L1_BASEFEE_ESTIMATE | GET_L1_GAS_PRICE_ESTIMATE => {
            read_l1_field(&mut input, L1_PRICE_PER_UNIT)
        }
        GET_MINIMUM_GAS_PRICE => read_l2_field(&mut input, L2_MIN_BASE_FEE),
        GET_PRICES_IN_WEI => handle_prices_in_wei(&mut input),
        GET_GAS_ACCOUNTING_PARAMS => handle_gas_accounting_params(&mut input),
        GET_CURRENT_TX_L1_FEES => {
            // Returns poster fee for current tx; context-dependent.
            let gas_cost = COPY_GAS.min(input.gas);
            Ok(PrecompileOutput::new(gas_cost, vec![0u8; 32].into()))
        }
        GET_PRICES_IN_ARBGAS => handle_prices_in_arbgas(&mut input),
        GET_L1_BASEFEE_ESTIMATE_INERTIA => read_l1_field(&mut input, L1_INERTIA),
        GET_GAS_BACKLOG => read_l2_field(&mut input, L2_GAS_BACKLOG),
        GET_PRICING_INERTIA => read_l2_field(&mut input, L2_PRICING_INERTIA),
        GET_GAS_BACKLOG_TOLERANCE => read_l2_field(&mut input, L2_BACKLOG_TOLERANCE),
        GET_L1_PRICING_SURPLUS => read_l1_field(&mut input, L1_LAST_SURPLUS),
        GET_PER_BATCH_GAS_CHARGE => read_l1_field(&mut input, L1_PER_BATCH_GAS_COST),
        GET_AMORTIZED_COST_CAP_BIPS => read_l1_field(&mut input, L1_AMORTIZED_COST_CAP_BIPS),
        GET_L1_FEES_AVAILABLE => read_l1_field(&mut input, L1_FEES_AVAILABLE),
        GET_L1_REWARD_RATE => read_l1_field(&mut input, L1_PER_UNIT_REWARD),
        GET_L1_REWARD_RECIPIENT => read_l1_field(&mut input, L1_PAY_REWARDS_TO),
        _ => Err(PrecompileError::other("unknown selector")),
    }
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
    let l1_slot = compute_storage_slot(&[], L1_PRICING_SPACE);
    let field_slot = compute_storage_slot(&[l1_slot], offset);
    let value = sload_field(input, field_slot)?;
    Ok(PrecompileOutput::new(
        (SLOAD_GAS + COPY_GAS).min(gas_limit),
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn read_l2_field(input: &mut PrecompileInput<'_>, offset: u64) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;
    let l2_slot = compute_storage_slot(&[], L2_PRICING_SPACE);
    let field_slot = compute_storage_slot(&[l2_slot], offset);
    let value = sload_field(input, field_slot)?;
    Ok(PrecompileOutput::new(
        (SLOAD_GAS + COPY_GAS).min(gas_limit),
        value.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_prices_in_wei(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let l1_slot = compute_storage_slot(&[], L1_PRICING_SPACE);
    let l2_slot = compute_storage_slot(&[], L2_PRICING_SPACE);

    let l1_price = sload_field(input, compute_storage_slot(&[l1_slot], L1_PRICE_PER_UNIT))?;
    let l2_base = sload_field(input, compute_storage_slot(&[l2_slot], L2_BASE_FEE))?;
    let l2_min = sload_field(input, compute_storage_slot(&[l2_slot], L2_MIN_BASE_FEE))?;

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

    Ok(PrecompileOutput::new(
        (3 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        out.into(),
    ))
}

fn handle_gas_accounting_params(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let l2_slot = compute_storage_slot(&[], L2_PRICING_SPACE);
    let speed_limit = sload_field(input, compute_storage_slot(&[l2_slot], L2_SPEED_LIMIT))?;
    let gas_limit_val =
        sload_field(input, compute_storage_slot(&[l2_slot], L2_PER_BLOCK_GAS_LIMIT))?;

    let mut out = Vec::with_capacity(96);
    out.extend_from_slice(&speed_limit.to_be_bytes::<32>());
    out.extend_from_slice(&gas_limit_val.to_be_bytes::<32>());
    out.extend_from_slice(&gas_limit_val.to_be_bytes::<32>());

    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        out.into(),
    ))
}

fn handle_prices_in_arbgas(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let l1_slot = compute_storage_slot(&[], L1_PRICING_SPACE);
    let l2_slot = compute_storage_slot(&[], L2_PRICING_SPACE);

    let l1_price = sload_field(input, compute_storage_slot(&[l1_slot], L1_PRICE_PER_UNIT))?;
    let l2_base = sload_field(input, compute_storage_slot(&[l2_slot], L2_BASE_FEE))?;

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

    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        out.into(),
    ))
}
