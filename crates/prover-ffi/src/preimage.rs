// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::{
    num::NonZeroUsize,
    sync::{Arc, Mutex},
};

use arbutil::{Bytes32, PreimageType};
use lru::LruCache;
use once_cell::sync::OnceCell;
use prover::{
    Machine,
    machine::PreimageResolver,
    utils::{self, CBytes},
};

use crate::ResolvedPreimage;

lazy_static::lazy_static! {
    static ref BLOBHASH_PREIMAGE_CACHE: Mutex<LruCache<Bytes32, Arc<OnceCell<CBytes>>>> = Mutex::new(LruCache::new(NonZeroUsize::new(12).unwrap()));
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn arbitrator_set_preimage_resolver(
    mach: *mut Machine,
    resolver: unsafe extern "C" fn(u64, u8, *const u8) -> ResolvedPreimage,
) { unsafe {
    (*mach).set_preimage_resolver(Arc::new(
        move |context: u64, ty: PreimageType, hash: Bytes32| -> Option<CBytes> {
            if ty == PreimageType::EthVersionedHash {
                let cache: Arc<OnceCell<CBytes>> = {
                    let mut locked = BLOBHASH_PREIMAGE_CACHE
                        .lock()
                        .unwrap_or_else(|e| e.into_inner());
                    locked.get_or_insert(hash, Default::default).clone()
                };
                return cache
                    .get_or_try_init(|| {
                        match handle_preimage_resolution(context, ty, hash, resolver) {
                            Some(data) => Ok(data),
                            None => {
                                eprintln!("Blob preimage resolution failed for hash {hash}");
                                Err(())
                            }
                        }
                    })
                    .ok()
                    .cloned();
            }
            handle_preimage_resolution(context, ty, hash, resolver)
        },
    ) as PreimageResolver);
}}

unsafe fn handle_preimage_resolution(
    context: u64,
    ty: PreimageType,
    hash: Bytes32,
    resolver: unsafe extern "C" fn(u64, u8, *const u8) -> ResolvedPreimage,
) -> Option<CBytes> { unsafe {
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
}}
