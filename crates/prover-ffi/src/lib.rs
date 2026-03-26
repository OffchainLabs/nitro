#![allow(clippy::missing_safety_doc, clippy::too_many_arguments)]

use arbutil::{Bytes32};
use lru::LruCache;
use once_cell::sync::OnceCell;
use prover::{
    utils::CBytes, RustBytes,
};
use std::sync::atomic;
use std::sync::atomic::AtomicU8;
use std::{
    num::NonZeroUsize


    ,
    sync::{Arc, Mutex},
};
pub mod c_strings;
pub mod machine;
pub mod preimage;

lazy_static::lazy_static! {
    static ref BLOBHASH_PREIMAGE_CACHE: Mutex<LruCache<Bytes32, Arc<OnceCell<CBytes>>>> = Mutex::new(LruCache::new(NonZeroUsize::new(12).unwrap()));
}

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
