use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{keccak256, Address, Log, B256, U256};
use alloy_sol_types::{SolError, SolEvent, SolInterface};
use arb_context::ArbPrecompileCtx;
use arb_storage::ARBOS_STATE_ADDRESS;
use arbos::merkle_accumulator::calc_num_partials;
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

    let call = match IArbSys::ArbSysCalls::abi_decode(data) {
        Ok(c) => c,
        Err(_) => return crate::burn_all_revert(gas_limit),
    };

    // withdrawEth and sendTxToL1 are payable; every other method rejects value.
    if let Some(r) = crate::reject_nonpayable_value(
        input.value,
        data,
        gas_limit,
        &[[0x25, 0xe1, 0x60, 0x63], [0x92, 0x8c, 0x16, 0x9a]],
    ) {
        return r;
    }
    if let Some(r) = crate::reject_static_write(
        input.is_static,
        input.data,
        gas_limit,
        &[[0x25, 0xe1, 0x60, 0x63], [0x92, 0x8c, 0x16, 0x9a]],
    ) {
        return r;
    }
    if let Some(r) = crate::reject_delegate_nonpure(
        input.target_address != input.bytecode_address,
        input.data,
        gas_limit,
        &[[0x4d, 0xbb, 0xd5, 0x06]],
    ) {
        return r;
    }

    // `mapL1SenderContractAddressToL2Alias` is `pure` (no state access), so
    // the framework skips the `OpenArbosState` SLOAD; every other method is at
    // least `view` and pays for it.
    let is_pure = matches!(
        call,
        IArbSys::ArbSysCalls::mapL1SenderContractAddressToL2Alias(_)
    );
    if is_pure {
        crate::init_precompile_gas_pure(&mut gas_used, ctx, data.len());
    } else {
        crate::init_precompile_gas(&mut gas_used, ctx, data.len());
    }

    use IArbSys::ArbSysCalls;
    let result = match call {
        ArbSysCalls::arbBlockNumber(_) => handle_arb_block_number(&mut input, &mut gas_used, ctx),
        ArbSysCalls::arbBlockHash(c) => {
            handle_arb_block_hash(&mut input, &mut gas_used, ctx, c.arbBlockNum)
        }
        ArbSysCalls::arbChainID(_) => handle_arb_chain_id(&mut input, &mut gas_used, ctx),
        ArbSysCalls::arbOSVersion(_) => handle_arbos_version(&mut input, &mut gas_used, ctx),
        ArbSysCalls::getStorageGasAvailable(_) => {
            handle_get_storage_gas(&mut input, &mut gas_used, ctx)
        }
        ArbSysCalls::isTopLevelCall(_) => handle_is_top_level_call(&mut input, &mut gas_used, ctx),
        ArbSysCalls::mapL1SenderContractAddressToL2Alias(c) => {
            handle_map_l1_sender(&mut input, &mut gas_used, ctx, c.sender)
        }
        ArbSysCalls::wasMyCallersAddressAliased(_) => {
            handle_was_aliased(&mut input, &mut gas_used, ctx)
        }
        ArbSysCalls::myCallersAddressWithoutAliasing(_) => {
            handle_caller_without_alias(&mut input, &mut gas_used, ctx)
        }
        ArbSysCalls::withdrawEth(c) => {
            handle_withdraw_eth(&mut input, &mut gas_used, ctx, c.destination)
        }
        ArbSysCalls::sendTxToL1(c) => handle_send_tx_to_l1(
            &mut input,
            &mut gas_used,
            ctx,
            c.destination,
            c.data.as_ref(),
        ),
        ArbSysCalls::sendMerkleTreeState(_) => {
            handle_send_merkle_tree_state(&mut input, &mut gas_used, ctx)
        }
    };
    crate::gas_check(ctx, gas_limit, gas_used, result)
}

// ── view functions ───────────────────────────────────────────────────

fn handle_arb_block_number(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let block_num = U256::from(ctx.block.l2_block_number);
    let gas_limit = input.gas;
    crate::charge_computation(gas_used, ctx, COPY_GAS * words_for_bytes(32));
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        block_num.to_be_bytes::<32>().to_vec().into(),
    ))
}

#[derive(Debug, thiserror::Error)]
#[error("arbBlockHash: L2 block {requested} hash unavailable (current {current})")]
struct MissingL2BlockHash {
    requested: u64,
    current: u64,
}

fn handle_arb_block_hash(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
    requested_u256: U256,
) -> PrecompileResult {
    let requested: u64 = requested_u256.try_into().unwrap_or(u64::MAX);
    let current = ctx.block.l2_block_number;
    let gas_limit = input.gas;

    if requested >= current || requested + 256 < current {
        let arbos_version = ctx.block.arbos_version;
        if arbos_version >= 11 {
            let revert_data = IArbSys::InvalidBlockNumber {
                requested: requested_u256,
                current: U256::from(current),
            }
            .abi_encode();
            crate::charge_computation(
                gas_used,
                ctx,
                COPY_GAS * words_for_bytes(revert_data.len() as u64),
            );
            return Ok(PrecompileOutput::new_reverted(
                (*gas_used).min(gas_limit),
                revert_data.into(),
            ));
        }
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }

    // The window is populated before execution, so an in-range miss is an
    // internal inconsistency — fail loudly instead of returning a zero hash.
    let hash = match ctx.block.cached_l2_block_hash(requested) {
        Some(hash) => hash,
        None => {
            return Err(ArbPrecompileError::fatal(MissingL2BlockHash { requested, current }).into())
        }
    };

    crate::charge_computation(gas_used, ctx, COPY_GAS * words_for_bytes(32));
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        hash.0.to_vec().into(),
    ))
}

fn handle_arb_chain_id(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let chain_id = input.internals().chain_id();
    let gas_limit = input.gas;
    crate::charge_computation(gas_used, ctx, COPY_GAS * words_for_bytes(32));
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        U256::from(chain_id).to_be_bytes::<32>().to_vec().into(),
    ))
}

/// User-visible ArbOS version: stored format version + 55.
fn arbos_version_from_format(format_version: U256) -> U256 {
    format_version + U256::from(55)
}

fn handle_arbos_version(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let internals = input.internals_mut();

    internals
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;

    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let version = arbos_version_from_format(U256::from(arb_state.arbos_version()));

    crate::charge_computation(gas_used, ctx, COPY_GAS * words_for_bytes(32));
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        version.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_is_top_level_call(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let depth = ctx.evm_depth();
    let is_top = depth <= 2;
    let val = if is_top { U256::from(1) } else { U256::ZERO };
    let gas_limit = input.gas;
    crate::charge_computation(gas_used, ctx, COPY_GAS * words_for_bytes(32));
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        val.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_was_aliased(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    let internals = input.internals_mut();
    internals
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;
    let arb_state = ctx
        .block
        .arbos_state(internals)
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
    crate::charge_computation(gas_used, ctx, COPY_GAS * words_for_bytes(32));
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        val.to_be_bytes::<32>().to_vec().into(),
    ))
}

fn handle_caller_without_alias(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
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

    let mut out = [0u8; 32];
    out[12..32].copy_from_slice(result_addr.as_slice());
    crate::charge_computation(gas_used, ctx, COPY_GAS * words_for_bytes(32));
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        out.to_vec().into(),
    ))
}

fn handle_map_l1_sender(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
    l1_addr: Address,
) -> PrecompileResult {
    let aliased = apply_l1_alias(l1_addr);
    let gas_limit = input.gas;
    let mut out = [0u8; 32];
    out[12..32].copy_from_slice(aliased.as_slice());
    // `mapL1SenderContractAddressToL2Alias` is `pure` — no OpenArbosState read,
    // init already charged argsCost only. Body adds result_cost as Computation.
    crate::charge_computation(gas_used, ctx, COPY_GAS * words_for_bytes(32));
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        out.to_vec().into(),
    ))
}

fn handle_get_storage_gas(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    let gas_limit = input.gas;
    crate::charge_computation(gas_used, ctx, COPY_GAS * words_for_bytes(32));
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        U256::ZERO.to_be_bytes::<32>().to_vec().into(),
    ))
}

// ── L2→L1 messaging ─────────────────────────────────────────────────

fn handle_withdraw_eth(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
    destination: Address,
) -> PrecompileResult {
    if input.is_static {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }
    do_send_tx_to_l1(input, gas_used, ctx, destination, &[])
}

fn handle_send_tx_to_l1(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
    destination: Address,
    calldata: &[u8],
) -> PrecompileResult {
    if input.is_static {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }
    do_send_tx_to_l1(input, gas_used, ctx, destination, calldata)
}

fn do_send_tx_to_l1(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
    destination: Address,
    calldata: &[u8],
) -> PrecompileResult {
    let caller = input.caller;
    let value = input.value;
    let gas_limit = input.gas;
    // Read the L1 block number recorded by StartBlock. `block_env.number` holds
    // the header's mix_hash L1 value, which can lag the StartBlock-updated one.
    let l1_block_number = U256::from(ctx.block.l1_block_number_for_evm);
    let l2_block_number = U256::from(ctx.block.l2_block_number);
    let timestamp = input.internals().block_timestamp();

    let internals = input.internals_mut();

    internals
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;

    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let arbos_version = arb_state.arbos_version();

    // ArbOS v41+: prevent sending value when native token owners exist.
    if !value.is_zero() && arbos_version >= 41 {
        crate::charge_storage_read(gas_used, ctx, STORAGE_READ_COST);
        let num_owners = arb_state
            .native_token_owners
            .size(internals)
            .map_err(ArbPrecompileError::fatal)?;
        if num_owners != 0 {
            return Err(ArbPrecompileError::empty_revert(*gas_used).into());
        }
    }

    // Merkle accumulator size: one read before append, one phantom read after.
    crate::charge_storage_read(gas_used, ctx, STORAGE_READ_COST);
    let old_size = arb_state
        .send_merkle_accumulator
        .size(internals)
        .map_err(ArbPrecompileError::fatal)?;

    // keccak hash burn — pure computation.
    let send_hash_input_len = 20 + 20 + 32 * 4 + calldata.len() as u64;
    crate::charge_computation(gas_used, ctx, keccak_gas(send_hash_input_len));
    let send_hash = compute_send_hash(
        caller,
        destination,
        l2_block_number,
        l1_block_number,
        timestamp,
        value,
        calldata,
    );

    let merkle_events = arb_state
        .send_merkle_accumulator
        .append(internals, send_hash)
        .map_err(ArbPrecompileError::fatal)?;
    let new_size = old_size + 1;

    // Per-level merge: one read + one keccak + one write at the reset price.
    // The append's outer terminator is either an extra read+write or just a
    // write depending on whether the last merge consumed all old partials.
    let num_partials_old = calc_num_partials(old_size);
    let n_events = merkle_events.len() as u64;
    let per_merge_keccak = keccak_gas(64);
    crate::charge_storage_read(gas_used, ctx, n_events * STORAGE_READ_COST);
    crate::charge_computation(gas_used, ctx, n_events * per_merge_keccak);
    crate::charge_storage_write(gas_used, ctx, n_events * STORAGE_WRITE_ZERO_COST);
    if n_events == num_partials_old {
        crate::charge_storage_write(gas_used, ctx, STORAGE_WRITE_COST);
    } else {
        crate::charge_storage_read(gas_used, ctx, STORAGE_READ_COST);
        crate::charge_storage_write(gas_used, ctx, STORAGE_WRITE_COST);
    }
    crate::charge_storage_write(gas_used, ctx, STORAGE_WRITE_COST); // size.set
    crate::charge_storage_read(gas_used, ctx, STORAGE_READ_COST); // phantom post-Append size

    // Emit SendMerkleUpdate events (one per intermediate node, all topics, empty data).
    let update_topic = send_merkle_update_topic();
    for evt in &merkle_events {
        // position = (level << 192) + numLeaves
        let position: U256 = (U256::from(evt.level) << 192) | U256::from(evt.num_leaves);
        internals.log(Log::new_unchecked(
            ARBSYS_ADDRESS,
            vec![
                update_topic,
                B256::from(U256::ZERO.to_be_bytes::<32>()),
                evt.hash,
                B256::from(position.to_be_bytes::<32>()),
            ],
            Default::default(),
        ));
        // 4 topics (event_id + 3 indexed), 0 data bytes.
        crate::charge_history_growth(gas_used, ctx, LOG_GAS + LOG_TOPIC_GAS * 4);
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
    let mut caller_padded = [0u8; 32];
    caller_padded[12..32].copy_from_slice(caller.as_slice());
    event_data.extend_from_slice(&caller_padded);
    event_data.extend_from_slice(&l2_block_number.to_be_bytes::<32>());
    event_data.extend_from_slice(&l1_block_number.to_be_bytes::<32>());
    event_data.extend_from_slice(&timestamp.to_be_bytes::<32>());
    event_data.extend_from_slice(&value.to_be_bytes::<32>());
    event_data.extend_from_slice(&U256::from(6 * 32).to_be_bytes::<32>());
    event_data.extend_from_slice(&U256::from(calldata.len()).to_be_bytes::<32>());
    event_data.extend_from_slice(calldata);
    let pad = (32 - calldata.len() % 32) % 32;
    event_data.extend(std::iter::repeat_n(0u8, pad));

    let l2l1_data_len = event_data.len() as u64;
    internals.log(Log::new_unchecked(
        ARBSYS_ADDRESS,
        vec![l2l1_topic, dest_topic, hash_topic, position_topic],
        event_data.into(),
    ));
    crate::charge_history_growth(
        gas_used,
        ctx,
        LOG_GAS + LOG_TOPIC_GAS * 4 + LOG_DATA_GAS * l2l1_data_len,
    );

    let return_val = if arbos_version >= 4 {
        U256::from(leaf_num)
    } else {
        U256::from_be_bytes(send_hash.0)
    };

    let output = return_val.to_be_bytes::<32>().to_vec();
    crate::charge_computation(
        gas_used,
        ctx,
        COPY_GAS * words_for_bytes(output.len() as u64),
    );

    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
        output.into(),
    ))
}

fn handle_send_merkle_tree_state(
    input: &mut PrecompileInput<'_>,
    gas_used: &mut u64,
    ctx: &ArbPrecompileCtx,
) -> PrecompileResult {
    // Only callable by address zero (for state export).
    if input.caller != Address::ZERO {
        return Err(ArbPrecompileError::empty_revert(*gas_used).into());
    }
    let gas_limit = input.gas;
    let internals = input.internals_mut();

    internals
        .load_account(ARBOS_STATE_ADDRESS)
        .map_err(ArbPrecompileError::fatal)?;

    let arb_state = ctx
        .block
        .arbos_state(internals)
        .map_err(ArbPrecompileError::fatal)?;

    crate::charge_storage_read(gas_used, ctx, STORAGE_READ_COST);
    let size_u64 = arb_state
        .send_merkle_accumulator
        .size(internals)
        .map_err(ArbPrecompileError::fatal)?;
    let size = U256::from(size_u64);

    let num_partials = calc_num_partials(size_u64);
    let mut partials = Vec::new();
    for i in 0..num_partials {
        crate::charge_storage_read(gas_used, ctx, STORAGE_READ_COST);
        let val = arb_state
            .send_merkle_accumulator
            .partial_at(internals, i)
            .map_err(ArbPrecompileError::fatal)?;
        partials.push(val);
    }

    let root = compute_merkle_root(&partials, size_u64);

    // ABI: uint256 size, bytes32 root, bytes32[] partials
    let num_partials = partials.len();
    let mut out = Vec::with_capacity(96 + num_partials * 32);
    out.extend_from_slice(&size.to_be_bytes::<32>());
    out.extend_from_slice(&root.0);
    out.extend_from_slice(&U256::from(96u64).to_be_bytes::<32>());
    out.extend_from_slice(&U256::from(num_partials).to_be_bytes::<32>());
    for p in &partials {
        out.extend_from_slice(p.0.as_slice());
    }

    crate::charge_computation(gas_used, ctx, COPY_GAS * words_for_bytes(out.len() as u64));
    Ok(PrecompileOutput::new(
        (*gas_used).min(gas_limit),
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
