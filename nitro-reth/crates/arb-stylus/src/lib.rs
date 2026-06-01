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
pub mod multi_gas;
pub mod native;
pub mod pricing;
pub mod run;
pub mod trace;

pub use cache::InitCache;
pub use config::{CompileConfig, StylusConfig};
pub use error::{MaybeEscape, StylusError};
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
pub fn strip_stylus_prefix(bytecode: &[u8]) -> Result<(&[u8], u8), StylusError> {
    if bytecode.len() < 4 {
        return Err(StylusError::InvalidProgram(
            "bytecode too short for Stylus prefix",
        ));
    }
    if bytecode[..3] != STYLUS_DISCRIMINANT {
        return Err(StylusError::InvalidProgram(
            "bytecode does not have Stylus discriminant",
        ));
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

/// Returns `true` if the bytecode is a runnable Stylus program: a classic or a
/// root program (a fragment is not runnable on its own). Root code can only
/// have been deployed at the contract-limit version, so no version gate is
/// needed here.
pub fn is_stylus_runnable(bytecode: &[u8]) -> bool {
    is_stylus_classic(bytecode) || is_stylus_root(bytecode)
}

/// Returns `true` if the bytecode is a deployable Stylus component: a classic
/// or root program, or (at the contract-limit version) a fragment. Used to
/// permit storing such code despite its `0xEF` prefix, mirroring
/// `IsStylusComponentPrefix`.
pub fn is_stylus_component(bytecode: &[u8], arbos_version: u64) -> bool {
    use arb_chainspec::arbos_version as av;
    if arbos_version < av::ARBOS_VERSION_STYLUS_CONTRACT_LIMIT {
        return is_stylus_deployable(bytecode, arbos_version);
    }
    is_stylus_deployable(bytecode, arbos_version) || is_stylus_fragment(bytecode)
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
///
/// The bytecode format is `[0xEF, 0xF0, 0x00, dict_byte, ...compressed_wasm]`.
pub fn decompress_wasm(bytecode: &[u8]) -> Result<Vec<u8>, StylusError> {
    if bytecode.len() < 4 || bytecode[..3] != STYLUS_DISCRIMINANT {
        return Err(StylusError::InvalidProgram("not a Stylus program"));
    }
    let dict_byte = bytecode[3];
    let compressed = &bytecode[4..];

    let dict = match dict_byte {
        0 => nitro_brotli::Dictionary::Empty,
        1 => nitro_brotli::Dictionary::StylusProgram,
        _ => return Err(StylusError::InvalidProgram("unsupported dictionary type")),
    };

    nitro_brotli::decompress(compressed, dict)
        .map_err(|e| StylusError::Decompression(format!("{e:?}")))
}

/// A parsed Stylus root program. The on-chain layout is
/// `[0xEF, 0xF0, 0x02, dict, decompressed_len(4, big-endian), addr×20...]`,
/// where each 20-byte address points to a fragment holding part of the
/// compressed WASM.
#[derive(Debug, Clone)]
pub struct StylusRoot {
    pub dictionary: u8,
    pub decompressed_length: u32,
    pub addresses: Vec<alloy_primitives::Address>,
}

impl StylusRoot {
    /// Parse a root program's contract bytecode.
    pub fn parse(bytecode: &[u8]) -> Result<Self, StylusError> {
        if !is_stylus_root(bytecode) {
            return Err(StylusError::InvalidProgram("not a Stylus program root"));
        }
        if bytecode.len() < 8 {
            return Err(StylusError::InvalidProgram("Stylus root too short"));
        }
        let address_data = &bytecode[8..];
        if !address_data.len().is_multiple_of(20) {
            return Err(StylusError::InvalidProgram(
                "Stylus root address data misaligned",
            ));
        }
        let addresses = address_data
            .chunks_exact(20)
            .map(alloy_primitives::Address::from_slice)
            .collect();
        Ok(Self {
            dictionary: bytecode[3],
            decompressed_length: u32::from_be_bytes([
                bytecode[4],
                bytecode[5],
                bytecode[6],
                bytecode[7],
            ]),
            addresses,
        })
    }
}

/// Reconstruct the WASM of a root Stylus program: read each fragment's deployed
/// bytecode via `read_code`, strip its prefix, concatenate the compressed
/// payloads, and decompress with the root's dictionary. When `enforce` is set
/// (i.e. activation), the decompressed-length and fragment-count limits are
/// applied; reads otherwise only reconstruct the program.
pub fn get_wasm_from_root(
    root: &[u8],
    max_wasm_size: u32,
    max_fragments: u8,
    enforce: bool,
    mut read_code: impl FnMut(alloy_primitives::Address) -> Result<Vec<u8>, StylusError>,
) -> Result<Vec<u8>, StylusError> {
    let parsed = StylusRoot::parse(root)?;
    if enforce {
        if parsed.decompressed_length > max_wasm_size {
            return Err(StylusError::InvalidProgram(
                "decompressed length exceeds max wasm size",
            ));
        }
        if parsed.addresses.len() > max_fragments as usize {
            return Err(StylusError::InvalidProgram("fragment count exceeds limit"));
        }
    }
    if parsed.addresses.is_empty() {
        return Err(StylusError::InvalidProgram("fragment count cannot be zero"));
    }
    let dict = match parsed.dictionary {
        0 => nitro_brotli::Dictionary::Empty,
        1 => nitro_brotli::Dictionary::StylusProgram,
        _ => return Err(StylusError::InvalidProgram("unsupported dictionary type")),
    };
    let mut compressed = Vec::new();
    for addr in &parsed.addresses {
        let fragment = read_code(*addr)?;
        if fragment.len() <= 3 || fragment[..3] != STYLUS_FRAGMENT_DISCRIMINANT {
            return Err(StylusError::InvalidProgram(
                "referenced code is not a Stylus fragment",
            ));
        }
        compressed.extend_from_slice(&fragment[3..]);
    }
    let wasm = nitro_brotli::decompress(&compressed, dict)
        .map_err(|e| StylusError::Decompression(format!("{e:?}")))?;
    if wasm.len() != parsed.decompressed_length as usize {
        return Err(StylusError::InvalidProgram(
            "decompressed length does not match the declared length",
        ));
    }
    Ok(wasm)
}

/// Gas charged for reading one fragment of `code_size` bytes during activation,
/// matching `fragmentReadGasCost`: a cold (or warm) account access plus the
/// per-word copy cost. The fragment-read charger uses this for both the
/// preflight affordability check (against the max code size) and the actual
/// per-fragment charge.
pub fn fragment_read_gas(warm: bool, code_size: u64) -> u64 {
    const WARM_ACCESS: u64 = 100; // WarmStorageReadCostEIP2929
    const COLD_ACCESS: u64 = 2_600; // ColdAccountAccessCostEIP2929
    const COPY: u64 = 3; // CopyGas
    let base = if warm { WARM_ACCESS } else { COLD_ACCESS };
    let words = code_size.div_ceil(32);
    base.saturating_add(words.saturating_mul(COPY))
}

/// Activate a Stylus program.
///
/// `wasm` must be the decompressed WASM bytes (call `decompress_wasm` first).
/// `gas` is decremented by the activation cost.
pub fn activate_program(
    wasm: &[u8],
    codehash: &[u8; 32],
    stylus_version: u16,
    arbos_version: u64,
    page_limit: u16,
    debug: bool,
    gas: &mut u64,
) -> Result<arbos::programs::types::ActivationResult, StylusError> {
    let codehash_bytes32 = nitro_arbutil::Bytes32(*codehash);
    let (module, stylus_data) = nitro_prover::machine::Module::activate(
        wasm,
        &codehash_bytes32,
        stylus_version,
        arbos_version,
        page_limit,
        debug,
        gas,
    )
    .map_err(|e| StylusError::Activation(format!("{e}")))?;

    Ok(arbos::programs::types::ActivationResult {
        module_hash: alloy_primitives::B256::from(module.hash().0),
        init_gas: stylus_data.init_cost,
        cached_init_gas: stylus_data.cached_init_cost,
        asm_estimate: stylus_data.asm_estimate,
        footprint: stylus_data.footprint,
    })
}

#[cfg(test)]
mod stylus_root_tests {
    use super::*;
    use alloy_primitives::Address;

    fn make_fragment(chunk: &[u8]) -> Vec<u8> {
        let mut f = STYLUS_FRAGMENT_DISCRIMINANT.to_vec();
        f.extend_from_slice(chunk);
        f
    }

    fn make_root(dict: u8, decompressed_len: u32, addrs: &[Address]) -> Vec<u8> {
        let mut r = STYLUS_ROOT_DISCRIMINANT.to_vec();
        r.push(dict);
        r.extend_from_slice(&decompressed_len.to_be_bytes());
        for a in addrs {
            r.extend_from_slice(a.as_slice());
        }
        r
    }

    #[test]
    fn parse_extracts_fields() {
        let a = Address::repeat_byte(0xab);
        let root = make_root(1, 0x1234_5678, &[a]);
        let p = StylusRoot::parse(&root).unwrap();
        assert_eq!(p.dictionary, 1);
        assert_eq!(p.decompressed_length, 0x1234_5678);
        assert_eq!(p.addresses, vec![a]);
    }

    #[test]
    fn root_reconstructs_wasm_across_fragments() {
        let payload =
            b"\x00asm\x01\x00\x00\x00 stylus wasm body bytes for the fragment round trip".to_vec();
        let compressed =
            nitro_brotli::compress(&payload, 0, 22, nitro_brotli::Dictionary::Empty).unwrap();
        let mid = compressed.len() / 2;
        let frag0 = make_fragment(&compressed[..mid]);
        let frag1 = make_fragment(&compressed[mid..]);
        let a0 = Address::repeat_byte(0x11);
        let a1 = Address::repeat_byte(0x22);
        let root = make_root(0, payload.len() as u32, &[a0, a1]);
        let out = get_wasm_from_root(&root, 100_000, 4, true, |a| {
            Ok(if a == a0 {
                frag0.clone()
            } else {
                frag1.clone()
            })
        })
        .unwrap();
        assert_eq!(out, payload);
    }

    #[test]
    fn decompressed_length_mismatch_rejected_even_unenforced() {
        let payload =
            b"\x00asm\x01\x00\x00\x00 stylus wasm body bytes for the fragment round trip".to_vec();
        let compressed =
            nitro_brotli::compress(&payload, 0, 22, nitro_brotli::Dictionary::Empty).unwrap();
        let frag = make_fragment(&compressed);
        let a = Address::repeat_byte(0x11);
        // Declare a length one byte longer than the fragments actually decompress to.
        let root = make_root(0, payload.len() as u32 + 1, &[a]);
        let err = get_wasm_from_root(&root, 100_000, 4, false, |_| Ok(frag.clone())).unwrap_err();
        assert!(matches!(
            err,
            StylusError::InvalidProgram("decompressed length does not match the declared length")
        ));
    }

    #[test]
    fn fragment_count_zero_rejected() {
        let root = make_root(0, 10, &[]);
        let err = get_wasm_from_root(&root, 100_000, 4, true, |_| Ok(Vec::new())).unwrap_err();
        assert!(matches!(err, StylusError::InvalidProgram(_)));
    }

    #[test]
    fn fragment_count_over_limit_rejected_when_enforced() {
        let addrs: Vec<Address> = (0..5).map(Address::repeat_byte).collect();
        let root = make_root(0, 10, &addrs);
        let err =
            get_wasm_from_root(&root, 100_000, 4, true, |_| Ok(make_fragment(&[]))).unwrap_err();
        assert!(matches!(
            err,
            StylusError::InvalidProgram("fragment count exceeds limit")
        ));
    }

    #[test]
    fn decompressed_length_over_max_rejected_when_enforced() {
        let root = make_root(0, 1000, &[Address::repeat_byte(1)]);
        let err = get_wasm_from_root(&root, 500, 4, true, |_| Ok(make_fragment(&[]))).unwrap_err();
        assert!(matches!(
            err,
            StylusError::InvalidProgram("decompressed length exceeds max wasm size")
        ));
    }

    #[test]
    fn fragment_read_gas_matches_reference_constants() {
        // cold: 2600 + ceil(64/32)*3 = 2606; warm: 100 + 2*3 = 106.
        assert_eq!(fragment_read_gas(false, 64), 2_600 + 2 * 3);
        assert_eq!(fragment_read_gas(true, 64), 100 + 2 * 3);
        // zero-length code: just the account access.
        assert_eq!(fragment_read_gas(false, 0), 2_600);
        // partial word rounds up.
        assert_eq!(fragment_read_gas(true, 33), 100 + 2 * 3);
    }

    #[test]
    fn non_fragment_code_rejected() {
        let root = make_root(0, 10, &[Address::repeat_byte(1)]);
        let err = get_wasm_from_root(&root, 100_000, 4, true, |_| {
            Ok(vec![0xEF, 0xF0, 0x00, 0x00, 0x99])
        })
        .unwrap_err();
        assert!(matches!(
            err,
            StylusError::InvalidProgram("referenced code is not a Stylus fragment")
        ));
    }
}
