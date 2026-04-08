//! Stylus WASM smart contract runtime.
//!
//! Provides the execution pipeline for Stylus programs: WASM compilation
//! and caching, ink metering, host I/O functions, and EVM interop.

pub mod cache;
pub mod config;
pub mod env;
pub mod error;
pub mod evm_api;
pub mod evm_api_impl;
#[allow(unused_mut)]
pub mod host;
pub mod ink;
pub mod meter;
pub mod middleware;
pub mod native;
pub mod pages;
pub mod pricing;
pub mod run;

pub use cache::InitCache;
pub use config::{CompileConfig, StylusConfig};
pub use evm_api::EvmApi;
pub use evm_api_impl::StylusEvmApi;
pub use ink::{Gas, Ink};
pub use meter::{MachineMeter, MeteredMachine, STYLUS_ENTRY_POINT};
pub use native::NativeInstance;
pub use run::RunProgram;

/// Prefix bytes that identify a Stylus WASM program in contract bytecode.
///
/// The discriminant is `[0xEF, 0xF0, 0x00]`. The `0xEF` byte conflicts with
/// EIP-3541, so EIP-3541 must be disabled for Stylus-era blocks to allow
/// deployment. The third byte `0x00` is the EOF version marker.
pub const STYLUS_DISCRIMINANT: [u8; 3] = [0xEF, 0xF0, 0x00];

/// Returns `true` if the bytecode is a Stylus WASM program.
///
/// Checks for the 3-byte discriminant prefix `[0xEF, 0xF0, 0x00]`.
pub fn is_stylus_program(bytecode: &[u8]) -> bool {
    bytecode.len() >= 4 && bytecode[..3] == STYLUS_DISCRIMINANT
}

/// Strips the 4-byte Stylus prefix from contract bytecode.
///
/// Returns `(stripped_bytecode, version_byte)` or an error if the bytecode
/// is too short or doesn't have the Stylus discriminant.
pub fn strip_stylus_prefix(bytecode: &[u8]) -> Result<(&[u8], u8), &'static str> {
    if bytecode.len() < 4 {
        return Err("bytecode too short for Stylus prefix");
    }
    if bytecode[..3] != STYLUS_DISCRIMINANT {
        return Err("bytecode does not have Stylus discriminant");
    }
    let version = bytecode[3];
    Ok((&bytecode[4..], version))
}

/// Root Stylus program prefix: `[0xEF, 0xF0, 0x02]`.
pub const STYLUS_ROOT_DISCRIMINANT: [u8; 3] = [0xEF, 0xF0, 0x02];

/// Fragment prefix: `[0xEF, 0xF0, 0x01]`.
pub const STYLUS_FRAGMENT_DISCRIMINANT: [u8; 3] = [0xEF, 0xF0, 0x01];

/// Returns `true` if the bytecode is a classic Stylus program (`[0xEF, 0xF0, 0x00, ...]`).
pub fn is_stylus_classic(bytecode: &[u8]) -> bool {
    bytecode.len() > 3 && bytecode[..3] == STYLUS_DISCRIMINANT
}

/// Returns `true` if the bytecode is a Stylus root program (`[0xEF, 0xF0, 0x02, ...]`).
pub fn is_stylus_root(bytecode: &[u8]) -> bool {
    bytecode.len() > 3 && bytecode[..3] == STYLUS_ROOT_DISCRIMINANT
}

/// Returns `true` if the bytecode is a Stylus fragment (`[0xEF, 0xF0, 0x01, ...]`).
pub fn is_stylus_fragment(bytecode: &[u8]) -> bool {
    bytecode.len() > 3 && bytecode[..3] == STYLUS_FRAGMENT_DISCRIMINANT
}

/// Returns `true` if the bytecode is a deployable Stylus program.
pub fn is_stylus_deployable(bytecode: &[u8], arbos_version: u64) -> bool {
    use arb_chainspec::arbos_version as av;
    if arbos_version < av::ARBOS_VERSION_STYLUS {
        return false;
    }
    if arbos_version < av::ARBOS_VERSION_STYLUS_CONTRACT_LIMIT {
        return is_stylus_classic(bytecode);
    }
    is_stylus_classic(bytecode) || is_stylus_root(bytecode)
}

/// Decompress a Stylus WASM program from its contract bytecode.
/// The bytecode format is `[0xEF, 0xF0, 0x00, dict_byte, ...compressed_wasm]`.
/// Returns the decompressed WASM bytes.
pub fn decompress_wasm(bytecode: &[u8]) -> eyre::Result<Vec<u8>> {
    if bytecode.len() < 4 || bytecode[..3] != STYLUS_DISCRIMINANT {
        eyre::bail!("not a Stylus program");
    }
    let dict_byte = bytecode[3];
    let compressed = &bytecode[4..];

    let dict = match dict_byte {
        0 => nitro_brotli::Dictionary::Empty,
        1 => nitro_brotli::Dictionary::StylusProgram,
        _ => eyre::bail!("unsupported dictionary type: {dict_byte}"),
    };

    nitro_brotli::decompress(compressed, dict)
        .map_err(|e| eyre::eyre!("brotli decompression failed: {e:?}"))
}

/// Activate a Stylus program using the official Nitro prover.
/// `wasm` must be the decompressed WASM bytes (call `decompress_wasm` first).
/// The `gas` parameter is decremented by the activation cost.
pub fn activate_program(
    wasm: &[u8],
    codehash: &[u8; 32],
    stylus_version: u16,
    arbos_version: u64,
    page_limit: u16,
    debug: bool,
    gas: &mut u64,
) -> eyre::Result<arbos::programs::types::ActivationResult> {
    let codehash_bytes32 = nitro_arbutil::Bytes32(*codehash);
    let (_module, stylus_data) = nitro_prover::machine::Module::activate(
        wasm,
        &codehash_bytes32,
        stylus_version,
        arbos_version,
        page_limit,
        debug,
        gas,
    )?;

    Ok(arbos::programs::types::ActivationResult {
        module_hash: alloy_primitives::B256::from(_module.hash().0),
        init_gas: stylus_data.init_cost,
        cached_init_gas: stylus_data.cached_init_cost,
        asm_estimate: stylus_data.asm_estimate,
        footprint: stylus_data.footprint,
    })
}
