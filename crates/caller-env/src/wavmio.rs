// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use alloc::{format, string::String};
use core::cmp::min;

use validation::ValidationInput;

use crate::{GuestPtr, MemAccess};

/// A protocol-level error produced by a wavmio host function.
#[derive(Debug)]
pub struct WavmioError(pub String);

impl From<String> for WavmioError {
    fn from(s: String) -> Self {
        Self(s)
    }
}

impl From<&str> for WavmioError {
    fn from(s: &str) -> Self {
        Self(s.into())
    }
}

/// Reads 32-bytes of global state and writes to guest memory.
pub fn get_global_state_bytes32(
    mem: &mut impl MemAccess,
    input: &ValidationInput,
    idx: u32,
    out_ptr: GuestPtr,
) -> Result<(), WavmioError> {
    let Some(global) = input.large_globals.get(idx as usize) else {
        return Err("global read out of bounds in wavmio.getGlobalStateBytes32".into());
    };
    mem.write_slice(out_ptr, &global[..]);
    Ok(())
}

/// Reads 32-bytes from guest memory and writes to global state.
pub fn set_global_state_bytes32(
    mem: &impl MemAccess,
    input: &mut ValidationInput,
    idx: u32,
    src_ptr: GuestPtr,
) -> Result<(), WavmioError> {
    let val = mem.read_fixed(src_ptr);
    let Some(g) = input.large_globals.get_mut(idx as usize) else {
        return Err("global write oob in wavmio.setGlobalStateBytes32".into());
    };
    *g = val;
    Ok(())
}

/// Reads 8-bytes of global state.
pub fn get_global_state_u64(
    _: &impl MemAccess,
    input: &ValidationInput,
    idx: u32,
) -> Result<u64, WavmioError> {
    input
        .small_globals
        .get(idx as usize)
        .copied()
        .ok_or(WavmioError(
            "global read out of bounds in wavmio.getGlobalStateU64".into(),
        ))
}

/// Writes 8-bytes of global state.
pub fn set_global_state_u64(
    _: &impl MemAccess,
    input: &mut ValidationInput,
    idx: u32,
    val: u64,
) -> Result<(), WavmioError> {
    let Some(g) = input.small_globals.get_mut(idx as usize) else {
        return Err("global write out of bounds in wavmio.setGlobalStateU64".into());
    };
    *g = val;
    Ok(())
}

/// Reads up to 32 bytes of a sequencer inbox message at the given offset.
pub fn read_inbox_message(
    mem: &mut impl MemAccess,
    input: &ValidationInput,
    msg_num: u64,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, WavmioError> {
    let message = input
        .sequencer_messages
        .get(&msg_num)
        .ok_or_else(|| WavmioError(format!("missing sequencer inbox message {msg_num}")))?;
    read_message(mem, message, offset, out_ptr)
}

/// Reads up to 32 bytes of a delayed inbox message at the given offset.
pub fn read_delayed_inbox_message(
    mem: &mut impl MemAccess,
    input: &ValidationInput,
    msg_num: u64,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, WavmioError> {
    let message = input
        .delayed_messages
        .get(&msg_num)
        .ok_or_else(|| WavmioError(format!("missing delayed inbox message {msg_num}")))?;
    read_message(mem, message, offset, out_ptr)
}

fn read_message(
    mem: &mut impl MemAccess,
    message: &[u8],
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, WavmioError> {
    let offset = offset as usize;
    let len = min(32, message.len().saturating_sub(offset));
    let read = message.get(offset..(offset + len)).unwrap_or_default();
    mem.write_slice(out_ptr, read);
    Ok(read.len() as u32)
}

/// Looks up a preimage by type and hash, reads up to 32 bytes at an aligned offset.
pub fn resolve_preimage(
    mem: &mut impl MemAccess,
    input: &ValidationInput,
    preimage_type: u8,
    hash_ptr: GuestPtr,
    offset: u32,
    out_ptr: GuestPtr,
    name: &str,
) -> Result<u32, WavmioError> {
    let hash = mem.read_fixed(hash_ptr);
    let offset = offset as usize;

    // Unknown preimage types are not an error — the prover returns 0 bytes for them.
    // Only validate when arbutil is available (both JIT and SP1 enable integrity_check).
    #[cfg(feature = "integrity_check")]
    if arbutil::PreimageType::try_from(preimage_type).is_err() {
        return Ok(0);
    }

    let Some(preimage) = input
        .preimages
        .get(&preimage_type)
        .and_then(|m| m.get(&hash))
    else {
        let hash_hex = hex::encode(hash);
        return Err(format!("Missing requested preimage for hash {hash_hex} in {name}").into());
    };

    #[cfg(all(debug_assertions, feature = "integrity_check"))]
    if let Ok(pt) = arbutil::PreimageType::try_from(preimage_type) {
        use arbutil::PreimageType;
        use sha2::{Digest, Sha256};
        use tiny_keccak::{Hasher, Keccak};

        let calculated: [u8; 32] = match pt {
            PreimageType::Keccak256 => {
                let mut k = Keccak::v256();
                k.update(preimage);
                let mut out = [0u8; 32];
                k.finalize(&mut out);
                out
            }
            PreimageType::Sha2_256 => Sha256::digest(preimage).into(),
            // EthVersionedHash and DACertificate: hash IS the identifier
            PreimageType::EthVersionedHash | PreimageType::DACertificate => hash,
        };
        if calculated != hash {
            return Err(WavmioError(format!(
                "preimage {} hashes to {} but stored under key {} in {name}",
                hex::encode(preimage),
                hex::encode(calculated),
                hex::encode(hash),
            )));
        }
    }

    if !offset.is_multiple_of(32) {
        return Err(format!("bad offset {offset} in {name}").into());
    }

    let len = min(32, preimage.len().saturating_sub(offset));
    let read = preimage.get(offset..(offset + len)).unwrap_or_default();
    mem.write_slice(out_ptr, read);
    Ok(read.len() as u32)
}

/// Returns 1 if a preimage exists for the given type and hash, 0 otherwise.
pub fn validate_certificate(
    mem: &impl MemAccess,
    input: &ValidationInput,
    preimage_type: u8,
    hash_ptr: GuestPtr,
) -> Result<u8, WavmioError> {
    let hash = mem.read_fixed(hash_ptr);
    match input
        .preimages
        .get(&preimage_type)
        .and_then(|m| m.get(&hash))
    {
        Some(_) => Ok(1),
        None => Ok(0),
    }
}

pub fn resolve_keccak_preimage<M: MemAccess>(
    mem: &mut M,
    input: &ValidationInput,
    hash_ptr: GuestPtr,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, WavmioError> {
    resolve_preimage(
        mem,
        input,
        0,
        hash_ptr,
        offset,
        out_ptr,
        "wavmio.ResolvePreImage",
    )
}

pub fn resolve_typed_preimage<M: MemAccess>(
    mem: &mut M,
    input: &ValidationInput,
    preimage_type: u8,
    hash_ptr: GuestPtr,
    offset: u32,
    out_ptr: GuestPtr,
) -> Result<u32, WavmioError> {
    resolve_preimage(
        mem,
        input,
        preimage_type,
        hash_ptr,
        offset,
        out_ptr,
        "wavmio.ResolveTypedPreimage",
    )
}

#[cfg(feature = "wasmer_traits")]
pub mod host {
    use crate::{
        GuestPtr,
        wasmer_traits::{HasMemory, WasmerMem},
    };

    /// Generates a wasmer host function that delegates to a `super::` inner function
    /// taking `(&mut impl MemAccess, &mut ValidationInput, args...)`.
    macro_rules! host_fn_wavmio {
        (fn $name:ident($($arg:ident : $ty:ty),*) -> $ret:ty) => {
            pub fn $name<T>(
                mut ctx: wasmer::FunctionEnvMut<T>,
                $($arg: $ty,)*
            ) -> Result<$ret, T::Escape>
            where
                T: HasMemory + $crate::wavmio::HasInput + Send + 'static,
                T::Escape: std::error::Error + Send + Sync + 'static,
            {
                let (data, store) = ctx.data_and_store_mut();
                let memory = data.memory();
                let input = data.input()?;
                Ok(super::$name(&mut WasmerMem::new(memory, store), input, $($arg,)*)?)
            }
        };
    }

    host_fn_wavmio!(fn get_global_state_bytes32(a: u32, b: GuestPtr) -> ());
    host_fn_wavmio!(fn set_global_state_bytes32(a: u32, b: GuestPtr) -> ());
    host_fn_wavmio!(fn get_global_state_u64(a: u32) -> u64);
    host_fn_wavmio!(fn set_global_state_u64(a: u32, b: u64) -> ());
    host_fn_wavmio!(fn read_inbox_message(a: u64, b: u32, c: GuestPtr) -> u32);
    host_fn_wavmio!(fn read_delayed_inbox_message(a: u64, b: u32, c: GuestPtr) -> u32);
    host_fn_wavmio!(fn resolve_keccak_preimage(a: GuestPtr, b: u32, c: GuestPtr) -> u32);
    host_fn_wavmio!(fn resolve_typed_preimage(a: u8, b: GuestPtr, c: u32, d: GuestPtr) -> u32);
    host_fn_wavmio!(fn validate_certificate(a: u8, b: GuestPtr) -> u8);
}

/// Provides access to the [`ValidationInput`] for a running machine.
///
/// For JIT, acquiring input may trigger lazy loading (socket connect, fork); the
/// associated `Escape` type carries any resulting error. For SP1 the load is
/// handled by `once_cell::Lazy` and the method is infallible in practice.
pub trait HasInput {
    type Escape: From<WavmioError>;
    fn input(&mut self) -> Result<&mut ValidationInput, Self::Escape>;
}
