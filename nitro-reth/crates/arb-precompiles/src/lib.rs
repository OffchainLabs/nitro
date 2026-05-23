//! Arbitrum precompile contracts.
//!
//! Implements the system contracts at addresses `0x64`+ that provide
//! on-chain access to ArbOS state, gas pricing, retryable tickets,
//! Stylus WASM management, and node interface queries.

mod error;
mod interfaces;

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
mod arbostest;
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
pub use arbostest::{create_arbostest_precompile, ARBOSTEST_ADDRESS};
pub use arbowner::{create_arbowner_precompile, ARBOWNER_ADDRESS};
pub use arbownerpublic::{create_arbownerpublic_precompile, ARBOWNERPUBLIC_ADDRESS};
pub use arbretryabletx::{
    create_arbretryabletx_precompile, redeem_scheduled_topic, ticket_created_topic,
    ARBRETRYABLETX_ADDRESS,
};
pub use arbstatistics::{create_arbstatistics_precompile, ARBSTATISTICS_ADDRESS};
pub use arbsys::{
    create_arbsys_precompile, get_cached_l1_block_number, get_current_l2_block, ARBSYS_ADDRESS,
};
pub use arbwasm::{create_arbwasm_precompile, ARBWASM_ADDRESS};
pub use arbwasmcache::{create_arbwasmcache_precompile, ARBWASMCACHE_ADDRESS};
pub use error::ArbPrecompileError;
pub use nodeinterface::{
    build_fake_tx_bytes, compute_l1_gas_for_estimate, create_nodeinterface_precompile,
    decode_estimate_args, NODE_INTERFACE_ADDRESS,
};
pub use nodeinterface_debug::{
    create_nodeinterface_debug_precompile, NODE_INTERFACE_DEBUG_ADDRESS,
};
pub use storage_slot::ARBOS_STATE_ADDRESS;

use alloy_evm::precompiles::{DynPrecompile, PrecompileInput, PrecompilesMap};
use alloy_primitives::B256;
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

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

fn create_p256verify_osaka_precompile() -> DynPrecompile {
    DynPrecompile::new(PrecompileId::P256Verify, |input: PrecompileInput<'_>| {
        revm::precompile::secp256r1::p256_verify_osaka(input.data, input.gas)
    })
}

fn create_modexp_osaka_precompile() -> DynPrecompile {
    DynPrecompile::new(PrecompileId::ModExp, |input: PrecompileInput<'_>| {
        revm::precompile::modexp::osaka_run(input.data, input.gas)
    })
}

/// Reset the recent WASMs cache for a new block, with the given capacity.
pub fn reset_recent_wasms(capacity: usize) {
    arb_context::with_active(|c| c.block.reset_recent_wasms(capacity));
}

/// Insert a Stylus program codehash into the recent WASMs cache.
/// Returns `true` if the codehash was already present (cache hit).
pub fn insert_recent_wasm(hash: B256) -> bool {
    arb_context::with_active(|c| c.block.insert_recent_wasm(hash)).unwrap_or(false)
}

/// Read the ArbOS version from the active block context.
pub fn get_arbos_version() -> u64 {
    arb_context::with_active(|c| c.block.arbos_version).unwrap_or(0)
}

/// Whether ArbDebug / ArbosTest debug precompiles are callable. Driven
/// by the chain spec's `AllowDebugPrecompiles` flag.
pub fn allow_debug_precompiles() -> bool {
    arb_context::with_active(|c| c.block.allow_debug_precompiles).unwrap_or(false)
}

/// Read the L1 block number associated with the current block.
pub fn get_l1_block_number_for_evm() -> u64 {
    arb_context::with_active(|c| c.block.l1_block_number_for_evm).unwrap_or(0)
}

/// Read the current gas backlog (mutated by the executor between transactions).
pub fn get_current_gas_backlog() -> u64 {
    arb_context::with_active(|c| c.block.current_gas_backlog()).unwrap_or(0)
}

/// Lookup an L2 block hash recorded for `arbBlockHash`.
pub fn get_l2_block_hash(l2_block_number: u64) -> Option<alloy_primitives::B256> {
    arb_context::with_active(|c| c.block.cached_l2_block_hash(l2_block_number)).flatten()
}

/// Read the current block timestamp.
pub fn get_block_timestamp() -> u64 {
    arb_context::with_active(|c| c.block.block_timestamp).unwrap_or(0)
}

pub fn charge_precompile_gas(gas_used: &mut u64, gas: u64) {
    *gas_used = gas_used.saturating_add(gas);
}

/// Initialize gas tracking for a precompile call: charge argsCost
/// (CopyGas * input words) and OpenArbosState (1 SLOAD = 800).
pub fn init_precompile_gas(gas_used: &mut u64, input_len: usize) {
    let args_cost = 3u64 * (input_len as u64).saturating_sub(4).div_ceil(32);
    charge_precompile_gas(gas_used, args_cost + 800);
}

/// Initialize gas tracking for a `pure` precompile method: like
/// `init_precompile_gas` but skips the OpenArbosState SLOAD (800), matching the
/// reference framework's pure-method path which does not open ArbOS state.
pub fn init_precompile_gas_pure(gas_used: &mut u64, input_len: usize) {
    let args_cost = 3u64 * (input_len as u64).saturating_sub(4).div_ceil(32);
    charge_precompile_gas(gas_used, args_cost);
}

pub fn set_stylus_activation_request(addr: Option<alloy_primitives::Address>) {
    arb_context::with_active(|c| c.set_stylus_activation_addr(addr));
}

pub fn take_stylus_activation_request() -> Option<alloy_primitives::Address> {
    arb_context::with_active(|c| c.take_stylus_activation_addr()).flatten()
}

pub fn set_stylus_keepalive_request(hash: Option<alloy_primitives::B256>) {
    arb_context::with_active(|c| c.set_stylus_keepalive_hash(hash));
}

pub fn take_stylus_keepalive_request() -> Option<alloy_primitives::B256> {
    arb_context::with_active(|c| c.take_stylus_keepalive_hash()).flatten()
}

pub fn set_stylus_activation_data_fee(fee: alloy_primitives::U256) {
    arb_context::with_active(|c| c.set_stylus_activation_data_fee(fee));
}

pub fn take_stylus_activation_data_fee() -> alloy_primitives::U256 {
    arb_context::with_active(|c| c.take_stylus_activation_data_fee())
        .unwrap_or(alloy_primitives::U256::ZERO)
}

pub fn set_stylus_call_value(value: alloy_primitives::U256) {
    arb_context::with_active(|c| c.set_stylus_call_value(value));
}

pub fn get_stylus_call_value() -> alloy_primitives::U256 {
    arb_context::with_active(|c| c.stylus_call_value()).unwrap_or(alloy_primitives::U256::ZERO)
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

/// Emit a pre-encoded Solidity custom-error payload (selector + ABI args)
/// as a revert. Adds the copy cost for the payload to the accumulated gas.
pub fn sol_error_revert(gas_used: &mut u64, payload: Vec<u8>, gas_limit: u64) -> PrecompileResult {
    let result_cost = 3u64 * (payload.len() as u64).div_ceil(32); // CopyGas * words
    charge_precompile_gas(gas_used, result_cost);
    Ok(PrecompileOutput::new_reverted(
        (*gas_used).min(gas_limit),
        payload.into(),
    ))
}

fn gas_check(gas_limit: u64, gas_used: u64, result: PrecompileResult) -> PrecompileResult {
    match result {
        Ok(ref output) if output.gas_used > gas_limit => Err(PrecompileError::OutOfGas),
        Err(PrecompileError::Other(_)) if get_arbos_version() >= 11 => Ok(
            PrecompileOutput::new_reverted(gas_used.min(gas_limit), Default::default()),
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
        (ARBOSTEST_ADDRESS, create_arbostest_precompile()),
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

    if arbos_version >= arb_chainspec::arbos_version::ARBOS_VERSION_50 {
        // P256VERIFY adopts the EIP-7951 Osaka schedule (6900 gas) at v50+.
        map.extend_precompiles([(P256VERIFY_ADDRESS, create_p256verify_osaka_precompile())]);
    } else if arbos_version >= arb_chainspec::arbos_version::ARBOS_VERSION_30 {
        // RIP-7212 P256VERIFY at 3450 gas (ArbOS 30..49).
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
mod recent_wasms_tests {
    use alloy_primitives::B256;
    use arb_context::BlockCtx;

    #[test]
    fn reset_clears_entries_and_sets_capacity() {
        let block = BlockCtx::default();
        let h1 = B256::repeat_byte(0xa1);
        let h2 = B256::repeat_byte(0xa2);
        block.reset_recent_wasms(8);
        assert!(!block.insert_recent_wasm(h1));
        assert!(!block.insert_recent_wasm(h2));
        assert!(block.insert_recent_wasm(h1));
        block.reset_recent_wasms(8);
        assert!(
            !block.insert_recent_wasm(h1),
            "reset must wipe prior entries"
        );
    }

    #[test]
    fn capacity_evicts_oldest() {
        let block = BlockCtx::default();
        let h1 = B256::repeat_byte(0x01);
        let h2 = B256::repeat_byte(0x02);
        let h3 = B256::repeat_byte(0x03);
        block.reset_recent_wasms(2);
        assert!(!block.insert_recent_wasm(h1));
        assert!(!block.insert_recent_wasm(h2));
        assert!(!block.insert_recent_wasm(h3));
        assert!(
            !block.insert_recent_wasm(h1),
            "h1 should be evicted after h3 push"
        );
    }

    #[test]
    fn zero_capacity_is_no_op_cache() {
        let block = BlockCtx::default();
        let h = B256::repeat_byte(0xff);
        block.reset_recent_wasms(0);
        assert!(!block.insert_recent_wasm(h));
        block.reset_recent_wasms(0);
        assert!(!block.insert_recent_wasm(h));
    }
}

#[cfg(test)]
mod p256_gas_tests {
    //! P256VERIFY gas: 3450 for ArbOS 30..49, 6900 for v50+ (EIP-7951 Osaka).
    use revm::precompile::secp256r1::{p256_verify, p256_verify_osaka};

    // Valid p256 signature input from the upstream RIP-7212 test vectors.
    const VALID_INPUT_HEX: &str = "4cee90eb86eaa050036147a12d49004b6b9c72bd725d39d4785011fe190f0b4da73bd4903f0ce3b639bbbf6e8e80d16931ff4bcf5993d58468e8fb19086e8cac36dbcd03009df8c59286b162af3bd7fcc0450c9aa81be5d10d312af6c66b1d604aebd3099c618202fcfe16ae7770b0c49ab5eadf74b754204a3bb6060e44eff37618b065f9832de4ca6ca971a7a1adc826d0f7c00181a5fb2ddf79ae00b4e10e";

    fn input_bytes() -> Vec<u8> {
        (0..VALID_INPUT_HEX.len() / 2)
            .map(|i| u8::from_str_radix(&VALID_INPUT_HEX[i * 2..i * 2 + 2], 16).unwrap())
            .collect()
    }

    #[test]
    fn rip7212_charges_3450() {
        let input = input_bytes();
        let out = p256_verify(&input, 10_000).expect("ok");
        assert_eq!(out.gas_used, 3450);
    }

    #[test]
    fn osaka_charges_6900() {
        let input = input_bytes();
        let out = p256_verify_osaka(&input, 10_000).expect("ok");
        assert_eq!(out.gas_used, 6900);
    }
}
