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
    arbos: HashMap<CacheKey, CacheItem>,
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
    fn new(size: usize) -> Self {
        Self {
            arbos: HashMap::new(),
            lru: LruCache::new(NonZeroUsize::new(size).unwrap()),
        }
    }

    /// Retrieves a cached value, updating items as necessary.
    pub fn get(module_hash: Bytes32, version: u16, debug: bool) -> Option<(Module, Store)> {
        let mut cache = cache!();
        let key = CacheKey::new(module_hash, version, debug);

        // See if the item is in the long term cache
        if let Some(item) = cache.arbos.get(&key) {
            return Some(item.data());
        }

        // See if the item is in the LRU cache, promoting if so
        if let Some(item) = cache.lru.get(&key) {
            return Some(item.data());
        }
        None
    }

    /// Inserts an item into the long term cache, cloning from the LRU cache if able.
    pub fn insert(
        module_hash: Bytes32,
        module: &[u8],
        version: u16,
        debug: bool,
    ) -> Result<(Module, Store)> {
        let key = CacheKey::new(module_hash, version, debug);

        // if in LRU, add to ArbOS
        let mut cache = cache!();
        if let Some(item) = cache.lru.peek(&key).cloned() {
            cache.arbos.insert(key, item.clone());
            return Ok(item.data());
        }
        drop(cache);

        let engine = CompileConfig::version(version, debug).engine();
        let module = unsafe { Module::deserialize_unchecked(&engine, module)? };

        let item = CacheItem::new(module, engine);
        let data = item.data();
        cache!().arbos.insert(key, item);
        Ok(data)
    }

    /// Inserts an item into the short-lived LRU cache.
    pub fn insert_lru(
        module_hash: Bytes32,
        module: &[u8],
        version: u16,
        debug: bool,
    ) -> Result<(Module, Store)> {
        let engine = CompileConfig::version(version, debug).engine();
        let module = unsafe { Module::deserialize_unchecked(&engine, module)? };

        let key = CacheKey::new(module_hash, version, debug);
        let item = CacheItem::new(module, engine);
        cache!().lru.put(key, item.clone());
        Ok(item.data())
    }

    /// Evicts an item in the long-term cache.
    pub fn evict(module_hash: Bytes32, version: u16, debug: bool) {
        let key = CacheKey::new(module_hash, version, debug);
        cache!().arbos.remove(&key);
    }

    /// Modifies the cache for reorg, dropping the long-term cache.
    pub fn reorg(_block: u64) {
        let mut cache = cache!();
        let cache = &mut *cache;
        for (key, item) in cache.arbos.drain() {
            cache.lru.put(key, item); // not all will fit, just a heuristic
        }
    }
}
