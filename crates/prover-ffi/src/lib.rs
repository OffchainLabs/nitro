#![allow(clippy::missing_safety_doc, clippy::too_many_arguments)]

use arbutil::{Bytes32, PreimageType};
use lru::LruCache;
use once_cell::sync::OnceCell;
use prover::machine::MachineStatus;
use prover::{
    utils,
    utils::CBytes
    , ResolvedPreimage, RustBytes,
};
use static_assertions::const_assert_eq;
use std::sync::atomic;
use std::sync::atomic::AtomicU8;
use std::{
    num::NonZeroUsize


    ,
    sync::{Arc, Mutex},
};
pub mod c_strings;
pub mod machine;

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

unsafe fn handle_preimage_resolution(
    context: u64,
    ty: PreimageType,
    hash: Bytes32,
    resolver: unsafe extern "C" fn(u64, u8, *const u8) -> ResolvedPreimage,
) -> Option<CBytes> {
    let res = resolver(context, ty.into(), hash.as_ptr());
    if res.len < 0 {
        return None;
    }
    let data = CBytes::from_raw_parts(res.ptr, res.len as usize);

    // Hash may not have a direct link to the data for DACertificate
    if ty == PreimageType::DACertificate {
        return Some(data);
    }

    // Check if preimage rehashes to the provided hash
    match utils::hash_preimage(&data, ty) {
        Ok(have_hash) if have_hash.as_slice() == *hash => {}
        Ok(got_hash) => panic!(
            "Resolved incorrect data for hash {} (rehashed to {})",
            hash,
            Bytes32(got_hash),
        ),
        Err(err) => panic!("Failed to hash preimage from resolver (expecting hash {hash}): {err}",),
    }
    Some(data)
}

