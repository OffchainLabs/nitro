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
    create_arbsys_precompile, get_cached_l1_block_number, get_tx_is_aliased,
    set_cached_l1_block_number, set_tx_is_aliased, store_arbsys_state, take_arbsys_state,
    ArbSysMerkleState, ARBSYS_ADDRESS,
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
    /// Current EVM call depth, incremented on each CALL/CREATE frame.
    /// Used by precompiles (e.g., ArbSys.isTopLevelCall) to determine
    /// the call stack position. Reset to 0 at transaction start.
    static EVM_CALL_DEPTH: Cell<usize> = const { Cell::new(0) };
    /// Current block timestamp, set before transaction execution.
    /// Used by ArbWasm to compute program age for expiry checks.
    static BLOCK_TIMESTAMP: Cell<u64> = const { Cell::new(0) };
}

/// Set the current ArbOS version for precompile version gating.
pub fn set_arbos_version(version: u64) {
    ARBOS_VERSION.with(|v| v.set(version));
}

/// Get the current ArbOS version.
pub fn get_arbos_version() -> u64 {
    ARBOS_VERSION.with(|v| v.get())
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
