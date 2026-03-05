use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::arbsys::get_cached_l1_block_number;
use crate::storage_slot::{
    root_slot, subspace_slot, ARBOS_STATE_ADDRESS, GENESIS_BLOCK_NUM_OFFSET,
    L1_PRICING_SUBSPACE, L2_PRICING_SUBSPACE,
};

/// NodeInterface virtual contract address (0xc8).
pub const NODE_INTERFACE_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0xc8,
]);

// Function selectors.
const GAS_ESTIMATE_COMPONENTS: [u8; 4] = [0xc9, 0x4e, 0x6e, 0xeb];
const GAS_ESTIMATE_L1_COMPONENT: [u8; 4] = [0x77, 0xd4, 0x88, 0xa2];
const NITRO_GENESIS_BLOCK: [u8; 4] = [0x93, 0xa2, 0xfe, 0x21];
const BLOCK_L1_NUM: [u8; 4] = [0x6f, 0x27, 0x5e, 0xf2];
const L2_BLOCK_RANGE_FOR_L1: [u8; 4] = [0x48, 0xe7, 0xf8, 0x11];
const ESTIMATE_RETRYABLE_TICKET: [u8; 4] = [0xc3, 0xdc, 0x58, 0x79];
const CONSTRUCT_OUTBOX_PROOF: [u8; 4] = [0x42, 0x69, 0x63, 0x50];
const FIND_BATCH_CONTAINING_BLOCK: [u8; 4] = [0x81, 0xf1, 0xad, 0xaf];
const GET_L1_CONFIRMATIONS: [u8; 4] = [0xe5, 0xca, 0x23, 0x8c];
const LEGACY_LOOKUP_MESSAGE_BATCH_PROOF: [u8; 4] = [0x89, 0x49, 0x62, 0x70];

// L1 pricing field offsets.
const L1_PRICE_PER_UNIT: u64 = 7;

// L2 pricing field offsets.
const L2_BASE_FEE: u64 = 2;

// Gas costs.
const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;

/// Non-zero calldata gas cost per byte.
const TX_DATA_NON_ZERO_GAS: u64 = 16;

/// Padding applied to L1 fee estimates (110% = 11000 bips).
const GAS_ESTIMATION_L1_PRICE_PADDING_BIPS: u64 = 11000;

pub fn create_nodeinterface_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("nodeinterface"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    let result = match selector {
        GAS_ESTIMATE_COMPONENTS => handle_gas_estimate_components(&mut input),
        GAS_ESTIMATE_L1_COMPONENT => handle_gas_estimate_l1_component(&mut input),
        NITRO_GENESIS_BLOCK => handle_nitro_genesis_block(&mut input),
        BLOCK_L1_NUM => handle_block_l1_num(&mut input),
        // Methods requiring chain-level access (blockchain history, batch data, logs).
        // These are handled at the RPC layer via InterceptRPCMessage, not as
        // EVM precompiles. Revert here since the required backend is not available.
        L2_BLOCK_RANGE_FOR_L1
        | ESTIMATE_RETRYABLE_TICKET
        | CONSTRUCT_OUTBOX_PROOF
        | FIND_BATCH_CONTAINING_BLOCK
        | GET_L1_CONFIRMATIONS
        | LEGACY_LOOKUP_MESSAGE_BATCH_PROOF => {
            Err(PrecompileError::other("method only available via RPC"))
        }
        _ => Err(PrecompileError::other("unknown selector")),
    };
    crate::gas_check(gas_limit, result)
}

/// gasEstimateComponents(address,bool,bytes) → (uint64, uint64, uint256, uint256)
///
/// Returns: (gasEstimate, gasEstimateForL1, baseFee, l1BaseFeeEstimate)
///
/// The full gas estimate requires calling back into eth_estimateGas which
/// is not possible from within a precompile. We return the L1 component
/// and basefee; the total estimate is left as 0 (callers should use
/// eth_estimateGas for the total).
fn handle_gas_estimate_components(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let l1_price =
        sload_field(input, subspace_slot(L1_PRICING_SUBSPACE, L1_PRICE_PER_UNIT))?;
    let basefee = sload_field(input, subspace_slot(L2_PRICING_SUBSPACE, L2_BASE_FEE))?;

    // Compute L1 gas cost for a simple transaction.
    // PosterDataCost computes the L1 fee from the message data, then divides by basefee.
    // Here we estimate using the calldata from the input parameters.
    let gas_for_l1 = estimate_l1_gas(input, l1_price, basefee);

    let mut out = Vec::with_capacity(128);
    // gasEstimate: 0 (full estimate requires eth_estimateGas)
    out.extend_from_slice(&U256::ZERO.to_be_bytes::<32>());
    // gasEstimateForL1
    out.extend_from_slice(&U256::from(gas_for_l1).to_be_bytes::<32>());
    // baseFee
    out.extend_from_slice(&basefee.to_be_bytes::<32>());
    // l1BaseFeeEstimate
    out.extend_from_slice(&l1_price.to_be_bytes::<32>());

    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        out.into(),
    ))
}

/// gasEstimateL1Component(address,bool,bytes) → (uint64, uint256, uint256)
///
/// Returns: (gasEstimateForL1, baseFee, l1BaseFeeEstimate)
fn handle_gas_estimate_l1_component(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let l1_price =
        sload_field(input, subspace_slot(L1_PRICING_SUBSPACE, L1_PRICE_PER_UNIT))?;
    let basefee = sload_field(input, subspace_slot(L2_PRICING_SUBSPACE, L2_BASE_FEE))?;

    let gas_for_l1 = estimate_l1_gas(input, l1_price, basefee);

    let mut out = Vec::with_capacity(96);
    // gasEstimateForL1
    out.extend_from_slice(&U256::from(gas_for_l1).to_be_bytes::<32>());
    // baseFee
    out.extend_from_slice(&basefee.to_be_bytes::<32>());
    // l1BaseFeeEstimate
    out.extend_from_slice(&l1_price.to_be_bytes::<32>());

    Ok(PrecompileOutput::new(
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
        out.into(),
    ))
}

/// nitroGenesisBlock() → uint64
fn handle_nitro_genesis_block(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let genesis_block_num =
        sload_field(input, root_slot(GENESIS_BLOCK_NUM_OFFSET))?;

    Ok(PrecompileOutput::new(
        (SLOAD_GAS + COPY_GAS).min(gas_limit),
        genesis_block_num.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// blockL1Num(uint64 blockNum) → uint64
///
/// Returns the L1 block number associated with the given L2 block.
/// Uses the cached L1→L2 block mapping populated during block execution.
fn handle_block_l1_num(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 + 32 {
        return Err(PrecompileError::other("input too short"));
    }

    let block_num: u64 = U256::from_be_slice(&data[4..36])
        .try_into()
        .unwrap_or(u64::MAX);

    let l1_block = get_cached_l1_block_number(block_num).unwrap_or(0);

    Ok(PrecompileOutput::new(
        COPY_GAS.min(input.gas),
        U256::from(l1_block).to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Estimate L1 gas from calldata in the input.
///
/// Computes: posterDataCost = l1PricePerUnit * txDataNonZeroGas * calldataLen
/// Then applies 110% padding and divides by basefee.
fn estimate_l1_gas(input: &PrecompileInput<'_>, l1_price: U256, basefee: U256) -> u64 {
    // Extract the `bytes data` parameter from calldata.
    // ABI: selector(4) + address(32) + bool(32) + offset(32) + length(32) + data...
    let calldata_len = if input.data.len() > 4 + 32 + 32 + 32 + 32 {
        let len_offset = 4 + 32 + 32 + 32;
        let len_bytes = &input.data[len_offset..len_offset + 32];
        U256::from_be_slice(len_bytes)
            .try_into()
            .unwrap_or(0u64)
    } else {
        0u64
    };

    if basefee.is_zero() || l1_price.is_zero() {
        return 0;
    }

    // L1 fee = l1PricePerUnit * txDataNonZeroGas * dataLength
    let l1_fee = l1_price
        .saturating_mul(U256::from(TX_DATA_NON_ZERO_GAS))
        .saturating_mul(U256::from(calldata_len));

    // Apply padding (110% = 11000/10000 bips).
    let padded = l1_fee
        .saturating_mul(U256::from(GAS_ESTIMATION_L1_PRICE_PADDING_BIPS))
        / U256::from(10000u64);

    // Convert to gas units: gasForL1 = paddedFee / basefee
    let gas_for_l1 = padded / basefee;

    gas_for_l1.try_into().unwrap_or(u64::MAX)
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
    Ok(val.data)
}
