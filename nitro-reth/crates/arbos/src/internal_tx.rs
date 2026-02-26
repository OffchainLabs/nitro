use alloy_primitives::{Address, B256, U256};

use arb_chainspec::arbos_version;

use crate::arbos_state::ArbosState;
use crate::arbos_types::{BatchDataStats, legacy_cost_for_stats};
use crate::burn::Burner;

/// Standard Ethereum base transaction gas.
const TX_GAS: u64 = 21_000;

// ---------------------------------------------------------------------------
// Method selectors (keccak256 of ABI signatures)
// ---------------------------------------------------------------------------

/// startBlock(uint256,uint64,uint64,uint64)
pub const INTERNAL_TX_START_BLOCK_METHOD_ID: [u8; 4] = [0x6b, 0xf6, 0xa4, 0x2d];

/// batchPostingReport(uint256,address,uint64,uint64,uint256)
pub const INTERNAL_TX_BATCH_POSTING_REPORT_METHOD_ID: [u8; 4] = [0xb6, 0x69, 0x37, 0x71];

/// batchPostingReportV2(uint256,address,uint64,uint64,uint64,uint64,uint256)
pub const INTERNAL_TX_BATCH_POSTING_REPORT_V2_METHOD_ID: [u8; 4] = [0xa6, 0xf3, 0xde, 0x31];

// ---------------------------------------------------------------------------
// Well-known system addresses
// ---------------------------------------------------------------------------

pub const ARB_RETRYABLE_TX_ADDRESS: Address = {
    let mut bytes = [0u8; 20];
    bytes[18] = 0x00;
    bytes[19] = 0x6e;
    Address::new(bytes)
};

pub const ARB_SYS_ADDRESS: Address = {
    let mut bytes = [0u8; 20];
    bytes[19] = 0x64;
    Address::new(bytes)
};

/// Additional tokens in the calldata for floor gas accounting.
///
/// Raw batch has a 40-byte header (5 uint64s) that doesn't come from calldata.
/// The addSequencerL2BatchFromOrigin call has a selector + 5 additional fields.
/// Token count: 4*4 (selector) + 4*24 (uint64 padding) + 4*12+12 (address) = 172
pub const FLOOR_GAS_ADDITIONAL_TOKENS: u64 = 172;

// ---------------------------------------------------------------------------
// L1 block info
// ---------------------------------------------------------------------------

/// L1 block info passed to internal transactions.
#[derive(Debug, Clone)]
pub struct L1Info {
    pub poster: Address,
    pub l1_block_number: u64,
    pub l1_timestamp: u64,
}

impl L1Info {
    pub fn new(poster: Address, l1_block_number: u64, l1_timestamp: u64) -> Self {
        Self {
            poster,
            l1_block_number,
            l1_timestamp,
        }
    }
}

// ---------------------------------------------------------------------------
// Event IDs
// ---------------------------------------------------------------------------

pub const L2_TO_L1_TRANSACTION_EVENT_ID: B256 = {
    let bytes: [u8; 32] = [
        0x5b, 0xaa, 0xbe, 0x19, 0x5c, 0x3e, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    ];
    B256::new(bytes)
};

pub const L2_TO_L1_TX_EVENT_ID: B256 = {
    let bytes: [u8; 32] = [
        0x3e, 0x7a, 0xdf, 0x9f, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    ];
    B256::new(bytes)
};

pub const REDEEM_SCHEDULED_EVENT_ID: B256 = {
    let bytes: [u8; 32] = [
        0x5a, 0x4c, 0x71, 0x5f, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
        0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    ];
    B256::new(bytes)
};

// ---------------------------------------------------------------------------
// Decoded internal tx data
// ---------------------------------------------------------------------------

/// Decoded startBlock(uint256 l1BaseFee, uint64 l1BlockNumber, uint64 l2BlockNumber, uint64 timePassed)
#[derive(Debug, Clone)]
pub struct StartBlockData {
    pub l1_base_fee: U256,
    pub l1_block_number: u64,
    pub l2_block_number: u64,
    pub time_passed: u64,
}

/// Decoded batchPostingReport(uint256, address, uint64, uint64, uint256)
#[derive(Debug, Clone)]
pub struct BatchPostingReportData {
    pub batch_timestamp: u64,
    pub batch_poster: Address,
    pub batch_data_gas: u64,
    pub l1_base_fee: U256,
}

/// Decoded batchPostingReportV2(uint256, address, uint64, uint64, uint64, uint64, uint256)
#[derive(Debug, Clone)]
pub struct BatchPostingReportV2Data {
    pub batch_timestamp: u64,
    pub batch_poster: Address,
    pub batch_calldata_length: u64,
    pub batch_calldata_non_zeros: u64,
    pub batch_extra_gas: u64,
    pub l1_base_fee: U256,
}

// ---------------------------------------------------------------------------
// ABI encoding
// ---------------------------------------------------------------------------

/// Creates the ABI-encoded data for a startBlock internal transaction.
pub fn encode_start_block(
    l1_base_fee: U256,
    l1_block_number: u64,
    l2_block_number: u64,
    time_passed: u64,
) -> Vec<u8> {
    let mut data = Vec::with_capacity(4 + 32 * 4);
    data.extend_from_slice(&INTERNAL_TX_START_BLOCK_METHOD_ID);
    data.extend_from_slice(&l1_base_fee.to_be_bytes::<32>());
    data.extend_from_slice(&B256::left_padding_from(&l1_block_number.to_be_bytes()).0);
    data.extend_from_slice(&B256::left_padding_from(&l2_block_number.to_be_bytes()).0);
    data.extend_from_slice(&B256::left_padding_from(&time_passed.to_be_bytes()).0);
    data
}

// ---------------------------------------------------------------------------
// ABI decoding
// ---------------------------------------------------------------------------

fn decode_start_block(data: &[u8]) -> Result<StartBlockData, String> {
    if data.len() < 4 + 32 * 4 {
        return Err(format!(
            "start block data too short: expected >= 132, got {}",
            data.len()
        ));
    }
    let args = &data[4..];
    Ok(StartBlockData {
        l1_base_fee: U256::from_be_slice(&args[0..32]),
        l1_block_number: U256::from_be_slice(&args[32..64]).to::<u64>(),
        l2_block_number: U256::from_be_slice(&args[64..96]).to::<u64>(),
        time_passed: U256::from_be_slice(&args[96..128]).to::<u64>(),
    })
}

fn decode_batch_posting_report(data: &[u8]) -> Result<BatchPostingReportData, String> {
    // 5 ABI words: uint256, address, uint64, uint64, uint256
    if data.len() < 4 + 32 * 5 {
        return Err(format!(
            "batch posting report data too short: expected >= 164, got {}",
            data.len()
        ));
    }
    let args = &data[4..];
    Ok(BatchPostingReportData {
        batch_timestamp: U256::from_be_slice(&args[0..32]).to::<u64>(),
        batch_poster: Address::from_slice(&args[44..64]),
        batch_data_gas: U256::from_be_slice(&args[96..128]).to::<u64>(),
        l1_base_fee: U256::from_be_slice(&args[128..160]),
    })
}

fn decode_batch_posting_report_v2(data: &[u8]) -> Result<BatchPostingReportV2Data, String> {
    // 7 ABI words: uint256, address, uint64, uint64, uint64, uint64, uint256
    if data.len() < 4 + 32 * 7 {
        return Err(format!(
            "batch posting report v2 data too short: expected >= 228, got {}",
            data.len()
        ));
    }
    let args = &data[4..];
    Ok(BatchPostingReportV2Data {
        batch_timestamp: U256::from_be_slice(&args[0..32]).to::<u64>(),
        batch_poster: Address::from_slice(&args[44..64]),
        batch_calldata_length: U256::from_be_slice(&args[96..128]).to::<u64>(),
        batch_calldata_non_zeros: U256::from_be_slice(&args[128..160]).to::<u64>(),
        batch_extra_gas: U256::from_be_slice(&args[160..192]).to::<u64>(),
        l1_base_fee: U256::from_be_slice(&args[192..224]),
    })
}

// ---------------------------------------------------------------------------
// Dispatch
// ---------------------------------------------------------------------------

/// Context needed by the internal transaction dispatch from the block executor.
pub struct InternalTxContext {
    pub block_number: u64,
    pub current_time: u64,
    pub prev_hash: B256,
}

/// Apply an internal transaction update to ArbOS state.
///
/// Dispatches on the 4-byte method selector to handle:
/// - StartBlock: records L1 block hashes, reaps expired retryables,
///   updates L2 pricing, and checks for ArbOS upgrades.
/// - BatchPostingReport (v1 and v2): updates L1 pricing based on
///   batch poster spending.
pub fn apply_internal_tx_update<D: revm::Database, B: Burner, F>(
    data: &[u8],
    state: &mut ArbosState<D, B>,
    ctx: &InternalTxContext,
    mut transfer_fn: F,
) -> Result<(), String>
where
    F: FnMut(Address, Address, U256) -> Result<(), ()>,
{
    if data.len() < 4 {
        return Err(format!(
            "internal tx data too short ({} bytes, need at least 4)",
            data.len()
        ));
    }

    let selector: [u8; 4] = data[0..4].try_into().unwrap();

    match selector {
        INTERNAL_TX_START_BLOCK_METHOD_ID => {
            let inputs = decode_start_block(data)?;
            apply_start_block(inputs, state, ctx, &mut transfer_fn)
        }
        INTERNAL_TX_BATCH_POSTING_REPORT_METHOD_ID => {
            let inputs = decode_batch_posting_report(data)?;
            apply_batch_posting_report(inputs, state, ctx, &mut transfer_fn)
        }
        INTERNAL_TX_BATCH_POSTING_REPORT_V2_METHOD_ID => {
            let inputs = decode_batch_posting_report_v2(data)?;
            apply_batch_posting_report_v2(inputs, state, ctx, &mut transfer_fn)
        }
        _ => Err(format!(
            "unknown internal tx selector: {:02x}{:02x}{:02x}{:02x}",
            selector[0], selector[1], selector[2], selector[3]
        )),
    }
}

fn apply_start_block<D: revm::Database, B: Burner, F>(
    inputs: StartBlockData,
    state: &mut ArbosState<D, B>,
    ctx: &InternalTxContext,
    transfer_fn: &mut F,
) -> Result<(), String>
where
    F: FnMut(Address, Address, U256) -> Result<(), ()>,
{
    let arbos_version = state.arbos_version();

    let mut l1_block_number = inputs.l1_block_number;
    let mut time_passed = inputs.time_passed;

    // Before ArbOS v3, incorrectly used the L2 block number as time_passed.
    if arbos_version < arbos_version::ARBOS_VERSION_3 {
        time_passed = inputs.l2_block_number;
    }

    // Before ArbOS v8, incorrectly used L1 block number one too high.
    if arbos_version < arbos_version::ARBOS_VERSION_8 {
        l1_block_number = l1_block_number.saturating_add(1);
    }

    // Record L1 block hashes if L1 block number advanced.
    let old_l1_block_number = state
        .blockhashes
        .l1_block_number()
        .map_err(|_| "failed to read l1 block number")?;

    if l1_block_number > old_l1_block_number {
        state
            .blockhashes
            .record_new_l1_block(l1_block_number - 1, ctx.prev_hash, arbos_version)
            .map_err(|_| "failed to record L1 block")?;
    }

    // Try to reap 2 expired retryables.
    let _ = state
        .retryable_state
        .try_to_reap_one_retryable(ctx.current_time, &mut *transfer_fn);
    let _ = state
        .retryable_state
        .try_to_reap_one_retryable(ctx.current_time, &mut *transfer_fn);

    // Update L2 pricing model.
    let _ = state
        .l2_pricing_state
        .update_pricing_model(time_passed, arbos_version);

    // Check for scheduled ArbOS upgrade.
    state
        .upgrade_arbos_version_if_necessary(ctx.current_time)
        .map_err(|_| "ArbOS upgrade failed (node may be out of date)")?;

    Ok(())
}

fn apply_batch_posting_report<D: revm::Database, B: Burner, F>(
    inputs: BatchPostingReportData,
    state: &mut ArbosState<D, B>,
    ctx: &InternalTxContext,
    transfer_fn: &mut F,
) -> Result<(), String>
where
    F: FnMut(Address, Address, U256) -> Result<(), ()>,
{
    let per_batch_gas = state
        .l1_pricing_state
        .per_batch_gas_cost()
        .unwrap_or(0);

    let gas_spent = (per_batch_gas as u64).saturating_add(inputs.batch_data_gas);
    let wei_spent = inputs.l1_base_fee.saturating_mul(U256::from(gas_spent));

    if let Err(e) = state.l1_pricing_state.update_for_batch_poster_spending(
        inputs.batch_timestamp,
        ctx.current_time,
        inputs.batch_poster,
        wei_spent,
        inputs.l1_base_fee,
        &mut *transfer_fn,
    ) {
        tracing::warn!(error = ?e, "L1 pricing update failed for batch posting report");
    }

    Ok(())
}

fn apply_batch_posting_report_v2<D: revm::Database, B: Burner, F>(
    inputs: BatchPostingReportV2Data,
    state: &mut ArbosState<D, B>,
    ctx: &InternalTxContext,
    transfer_fn: &mut F,
) -> Result<(), String>
where
    F: FnMut(Address, Address, U256) -> Result<(), ()>,
{
    let arbos_version = state.arbos_version();

    // Compute gas from calldata stats (legacy cost model).
    let mut gas_spent = legacy_cost_for_stats(&BatchDataStats {
        length: inputs.batch_calldata_length,
        non_zeros: inputs.batch_calldata_non_zeros,
    });

    gas_spent = gas_spent.saturating_add(inputs.batch_extra_gas);

    // Add per-batch gas overhead.
    let per_batch_gas = state
        .l1_pricing_state
        .per_batch_gas_cost()
        .unwrap_or(0);

    gas_spent = gas_spent.saturating_add(per_batch_gas.max(0) as u64);

    // Floor gas computation (ArbOS v50+).
    if arbos_version >= arbos_version::ARBOS_VERSION_50 {
        let gas_floor_per_token = state
            .l1_pricing_state
            .parent_gas_floor_per_token()
            .unwrap_or(0);

        let total_tokens = inputs
            .batch_calldata_length
            .saturating_add(inputs.batch_calldata_non_zeros.saturating_mul(3))
            .saturating_add(FLOOR_GAS_ADDITIONAL_TOKENS);

        let floor_gas_spent = gas_floor_per_token
            .saturating_mul(total_tokens)
            .saturating_add(TX_GAS);

        if floor_gas_spent > gas_spent {
            gas_spent = floor_gas_spent;
        }
    }

    let wei_spent = inputs.l1_base_fee.saturating_mul(U256::from(gas_spent));

    if let Err(e) = state.l1_pricing_state.update_for_batch_poster_spending(
        inputs.batch_timestamp,
        ctx.current_time,
        inputs.batch_poster,
        wei_spent,
        inputs.l1_base_fee,
        &mut *transfer_fn,
    ) {
        tracing::warn!(error = ?e, "L1 pricing update failed for batch posting report v2");
    }

    Ok(())
}
