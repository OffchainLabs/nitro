mod arbaddresstable;
mod arbaggregator;
mod arbbls;
mod arbdebug;
mod arbfilteredtxmanager;
mod arbfunctiontable;
mod arbgasinfo;
mod arbinfo;
mod arbnativetokenmanager;
mod arbosacts;
mod arbowner;
mod arbownerpublic;
mod arbretryabletx;
mod arbstatistics;
mod arbsys;
mod arbwasm;
mod arbwasmcache;
mod nodeinterface;
pub mod storage_slot;

pub use arbaddresstable::{create_arbaddresstable_precompile, ARBADDRESSTABLE_ADDRESS};
pub use arbaggregator::{create_arbaggregator_precompile, ARBAGGREGATOR_ADDRESS};
pub use arbbls::{create_arbbls_precompile, ARBBLS_ADDRESS};
pub use arbdebug::{create_arbdebug_precompile, ARBDEBUG_ADDRESS};
pub use arbfilteredtxmanager::{
    create_arbfilteredtxmanager_precompile, ARBFILTEREDTXMANAGER_ADDRESS,
};
pub use arbfunctiontable::{create_arbfunctiontable_precompile, ARBFUNCTIONTABLE_ADDRESS};
pub use arbgasinfo::{create_arbgasinfo_precompile, ARBGASINFO_ADDRESS};
pub use arbinfo::{create_arbinfo_precompile, ARBINFO_ADDRESS};
pub use arbnativetokenmanager::{
    create_arbnativetokenmanager_precompile, ARBNATIVETOKENMANAGER_ADDRESS,
};
pub use arbosacts::{create_arbosacts_precompile, ARBOSACTS_ADDRESS};
pub use arbowner::{create_arbowner_precompile, ARBOWNER_ADDRESS};
pub use arbownerpublic::{create_arbownerpublic_precompile, ARBOWNERPUBLIC_ADDRESS};
pub use arbretryabletx::{
    create_arbretryabletx_precompile, redeem_scheduled_topic, ticket_created_topic,
    ARBRETRYABLETX_ADDRESS,
};
pub use arbstatistics::{create_arbstatistics_precompile, ARBSTATISTICS_ADDRESS};
pub use arbsys::{
    create_arbsys_precompile, get_cached_l1_block_number, get_current_l2_block, get_tx_is_aliased,
    set_cached_l1_block_number, set_current_l2_block, set_tx_is_aliased, store_arbsys_state,
    take_arbsys_state, ArbSysMerkleState, ARBSYS_ADDRESS,
};
pub use arbwasm::{create_arbwasm_precompile, ARBWASM_ADDRESS};
pub use arbwasmcache::{create_arbwasmcache_precompile, ARBWASMCACHE_ADDRESS};
pub use nodeinterface::{create_nodeinterface_precompile, NODE_INTERFACE_ADDRESS};
pub use storage_slot::ARBOS_STATE_ADDRESS;

use alloy_evm::precompiles::PrecompilesMap;
use revm::precompile::{PrecompileError, PrecompileOutput, PrecompileResult};
use std::cell::Cell;

// ── ArbOS version thread-local ──────────────────────────────────────

thread_local! {
    /// Current ArbOS version, set by the block executor before transaction execution.
    static ARBOS_VERSION: Cell<u64> = const { Cell::new(0) };
    /// L1 block number for the NUMBER opcode, from ArbOS state after StartBlock.
    static L1_BLOCK_NUMBER_FOR_EVM: Cell<u64> = const { Cell::new(0) };
    /// Current EVM call depth, incremented on each CALL/CREATE frame.
    /// Used by precompiles (e.g., ArbSys.isTopLevelCall) to determine
    /// the call stack position. Reset to 0 at transaction start.
    static EVM_CALL_DEPTH: Cell<usize> = const { Cell::new(0) };
    /// Current block timestamp, set before transaction execution.
    /// Used by ArbWasm to compute program age for expiry checks.
    static BLOCK_TIMESTAMP: Cell<u64> = const { Cell::new(0) };
    /// Current gas backlog value, set by executor before each tx.
    /// Used by Redeem precompile to determine ShrinkBacklog write cost.
    static CURRENT_GAS_BACKLOG: Cell<u64> = const { Cell::new(0) };
    /// Current tx poster fee (wei), set by executor before each tx.
    /// Used by ArbGasInfo.getCurrentTxL1GasFees to avoid storage reads.
    /// In Nitro, this is read from c.txProcessor.PosterFee (a memory field).
    static CURRENT_TX_POSTER_FEE: Cell<u128> = const { Cell::new(0) };
    /// Poster fee balance correction for BALANCE opcode.
    /// Nitro's BuyGas charges gas_limit * baseFee, but our reduced gas_limit
    /// charges less by posterGas * baseFee. The BALANCE opcode handler subtracts
    /// this amount when checking the sender's balance to match Nitro.
    static POSTER_BALANCE_CORRECTION: Cell<u128> = const { Cell::new(0) };
    /// Current transaction sender address (first 20 bytes as u128 + extra Cell).
    static TX_SENDER_LO: Cell<u128> = const { Cell::new(0) };
    static TX_SENDER_HI: Cell<u32> = const { Cell::new(0) };
}

use std::sync::Mutex as StdMutex;

/// Cache of L2 block hashes for the arbBlockHash() precompile.
/// Populated from the header chain during apply_pre_execution_changes.
/// Separate from the journal's block_hashes (which holds L1 hashes for BLOCKHASH opcode).
static L2_BLOCKHASH_CACHE: StdMutex<Option<std::collections::HashMap<u64, alloy_primitives::B256>>> =
    StdMutex::new(None);

/// Set an L2 block hash in the arbBlockHash cache.
pub fn set_l2_block_hash(l2_block_number: u64, hash: alloy_primitives::B256) {
    let mut cache = L2_BLOCKHASH_CACHE.lock().unwrap();
    let map = cache.get_or_insert_with(std::collections::HashMap::new);
    map.insert(l2_block_number, hash);
}

/// Get an L2 block hash from the arbBlockHash cache.
pub fn get_l2_block_hash(l2_block_number: u64) -> Option<alloy_primitives::B256> {
    let cache = L2_BLOCKHASH_CACHE.lock().unwrap();
    cache.as_ref()?.get(&l2_block_number).copied()
}

/// Set the current ArbOS version for precompile version gating.
pub fn set_arbos_version(version: u64) {
    ARBOS_VERSION.with(|v| v.set(version));
}

/// Get the current ArbOS version.
pub fn get_arbos_version() -> u64 {
    ARBOS_VERSION.with(|v| v.get())
}

/// Set the L1 block number for the NUMBER opcode.
pub fn set_l1_block_number_for_evm(number: u64) {
    L1_BLOCK_NUMBER_FOR_EVM.with(|v| v.set(number));
}

/// Get the L1 block number for the NUMBER opcode.
pub fn get_l1_block_number_for_evm() -> u64 {
    L1_BLOCK_NUMBER_FOR_EVM.with(|v| v.get())
}

/// Set the current gas backlog value for the Redeem precompile.
pub fn set_current_gas_backlog(backlog: u64) {
    CURRENT_GAS_BACKLOG.with(|v| v.set(backlog));
}

/// Get the current gas backlog value.
pub fn get_current_gas_backlog() -> u64 {
    CURRENT_GAS_BACKLOG.with(|v| v.get())
}

/// Set the current tx poster fee for ArbGasInfo.getCurrentTxL1GasFees.
pub fn set_current_tx_poster_fee(fee_wei: u128) {
    CURRENT_TX_POSTER_FEE.with(|v| v.set(fee_wei));
}

/// Get the current tx poster fee.
pub fn get_current_tx_poster_fee() -> u128 {
    CURRENT_TX_POSTER_FEE.with(|v| v.get())
}

/// Set the poster balance correction for BALANCE opcode adjustment.
pub fn set_poster_balance_correction(correction: alloy_primitives::U256) {
    let val: u128 = correction.try_into().unwrap_or(u128::MAX);
    POSTER_BALANCE_CORRECTION.with(|v| v.set(val));
}

/// Get the poster balance correction.
pub fn get_poster_balance_correction() -> alloy_primitives::U256 {
    alloy_primitives::U256::from(POSTER_BALANCE_CORRECTION.with(|v| v.get()))
}

/// Set the current tx sender for BALANCE correction.
pub fn set_current_tx_sender(addr: alloy_primitives::Address) {
    let bytes = addr.as_slice();
    let lo = u128::from_be_bytes(bytes[4..20].try_into().unwrap_or([0u8; 16]));
    let hi = u32::from_be_bytes(bytes[0..4].try_into().unwrap_or([0u8; 4]));
    TX_SENDER_LO.with(|v| v.set(lo));
    TX_SENDER_HI.with(|v| v.set(hi));
}

/// Get the current tx sender.
pub fn get_current_tx_sender() -> alloy_primitives::Address {
    let lo = TX_SENDER_LO.with(|v| v.get());
    let hi = TX_SENDER_HI.with(|v| v.get());
    let mut bytes = [0u8; 20];
    bytes[0..4].copy_from_slice(&hi.to_be_bytes());
    bytes[4..20].copy_from_slice(&lo.to_be_bytes());
    alloy_primitives::Address::new(bytes)
}

/// Set the EVM call depth to a specific value.
/// Called by the precompile provider which reads the depth from revm's journal.
pub fn set_evm_depth(depth: usize) {
    EVM_CALL_DEPTH.with(|v| v.set(depth));
}

/// Get the current EVM call depth.
pub fn get_evm_depth() -> usize {
    EVM_CALL_DEPTH.with(|v| v.get())
}

/// Set the current block timestamp for precompile queries.
pub fn set_block_timestamp(timestamp: u64) {
    BLOCK_TIMESTAMP.with(|v| v.set(timestamp));
}

/// Get the current block timestamp.
pub fn get_block_timestamp() -> u64 {
    BLOCK_TIMESTAMP.with(|v| v.get())
}

/// Check precompile-level version gate. If the current ArbOS version is below
/// `min_version`, the precompile is not yet active and we return success with
/// empty bytes (as if calling a contract that doesn't exist).
fn check_precompile_version(min_version: u64) -> Option<PrecompileResult> {
    if get_arbos_version() < min_version {
        Some(Ok(PrecompileOutput::new(0, Default::default())))
    } else {
        None
    }
}

/// Ensure precompile gas_used does not exceed the gas limit.
/// Returns `OutOfGas` if it does, preventing an assertion panic in alloy-evm.
fn gas_check(gas_limit: u64, result: PrecompileResult) -> PrecompileResult {
    match result {
        Ok(ref output) if output.gas_used > gas_limit => Err(PrecompileError::OutOfGas),
        other => other,
    }
}

/// Check method-level version gate. If the current ArbOS version is below
/// `min_version` or above `max_version` (when non-zero), the method reverts.
fn check_method_version(min_version: u64, max_version: u64) -> Option<PrecompileResult> {
    let v = get_arbos_version();
    if v < min_version || (max_version > 0 && v > max_version) {
        Some(Err(PrecompileError::other("method not available at this ArbOS version")))
    } else {
        None
    }
}

/// Register all Arbitrum precompiles into a [`PrecompilesMap`].
pub fn register_arb_precompiles(map: &mut PrecompilesMap) {
    map.extend_precompiles([
        (ARBSYS_ADDRESS, create_arbsys_precompile()),
        (ARBGASINFO_ADDRESS, create_arbgasinfo_precompile()),
        (ARBINFO_ADDRESS, create_arbinfo_precompile()),
        (ARBSTATISTICS_ADDRESS, create_arbstatistics_precompile()),
        (
            ARBFUNCTIONTABLE_ADDRESS,
            create_arbfunctiontable_precompile(),
        ),
        (ARBOSACTS_ADDRESS, create_arbosacts_precompile()),
        (
            ARBOWNERPUBLIC_ADDRESS,
            create_arbownerpublic_precompile(),
        ),
        (
            ARBADDRESSTABLE_ADDRESS,
            create_arbaddresstable_precompile(),
        ),
        (ARBAGGREGATOR_ADDRESS, create_arbaggregator_precompile()),
        (
            ARBRETRYABLETX_ADDRESS,
            create_arbretryabletx_precompile(),
        ),
        (ARBOWNER_ADDRESS, create_arbowner_precompile()),
        (ARBBLS_ADDRESS, create_arbbls_precompile()),
        (ARBDEBUG_ADDRESS, create_arbdebug_precompile()),
        (ARBWASM_ADDRESS, create_arbwasm_precompile()),
        (ARBWASMCACHE_ADDRESS, create_arbwasmcache_precompile()),
        (
            ARBFILTEREDTXMANAGER_ADDRESS,
            create_arbfilteredtxmanager_precompile(),
        ),
        (
            ARBNATIVETOKENMANAGER_ADDRESS,
            create_arbnativetokenmanager_precompile(),
        ),
        (
            NODE_INTERFACE_ADDRESS,
            create_nodeinterface_precompile(),
        ),
    ]);
}
