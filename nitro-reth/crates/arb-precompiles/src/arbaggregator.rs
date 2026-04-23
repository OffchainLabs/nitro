use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, B256, U256};
use alloy_sol_types::SolInterface;
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

use crate::{
    interfaces::IArbAggregator,
    storage_slot::{
        derive_subspace_key, map_slot, map_slot_b256, ARBOS_STATE_ADDRESS, CHAIN_OWNER_SUBSPACE,
        L1_PRICING_SUBSPACE, ROOT_STORAGE_KEY,
    },
};

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
    crate::init_precompile_gas(input.data.len());

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
            Ok(PrecompileOutput::new(
                (SLOAD_GAS + 6).min(gas_limit),
                out.into(),
            ))
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
        Calls::getFeeCollector(c) => handle_get_fee_collector(&mut input, c.batchPoster),
        Calls::setFeeCollector(c) => {
            handle_set_fee_collector(&mut input, c.batchPoster, c.newFeeCollector)
        }
        Calls::getBatchPosters(_) => handle_get_batch_posters(&mut input),
        Calls::addBatchPoster(c) => handle_add_batch_poster(&mut input, c.newBatchPoster),
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

fn sstore_field(
    input: &mut PrecompileInput<'_>,
    slot: U256,
    value: U256,
) -> Result<(), PrecompileError> {
    input
        .internals_mut()
        .sstore(ARBOS_STATE_ADDRESS, slot, value)
        .map_err(|_| PrecompileError::other("sstore failed"))?;
    crate::charge_precompile_gas(SSTORE_GAS);
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

fn handle_get_fee_collector(input: &mut PrecompileInput<'_>, poster: Address) -> PrecompileResult {
    let gas_limit = input.gas;
    load_arbos(input)?;

    let info_key = poster_info_key(poster);
    let pay_to_slot = map_slot(info_key.as_slice(), PAY_TO_OFFSET);
    let pay_to = sload_field(input, pay_to_slot)?;

    // OAS(1) + OpenPoster IsMember(1) + payTo.Get(1) + argsCost(3) + resultCost(3).
    Ok(PrecompileOutput::new(
        (3 * SLOAD_GAS + 2 * COPY_GAS).min(gas_limit),
        pay_to.to_be_bytes::<32>().to_vec().into(),
    ))
}

/// Caller must be the batch poster, its current fee collector, or a chain owner.
fn handle_set_fee_collector(
    input: &mut PrecompileInput<'_>,
    poster: Address,
    new_collector: Address,
) -> PrecompileResult {
    let gas_limit = input.gas;
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

    // OAS(1) + OpenPoster IsMember(1) + PayTo.Get(1) + SetPayTo(1 SSTORE) + argsCost(6).
    // Owner check adds IsMember(1 SLOAD) only when caller is neither poster nor collector.
    let mut gas_used = 3 * SLOAD_GAS + SSTORE_GAS + 2 * COPY_GAS;
    if caller != poster && caller != old_collector {
        gas_used += SLOAD_GAS;
    }
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        vec![].into(),
    ))
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

    // resultCost = (2 + N) words for dynamic array encoding.
    let gas_used = (2 + count) * SLOAD_GAS + (2 + count) * COPY_GAS;
    Ok(PrecompileOutput::new(gas_used.min(gas_limit), out.into()))
}

/// Caller must be a chain owner.
fn handle_add_batch_poster(
    input: &mut PrecompileInput<'_>,
    new_poster: Address,
) -> PrecompileResult {
    let gas_limit = input.gas;
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

    // IsMember(caller)(1) + ContainsPoster IsMember(1) + AddPoster[IsMember(1) +
    // fundsDue.SetChecked(0)(5000) + payTo.Set(20000) + Add(IsMember(1) + size.Get(1) +
    // byAddress.Set(20000) + backingStorage.Set(20000) + size.Increment Get(1)+Set(20000))]
    // + argsCost(3).
    let gas_used = 6 * SLOAD_GAS + SSTORE_ZERO_GAS + 4 * SSTORE_GAS + COPY_GAS;
    Ok(PrecompileOutput::new(
        gas_used.min(gas_limit),
        vec![].into(),
    ))
}
