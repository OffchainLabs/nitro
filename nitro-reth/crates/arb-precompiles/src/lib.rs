//! Arbitrum precompile contracts.
//!
//! Implements the system contracts at addresses `0x64`+ that provide
//! on-chain access to ArbOS state, gas pricing, retryable tickets,
//! Stylus WASM management, and node interface queries.

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
pub mod arbsys;
mod arbwasm;
mod arbwasmcache;
mod nodeinterface;
mod nodeinterface_debug;
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
pub use nodeinterface_debug::{
    create_nodeinterface_debug_precompile, NODE_INTERFACE_DEBUG_ADDRESS,
};
pub use storage_slot::ARBOS_STATE_ADDRESS;

use alloy_evm::precompiles::{DynPrecompile, PrecompileInput, PrecompilesMap};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};
use std::cell::Cell;

/// RIP-7212 P256VERIFY precompile address (ArbOS v30+).
pub const P256VERIFY_ADDRESS: alloy_primitives::Address =
    alloy_primitives::address!("0000000000000000000000000000000000000100");

/// modexp precompile address (0x05).
const MODEXP_ADDRESS: alloy_primitives::Address =
    alloy_primitives::address!("0000000000000000000000000000000000000005");

/// BLS12-381 precompile addresses (EIP-2537), enabled from ArbOS v50.
const BLS12_381_ADDRESSES: [alloy_primitives::Address; 7] = [
    alloy_primitives::address!("000000000000000000000000000000000000000b"),
    alloy_primitives::address!("000000000000000000000000000000000000000c"),
    alloy_primitives::address!("000000000000000000000000000000000000000d"),
    alloy_primitives::address!("000000000000000000000000000000000000000e"),
    alloy_primitives::address!("000000000000000000000000000000000000000f"),
    alloy_primitives::address!("0000000000000000000000000000000000000010"),
    alloy_primitives::address!("0000000000000000000000000000000000000011"),
];

fn create_p256verify_precompile() -> DynPrecompile {
    DynPrecompile::new(PrecompileId::P256Verify, |input: PrecompileInput<'_>| {
        revm::precompile::secp256r1::p256_verify(input.data, input.gas)
    })
}

fn create_modexp_osaka_precompile() -> DynPrecompile {
    DynPrecompile::new(PrecompileId::ModExp, |input: PrecompileInput<'_>| {
        revm::precompile::modexp::osaka_run(input.data, input.gas)
    })
}

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
    /// Gas consumed by precompile operations before an error.
    static PRECOMPILE_GAS_USED: Cell<u64> = const { Cell::new(0) };
    /// Current tx poster fee (wei), set by executor before each tx.
    /// Used by ArbGasInfo.getCurrentTxL1GasFees to avoid storage reads.
    static CURRENT_TX_POSTER_FEE: Cell<u128> = const { Cell::new(0) };
    /// Poster fee balance correction for BALANCE opcode.
    /// The canonical implementation charges gas_limit * baseFee, but our reduced
    /// gas_limit charges less by posterGas * baseFee. The BALANCE opcode handler
    /// subtracts this amount when checking the sender's balance.
    static POSTER_BALANCE_CORRECTION: Cell<u128> = const { Cell::new(0) };
    /// Current transaction sender address (first 20 bytes as u128 + extra Cell).
    static TX_SENDER_LO: Cell<u128> = const { Cell::new(0) };
    static TX_SENDER_HI: Cell<u32> = const { Cell::new(0) };
    static STYLUS_ACTIVATION_ADDR: Cell<Option<[u8; 20]>> = const { Cell::new(None) };
    static STYLUS_KEEPALIVE_HASH: Cell<Option<[u8; 32]>> = const { Cell::new(None) };
    static STYLUS_ACTIVATION_DATA_FEE: Cell<u128> = const { Cell::new(0) };
}

use std::cell::RefCell;

thread_local! {
    static PENDING_PRECOMPILE_LOGS: RefCell<Vec<(alloy_primitives::Address, Vec<alloy_primitives::B256>, Vec<u8>)>> = const { RefCell::new(Vec::new()) };
    /// Per-block LRU of recently invoked Stylus program codehashes. Used by
    /// ArbOS v60+ pricing; capacity set per-block from `params.BlockCacheSize`.
    static RECENT_WASMS: RefCell<(Vec<alloy_primitives::B256>, usize)> = const { RefCell::new((Vec::new(), 0)) };
}

/// Reset the recent WASMs cache for a new block, with the given capacity.
pub fn reset_recent_wasms(capacity: usize) {
    RECENT_WASMS.with(|c| {
        let mut cache = c.borrow_mut();
        cache.0.clear();
        cache.1 = capacity;
    });
}

/// Insert a Stylus program codehash into the recent WASMs cache.
/// Returns `true` if the codehash was already present (cache hit).
pub fn insert_recent_wasm(hash: alloy_primitives::B256) -> bool {
    RECENT_WASMS.with(|c| {
        let mut cache = c.borrow_mut();
        let was_present = if let Some(pos) = cache.0.iter().position(|h| *h == hash) {
            cache.0.remove(pos);
            true
        } else {
            false
        };
        cache.0.push(hash);
        let max = cache.1;
        if max > 0 && cache.0.len() > max {
            cache.0.remove(0);
        }
        was_present
    })
}

use std::sync::Mutex as StdMutex;

/// Cache of L2 block hashes for the arbBlockHash() precompile.
/// Populated from the header chain during apply_pre_execution_changes.
/// Separate from the journal's block_hashes (which holds L1 hashes for BLOCKHASH opcode).
static L2_BLOCKHASH_CACHE: StdMutex<
    Option<std::collections::HashMap<u64, alloy_primitives::B256>>,
> = StdMutex::new(None);

/// Set an L2 block hash in the arbBlockHash cache.
pub fn set_l2_block_hash(l2_block_number: u64, hash: alloy_primitives::B256) {
    let mut cache = L2_BLOCKHASH_CACHE
        .lock()
        .expect("L2 blockhash cache lock poisoned");
    let map = cache.get_or_insert_with(std::collections::HashMap::new);
    map.insert(l2_block_number, hash);
}

/// Get an L2 block hash from the arbBlockHash cache.
pub fn get_l2_block_hash(l2_block_number: u64) -> Option<alloy_primitives::B256> {
    let cache = L2_BLOCKHASH_CACHE
        .lock()
        .expect("L2 blockhash cache lock poisoned");
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

pub fn reset_precompile_gas() {
    PRECOMPILE_GAS_USED.with(|v| v.set(0));
}

pub fn charge_precompile_gas(gas: u64) {
    PRECOMPILE_GAS_USED.with(|v| v.set(v.get() + gas));
}

pub fn get_precompile_gas() -> u64 {
    PRECOMPILE_GAS_USED.with(|v| v.get())
}

/// Initialize gas tracking for a precompile call: reset accumulator, charge
/// argsCost (CopyGas * input words) and OpenArbosState (1 SLOAD = 800).
pub fn init_precompile_gas(input_len: usize) {
    reset_precompile_gas();
    let args_cost = 3u64 * (input_len as u64).saturating_sub(4).div_ceil(32);
    charge_precompile_gas(args_cost + 800);
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

pub fn set_stylus_activation_request(addr: Option<alloy_primitives::Address>) {
    STYLUS_ACTIVATION_ADDR.with(|v| v.set(addr.map(|a| *a.as_ref())));
}

pub fn take_stylus_activation_request() -> Option<alloy_primitives::Address> {
    STYLUS_ACTIVATION_ADDR.with(|v| {
        let val = v.get();
        v.set(None);
        val.map(alloy_primitives::Address::from)
    })
}

pub fn set_stylus_keepalive_request(hash: Option<alloy_primitives::B256>) {
    STYLUS_KEEPALIVE_HASH.with(|v| v.set(hash.map(|h| h.0)));
}

pub fn take_stylus_keepalive_request() -> Option<alloy_primitives::B256> {
    STYLUS_KEEPALIVE_HASH.with(|v| {
        let val = v.get();
        v.set(None);
        val.map(alloy_primitives::B256::from)
    })
}

pub fn set_stylus_activation_data_fee(fee: alloy_primitives::U256) {
    STYLUS_ACTIVATION_DATA_FEE.with(|v| v.set(fee.try_into().unwrap_or(u128::MAX)));
}

pub fn take_stylus_activation_data_fee() -> alloy_primitives::U256 {
    STYLUS_ACTIVATION_DATA_FEE.with(|v| {
        let val = v.get();
        v.set(0);
        alloy_primitives::U256::from(val)
    })
}

pub fn emit_log(
    address: alloy_primitives::Address,
    topics: &[alloy_primitives::B256],
    data: &[u8],
) {
    PENDING_PRECOMPILE_LOGS.with(|logs| {
        logs.borrow_mut()
            .push((address, topics.to_vec(), data.to_vec()));
    });
}

pub fn take_pending_precompile_logs() -> Vec<(
    alloy_primitives::Address,
    Vec<alloy_primitives::B256>,
    Vec<u8>,
)> {
    PENDING_PRECOMPILE_LOGS.with(|logs| std::mem::take(&mut *logs.borrow_mut()))
}

fn check_precompile_version(min_version: u64) -> Option<PrecompileResult> {
    if get_arbos_version() < min_version {
        Some(Ok(PrecompileOutput::new(0, Default::default())))
    } else {
        None
    }
}

/// Pre-dispatch error: consumes all supplied gas and reverts.
fn burn_all_revert(gas_limit: u64) -> PrecompileResult {
    Ok(PrecompileOutput::new_reverted(
        gas_limit,
        Default::default(),
    ))
}

/// SolError revert: accumulated gas + result-cost, with the error selector.
pub fn sol_error_revert(error_selector: [u8; 4], gas_limit: u64) -> PrecompileResult {
    sol_error_revert_with_args(error_selector, &[], gas_limit)
}

/// SolError revert with ABI-encoded arguments. `args` is the already-encoded
/// argument tail (one 32-byte word per static parameter, head-then-tail layout
/// for dynamic types).
pub fn sol_error_revert_with_args(
    error_selector: [u8; 4],
    args: &[u8],
    gas_limit: u64,
) -> PrecompileResult {
    let mut payload = Vec::with_capacity(4 + args.len());
    payload.extend_from_slice(&error_selector);
    payload.extend_from_slice(args);

    let result_cost = 3u64 * (payload.len() as u64).div_ceil(32); // CopyGas * words
    charge_precompile_gas(result_cost);
    let gas = get_precompile_gas();
    Ok(PrecompileOutput::new_reverted(
        gas.min(gas_limit),
        payload.into(),
    ))
}

/// ABI-encode a u64 as a 32-byte right-aligned word.
pub fn abi_word_u64(v: u64) -> [u8; 32] {
    let mut out = [0u8; 32];
    out[24..].copy_from_slice(&v.to_be_bytes());
    out
}

/// ABI-encode a u16 as a 32-byte right-aligned word.
pub fn abi_word_u16(v: u16) -> [u8; 32] {
    let mut out = [0u8; 32];
    out[30..].copy_from_slice(&v.to_be_bytes());
    out
}

fn gas_check(gas_limit: u64, result: PrecompileResult) -> PrecompileResult {
    let accumulated_gas = get_precompile_gas();
    reset_precompile_gas();
    match result {
        Ok(ref output) if output.gas_used > gas_limit => Err(PrecompileError::OutOfGas),
        Err(PrecompileError::Other(_)) if get_arbos_version() >= 11 => Ok(
            PrecompileOutput::new_reverted(accumulated_gas.min(gas_limit), Default::default()),
        ),
        other => other,
    }
}

/// Returns a revert that consumes the full `gas_limit` if the current ArbOS
/// version is outside `[min_version, max_version]`. `max_version == 0` is
/// unbounded.
fn check_method_version(
    gas_limit: u64,
    min_version: u64,
    max_version: u64,
) -> Option<PrecompileResult> {
    let v = get_arbos_version();
    if v < min_version || (max_version > 0 && v > max_version) {
        Some(burn_all_revert(gas_limit))
    } else {
        None
    }
}

const KZG_POINT_EVALUATION_ADDRESS: alloy_primitives::Address =
    alloy_primitives::address!("000000000000000000000000000000000000000a");

/// Registers Arbitrum precompiles into `map` and applies the per-ArbOS-version
/// adjustments to the standard Ethereum precompile set.
pub fn register_arb_precompiles(map: &mut PrecompilesMap, arbos_version: u64) {
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
        (ARBOWNERPUBLIC_ADDRESS, create_arbownerpublic_precompile()),
        (ARBADDRESSTABLE_ADDRESS, create_arbaddresstable_precompile()),
        (ARBAGGREGATOR_ADDRESS, create_arbaggregator_precompile()),
        (ARBRETRYABLETX_ADDRESS, create_arbretryabletx_precompile()),
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
        (NODE_INTERFACE_ADDRESS, create_nodeinterface_precompile()),
        (
            NODE_INTERFACE_DEBUG_ADDRESS,
            create_nodeinterface_debug_precompile(),
        ),
    ]);

    if arbos_version >= arb_chainspec::arbos_version::ARBOS_VERSION_30 {
        // P256VERIFY stays at 3450 gas on Arbitrum for all ArbOS >= 30,
        // regardless of the underlying EVM spec's Osaka rules.
        map.extend_precompiles([(P256VERIFY_ADDRESS, create_p256verify_precompile())]);
    } else {
        map.apply_precompile(&KZG_POINT_EVALUATION_ADDRESS, |_| None);
        map.apply_precompile(&P256VERIFY_ADDRESS, |_| None);
    }

    if arbos_version >= arb_chainspec::arbos_version::ARBOS_VERSION_50 {
        // ArbOS 50+ switches modexp to the EIP-7823 + EIP-7883 gas schedule.
        map.extend_precompiles([(MODEXP_ADDRESS, create_modexp_osaka_precompile())]);
    } else {
        // BLS12-381 precompiles are not available before ArbOS 50.
        for addr in &BLS12_381_ADDRESSES {
            map.apply_precompile(addr, |_| None);
        }
    }
}

#[cfg(test)]
mod selector_audit {
    #[test]
    fn verify_selectors() {
        fn check(sig: &str, expected: &[u8; 4]) {
            let h = alloy_primitives::keccak256(sig.as_bytes());
            let actual = [h[0], h[1], h[2], h[3]];
            assert_eq!(actual, *expected, "selector mismatch for {sig}: expected 0x{:02x}{:02x}{:02x}{:02x} got 0x{:02x}{:02x}{:02x}{:02x}",
                expected[0], expected[1], expected[2], expected[3], actual[0], actual[1], actual[2], actual[3]);
        }
        check("rectifyChainOwner(address)", &[0x6f, 0xe8, 0x63, 0x73]);
        check("addChainOwner(address)", &[0x48, 0x1f, 0x8d, 0xbf]);
        check("removeChainOwner(address)", &[0x87, 0x92, 0x70, 0x1a]);
        check(
            "releaseL1PricerSurplusFunds(uint256)",
            &[0x31, 0x4b, 0xcf, 0x05],
        );
        check("withdrawEth(address)", &[0x25, 0xe1, 0x60, 0x63]);
        check("sendTxToL1(address,bytes)", &[0x92, 0x8c, 0x16, 0x9a]);
        check("arbBlockNumber()", &[0xa3, 0xb1, 0xb3, 0x1d]);
        check("arbBlockHash(uint256)", &[0x2b, 0x40, 0x7a, 0x82]);
        check("arbChainID()", &[0xd1, 0x27, 0xf5, 0x4a]);
        check("isTopLevelCall()", &[0x08, 0xbd, 0x62, 0x4c]);
        // ArbWasm selectors
        check("activateProgram(address)", &[0x58, 0xc7, 0x80, 0xc2]);
        check("codehashKeepalive(bytes32)", &[0xc6, 0x89, 0xba, 0xd5]);
    }
}
