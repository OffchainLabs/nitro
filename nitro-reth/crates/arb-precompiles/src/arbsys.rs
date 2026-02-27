use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{keccak256, Address, Log, B256, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};
use std::cell::RefCell;
use std::collections::HashMap;
use std::sync::Mutex;

use crate::storage_slot::{
    derive_subspace_key, map_slot, root_slot, ARBOS_STATE_ADDRESS, ROOT_STORAGE_KEY,
    SEND_MERKLE_SUBSPACE,
};

/// ArbSys precompile address (0x64).
pub const ARBSYS_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x64,
]);

// Function selectors.
const WITHDRAW_ETH: [u8; 4] = [0x25, 0xe1, 0x60, 0x63];
const SEND_TX_TO_L1: [u8; 4] = [0x92, 0x8c, 0x16, 0x9a];
const ARB_BLOCK_NUMBER: [u8; 4] = [0xa3, 0xb1, 0xb3, 0x1d];
const ARB_BLOCK_HASH: [u8; 4] = [0x2b, 0x40, 0x7a, 0x49];
const ARB_CHAIN_ID: [u8; 4] = [0xd1, 0x27, 0x00, 0x44];
const ARB_OS_VERSION: [u8; 4] = [0x05, 0x1e, 0xd6, 0xa3];
const GET_STORAGE_GAS_AVAILABLE: [u8; 4] = [0xf3, 0x38, 0x14, 0x0e];
const IS_TOP_LEVEL_CALL: [u8; 4] = [0x08, 0xbd, 0x62, 0x4c];
const MAP_L1_SENDER: [u8; 4] = [0xb6, 0xc6, 0xb7, 0x05]; // mapL1SenderContractAddressToL2Alias
const WAS_ALIASED: [u8; 4] = [0x69, 0x52, 0x75, 0xf7]; // wasMyCallersAddressAliased
const CALLER_WITHOUT_ALIAS: [u8; 4] = [0xd7, 0x4c, 0x83, 0xa3]; // myCallersAddressWithoutAliasing
const SEND_MERKLE_TREE_STATE: [u8; 4] = [0xae, 0x6e, 0x33, 0x08];

// L1 alias offset: 0x1111000000000000000000000000000000001111
const L1_ALIAS_OFFSET: Address = Address::new([
    0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x11, 0x11,
]);

// MerkleAccumulator: size at offset 0, partials at offset (2 + level).

// Gas costs.
const SLOAD_GAS: u64 = 800;
const COPY_GAS: u64 = 3;

// Event topics.
fn l2_to_l1_tx_topic() -> B256 {
    keccak256(b"L2ToL1Tx(address,address,uint256,uint256,uint256,uint256,uint256,bytes)")
}

fn send_merkle_update_topic() -> B256 {
    keccak256(b"SendMerkleUpdate(uint256,bytes32,uint256)")
}

/// State changes from an ArbSys call for post-execution application.
#[derive(Debug, Clone, Default)]
pub struct ArbSysMerkleState {
    pub new_size: u64,
    pub partials: Vec<(u64, B256)>,
    pub send_hash: B256,
    pub leaf_num: u64,
    pub value_to_burn: U256,
    pub block_number: u64,
}

thread_local! {
    static ARBSYS_STATE: RefCell<Option<ArbSysMerkleState>> = RefCell::new(None);
}

static L1_BLOCK_CACHE: Mutex<Option<HashMap<u64, u64>>> = Mutex::new(None);

/// Store ArbSys state changes for post-execution application.
pub fn store_arbsys_state(state: ArbSysMerkleState) {
    ARBSYS_STATE.with(|cell| *cell.borrow_mut() = Some(state));
}

/// Take the stored ArbSys state (clears it).
pub fn take_arbsys_state() -> Option<ArbSysMerkleState> {
    ARBSYS_STATE.with(|cell| cell.borrow_mut().take())
}

/// Set the cached L1 block number for a given L2 block.
pub fn set_cached_l1_block_number(l2_block: u64, l1_block: u64) {
    let mut cache = L1_BLOCK_CACHE.lock().unwrap();
    let map = cache.get_or_insert_with(HashMap::new);
    map.insert(l2_block, l1_block);
    if l2_block > 100 {
        map.retain(|&k, _| k >= l2_block - 100);
    }
}

/// Get the cached L1 block number for a given L2 block.
pub fn get_cached_l1_block_number(l2_block: u64) -> Option<u64> {
    let cache = L1_BLOCK_CACHE.lock().unwrap();
    cache.as_ref().and_then(|m| m.get(&l2_block).copied())
}

pub fn create_arbsys_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbsys"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];

    match selector {
        ARB_BLOCK_NUMBER => handle_arb_block_number(&mut input),
        ARB_BLOCK_HASH => handle_arb_block_hash(&mut input),
        ARB_CHAIN_ID => handle_arb_chain_id(&mut input),
        ARB_OS_VERSION => handle_arbos_version(&mut input),
        IS_TOP_LEVEL_CALL => handle_is_top_level_call(&mut input),
        WAS_ALIASED => handle_was_aliased(&mut input),
        CALLER_WITHOUT_ALIAS => handle_caller_without_alias(&mut input),
        MAP_L1_SENDER => handle_map_l1_sender(&mut input),
        GET_STORAGE_GAS_AVAILABLE => handle_get_storage_gas(&mut input),
        WITHDRAW_ETH => handle_withdraw_eth(&mut input),
        SEND_TX_TO_L1 => handle_send_tx_to_l1(&mut input),
        SEND_MERKLE_TREE_STATE => handle_send_merkle_tree_state(&mut input),
        _ => Err(PrecompileError::other("unknown ArbSys selector")),
    }
}

// ── view functions ───────────────────────────────────────────────────

fn handle_arb_block_number(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let block_num = input.internals().block_number();
    let gas_cost = COPY_GAS.min(input.gas);
    Ok(PrecompileOutput::new(
        gas_cost,
        block_num.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_arb_block_hash(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    // ArbBlockHash returns the block hash for a given block number.
    // The actual block hash lookup requires state access; return zero for now.
    let gas_cost = COPY_GAS.min(input.gas);
    Ok(PrecompileOutput::new(gas_cost, vec![0u8; 32].into()))
}

fn handle_arb_chain_id(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let chain_id = input.internals().chain_id();
    let gas_cost = COPY_GAS.min(input.gas);
    Ok(PrecompileOutput::new(
        gas_cost,
        U256::from(chain_id).to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_arbos_version(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    let internals = input.internals_mut();

    internals
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;

    // ArbOS version is at root offset 0.
    let version = internals
        .sload(ARBOS_STATE_ADDRESS, root_slot(0))
        .map_err(|_| PrecompileError::other("sload failed"))?;

    Ok(PrecompileOutput::new(
        (SLOAD_GAS + COPY_GAS).min(gas_limit),
        version.data.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_is_top_level_call(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    // A call is top-level if the caller is the tx origin.
    let is_top = input.caller == input.internals().tx_origin();
    let val = if is_top { U256::from(1) } else { U256::ZERO };
    let gas_cost = COPY_GAS.min(input.gas);
    Ok(PrecompileOutput::new(
        gas_cost,
        val.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_was_aliased(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    // Check if the caller's address was aliased (L1 → L2 address translation).
    // This is determined by the tx processor, not accessible here directly.
    // Return false (0) as default.
    let gas_cost = COPY_GAS.min(input.gas);
    Ok(PrecompileOutput::new(gas_cost, vec![0u8; 32].into()))
}

fn handle_caller_without_alias(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    // Return the caller with the alias undone.
    let caller = input.caller;
    let unaliased = undo_l1_alias(caller);
    let gas_cost = COPY_GAS.min(input.gas);
    let mut out = [0u8; 32];
    out[12..32].copy_from_slice(unaliased.as_slice());
    Ok(PrecompileOutput::new(gas_cost, out.to_vec().into()))
}

fn handle_map_l1_sender(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 + 64 {
        return Err(PrecompileError::other("input too short"));
    }
    // mapL1SenderContractAddressToL2Alias(address l1_addr, address _unused)
    let l1_addr = Address::from_slice(&data[16..36]);
    let aliased = apply_l1_alias(l1_addr);
    let gas_cost = COPY_GAS.min(input.gas);
    let mut out = [0u8; 32];
    out[12..32].copy_from_slice(aliased.as_slice());
    Ok(PrecompileOutput::new(gas_cost, out.to_vec().into()))
}

fn handle_get_storage_gas(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    // Returns remaining gas available for storage operations.
    let gas_cost = COPY_GAS.min(input.gas);
    Ok(PrecompileOutput::new(
        gas_cost,
        U256::from(input.gas).to_be_bytes::<32>().to_vec().into(),
    ))
}

// ── L2→L1 messaging ─────────────────────────────────────────────────

fn handle_withdraw_eth(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    if input.is_static {
        return Err(PrecompileError::other("cannot call withdrawEth in static context"));
    }

    let data = input.data;
    if data.len() < 4 + 32 {
        return Err(PrecompileError::other("input too short"));
    }

    let destination = Address::from_slice(&data[16..36]);
    // WithdrawEth calls SendTxToL1 with the destination and empty calldata.
    do_send_tx_to_l1(input, destination, &[])
}

fn handle_send_tx_to_l1(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    if input.is_static {
        return Err(PrecompileError::other("cannot call sendTxToL1 in static context"));
    }

    let data = input.data;
    if data.len() < 4 + 64 {
        return Err(PrecompileError::other("input too short"));
    }

    // sendTxToL1(address destination, bytes calldata)
    let destination = Address::from_slice(&data[16..36]);

    // Decode the dynamic bytes parameter.
    let offset =
        U256::from_be_slice(&data[36..68]).try_into().unwrap_or(0usize);
    let abs_offset = 4 + offset;
    if abs_offset + 32 > data.len() {
        return Err(PrecompileError::other("calldata offset out of bounds"));
    }
    let length: usize =
        U256::from_be_slice(&data[abs_offset..abs_offset + 32])
            .try_into()
            .unwrap_or(0);
    let calldata_start = abs_offset + 32;
    let calldata_end = calldata_start + length;
    if calldata_end > data.len() {
        return Err(PrecompileError::other("calldata length out of bounds"));
    }
    let calldata = &data[calldata_start..calldata_end];

    do_send_tx_to_l1(input, destination, calldata)
}

fn do_send_tx_to_l1(
    input: &mut PrecompileInput<'_>,
    destination: Address,
    calldata: &[u8],
) -> PrecompileResult {
    let gas_limit = input.gas;
    let caller = input.caller;
    let value = input.value;
    let block_number = input.internals().block_number();

    let internals = input.internals_mut();

    // Load the ArbOS state account.
    internals
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;

    // Read current Merkle accumulator size.
    let merkle_key = derive_subspace_key(ROOT_STORAGE_KEY, SEND_MERKLE_SUBSPACE);
    let size_slot = map_slot(merkle_key.as_slice(), 0);
    let current_size = internals
        .sload(ARBOS_STATE_ADDRESS, size_slot)
        .map_err(|_| PrecompileError::other("sload failed"))?
        .data;
    let leaf_num: u64 = current_size.try_into().unwrap_or(0);

    // Compute the send hash.
    let send_hash = compute_send_hash(
        caller,
        destination,
        leaf_num,
        value,
        calldata,
    );

    // Update Merkle accumulator: insert leaf and update partials.
    let new_size = leaf_num + 1;
    let partials = update_merkle_accumulator(
        internals,
        &merkle_key,
        send_hash,
        leaf_num,
    )?;

    // Write new size.
    internals
        .sstore(ARBOS_STATE_ADDRESS, size_slot, U256::from(new_size))
        .map_err(|_| PrecompileError::other("sstore failed"))?;

    // Emit SendMerkleUpdate event.
    let merkle_root = compute_merkle_root(&partials, new_size);
    let update_topic = send_merkle_update_topic();
    let mut update_data = Vec::with_capacity(96);
    update_data.extend_from_slice(&U256::from(0u64).to_be_bytes::<32>()); // reserved
    update_data.extend_from_slice(&merkle_root.0);
    update_data.extend_from_slice(&U256::from(new_size).to_be_bytes::<32>());
    internals.log(Log::new_unchecked(
        ARBSYS_ADDRESS,
        vec![update_topic],
        update_data.into(),
    ));

    // Emit L2ToL1Tx event.
    let l2l1_topic = l2_to_l1_tx_topic();
    let mut event_data = Vec::with_capacity(256);
    // caller (indexed topic)
    let caller_topic = B256::left_padding_from(caller.as_slice());
    // destination (indexed topic)
    let dest_topic = B256::left_padding_from(destination.as_slice());
    // hash, position, arbBlockNum, ethBlockNum, timestamp, callvalue, data
    event_data.extend_from_slice(&send_hash.0);
    event_data.extend_from_slice(&U256::from(leaf_num).to_be_bytes::<32>());
    event_data.extend_from_slice(&block_number.to_be_bytes::<32>());
    event_data.extend_from_slice(&U256::ZERO.to_be_bytes::<32>()); // ethBlockNum
    event_data.extend_from_slice(&U256::ZERO.to_be_bytes::<32>()); // timestamp
    event_data.extend_from_slice(&value.to_be_bytes::<32>());
    // bytes offset + length + data
    event_data.extend_from_slice(&U256::from(7 * 32).to_be_bytes::<32>()); // offset to data
    event_data.extend_from_slice(&U256::from(calldata.len()).to_be_bytes::<32>());
    event_data.extend_from_slice(calldata);
    // Pad to 32-byte boundary.
    let pad = (32 - calldata.len() % 32) % 32;
    event_data.extend(std::iter::repeat_n(0u8, pad));

    internals.log(Log::new_unchecked(
        ARBSYS_ADDRESS,
        vec![l2l1_topic, caller_topic, dest_topic],
        event_data.into(),
    ));

    // Store state for post-execution (value burn, etc.)
    store_arbsys_state(ArbSysMerkleState {
        new_size,
        partials: partials
            .iter()
            .enumerate()
            .map(|(i, h)| (i as u64, *h))
            .collect(),
        send_hash,
        leaf_num,
        value_to_burn: value,
        block_number: block_number.try_into().unwrap_or(0),
    });

    // Return the leaf number.
    let gas_cost = (4 * SLOAD_GAS + COPY_GAS).min(gas_limit);
    Ok(PrecompileOutput::new(
        gas_cost,
        U256::from(leaf_num).to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_send_merkle_tree_state(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let gas_limit = input.gas;
    let internals = input.internals_mut();

    internals
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;

    let merkle_key = derive_subspace_key(ROOT_STORAGE_KEY, SEND_MERKLE_SUBSPACE);
    let size_slot = map_slot(merkle_key.as_slice(), 0);
    let size = internals
        .sload(ARBOS_STATE_ADDRESS, size_slot)
        .map_err(|_| PrecompileError::other("sload failed"))?
        .data;

    let size_u64: u64 = size.try_into().unwrap_or(0);

    // Read partials — stored at offset (2 + level) in the accumulator storage.
    let mut partials = Vec::new();
    for i in 0..64u64 {
        if (size_u64 >> i) == 0 {
            break;
        }
        let slot = map_slot(merkle_key.as_slice(), 2 + i);
        let val = internals
            .sload(ARBOS_STATE_ADDRESS, slot)
            .map_err(|_| PrecompileError::other("sload failed"))?
            .data;
        partials.push(val);
    }

    let root = compute_merkle_root_from_u256(&partials, size_u64);

    // Return (size, root, partials...)
    // ABI: uint256 size, bytes32 root, bytes32[] partials
    let num_partials = partials.len();
    let mut out = Vec::with_capacity(96 + num_partials * 32);
    out.extend_from_slice(&size.to_be_bytes::<32>());
    out.extend_from_slice(&root.0);
    // Dynamic array: offset, length, elements
    out.extend_from_slice(&U256::from(96u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(num_partials).to_be_bytes::<32>());
    for p in &partials {
        out.extend_from_slice(&p.to_be_bytes::<32>());
    }

    let gas_cost = ((1 + num_partials as u64) * SLOAD_GAS + COPY_GAS).min(gas_limit);
    Ok(PrecompileOutput::new(gas_cost, out.into()))
}

// ── Merkle helpers ───────────────────────────────────────────────────

fn compute_send_hash(
    sender: Address,
    dest: Address,
    position: u64,
    value: U256,
    data: &[u8],
) -> B256 {
    let mut preimage = Vec::with_capacity(160 + data.len());
    preimage.extend_from_slice(&[0u8; 12]);
    preimage.extend_from_slice(sender.as_slice());
    preimage.extend_from_slice(&[0u8; 12]);
    preimage.extend_from_slice(dest.as_slice());
    preimage.extend_from_slice(&U256::from(position).to_be_bytes::<32>());
    preimage.extend_from_slice(&value.to_be_bytes::<32>());
    preimage.extend_from_slice(data);
    keccak256(&preimage)
}

fn update_merkle_accumulator(
    internals: &mut alloy_evm::EvmInternals<'_>,
    merkle_key: &B256,
    leaf_hash: B256,
    index: u64,
) -> Result<Vec<B256>, PrecompileError> {
    let mut hash = U256::from_be_bytes(leaf_hash.0);
    let mut level = 0u64;
    let mut idx = index;
    let mut updated_partials = Vec::new();

    while idx & 1 == 1 {
        // Read the sibling partial at this level (offset = 2 + level).
        let slot = map_slot(merkle_key.as_slice(), 2 + level);
        let sibling = internals
            .sload(ARBOS_STATE_ADDRESS, slot)
            .map_err(|_| PrecompileError::other("sload failed"))?
            .data;

        // hash = keccak256(sibling || hash)
        let mut preimage = [0u8; 64];
        preimage[..32].copy_from_slice(&sibling.to_be_bytes::<32>());
        preimage[32..].copy_from_slice(&hash.to_be_bytes::<32>());
        hash = U256::from_be_bytes(keccak256(preimage).0);

        idx >>= 1;
        level += 1;
    }

    // Store the hash at the current level (offset = 2 + level).
    let slot = map_slot(merkle_key.as_slice(), 2 + level);
    internals
        .sstore(ARBOS_STATE_ADDRESS, slot, hash)
        .map_err(|_| PrecompileError::other("sstore failed"))?;

    // Re-read all partials for root computation.
    let new_size = index + 1;
    for i in 0..64u64 {
        if (new_size >> i) == 0 {
            break;
        }
        let pslot = map_slot(merkle_key.as_slice(), 2 + i);
        let val = internals
            .sload(ARBOS_STATE_ADDRESS, pslot)
            .map_err(|_| PrecompileError::other("sload failed"))?
            .data;
        updated_partials.push(B256::from(val.to_be_bytes::<32>()));
    }

    Ok(updated_partials)
}

fn compute_merkle_root(partials: &[B256], size: u64) -> B256 {
    if partials.is_empty() || size == 0 {
        return B256::ZERO;
    }

    let mut accumulator = B256::ZERO;
    let mut started = false;

    for (i, partial) in partials.iter().enumerate() {
        if size & (1 << i) != 0 {
            if !started {
                accumulator = *partial;
                started = true;
            } else {
                // accumulator = keccak256(partial || accumulator)
                let mut preimage = [0u8; 64];
                preimage[..32].copy_from_slice(partial.as_slice());
                preimage[32..].copy_from_slice(accumulator.as_slice());
                accumulator = keccak256(preimage);
            }
        }
    }

    accumulator
}

fn compute_merkle_root_from_u256(partials: &[U256], size: u64) -> B256 {
    let b256_partials: Vec<B256> = partials
        .iter()
        .map(|p| B256::from(p.to_be_bytes::<32>()))
        .collect();
    compute_merkle_root(&b256_partials, size)
}

// ── L1 alias helpers ─────────────────────────────────────────────────

fn apply_l1_alias(addr: Address) -> Address {
    let mut bytes = [0u8; 20];
    for i in 0..20 {
        bytes[i] = addr.0[i].wrapping_add(L1_ALIAS_OFFSET.0[i]);
    }
    Address::new(bytes)
}

fn undo_l1_alias(addr: Address) -> Address {
    let mut bytes = [0u8; 20];
    for i in 0..20 {
        bytes[i] = addr.0[i].wrapping_sub(L1_ALIAS_OFFSET.0[i]);
    }
    Address::new(bytes)
}
