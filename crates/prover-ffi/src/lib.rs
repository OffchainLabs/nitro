#![allow(clippy::missing_safety_doc, clippy::too_many_arguments)]

use prover::RustBytes;
use std::sync::atomic;
use std::sync::atomic::AtomicU8;
pub mod c_strings;
pub mod machine;
pub mod preimage;

/// Frees the vector. Does nothing when the vector is null.
///
/// # Safety
///
/// Must only be called once per vec.
#[no_mangle]
pub unsafe extern "C" fn free_rust_bytes(vec: RustBytes) {
    if !vec.ptr.is_null() {
        drop(vec.into_vec())
    }
}

/// Go doesn't have this functionality builtin for whatever reason. Uses relaxed ordering.
#[no_mangle]
pub unsafe extern "C" fn atomic_u8_store(ptr: *mut u8, contents: u8) {
    (*(ptr as *mut AtomicU8)).store(contents, atomic::Ordering::Relaxed);
}
