use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, B256, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::storage_slot::{
    derive_subspace_key, map_slot, map_slot_b256, ARBOS_STATE_ADDRESS, CHAIN_OWNER_SUBSPACE,
    L1_PRICING_SUBSPACE, ROOT_STORAGE_KEY,
};

/// ArbAggregator precompile address (0x6d).
pub const ARBAGGREGATOR_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x6d,
]);

/// Default batch poster address (the sequencer).
const BATCH_POSTER_ADDRESS: Address = Address::new([
    0xa4, 0xb0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x65, 0x71, 0x75,
    0x65, 0x6e, 0x63, 0x65, 0x72,
]);

// Function selectors.
const GET_PREFERRED_AGGREGATOR: [u8; 4] = [0x52, 0xf1, 0x07, 0x40];
const GET_DEFAULT_AGGREGATOR: [u8; 4] = [0x87, 0x58, 0x83, 0xf2];
const GET_BATCH_POSTERS: [u8; 4] = [0xe1, 0x05, 0x73, 0xa3];
const ADD_BATCH_POSTER: [u8; 4] = [0xdf, 0x41, 0xe1, 0xe2];
const GET_FEE_COLLECTOR: [u8; 4] = [0x9c, 0x2c, 0x5b, 0xb5];
const SET_FEE_COLLECTOR: [u8; 4] = [0x29, 0x14, 0x97, 0x99];
const GET_TX_BASE_FEE: [u8; 4] = [0x04, 0x97, 0x64, 0xaf];
const SET_TX_BASE_FEE: [u8; 4] = [0x5b, 0xe6, 0x88, 0x8b];

const SLOAD_GAS: u64 = 800;
const SSTORE_GAS: u64 = 20_000;
const COPY_GAS: u64 = 3;

// Batch poster table storage layout constants.
const BATCH_POSTER_TABLE_KEY: &[u8] = &[0];
const POSTER_ADDRS_KEY: &[u8] = &[0];
const POSTER_INFO_KEY: &[u8] = &[1];
const PAY_TO_OFFSET: u64 = 1;

pub fn create_arbaggregator_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbaggregator"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];
    let gas_limit = input.gas;

    match selector {
        GET_PREFERRED_AGGREGATOR => {
            // Deprecated: always returns (BatchPosterAddress, true).
            let mut out = Vec::with_capacity(96);
            // ABI offset for the tuple
            out.extend_from_slice(&U256::from(0x40u64).to_be_bytes::<32>());
            // isDefault = true
            out.extend_from_slice(&U256::from(1u64).to_be_bytes::<32>());
            // address (left-padded)
            let mut addr_word = [0u8; 32];
            addr_word[12..32].copy_from_slice(BATCH_POSTER_ADDRESS.as_slice());
            out.extend_from_slice(&addr_word);
            Ok(PrecompileOutput::new(COPY_GAS.min(gas_limit), out.into()))
        }
        GET_DEFAULT_AGGREGATOR => {
            // Deprecated: always returns BatchPosterAddress.
            let mut out = [0u8; 32];
            out[12..32].copy_from_slice(BATCH_POSTER_ADDRESS.as_slice());
            Ok(PrecompileOutput::new(
                COPY_GAS.min(gas_limit),
                out.to_vec().into(),
            ))
        }
        GET_TX_BASE_FEE => {
            // Deprecated: always returns 0.
            Ok(PrecompileOutput::new(
                COPY_GAS.min(gas_limit),
                U256::ZERO.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        SET_TX_BASE_FEE => {
            // Deprecated: no-op.
            Ok(PrecompileOutput::new(COPY_GAS.min(gas_limit), vec![].into()))
        }
        GET_FEE_COLLECTOR => handle_get_fee_collector(&mut input),
        SET_FEE_COLLECTOR => handle_set_fee_collector(&mut input),
        GET_BATCH_POSTERS | ADD_BATCH_POSTER => {
            // These methods require address set iteration/modification.
            Err(PrecompileError::other(
                "batch poster list operations not yet supported",
            ))
        }
        _ => Err(PrecompileError::other(
            "unknown ArbAggregator selector",
        )),
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

fn sstore_field(
    input: &mut PrecompileInput<'_>,
    slot: U256,
    value: U256,
) -> Result<(), PrecompileError> {
    input
        .internals_mut()
        .sstore(ARBOS_STATE_ADDRESS, slot, value)
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    Ok(())
}

/// Derive the poster info sub-storage key for a specific batch poster.
/// Path: root → l1pricing → batchPosterTable([0]) → posterInfo([1]) → poster_address
fn poster_info_key(poster: Address) -> B256 {
    let l1_pricing_key = derive_subspace_key(ROOT_STORAGE_KEY, L1_PRICING_SUBSPACE);
    let bpt_key = derive_subspace_key(l1_pricing_key.as_slice(), BATCH_POSTER_TABLE_KEY);
    let poster_info = derive_subspace_key(bpt_key.as_slice(), POSTER_INFO_KEY);
    derive_subspace_key(poster_info.as_slice(), poster.as_slice())
}

/// Check if caller is a chain owner via the address set membership check.
fn is_chain_owner(input: &mut PrecompileInput<'_>, addr: Address) -> Result<bool, PrecompileError> {
    let owner_key = derive_subspace_key(ROOT_STORAGE_KEY, CHAIN_OWNER_SUBSPACE);
    let by_address_key = derive_subspace_key(owner_key.as_slice(), &[]);
    let addr_b256 = B256::left_padding_from(addr.as_slice());
    let slot = map_slot_b256(by_address_key.as_slice(), &addr_b256);
    let val = sload_field(input, slot)?;
    Ok(val != U256::ZERO)
}

/// GetFeeCollector reads the payTo address for a batch poster.
fn handle_get_fee_collector(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let poster = Address::from_slice(&data[16..36]);
    load_arbos(input)?;

    let info_key = poster_info_key(poster);
    let pay_to_slot = map_slot(info_key.as_slice(), PAY_TO_OFFSET);
    let pay_to = sload_field(input, pay_to_slot)?;

    Ok(PrecompileOutput::new(
        (SLOAD_GAS + COPY_GAS).min(gas_limit),
        pay_to.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// SetFeeCollector sets the payTo address for a batch poster.
/// Caller must be the batch poster, its current fee collector, or a chain owner.
fn handle_set_fee_collector(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 68 {
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let poster = Address::from_slice(&data[16..36]);
    let new_collector = Address::from_slice(&data[48..68]);
    let caller = input.caller;

    load_arbos(input)?;

    // Read the current fee collector.
    let info_key = poster_info_key(poster);
    let pay_to_slot = map_slot(info_key.as_slice(), PAY_TO_OFFSET);
    let old_collector_u256 = sload_field(input, pay_to_slot)?;
    let old_collector_bytes = old_collector_u256.to_be_bytes::<32>();
    let old_collector = Address::from_slice(&old_collector_bytes[12..32]);

    // Verify authorization: caller must be poster, old fee collector, or chain owner.
    if caller != poster && caller != old_collector {
        let is_owner = is_chain_owner(input, caller)?;
        if !is_owner {
            return Err(PrecompileError::other(
                "only a batch poster, its fee collector, or chain owner may change the fee collector",
            ));
        }
    }

    // Write the new fee collector.
    let new_val = U256::from_be_slice(new_collector.as_slice());
    sstore_field(input, pay_to_slot, new_val)?;

    let gas_used = 2 * SLOAD_GAS + SSTORE_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(gas_used.min(gas_limit), vec![].into()))
}
