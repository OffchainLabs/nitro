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
pub mod native;
pub mod pages;
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
