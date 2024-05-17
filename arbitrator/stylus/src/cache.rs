// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

use arbutil::Bytes32;
use eyre::Result;
use lazy_static::lazy_static;
use lru::LruCache;
use parking_lot::Mutex;
use prover::programs::config::CompileConfig;
use std::{collections::HashMap, num::NonZeroUsize};
use wasmer::{Engine, Module, Store};

lazy_static! {
    static ref INIT_CACHE: Mutex<InitCache> = Mutex::new(InitCache::new(256));
}

macro_rules! cache {
    () => {
        INIT_CACHE.lock()
    };
}

pub struct InitCache {
    long_term: HashMap<CacheKey, CacheItem>,
    lru: LruCache<CacheKey, CacheItem>,
}

#[derive(Clone, Copy, Hash, PartialEq, Eq)]
struct CacheKey {
    module_hash: Bytes32,
    version: u16,
    debug: bool,
}

impl CacheKey {
    fn new(module_hash: Bytes32, version: u16, debug: bool) -> Self {
        Self {
            module_hash,
            version,
            debug,
        }
    }
}

#[derive(Clone)]
struct CacheItem {
    module: Module,
    engine: Engine,
}

impl CacheItem {
    fn new(module: Module, engine: Engine) -> Self {
        Self { module, engine }
    }

    fn data(&self) -> (Module, Store) {
        (self.module.clone(), Store::new(self.engine.clone()))
    }
}

impl InitCache {
    // current implementation only has one tag that stores to the long_term
    // future implementations might have more, but 0 is a reserved tag
    // that will never modify long_term state
    const ARBOS_TAG: u32 = 1;

    fn new(size: usize) -> Self {
        Self {
            long_term: HashMap::new(),
            lru: LruCache::new(NonZeroUsize::new(size).unwrap()),
        }
    }

    pub fn set_lru_size(size: u32) {
        cache!().lru.resize(NonZeroUsize::new(size.try_into().unwrap()).unwrap())
    }

    /// Retrieves a cached value, updating items as necessary.
    pub fn get(module_hash: Bytes32, version: u16, debug: bool) -> Option<(Module, Store)> {
        let mut cache = cache!();
        let key = CacheKey::new(module_hash, version, debug);

        // See if the item is in the long term cache
        if let Some(item) = cache.long_term.get(&key) {
            return Some(item.data());
        }

        // See if the item is in the LRU cache, promoting if so
        if let Some(item) = cache.lru.get(&key) {
            return Some(item.data());
        }
        None
    }

    /// Inserts an item into the long term cache, cloning from the LRU cache if able.
    /// If long_term_tag is 0 will only insert to LRU
    pub fn insert(
        module_hash: Bytes32,
        module: &[u8],
        version: u16,
        long_term_tag: u32,
        debug: bool,
    ) -> Result<(Module, Store)> {
        let key = CacheKey::new(module_hash, version, debug);

        // if in LRU, add to ArbOS
        let mut cache = cache!();
        if let Some(item) = cache.long_term.get(&key) {
            return Ok(item.data())
        }
        if let Some(item) = cache.lru.peek(&key).cloned() {
            if long_term_tag == Self::ARBOS_TAG {
                cache.long_term.insert(key, item.clone());
            } else {
                cache.lru.promote(&key)
            }
            return Ok(item.data());
        }
        drop(cache);

        let engine = CompileConfig::version(version, debug).engine();
        let module = unsafe { Module::deserialize_unchecked(&engine, module)? };

        let item = CacheItem::new(module, engine);
        let data = item.data();
        let mut cache = cache!();
        if long_term_tag != Self::ARBOS_TAG {
            cache.lru.put(key, item);
        } else {
            cache.long_term.insert(key, item);
        }
        Ok(data)
    }

    /// Evicts an item in the long-term cache.
    pub fn evict(module_hash: Bytes32, version: u16, long_term_tag: u32, debug: bool) {
        if long_term_tag != Self::ARBOS_TAG {
            return
        }
        let key = CacheKey::new(module_hash, version, debug);
        let mut cache = cache!();
        if let Some(item) = cache.long_term.remove(&key) {
            cache.lru.put(key, item);
        }
    }

    pub fn clear_long_term(long_term_tag: u32) {
        if long_term_tag != Self::ARBOS_TAG {
            return
        }        
        let mut cache = cache!();
        let cache = &mut *cache;
        for (key, item) in cache.long_term.drain() {
            cache.lru.put(key, item); // not all will fit, just a heuristic
        }
    }
}
