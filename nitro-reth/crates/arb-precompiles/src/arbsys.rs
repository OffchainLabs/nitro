use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{keccak256, Address, Log, B256, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};
use revm::Database;
use std::cell::RefCell;
use std::collections::HashMap;
use std::sync::Mutex;

use crate::storage_slot::{
    derive_subspace_key, map_slot, root_slot, ARBOS_STATE_ADDRESS, NATIVE_TOKEN_SUBSPACE,
    ROOT_STORAGE_KEY, SEND_MERKLE_SUBSPACE,
};

/// ArbSys precompile address (0x64).
pub const ARBSYS_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x64,
]);

// Function selectors (keccak256 of canonical signature, first 4 bytes).
const WITHDRAW_ETH: [u8; 4] = [0x25, 0xe1, 0x60, 0x63]; // withdrawEth(address)
const SEND_TX_TO_L1: [u8; 4] = [0x92, 0x8c, 0x16, 0x9a]; // sendTxToL1(address,bytes)
const ARB_BLOCK_NUMBER: [u8; 4] = [0xa3, 0xb1, 0xb3, 0x1d]; // arbBlockNumber()
const ARB_BLOCK_HASH: [u8; 4] = [0x2b, 0x40, 0x7a, 0x82]; // arbBlockHash(uint256)
const ARB_CHAIN_ID: [u8; 4] = [0xd1, 0x27, 0xf5, 0x4a]; // arbChainID()
const ARB_OS_VERSION: [u8; 4] = [0x05, 0x10, 0x38, 0xf2]; // arbOSVersion()
const GET_STORAGE_GAS_AVAILABLE: [u8; 4] = [0xa9, 0x45, 0x97, 0xff]; // getStorageGasAvailable()
const IS_TOP_LEVEL_CALL: [u8; 4] = [0x08, 0xbd, 0x62, 0x4c]; // isTopLevelCall()
const MAP_L1_SENDER: [u8; 4] = [0x4d, 0xbb, 0xd5, 0x06]; // mapL1SenderContractAddressToL2Alias(address,address)
const WAS_ALIASED: [u8; 4] = [0x17, 0x5a, 0x26, 0x0b]; // wasMyCallersAddressAliased()
const CALLER_WITHOUT_ALIAS: [u8; 4] = [0xd7, 0x45, 0x23, 0xb3]; // myCallersAddressWithoutAliasing()
const SEND_MERKLE_TREE_STATE: [u8; 4] = [0x7a, 0xee, 0xcd, 0x2a]; // sendMerkleTreeState()

// L1 alias offset: 0x1111000000000000000000000000000000001111
const L1_ALIAS_OFFSET: Address = Address::new([
    0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x11, 0x11,
]);

// MerkleAccumulator: size at offset 0, partials at offset (2 + level).

// Gas costs matching Go's precompile framework (params package).
const COPY_GAS: u64 = 3; // per 32-byte word
const LOG_GAS: u64 = 375;
const LOG_TOPIC_GAS: u64 = 375;
const LOG_DATA_GAS: u64 = 8; // per byte

// Storage gas costs matching Go's arbos/storage/storage.go.
const STORAGE_READ_COST: u64 = 800; // params.SloadGasEIP2200
const STORAGE_WRITE_COST: u64 = 20_000; // params.SstoreSetGasEIP2200
const STORAGE_WRITE_ZERO_COST: u64 = 5_000; // params.SstoreResetGasEIP2200

fn storage_write_cost(value: U256) -> u64 {
    if value.is_zero() {
        STORAGE_WRITE_ZERO_COST
    } else {
        STORAGE_WRITE_COST
    }
}

fn words_for_bytes(n: u64) -> u64 {
    (n + 31) / 32
}

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
    /// Set to `true` when the current transaction is an aliasing type
    /// (unsigned, contract, or retryable L1→L2 message).
    static TX_IS_ALIASED: RefCell<bool> = const { RefCell::new(false) };
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

/// Mark the current transaction as an aliased L1→L2 type.
pub fn set_tx_is_aliased(aliased: bool) {
    TX_IS_ALIASED.with(|cell| *cell.borrow_mut() = aliased);
}

/// Check whether the current transaction uses address aliasing.
pub fn get_tx_is_aliased() -> bool {
    TX_IS_ALIASED.with(|cell| *cell.borrow())
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
    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    Ok(PrecompileOutput::new(
        STORAGE_READ_COST + args_cost + result_cost,
        block_num.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_arb_block_hash(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 + 32 {
        return Err(PrecompileError::other("input too short"));
    }

    let requested: u64 = U256::from_be_slice(&data[4..36])
        .try_into()
        .unwrap_or(u64::MAX);
    let current: u64 = input
        .internals()
        .block_number()
        .try_into()
        .unwrap_or(u64::MAX);

    // Must be strictly less than current and within 256 blocks.
    if requested >= current || requested + 256 < current {
        return Err(PrecompileError::other("invalid block number"));
    }

    let hash = input
        .internals_mut()
        .db_mut()
        .block_hash(requested)
        .map_err(|e| PrecompileError::other(format!("block_hash: {e}")))?;

    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    Ok(PrecompileOutput::new(STORAGE_READ_COST + args_cost + result_cost, hash.0.to_vec().into()))
}

fn handle_arb_chain_id(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let chain_id = input.internals().chain_id();
    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    Ok(PrecompileOutput::new(
        STORAGE_READ_COST + args_cost + result_cost,
        U256::from(chain_id).to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_arbos_version(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let internals = input.internals_mut();

    internals
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;

    // ArbOS version is at root offset 0. Add 55 because Nitro starts at version 56.
    let raw_version = internals
        .sload(ARBOS_STATE_ADDRESS, root_slot(0))
        .map_err(|_| PrecompileError::other("sload failed"))?;
    let version = raw_version.data + U256::from(55);

    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    Ok(PrecompileOutput::new(
        STORAGE_READ_COST + args_cost + result_cost,
        version.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_is_top_level_call(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    // Go returns `evm.Depth() <= 2`.
    // Depth 1 = direct precompile call from tx, depth 2 = one intermediate contract.
    let depth = crate::get_evm_depth();
    let is_top = depth <= 2;
    let val = if is_top { U256::from(1) } else { U256::ZERO };
    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    Ok(PrecompileOutput::new(
        STORAGE_READ_COST + args_cost + result_cost,
        val.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_was_aliased(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let tx_origin = input.internals().tx_origin();
    let caller = input.caller;

    // Read ArbOS version for version-gated behavior.
    let internals = input.internals_mut();
    internals
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;
    let raw_version = internals
        .sload(ARBOS_STATE_ADDRESS, root_slot(0))
        .map_err(|_| PrecompileError::other("sload failed"))?
        .data;
    let arbos_version: u64 = raw_version.try_into().unwrap_or(0);

    // Go: topLevel = isTopLevel(depth < 2 || origin == Contracts[depth-2].Caller())
    // ArbOS < 6: topLevel = depth == 2
    // aliased = topLevel && DoesTxTypeAlias(TopTxType)
    let depth = crate::get_evm_depth();
    let is_top_level = if arbos_version < 6 {
        depth == 2
    } else {
        depth <= 2 || tx_origin == caller
    };

    let aliased = is_top_level && get_tx_is_aliased();
    let val = if aliased { U256::from(1) } else { U256::ZERO };
    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    Ok(PrecompileOutput::new(
        STORAGE_READ_COST + args_cost + result_cost,
        val.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_caller_without_alias(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    // Go returns Contracts[depth-2].Caller() (potentially unaliased).
    // At depth 2 (common case): Contracts[0].Caller() == tx_origin.
    // For deeper calls we'd need the call stack, which isn't available
    // through PrecompileInput. tx_origin is correct at depth <= 2.
    let tx_origin = input.internals().tx_origin();
    let address = tx_origin;

    let result_addr = if get_tx_is_aliased() {
        undo_l1_alias(address)
    } else {
        address
    };

    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    let mut out = [0u8; 32];
    out[12..32].copy_from_slice(result_addr.as_slice());
    Ok(PrecompileOutput::new(STORAGE_READ_COST + args_cost + result_cost, out.to_vec().into()))
}

fn handle_map_l1_sender(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let data = input.data;
    if data.len() < 4 + 64 {
        return Err(PrecompileError::other("input too short"));
    }
    // mapL1SenderContractAddressToL2Alias(address l1_addr, address _unused)
    let l1_addr = Address::from_slice(&data[16..36]);
    let aliased = apply_l1_alias(l1_addr);
    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    let mut out = [0u8; 32];
    out[12..32].copy_from_slice(aliased.as_slice());
    Ok(PrecompileOutput::new(args_cost + result_cost, out.to_vec().into()))
}

fn handle_get_storage_gas(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    // Returns 0 — Nitro has no concept of storage gas.
    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    Ok(PrecompileOutput::new(
        STORAGE_READ_COST + args_cost + result_cost,
        U256::ZERO.to_be_bytes::<32>().to_vec().into(),
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
    let caller = input.caller;
    let value = input.value;
    let block_number = input.internals().block_number();
    let timestamp = input.internals().block_timestamp();
    let l2_block_u64: u64 = block_number.try_into().unwrap_or(0);
    let l1_block_num = get_cached_l1_block_number(l2_block_u64).unwrap_or(0);

    // Gas tracking: match Go's precompile framework burn pattern.
    let mut gas_used = 0u64;
    // Argument copy cost.
    gas_used += COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    // OpenArbosState overhead: Go's makeContext reads version (800 gas) for all non-pure methods.
    gas_used += STORAGE_READ_COST;

    let internals = input.internals_mut();

    // Load the ArbOS state account.
    internals
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;

    // ArbOS v41+: prevent sending value when native token owners exist.
    if !value.is_zero() {
        // Version read gas already covered by OpenArbosState overhead above.
        let raw_version = internals
            .sload(ARBOS_STATE_ADDRESS, root_slot(0))
            .map_err(|_| PrecompileError::other("sload failed"))?
            .data;
        let arbos_version: u64 = raw_version.try_into().unwrap_or(0);
        if arbos_version >= 41 {
            let nt_key = derive_subspace_key(ROOT_STORAGE_KEY, NATIVE_TOKEN_SUBSPACE);
            let nt_size_slot = map_slot(nt_key.as_slice(), 0);
            gas_used += STORAGE_READ_COST;
            let num_owners = internals
                .sload(ARBOS_STATE_ADDRESS, nt_size_slot)
                .map_err(|_| PrecompileError::other("sload failed"))?
                .data;
            if !num_owners.is_zero() {
                return Err(PrecompileError::other(
                    "not allowed to send value when native token owners exist",
                ));
            }
        }
    }

    // Read current Merkle accumulator size.
    let merkle_key = derive_subspace_key(ROOT_STORAGE_KEY, SEND_MERKLE_SUBSPACE);
    let size_slot = map_slot(merkle_key.as_slice(), 0);
    gas_used += STORAGE_READ_COST;
    let current_size = internals
        .sload(ARBOS_STATE_ADDRESS, size_slot)
        .map_err(|_| PrecompileError::other("sload failed"))?
        .data;
    let old_size: u64 = current_size.try_into().unwrap_or(0);

    // Compute the send hash.
    let send_hash = compute_send_hash(
        caller,
        destination,
        block_number,
        U256::from(l1_block_num),
        timestamp,
        value,
        calldata,
    );

    // Update Merkle accumulator: insert leaf and collect intermediate node events.
    let (new_size, merkle_events, partials) = update_merkle_accumulator(
        internals,
        &merkle_key,
        send_hash,
        old_size,
        &mut gas_used,
    )?;

    // Go calls merkleAcc.Size() after Append, which does another storage read.
    gas_used += STORAGE_READ_COST;

    // Write new size.
    let new_size_val = U256::from(new_size);
    gas_used += storage_write_cost(new_size_val);
    internals
        .sstore(ARBOS_STATE_ADDRESS, size_slot, new_size_val)
        .map_err(|_| PrecompileError::other("sstore failed"))?;

    // Emit SendMerkleUpdate events (one per intermediate node, all topics, empty data).
    let update_topic = send_merkle_update_topic();
    for evt in &merkle_events {
        // position = (level << 192) + numLeaves
        let position: U256 = (U256::from(evt.level) << 192) | U256::from(evt.num_leaves);
        internals.log(Log::new_unchecked(
            ARBSYS_ADDRESS,
            vec![
                update_topic,
                B256::from(U256::ZERO.to_be_bytes::<32>()), // reserved = 0
                B256::from(evt.hash.to_be_bytes::<32>()),    // hash
                B256::from(position.to_be_bytes::<32>()),     // position
            ],
            Default::default(), // empty data (all fields indexed)
        ));
        // Gas: 4 topics (event_id + 3 indexed), 0 data bytes.
        gas_used += LOG_GAS + LOG_TOPIC_GAS * 4;
    }

    let leaf_num = new_size - 1;

    // Emit L2ToL1Tx event.
    // Topics: [event_id, destination (indexed), hash (indexed), position (indexed)]
    // Data: ABI-encoded [caller, arbBlockNum, ethBlockNum, timestamp, callvalue, bytes]
    let l2l1_topic = l2_to_l1_tx_topic();
    let dest_topic = B256::left_padding_from(destination.as_slice());
    let hash_topic = B256::from(U256::from_be_bytes(send_hash.0).to_be_bytes::<32>());
    let position_topic = B256::from(U256::from(leaf_num).to_be_bytes::<32>());

    let mut event_data = Vec::with_capacity(256);
    // address caller (left-padded to 32 bytes)
    let mut caller_padded = [0u8; 32];
    caller_padded[12..32].copy_from_slice(caller.as_slice());
    event_data.extend_from_slice(&caller_padded);
    // uint256 arbBlockNum
    event_data.extend_from_slice(&block_number.to_be_bytes::<32>());
    // uint256 ethBlockNum
    event_data.extend_from_slice(&U256::from(l1_block_num).to_be_bytes::<32>());
    // uint256 timestamp
    event_data.extend_from_slice(&timestamp.to_be_bytes::<32>());
    // uint256 callvalue
    event_data.extend_from_slice(&value.to_be_bytes::<32>());
    // bytes data (ABI dynamic type: offset, then length, then data, then padding)
    event_data.extend_from_slice(&U256::from(6 * 32).to_be_bytes::<32>()); // offset = 6 words
    event_data.extend_from_slice(&U256::from(calldata.len()).to_be_bytes::<32>());
    event_data.extend_from_slice(calldata);
    // Pad to 32-byte boundary.
    let pad = (32 - calldata.len() % 32) % 32;
    event_data.extend(std::iter::repeat_n(0u8, pad));

    let l2l1_data_len = event_data.len() as u64;
    internals.log(Log::new_unchecked(
        ARBSYS_ADDRESS,
        vec![l2l1_topic, dest_topic, hash_topic, position_topic],
        event_data.into(),
    ));
    // Gas: 4 topics (event_id + 3 indexed), data = ABI-encoded non-indexed fields.
    gas_used += LOG_GAS + LOG_TOPIC_GAS * 4 + LOG_DATA_GAS * l2l1_data_len;

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

    // Read ArbOS version for return value versioning (no gas — Go uses cached value).
    let raw_version = internals
        .sload(ARBOS_STATE_ADDRESS, root_slot(0))
        .map_err(|_| PrecompileError::other("sload failed"))?
        .data;
    let arbos_version: u64 = raw_version.try_into().unwrap_or(0);

    // ArbOS >= 4: return leafNum; older versions return sendHash.
    let return_val = if arbos_version >= 4 {
        U256::from(leaf_num)
    } else {
        U256::from_be_bytes(send_hash.0)
    };

    // Result copy cost.
    let output = return_val.to_be_bytes::<32>().to_vec();
    gas_used += COPY_GAS * words_for_bytes(output.len() as u64);

    Ok(PrecompileOutput::new(gas_used, output.into()))
}

fn handle_send_merkle_tree_state(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    // Only callable by address zero (for state export).
    if input.caller != Address::ZERO {
        return Err(PrecompileError::other(
            "method can only be called by address zero",
        ));
    }
    let mut gas_used = 0u64;
    let internals = input.internals_mut();

    internals
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(|e| PrecompileError::other(format!("load_account: {e:?}")))?;

    let merkle_key = derive_subspace_key(ROOT_STORAGE_KEY, SEND_MERKLE_SUBSPACE);
    let size_slot = map_slot(merkle_key.as_slice(), 0);
    gas_used += STORAGE_READ_COST;
    let size = internals
        .sload(ARBOS_STATE_ADDRESS, size_slot)
        .map_err(|_| PrecompileError::other("sload failed"))?
        .data;

    let size_u64: u64 = size.try_into().unwrap_or(0);

    // Read partials — stored at offset (2 + level) in the accumulator storage.
    let num_partials = calc_num_partials(size_u64);
    let mut partials = Vec::new();
    for i in 0..num_partials {
        let slot = map_slot(merkle_key.as_slice(), 2 + i);
        gas_used += STORAGE_READ_COST;
        let val = internals
            .sload(ARBOS_STATE_ADDRESS, slot)
            .map_err(|_| PrecompileError::other("sload failed"))?
            .data;
        partials.push(val);
    }

    let b256_partials: Vec<B256> = partials
        .iter()
        .map(|p| B256::from(p.to_be_bytes::<32>()))
        .collect();
    let root = compute_merkle_root(&b256_partials, size_u64);

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

    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(out.len() as u64);
    Ok(PrecompileOutput::new(gas_used + args_cost + result_cost, out.into()))
}

// ── Merkle helpers ───────────────────────────────────────────────────

fn compute_send_hash(
    sender: Address,
    dest: Address,
    arb_block_num: U256,
    eth_block_num: U256,
    timestamp: U256,
    value: U256,
    data: &[u8],
) -> B256 {
    // Go uses raw 20-byte addresses (no left-padding to 32 bytes).
    let mut preimage = Vec::with_capacity(200 + data.len());
    preimage.extend_from_slice(sender.as_slice()); // 20 bytes
    preimage.extend_from_slice(dest.as_slice()); // 20 bytes
    preimage.extend_from_slice(&arb_block_num.to_be_bytes::<32>());
    preimage.extend_from_slice(&eth_block_num.to_be_bytes::<32>());
    preimage.extend_from_slice(&timestamp.to_be_bytes::<32>());
    preimage.extend_from_slice(&value.to_be_bytes::<32>());
    preimage.extend_from_slice(data);
    keccak256(&preimage)
}

/// Intermediate node event from merkle accumulator append.
struct MerkleTreeNodeEvent {
    level: u64,
    num_leaves: u64,
    hash: U256,
}

/// Append a leaf to the merkle accumulator, matching Go's MerkleAccumulator.Append.
///
/// Returns (new_size, events, partials_for_root_computation).
fn update_merkle_accumulator(
    internals: &mut alloy_evm::EvmInternals<'_>,
    merkle_key: &B256,
    item_hash: B256,
    old_size: u64,
    gas_used: &mut u64,
) -> Result<(u64, Vec<MerkleTreeNodeEvent>, Vec<B256>), PrecompileError> {
    let new_size = old_size + 1;
    let mut events = Vec::new();

    // Hash the leaf before insertion (Go: soFar = crypto.Keccak256(itemHash.Bytes())).
    let mut so_far = keccak256(item_hash.as_slice()).to_vec();

    let num_partials_old = calc_num_partials(old_size);
    let mut level = 0u64;

    loop {
        if level == num_partials_old {
            // Store at new top level.
            let h = U256::from_be_slice(&so_far);
            let slot = map_slot(merkle_key.as_slice(), 2 + level);
            *gas_used += storage_write_cost(h);
            internals
                .sstore(ARBOS_STATE_ADDRESS, slot, h)
                .map_err(|_| PrecompileError::other("sstore failed"))?;
            break;
        }

        // Read partial at this level.
        let slot = map_slot(merkle_key.as_slice(), 2 + level);
        *gas_used += STORAGE_READ_COST;
        let this_level = internals
            .sload(ARBOS_STATE_ADDRESS, slot)
            .map_err(|_| PrecompileError::other("sload failed"))?
            .data;

        if this_level.is_zero() {
            // Empty slot: store and stop.
            let h = U256::from_be_slice(&so_far);
            *gas_used += storage_write_cost(h);
            internals
                .sstore(ARBOS_STATE_ADDRESS, slot, h)
                .map_err(|_| PrecompileError::other("sstore failed"))?;
            break;
        }

        // Combine: soFar = keccak256(thisLevel || soFar)
        let mut preimage = [0u8; 64];
        preimage[..32].copy_from_slice(&this_level.to_be_bytes::<32>());
        preimage[32..].copy_from_slice(&so_far);
        so_far = keccak256(preimage).to_vec();

        // Clear the partial at this level (Go sets it to zero hash).
        *gas_used += STORAGE_WRITE_ZERO_COST;
        internals
            .sstore(ARBOS_STATE_ADDRESS, slot, U256::ZERO)
            .map_err(|_| PrecompileError::other("sstore failed"))?;

        level += 1;

        // Record event for this intermediate node.
        events.push(MerkleTreeNodeEvent {
            level,
            num_leaves: new_size - 1,
            hash: U256::from_be_slice(&so_far),
        });
    }

    // Read all partials for root computation.
    // No gas charge here: Go's Append doesn't read partials for root.
    // The root is computed later in the block builder (Go's Root() call).
    let num_partials = calc_num_partials(new_size);
    let mut partials = Vec::with_capacity(num_partials as usize);
    for i in 0..num_partials {
        let pslot = map_slot(merkle_key.as_slice(), 2 + i);
        let val = internals
            .sload(ARBOS_STATE_ADDRESS, pslot)
            .map_err(|_| PrecompileError::other("sload failed"))?
            .data;
        partials.push(B256::from(val.to_be_bytes::<32>()));
    }

    Ok((new_size, events, partials))
}

/// Calculate number of partials for a given size (Go: Log2ceil).
fn calc_num_partials(size: u64) -> u64 {
    if size == 0 {
        return 0;
    }
    64 - size.leading_zeros() as u64
}

/// Compute the merkle root from partials, matching Go's MerkleAccumulator.Root().
///
/// The Go algorithm pads with zero hashes when capacity gaps exist between
/// populated partial levels.
fn compute_merkle_root(partials: &[B256], size: u64) -> B256 {
    if partials.is_empty() || size == 0 {
        return B256::ZERO;
    }

    let num_partials = calc_num_partials(size);
    let mut hash_so_far: Option<B256> = None;
    let mut capacity_in_hash: u64 = 0;
    let mut capacity: u64 = 1;

    for level in 0..num_partials {
        let partial = if (level as usize) < partials.len() {
            partials[level as usize]
        } else {
            B256::ZERO
        };

        if partial != B256::ZERO {
            match hash_so_far {
                None => {
                    hash_so_far = Some(partial);
                    capacity_in_hash = capacity;
                }
                Some(ref h) => {
                    // Pad with zero hashes until capacity matches.
                    let mut current = *h;
                    let mut cap = capacity_in_hash;
                    while cap < capacity {
                        let mut preimage = [0u8; 64];
                        preimage[..32].copy_from_slice(current.as_slice());
                        // second 32 bytes remain zero
                        current = keccak256(preimage);
                        cap *= 2;
                    }
                    // Combine: keccak256(partial || current)
                    let mut preimage = [0u8; 64];
                    preimage[..32].copy_from_slice(partial.as_slice());
                    preimage[32..].copy_from_slice(current.as_slice());
                    let combined = keccak256(preimage);
                    hash_so_far = Some(combined);
                    capacity_in_hash = 2 * capacity;
                }
            }
        }
        capacity *= 2;
    }

    hash_so_far.unwrap_or(B256::ZERO)
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
