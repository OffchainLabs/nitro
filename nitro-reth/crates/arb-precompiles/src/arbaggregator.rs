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
    let gas_limit = input.gas;
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    let result = match selector {
        GET_PREFERRED_AGGREGATOR => {
            // Deprecated view method: always returns (BatchPosterAddress, true).
            // Go charges OpenArbosState (800) + argsCost (3) + resultCost (6) = 809.
            let mut out = Vec::with_capacity(96);
            out.extend_from_slice(&U256::from(0x40u64).to_be_bytes::<32>());
            out.extend_from_slice(&U256::from(1u64).to_be_bytes::<32>());
            let mut addr_word = [0u8; 32];
            addr_word[12..32].copy_from_slice(BATCH_POSTER_ADDRESS.as_slice());
            out.extend_from_slice(&addr_word);
            Ok(PrecompileOutput::new(
                (SLOAD_GAS + 9).min(gas_limit),
                out.into(),
            ))
        }
        GET_DEFAULT_AGGREGATOR => {
            // Deprecated view method: always returns BatchPosterAddress.
            // Go charges OpenArbosState (800) + resultCost (3) = 803.
            let mut out = [0u8; 32];
            out[12..32].copy_from_slice(BATCH_POSTER_ADDRESS.as_slice());
            Ok(PrecompileOutput::new(
                (SLOAD_GAS + COPY_GAS).min(gas_limit),
                out.to_vec().into(),
            ))
        }
        GET_TX_BASE_FEE => {
            // Deprecated view method: always returns 0.
            // Go charges OpenArbosState (800) + argsCost (3) + resultCost (3) = 806.
            Ok(PrecompileOutput::new(
                (SLOAD_GAS + 6).min(gas_limit),
                U256::ZERO.to_be_bytes::<32>().to_vec().into(),
            ))
        }
        SET_TX_BASE_FEE => {
            // Deprecated write method: no-op.
            // Go charges OpenArbosState (800) + argsCost (6) = 806.
            Ok(PrecompileOutput::new(
                (SLOAD_GAS + 6).min(gas_limit),
                vec![].into(),
            ))
        }
        GET_FEE_COLLECTOR => handle_get_fee_collector(&mut input),
        SET_FEE_COLLECTOR => handle_set_fee_collector(&mut input),
        GET_BATCH_POSTERS => handle_get_batch_posters(&mut input),
        ADD_BATCH_POSTER => handle_add_batch_poster(&mut input),
        _ => Err(PrecompileError::other(
            "unknown ArbAggregator selector",
        )),
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

/// Derive the batch poster table sub-storage key.
fn batch_poster_table_key() -> B256 {
    let l1_pricing_key = derive_subspace_key(ROOT_STORAGE_KEY, L1_PRICING_SUBSPACE);
    derive_subspace_key(l1_pricing_key.as_slice(), BATCH_POSTER_TABLE_KEY)
}

/// Derive the posterAddrs (AddressSet) sub-storage key.
fn poster_addrs_key() -> B256 {
    let bpt_key = batch_poster_table_key();
    derive_subspace_key(bpt_key.as_slice(), POSTER_ADDRS_KEY)
}

/// Derive the poster info sub-storage key for a specific batch poster.
fn poster_info_key(poster: Address) -> B256 {
    let bpt_key = batch_poster_table_key();
    let poster_info = derive_subspace_key(bpt_key.as_slice(), POSTER_INFO_KEY);
    derive_subspace_key(poster_info.as_slice(), poster.as_slice())
}

/// Check if caller is a chain owner via the address set membership check.
fn is_chain_owner(input: &mut PrecompileInput<'_>, addr: Address) -> Result<bool, PrecompileError> {
    let owner_key = derive_subspace_key(ROOT_STORAGE_KEY, CHAIN_OWNER_SUBSPACE);
    let by_address_key = derive_subspace_key(owner_key.as_slice(), &[0]);
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
        (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
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

    let gas_used = 3 * SLOAD_GAS + SSTORE_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(gas_used.min(gas_limit), vec![].into()))
}

/// GetBatchPosters returns all batch poster addresses from the AddressSet.
fn handle_get_batch_posters(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let addrs_key = poster_addrs_key();
    // AddressSet size is at offset 0.
    let size_slot = map_slot(addrs_key.as_slice(), 0);
    let size = sload_field(input, size_slot)?;
    let count: u64 = size
        .try_into()
        .map_err(|_| PrecompileError::other("invalid address set size"))?;

    const MAX_MEMBERS: u64 = 1024;
    let count = count.min(MAX_MEMBERS);

    // Read each member address from positions 1..=count.
    let mut addresses = Vec::with_capacity(count as usize);
    for i in 1..=count {
        let member_slot = map_slot(addrs_key.as_slice(), i);
        let val = sload_field(input, member_slot)?;
        addresses.push(val);
    }

    // ABI-encode as dynamic address array: offset, length, then elements.
    let mut out = Vec::with_capacity(64 + 32 * addresses.len());
    out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(count).to_be_bytes::<32>());
    for addr_val in &addresses {
        out.extend_from_slice(&addr_val.to_be_bytes::<32>());
    }

    let gas_used = (2 + count) * SLOAD_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(gas_used.min(gas_limit), out.into()))
}

/// AddBatchPoster adds a new batch poster. Caller must be a chain owner.
fn handle_add_batch_poster(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 36 {
        return Err(PrecompileError::other("input too short"));
    }

    let gas_limit = input.gas;
    let new_poster = Address::from_slice(&data[16..36]);
    let caller = input.caller;
    load_arbos(input)?;

    // Verify caller is a chain owner.
    if !is_chain_owner(input, caller)? {
        return Err(PrecompileError::other("must be called by chain owner"));
    }

    let addrs_key = poster_addrs_key();

    // Check if already a batch poster via byAddress sub-storage.
    let by_address_key = derive_subspace_key(addrs_key.as_slice(), &[0]);
    let addr_hash = B256::left_padding_from(new_poster.as_slice());
    let member_slot = map_slot_b256(by_address_key.as_slice(), &addr_hash);
    let existing = sload_field(input, member_slot)?;

    if existing != U256::ZERO {
        // Already a batch poster — no-op.
        return Ok(PrecompileOutput::new(
            (2 * SLOAD_GAS + COPY_GAS).min(gas_limit),
            vec![].into(),
        ));
    }

    // Read current size and increment.
    let size_slot = map_slot(addrs_key.as_slice(), 0);
    let size = sload_field(input, size_slot)?;
    let size_u64: u64 = size
        .try_into()
        .map_err(|_| PrecompileError::other("invalid address set size"))?;
    let new_size = size_u64 + 1;

    // Store the new poster at position (1 + size) in the backing storage.
    let new_pos_slot = map_slot(addrs_key.as_slice(), new_size);
    let addr_as_u256 = U256::from_be_slice(new_poster.as_slice());
    sstore_field(input, new_pos_slot, addr_as_u256)?;

    // Store in byAddress mapping: byAddress[addr_hash] = 1-based position.
    let slot_value = U256::from(new_size);
    sstore_field(input, member_slot, slot_value)?;

    // Increment size.
    sstore_field(input, size_slot, U256::from(new_size))?;

    // Initialize poster info: set payTo = newPoster (the poster pays itself initially).
    let info_key = poster_info_key(new_poster);
    let pay_to_slot = map_slot(info_key.as_slice(), PAY_TO_OFFSET);
    sstore_field(input, pay_to_slot, addr_as_u256)?;

    let gas_used = 4 * SLOAD_GAS + 4 * SSTORE_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(gas_used.min(gas_limit), vec![].into()))
}
