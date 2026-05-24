use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{keccak256, Address, Log, B256, U256};
use alloy_sol_types::{SolError, SolEvent, SolInterface};
use arb_context::ArbPrecompileCtx;
use arb_storage::ARBOS_STATE_ADDRESS;
use arbos::{
    arbos_state::arbos_from_input, burn::SystemBurner, merkle_accumulator::calc_num_partials,
};
use revm::precompile::{PrecompileId, PrecompileOutput, PrecompileResult};
use std::sync::Arc;

use crate::{interfaces::IArbSys, ArbPrecompileError};

/// ArbSys precompile address (0x64).
pub const ARBSYS_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x64,
]);

// L1 alias offset: 0x1111000000000000000000000000000000001111
const L1_ALIAS_OFFSET: Address = Address::new([
    0x11, 0x11, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x11, 0x11,
]);

// MerkleAccumulator: size at offset 0, partials at offset (2 + level).

// Gas costs from the precompile framework (params package).
const COPY_GAS: u64 = 3; // per 32-byte word
const LOG_GAS: u64 = 375;
const LOG_TOPIC_GAS: u64 = 375;
const LOG_DATA_GAS: u64 = 8; // per byte

// Storage gas costs from ArbOS storage accounting.
const STORAGE_READ_COST: u64 = 800; // params.SloadGasEIP2200
const STORAGE_WRITE_COST: u64 = 20_000; // params.SstoreSetGasEIP2200
const STORAGE_WRITE_ZERO_COST: u64 = 5_000; // params.SstoreResetGasEIP2200

fn words_for_bytes(n: u64) -> u64 {
    n.div_ceil(32)
}

/// Keccak gas from the storage burner: 30 + 6*words.
fn keccak_gas(byte_count: u64) -> u64 {
    30 + 6 * words_for_bytes(byte_count)
}

pub fn l2_to_l1_tx_topic() -> B256 {
    IArbSys::L2ToL1Tx::SIGNATURE_HASH
}

pub fn send_merkle_update_topic() -> B256 {
    IArbSys::SendMerkleUpdate::SIGNATURE_HASH
}

pub fn create_arbsys_precompile(ctx: Arc<ArbPrecompileCtx>) -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbsys"), move |input| {
        handler(input, &ctx)
    })
}

fn handler(mut input: PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    let mut gas_used = 0u64;
    let gas_limit = input.gas;
    let data = input.data;
    crate::init_precompile_gas(&mut gas_used, data.len());

    let call = match IArbSys::ArbSysCalls::abi_decode(data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    use IArbSys::ArbSysCalls;
    let result = match call {
        ArbSysCalls::arbBlockNumber(_) => handle_arb_block_number(&mut input, ctx),
        ArbSysCalls::arbBlockHash(c) => {
            handle_arb_block_hash(&mut input, ctx, gas_used, c.arbBlockNum)
        }
        ArbSysCalls::arbChainID(_) => handle_arb_chain_id(&mut input),
        ArbSysCalls::arbOSVersion(_) => handle_arbos_version(&mut input),
        ArbSysCalls::getStorageGasAvailable(_) => handle_get_storage_gas(&mut input),
        ArbSysCalls::isTopLevelCall(_) => handle_is_top_level_call(&mut input, ctx),
        ArbSysCalls::mapL1SenderContractAddressToL2Alias(c) => {
            handle_map_l1_sender(&mut input, c.sender)
        }
        ArbSysCalls::wasMyCallersAddressAliased(_) => handle_was_aliased(&mut input, ctx),
        ArbSysCalls::myCallersAddressWithoutAliasing(_) => {
            handle_caller_without_alias(&mut input, ctx)
        }
        ArbSysCalls::withdrawEth(c) => {
            handle_withdraw_eth(&mut input, ctx, gas_used, c.destination)
        }
        ArbSysCalls::sendTxToL1(c) => {
            handle_send_tx_to_l1(&mut input, ctx, gas_used, c.destination, c.data.as_ref())
        }
        ArbSysCalls::sendMerkleTreeState(_) => handle_send_merkle_tree_state(&mut input, gas_used),
    };
    crate::gas_check(ctx, gas_limit, gas_used, result)
}

// ── view functions ───────────────────────────────────────────────────

fn handle_arb_block_number(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let block_num = U256::from(ctx.block.l2_block_number);
    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    Ok(PrecompileOutput::new(
        STORAGE_READ_COST + args_cost + result_cost,
        block_num.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_arb_block_hash(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    gas_used: u64,
    requested_u256: U256,
) -> PrecompileResult {
    let requested: u64 = requested_u256.try_into().unwrap_or(u64::MAX);
    let current = ctx.block.l2_block_number;

    if requested >= current || requested + 256 < current {
        let arbos_version = ctx.block.arbos_version;
        if arbos_version >= 11 {
            let revert_data = IArbSys::InvalidBlockNumber {
                requested: requested_u256,
                current: U256::from(current),
            }
            .abi_encode();
            let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
            let result_cost = COPY_GAS * words_for_bytes(revert_data.len() as u64);
            return Ok(PrecompileOutput::new_reverted(
                STORAGE_READ_COST + args_cost + result_cost,
                revert_data.into(),
            ));
        }
        return Err(ArbPrecompileError::empty_revert(gas_used).into());
    }

    // L2 block hashes come from the header chain cache — the journal's
    // block_hashes map is pre-populated with L1 hashes for the BLOCKHASH opcode.
    let hash = ctx
        .block
        .cached_l2_block_hash(requested)
        .unwrap_or(B256::ZERO);

    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    Ok(PrecompileOutput::new(
        STORAGE_READ_COST + args_cost + result_cost,
        hash.0.to_vec().into(),
    ))
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

/// User-visible ArbOS version: stored format version + 55.
fn arbos_version_from_format(format_version: U256) -> U256 {
    format_version + U256::from(55)
}

fn handle_arbos_version(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    let internals = input.internals_mut();

    internals
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;

    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let version = arbos_version_from_format(U256::from(arb_state.arbos_version()));

    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    Ok(PrecompileOutput::new(
        STORAGE_READ_COST + args_cost + result_cost,
        version.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_is_top_level_call(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let depth = ctx.evm_depth();
    let is_top = depth <= 2;
    let val = if is_top { U256::from(1) } else { U256::ZERO };
    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    Ok(PrecompileOutput::new(
        STORAGE_READ_COST + args_cost + result_cost,
        val.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_was_aliased(input: &mut PrecompileInput<'_>, ctx: &ArbPrecompileCtx) -> PrecompileResult {
    let internals = input.internals_mut();
    internals
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;
    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let arbos_version = arb_state.arbos_version();

    let tx_origin = input.internals().tx_origin();
    let depth = ctx.evm_depth();
    let is_top_level = if arbos_version < 6 {
        depth == 2
    } else if depth <= 2 {
        true
    } else {
        ctx.caller_at_depth(depth - 1)
            .map(|c| tx_origin == c)
            .unwrap_or(false)
    };

    let aliased = is_top_level && ctx.tx_is_aliased();
    let val = if aliased { U256::from(1) } else { U256::ZERO };
    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    Ok(PrecompileOutput::new(
        STORAGE_READ_COST + args_cost + result_cost,
        val.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_caller_without_alias(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let depth = ctx.evm_depth();
    let address = if depth > 1 {
        ctx.caller_at_depth(depth - 1).unwrap_or(Address::ZERO)
    } else {
        Address::ZERO
    };

    let arbos_version = ctx.block.arbos_version;
    let is_top_level = if arbos_version < 6 {
        depth == 2
    } else if depth <= 2 {
        true
    } else {
        let tx_origin = input.internals().tx_origin();
        ctx.caller_at_depth(depth - 1)
            .map(|c| tx_origin == c)
            .unwrap_or(false)
    };
    let aliased = is_top_level && ctx.tx_is_aliased();
    let result_addr = if aliased {
        undo_l1_alias(address)
    } else {
        address
    };

    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    let mut out = [0u8; 32];
    out[12..32].copy_from_slice(result_addr.as_slice());
    Ok(PrecompileOutput::new(
        STORAGE_READ_COST + args_cost + result_cost,
        out.to_vec().into(),
    ))
}

fn handle_map_l1_sender(input: &mut PrecompileInput<'_>, l1_addr: Address) -> PrecompileResult {
    let aliased = apply_l1_alias(l1_addr);
    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    let mut out = [0u8; 32];
    out[12..32].copy_from_slice(aliased.as_slice());
    Ok(PrecompileOutput::new(
        args_cost + result_cost,
        out.to_vec().into(),
    ))
}

fn handle_get_storage_gas(input: &mut PrecompileInput<'_>) -> PrecompileResult {
    // Returns 0 — ArbOS has no concept of storage gas.
    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(32);
    Ok(PrecompileOutput::new(
        STORAGE_READ_COST + args_cost + result_cost,
        U256::ZERO.to_be_bytes::<32>().to_vec().into(),
    ))
}

// ── L2→L1 messaging ─────────────────────────────────────────────────

fn handle_withdraw_eth(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    gas_used: u64,
    destination: Address,
) -> PrecompileResult {
    if input.is_static {
        return Err(ArbPrecompileError::empty_revert(gas_used).into());
    }
    do_send_tx_to_l1(input, ctx, gas_used, destination, &[])
}

fn handle_send_tx_to_l1(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    gas_used: u64,
    destination: Address,
    calldata: &[u8],
) -> PrecompileResult {
    if input.is_static {
        return Err(ArbPrecompileError::empty_revert(gas_used).into());
    }
    do_send_tx_to_l1(input, ctx, gas_used, destination, calldata)
}

fn do_send_tx_to_l1(
    input: &mut PrecompileInput<'_>,
    ctx: &ArbPrecompileCtx,
    outer_gas_used: u64,
    destination: Address,
    calldata: &[u8],
) -> PrecompileResult {
    let caller = input.caller;
    let value = input.value;
    // Read the L1 block number recorded by StartBlock. `block_env.number` holds
    // the header's mix_hash L1 value, which can lag the StartBlock-updated one.
    let l1_block_number = U256::from(ctx.block.l1_block_number_for_evm);
    let l2_block_number = U256::from(ctx.block.l2_block_number);
    let timestamp = input.internals().block_timestamp();

    let mut gas_used = 0u64;
    // Argument copy cost.
    gas_used += COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    // OpenArbosState overhead: makeContext reads version (800 gas) for all non-pure methods.
    gas_used += STORAGE_READ_COST;

    let internals = input.internals_mut();

    // Load the ArbOS state account.
    internals
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;

    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;
    let arbos_version = arb_state.arbos_version();

    // ArbOS v41+: prevent sending value when native token owners exist.
    if !value.is_zero() && arbos_version >= 41 {
        gas_used += STORAGE_READ_COST;
        let num_owners = arb_state
            .native_token_owners
            .size(internals)
            .map_err(ArbPrecompileError::fatal)?;
        if num_owners != 0 {
            return Err(ArbPrecompileError::empty_revert(outer_gas_used).into());
        }
    }

    // Read current Merkle accumulator size — accumulator's append() will also
    // read+write this slot internally; the precompile charges Go-parity gas
    // for two reads (one before, one phantom "post-append") plus the write.
    gas_used += STORAGE_READ_COST;
    let old_size = arb_state
        .send_merkle_accumulator
        .size(internals)
        .map_err(ArbPrecompileError::fatal)?;

    // Compute the send hash (arbosState.KeccakHash charges gas via burner).
    // Preimage: caller(20) + dest(20) + blockNum(32) + l1BlockNum(32) + time(32) + value(32) +
    // calldata
    let send_hash_input_len = 20 + 20 + 32 * 4 + calldata.len() as u64;
    gas_used += keccak_gas(send_hash_input_len);
    let send_hash = compute_send_hash(
        caller,
        destination,
        l2_block_number,
        l1_block_number,
        timestamp,
        value,
        calldata,
    );

    // Append leaf and collect intermediate node events for emission.
    let merkle_events = arb_state
        .send_merkle_accumulator
        .append(internals, send_hash)
        .map_err(ArbPrecompileError::fatal)?;
    let new_size = old_size + 1;

    // Gas for the partial reads/writes inside append(), plus the new-size
    // write and the phantom "post-append size read" tracked by Go's burner.
    let num_partials_old = calc_num_partials(old_size);
    let n_events = merkle_events.len() as u64;
    let per_merge_gas = STORAGE_READ_COST + keccak_gas(64) + STORAGE_WRITE_ZERO_COST;
    let terminator_gas = if n_events == num_partials_old {
        STORAGE_WRITE_COST
    } else {
        STORAGE_READ_COST + STORAGE_WRITE_COST
    };
    gas_used += n_events * per_merge_gas + terminator_gas;
    gas_used += STORAGE_WRITE_COST; // size.set inside append()
    gas_used += STORAGE_READ_COST; // phantom "merkleAcc.Size() after Append"

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
                evt.hash,                                   // hash
                B256::from(position.to_be_bytes::<32>()),   // position
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
    // uint256 arbBlockNum (L2 block number)
    event_data.extend_from_slice(&l2_block_number.to_be_bytes::<32>());
    // uint256 ethBlockNum (L1 block number)
    event_data.extend_from_slice(&l1_block_number.to_be_bytes::<32>());
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

    // ArbOS >= 4: return leafNum; older versions return sendHash. The version
    // was already read by `arbos_from_input` above (no extra gas charged).
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

fn handle_send_merkle_tree_state(
    input: &mut PrecompileInput<'_>,
    outer_gas_used: u64,
) -> PrecompileResult {
    // Only callable by address zero (for state export).
    if input.caller != Address::ZERO {
        return Err(ArbPrecompileError::empty_revert(outer_gas_used).into());
    }
    let mut gas_used = 0u64;
    let internals = input.internals_mut();

    internals
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;

    let arb_state = arbos_from_input(internals, SystemBurner::new(None, false))
        .map_err(ArbPrecompileError::fatal)?;

    gas_used += STORAGE_READ_COST;
    let size_u64 = arb_state
        .send_merkle_accumulator
        .size(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let size = U256::from(size_u64);

    // Read partials — stored at offset (2 + level) in the accumulator storage.
    let num_partials = calc_num_partials(size_u64);
    let mut partials = Vec::new();
    for i in 0..num_partials {
        gas_used += STORAGE_READ_COST;
        let val = arb_state
            .send_merkle_accumulator
            .partial_at(internals, i)
            .map_err(ArbPrecompileError::fatal)?;
        partials.push(val);
    }

    let root = compute_merkle_root(&partials, size_u64);

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
        out.extend_from_slice(p.0.as_slice());
    }

    let args_cost = COPY_GAS * words_for_bytes(input.data.len().saturating_sub(4) as u64);
    let result_cost = COPY_GAS * words_for_bytes(out.len() as u64);
    Ok(PrecompileOutput::new(
        gas_used + args_cost + result_cost,
        out.into(),
    ))
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
    // Uses raw 20-byte addresses (no left-padding to 32 bytes).
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

/// Compute the merkle root from partials (MerkleAccumulator.Root()).
///
/// Pads with zero hashes when capacity gaps exist between populated partial levels.
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

fn alias_offset_u256() -> U256 {
    U256::from_be_slice(L1_ALIAS_OFFSET.as_slice())
}

fn truncate_to_address(v: U256) -> Address {
    let bytes = v.to_be_bytes::<32>();
    Address::from_slice(&bytes[12..])
}

fn apply_l1_alias(addr: Address) -> Address {
    let val = U256::from_be_slice(addr.as_slice());
    truncate_to_address(val.wrapping_add(alias_offset_u256()))
}

fn undo_l1_alias(addr: Address) -> Address {
    let val = U256::from_be_slice(addr.as_slice());
    truncate_to_address(val.wrapping_sub(alias_offset_u256()))
}

#[cfg(test)]
mod alias_tests {
    use super::*;
    use alloy_primitives::address;

    #[test]
    fn alias_simple_no_carry() {
        let l1 = address!("0000000000000000000000000000000000000000");
        let aliased = apply_l1_alias(l1);
        assert_eq!(aliased, L1_ALIAS_OFFSET);
        assert_eq!(undo_l1_alias(aliased), l1);
    }

    #[test]
    fn alias_carry_propagates_across_bytes() {
        let l1 = address!("00ef000000000000000000000000000000000000");
        let expected = address!("1200000000000000000000000000000000001111");
        assert_eq!(apply_l1_alias(l1), expected);
        assert_eq!(undo_l1_alias(expected), l1);
    }

    #[test]
    fn alias_wraps_at_160_bits() {
        // (2^160 - 1) + 0x1111000000000000000000000000000000001111
        //   = 2^160 + (0x1111000000000000000000000000000000001110)
        //   ≡ 0x1111000000000000000000000000000000001110 (mod 2^160)
        let l1 = address!("ffffffffffffffffffffffffffffffffffffffff");
        let expected = address!("1111000000000000000000000000000000001110");
        assert_eq!(apply_l1_alias(l1), expected);
        assert_eq!(undo_l1_alias(expected), l1);
    }

    #[test]
    fn alias_inverse_round_trip() {
        let cases = [
            address!("0123456789abcdef0123456789abcdef01234567"),
            address!("deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
            address!("ffeeffeeffeeffeeffeeffeeffeeffeeffeeffee"),
        ];
        for addr in cases {
            let aliased = apply_l1_alias(addr);
            let restored = undo_l1_alias(aliased);
            assert_eq!(restored, addr, "round trip failed for {addr}");
        }
    }
}

#[cfg(test)]
mod version_tests {
    use super::*;

    #[test]
    fn arb_os_version_returns_format_plus_55() {
        // formatVersion 51 → user-visible ArbOS version 106 (0x6a)
        assert_eq!(arbos_version_from_format(U256::from(51)), U256::from(106),);
        // formatVersion 1 → 56 (the lowest publicly used version)
        assert_eq!(arbos_version_from_format(U256::from(1)), U256::from(56),);
        // formatVersion 0 → 55
        assert_eq!(arbos_version_from_format(U256::ZERO), U256::from(55),);
    }
}
