use alloy_evm::precompiles::{DynPrecompile, PrecompileInput};
use alloy_primitives::{Address, U256};
use revm::precompile::{PrecompileError, PrecompileId, PrecompileOutput, PrecompileResult};

/// ArbWasmCache precompile address (0x72).
pub const ARBWASMCACHE_ADDRESS: Address = Address::new([
    0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
    0x00, 0x00, 0x00, 0x00, 0x72,
]);

// Function selectors.
const IS_CACHE_MANAGER: [u8; 4] = [0xf1, 0x37, 0xfc, 0xda];
const ALL_CACHE_MANAGERS: [u8; 4] = [0x35, 0x17, 0x3c, 0x26];
const CACHE_CODEHASH: [u8; 4] = [0x0e, 0xa0, 0x7a, 0x7a];
const CACHE_PROGRAM: [u8; 4] = [0xb6, 0xf4, 0xfb, 0x22];
const EVICT_CODEHASH: [u8; 4] = [0xd4, 0x56, 0xcd, 0x34];
const CODEHASH_IS_CACHED: [u8; 4] = [0x47, 0x97, 0x00, 0xf6];

const COPY_GAS: u64 = 3;

pub fn create_arbwasmcache_precompile() -> DynPrecompile {
    DynPrecompile::new_stateful(PrecompileId::custom("arbwasmcache"), handler)
}

fn handler(mut input: PrecompileInput<'_>) -> PrecompileResult {
    // ArbWasmCache requires ArbOS >= 30 (Stylus).
    if let Some(result) = crate::check_precompile_version(
        arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS,
    ) {
        return result;
    }

    let data = input.data;
    if data.len() < 4 {
        return Err(PrecompileError::other("input too short"));
    }

    let selector: [u8; 4] = [data[0], data[1], data[2], data[3]];
    let gas_cost = COPY_GAS.min(input.gas);

    match selector {
        // CacheCodehash: available only on ArbOS 30, replaced by CacheProgram at 31.
        CACHE_CODEHASH => {
            if let Some(result) = crate::check_method_version(
                arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS,
                arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS,
            ) {
                return result;
            }
            let _ = &mut input;
            Err(PrecompileError::other("caller is not a cache manager"))
        }
        // CacheProgram: requires ArbOS >= 31 (StylusFixes).
        CACHE_PROGRAM => {
            if let Some(result) = crate::check_method_version(
                arb_chainspec::arbos_version::ARBOS_VERSION_STYLUS_FIXES,
                0,
            ) {
                return result;
            }
            let _ = &mut input;
            Err(PrecompileError::other("caller is not a cache manager"))
        }
        IS_CACHE_MANAGER => {
            // Returns false (no cache managers registered).
            Ok(PrecompileOutput::new(gas_cost, vec![0u8; 32].into()))
        }
        ALL_CACHE_MANAGERS => {
            // Returns empty array.
            let mut out = Vec::with_capacity(64);
            out.extend_from_slice(&U256::from(32u64).to_be_bytes::<32>());
            out.extend_from_slice(&U256::ZERO.to_be_bytes::<32>());
            Ok(PrecompileOutput::new(gas_cost, out.into()))
        }
        CODEHASH_IS_CACHED => {
            // Returns false.
            Ok(PrecompileOutput::new(gas_cost, vec![0u8; 32].into()))
        }
        EVICT_CODEHASH => {
            let _ = &mut input;
            Err(PrecompileError::other("caller is not a cache manager"))
        }
        _ => Err(PrecompileError::other("unknown ArbWasmCache selector")),
    }
}
