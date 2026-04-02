// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

#![allow(clippy::missing_safety_doc, clippy::too_many_arguments)]

use std::sync::{atomic, atomic::AtomicU8};

pub mod c_strings;
pub mod machine;
pub mod preimage;
pub mod types;

pub use types::{CByteArray, ResolvedPreimage, RustBytes, RustSlice};

/// Frees the vector. Does nothing when the vector is null.
///
/// # Safety
///
/// Must only be called once per vec.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn free_rust_bytes(vec: RustBytes) { unsafe {
    if !vec.ptr.is_null() {
        drop(vec.into_vec())
    }
}}

/// Go doesn't have this functionality builtin for whatever reason. Uses relaxed ordering.
#[unsafe(no_mangle)]
pub unsafe extern "C" fn atomic_u8_store(ptr: *mut u8, contents: u8) { unsafe {
    (*(ptr as *mut AtomicU8)).store(contents, atomic::Ordering::Relaxed);
}}
