use alloy_consensus::{SignableTransaction, TxEip1559, TxEnvelope};
use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{keccak256, Address, Bytes, ChainId, Signature, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::{
    arbsys::get_cached_l1_block_number,
    storage_slot::{
        root_slot, subspace_slot, ARBOS_STATE_ADDRESS, GENESIS_BLOCK_NUM_OFFSET,
        L1_PRICING_SUBSPACE, L2_PRICING_SUBSPACE,
    },
};

/// NodeInterface virtual contract address (0xc8).
pub const NODE_INTERFACE_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0xc8,
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
const L2_MIN_BASE_FEE: u64 = 3;

// Gas costs.
const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;

pub fn create_nodeinterface_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("nodeinterface"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    let data = input.data;
    if data.len() < 4 {
        return crate::burn_all_revert(gas_limit);
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    crate::init_precompile_gas(data.len());

    let result = match selector {
        GAS_ESTIMATE_COMPONENTS => handle_gas_estimate_components(&mut input),
        GAS_ESTIMATE_L1_COMPONENT => handle_gas_estimate_l1_component(&mut input),
        NITRO_GENESIS_BLOCK => handle_nitro_genesis_block(&mut input),
        BLOCK_L1_NUM => handle_block_l1_num(&mut input),
        // Batch-fetcher-dependent methods: return 0 when no batch fetcher is
        // wired (validators, replayers, nodes not following L1) so bridge
        // tooling can distinguish "unknown/pending" from "not implemented".
        GET_L1_CONFIRMATIONS => handle_zero_u64(&input),
        FIND_BATCH_CONTAINING_BLOCK => handle_zero_u64(&input),
        // Legacy classic-chain outbox lookup — no classic chain, so return
        // the 9-element all-zero tuple.
        LEGACY_LOOKUP_MESSAGE_BATCH_PROOF => handle_legacy_lookup_empty(&input),
        // Still RPC-layer only (need header scans or tx construction).
        L2_BLOCK_RANGE_FOR_L1 | ESTIMATE_RETRYABLE_TICKET | CONSTRUCT_OUTBOX_PROOF => {
            Err(PrecompileError::other("method only available via RPC"))
        }
        _ => return crate::burn_all_revert(gas_limit),
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

    let l1_price = sload_field(input, subspace_slot(L1_PRICING_SUBSPACE, L1_PRICE_PER_UNIT))?;
    let basefee = sload_field(input, subspace_slot(L2_PRICING_SUBSPACE, L2_BASE_FEE))?;
    let min_basefee = sload_field(input, subspace_slot(L2_PRICING_SUBSPACE, L2_MIN_BASE_FEE))?;
    let chain_id_u256 = sload_field(input, root_slot(crate::storage_slot::CHAIN_ID_OFFSET))?;
    let chain_id: ChainId = chain_id_u256.try_into().unwrap_or(0);
    let brotli_level = sload_field(input, root_slot(crate::storage_slot::BROTLI_COMPRESSION_LEVEL_OFFSET))?
        .try_into()
        .unwrap_or(0u64);

    let gas_for_l1 = estimate_l1_gas(input, l1_price, basefee, min_basefee, chain_id, brotli_level);

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

    let l1_price = sload_field(input, subspace_slot(L1_PRICING_SUBSPACE, L1_PRICE_PER_UNIT))?;
    let basefee = sload_field(input, subspace_slot(L2_PRICING_SUBSPACE, L2_BASE_FEE))?;
    let min_basefee = sload_field(input, subspace_slot(L2_PRICING_SUBSPACE, L2_MIN_BASE_FEE))?;
    let chain_id_u256 = sload_field(input, root_slot(crate::storage_slot::CHAIN_ID_OFFSET))?;
    let chain_id: ChainId = chain_id_u256.try_into().unwrap_or(0);
    let brotli_level = sload_field(input, root_slot(crate::storage_slot::BROTLI_COMPRESSION_LEVEL_OFFSET))?
        .try_into()
        .unwrap_or(0u64);

    let gas_for_l1 = estimate_l1_gas(input, l1_price, basefee, min_basefee, chain_id, brotli_level);

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

    let genesis_block_num = sload_field(input, root_slot(GENESIS_BLOCK_NUM_OFFSET))?;

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
        return crate::burn_all_revert(input.gas);
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

/// Encode a single uint64/uint256 zero — used when batch-fetcher methods
/// can't resolve data.
fn handle_zero_u64(input: &PrecompileInput<'_>) -> PrecompileResult {
    Ok(PrecompileOutput::new(
        COPY_GAS.min(input.gas),
        U256::ZERO.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// legacyLookupMessageBatchProof returns the 9-value all-zero tuple —
/// arbreth has no classic-chain outbox to look up.
///
/// ABI return:
///   (bytes32[] proof, uint256 path, address l2Sender, address l1Dest,
///    uint256 l2Block, uint256 l1Block, uint256 timestamp, uint256 amount,
///    bytes calldataForL1)
fn handle_legacy_lookup_empty(input: &PrecompileInput<'_>) -> PrecompileResult {
    // 9 head slots for the tuple + 1 slot each for the two dynamic arrays'
    // length (proof and calldataForL1), emitted inline since length is 0.
    // Head layout:
    //   offset 0x00: proof offset (dynamic)       → points to trailing data
    //   offset 0x20: path
    //   offset 0x40: l2Sender
    //   offset 0x60: l1Dest
    //   offset 0x80: l2Block
    //   offset 0xA0: l1Block
    //   offset 0xC0: timestamp
    //   offset 0xE0: amount
    //   offset 0x100: calldataForL1 offset (dynamic)
    //   offset 0x120: proof length (0)
    //   offset 0x140: calldataForL1 length (0)
    let mut out = vec![0u8; 0x160];
    // proof offset = 0x140 (after the 9 head words, points to proof length)
    U256::from(0x140u64)
        .to_be_bytes::<32>()
        .iter()
        .enumerate()
        .for_each(|(i, b)| out[i] = *b);
    // calldataForL1 offset = 0x140 + 0x20 (after proof length = 0)
    U256::from(0x160u64)
        .to_be_bytes::<32>()
        .iter()
        .enumerate()
        .for_each(|(i, b)| out[0x100 + i] = *b);
    Ok(PrecompileOutput::new(COPY_GAS.min(input.gas), out.into()))
}

fn estimate_l1_gas(
    input: &PrecompileInput<'_>,
    l1_price: U256,
    basefee: U256,
    min_basefee: U256,
    chain_id: ChainId,
    brotli_level: u64,
) -> u64 {
    let (to_addr, contract_creation, data) = match decode_estimate_args(input.data) {
        Some(v) => v,
        None => return 0,
    };
    compute_l1_gas_for_estimate(
        chain_id,
        to_addr,
        contract_creation,
        U256::ZERO,
        data,
        l1_price,
        basefee,
        min_basefee,
        brotli_level,
    )
}

/// L1 gas estimate: brotli-compress a fake EIP-1559 tx, pad units by
/// `(units + 256) * 1.01`, multiply by `pricePerUnit`, pad posterCost by
/// `1.10`, then divide by `max(basefee * 7/8, minBaseFee)`.
pub fn compute_l1_gas_for_estimate(
    chain_id: ChainId,
    to: Address,
    contract_creation: bool,
    value: U256,
    data: Bytes,
    l1_price: U256,
    basefee: U256,
    min_basefee: U256,
    brotli_level: u64,
) -> u64 {
    if basefee.is_zero() || l1_price.is_zero() {
        return 0;
    }
    let tx_bytes = build_fake_tx_bytes(chain_id, to, contract_creation, value, data);
    let raw_units = arbos::l1_pricing::poster_units_from_bytes(&tx_bytes, brotli_level);
    let padded_units = raw_units
        .saturating_add(arbos::l1_pricing::ESTIMATION_PADDING_UNITS)
        .saturating_mul(10_000 + arbos::l1_pricing::ESTIMATION_PADDING_BASIS_POINTS)
        / 10_000;
    let poster_cost = l1_price.saturating_mul(U256::from(padded_units));
    let posting_padded = poster_cost.saturating_mul(U256::from(11_000u64)) / U256::from(10_000u64);
    let adjusted = basefee.saturating_mul(U256::from(7u64)) / U256::from(8u64);
    let gas_price = if adjusted < min_basefee { min_basefee } else { adjusted };
    if gas_price.is_zero() {
        return 0;
    }
    (posting_padded / gas_price).try_into().unwrap_or(u64::MAX)
}

/// Decode `gasEstimateComponents(address,bool,bytes)` calldata into
/// `(to, contractCreation, data)`.
pub fn decode_estimate_args(data: &[u8]) -> Option<(Address, bool, Bytes)> {
    if data.len() < 4 + 4 * 32 {
        return None;
    }
    let to = Address::from_slice(&data[16..36]);
    let creation = data[4 + 32 + 31] != 0;
    let bytes_offset: usize = U256::from_be_slice(&data[4 + 64..4 + 96]).try_into().ok()?;
    let bytes_pos = 4usize.checked_add(bytes_offset)?;
    if data.len() < bytes_pos + 32 {
        return None;
    }
    let bytes_len: usize = U256::from_be_slice(&data[bytes_pos..bytes_pos + 32])
        .try_into()
        .ok()?;
    let data_start = bytes_pos + 32;
    if data.len() < data_start + bytes_len {
        return None;
    }
    Some((
        to,
        creation,
        Bytes::copy_from_slice(&data[data_start..data_start + bytes_len]),
    ))
}

/// Build the EIP-2718 envelope of a fake EIP-1559 tx used to size the
/// calldata payload for gas estimation (hard-coded random
/// nonce/tip/feeCap/gas/sig fields).
pub fn build_fake_tx_bytes(
    chain_id: ChainId,
    to: Address,
    contract_creation: bool,
    value: U256,
    data: Bytes,
) -> Vec<u8> {
    let nonce = u64::from_be_bytes(keccak256(b"Nonce")[..8].try_into().unwrap());
    let max_priority = u128::from(u32::from_be_bytes(
        keccak256(b"GasTipCap")[..4].try_into().unwrap(),
    ));
    let max_fee = u128::from(u32::from_be_bytes(
        keccak256(b"GasFeeCap")[..4].try_into().unwrap(),
    ));
    let gas_limit = u64::from(u32::from_be_bytes(
        keccak256(b"Gas")[..4].try_into().unwrap(),
    ));
    let r = U256::from_be_bytes(keccak256(b"R").0);
    let s = U256::from_be_bytes(keccak256(b"S").0);

    let kind = if contract_creation {
        revm::primitives::TxKind::Create
    } else {
        revm::primitives::TxKind::Call(to)
    };

    let tx = TxEip1559 {
        chain_id,
        nonce,
        gas_limit,
        max_fee_per_gas: max_fee,
        max_priority_fee_per_gas: max_priority,
        to: kind,
        value,
        access_list: Default::default(),
        input: data,
    };

    let signature = Signature::new(r, s, false);
    let signed = tx.into_signed(signature);
    use alloy_eips::eip2718::Encodable2718;
    let envelope = TxEnvelope::Eip1559(signed);
    envelope.encoded_2718()
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
