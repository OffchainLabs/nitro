use alloy_primitives::B256;
use parking_lot::Mutex;
use std::collections::HashMap;
use wasmer::{Engine, Module, Store};

use crate::config::CompileConfig;

lazy_static::lazy_static! {
    static ref INIT_CACHE: Mutex<InitCache> = Mutex::new(InitCache::new());
}

macro_rules! cache {
    () => {
        INIT_CACHE.lock()
    };
}

/// Counters for LRU cache hit/miss tracking.
#[derive(Debug, Default)]
pub struct LruCounters {
    pub hits: u32,
    pub misses: u32,
    pub does_not_fit: u32,
}

/// Counters for long-term cache hit/miss tracking.
#[derive(Debug, Default)]
pub struct LongTermCounters {
    pub hits: u32,
    pub misses: u32,
}

/// Two-tier module cache: LRU for hot modules, long-term for ArbOS-pinned modules.
pub struct InitCache {
    long_term: HashMap<CacheKey, CacheItem>,
    long_term_size_bytes: usize,
    long_term_counters: LongTermCounters,

    lru: HashMap<CacheKey, CacheItem>,
    lru_capacity: usize,
    lru_counters: LruCounters,
}

#[derive(Clone, Copy, Hash, PartialEq, Eq)]
struct CacheKey {
    module_hash: B256,
    version: u16,
    debug: bool,
}

impl CacheKey {
    fn new(module_hash: B256, version: u16, debug: bool) -> Self {
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
    entry_size_estimate_bytes: usize,
}

impl CacheItem {
    fn new(module: Module, engine: Engine, entry_size_estimate_bytes: usize) -> Self {
        Self {
            module,
            engine,
            entry_size_estimate_bytes,
        }
    }

    fn data(&self) -> (Module, Store) {
        (self.module.clone(), Store::new(self.engine.clone()))
    }
}

/// LRU cache metrics.
#[derive(Debug, Default)]
pub struct LruCacheMetrics {
    pub size_bytes: u64,
    pub count: u32,
    pub hits: u32,
    pub misses: u32,
    pub does_not_fit: u32,
}

/// Long-term cache metrics.
#[derive(Debug, Default)]
pub struct LongTermCacheMetrics {
    pub size_bytes: u64,
    pub count: u32,
    pub hits: u32,
    pub misses: u32,
}

/// Combined cache metrics.
#[derive(Debug, Default)]
pub struct CacheMetrics {
    pub lru: LruCacheMetrics,
    pub long_term: LongTermCacheMetrics,
}

/// Deserialize a WASM module from compiled bytes.
pub fn deserialize_module(
    module: &[u8],
    version: u16,
    debug: bool,
) -> eyre::Result<(Module, Engine, usize)> {
    let compile = CompileConfig::version(version, debug);
    let engine = compile.engine();
    let module = unsafe { Module::deserialize_unchecked(&engine, module)? };
    let asm_size_estimate_bytes = module.serialize()?.len();
    let entry_size_estimate_bytes = asm_size_estimate_bytes + 128;
    Ok((module, engine, entry_size_estimate_bytes))
}

impl CompileConfig {
    /// Create a wasmer Engine with the configured middleware.
    pub fn engine(&self) -> Engine {
        use std::sync::Arc;
        use wasmer::{sys::EngineBuilder, CompilerConfig, Cranelift, CraneliftOptLevel};

        use crate::middleware;

        let mut cranelift = Cranelift::new();
        cranelift.opt_level(CraneliftOptLevel::Speed);
        cranelift.canonicalize_nans(true);

        if self.pricing.ink_header_cost > 0 {
            // Middleware order:
            //   StartMover -> InkMeter -> DynamicMeter -> DepthChecker -> HeapBound
            cranelift.push_middleware(Arc::new(middleware::StartMover::new(
                self.debug.debug_info,
            )));
            cranelift.push_middleware(Arc::new(middleware::InkMeter::new(
                self.pricing.ink_header_cost,
            )));
            cranelift.push_middleware(Arc::new(middleware::DynamicMeter::new(
                self.pricing.memory_fill_ink,
                self.pricing.memory_copy_ink,
            )));
            cranelift.push_middleware(Arc::new(middleware::DepthChecker::new(
                self.bounds.max_frame_size,
                self.bounds.max_frame_contention,
            )));
            cranelift.push_middleware(Arc::new(middleware::HeapBound::new()));
        }

        EngineBuilder::new(cranelift).into()
    }

    /// Create a wasmer Store from this config.
    pub fn store(&self) -> Store {
        Store::new(self.engine())
    }
}

impl InitCache {
    const ARBOS_TAG: u32 = 1;
    const DEFAULT_LRU_CAPACITY: usize = 1024;

    fn new() -> Self {
        Self {
            long_term: HashMap::new(),
            long_term_size_bytes: 0,
            long_term_counters: LongTermCounters::default(),
            lru: HashMap::new(),
            lru_capacity: Self::DEFAULT_LRU_CAPACITY,
            lru_counters: LruCounters::default(),
        }
    }

    /// Set the LRU cache capacity.
    pub fn set_lru_capacity(capacity: u32) {
        cache!().lru_capacity = capacity as usize;
    }

    /// Retrieve a cached module.
    pub fn get(
        module_hash: B256,
        version: u16,
        long_term_tag: u32,
        debug: bool,
    ) -> Option<(Module, Store)> {
        let key = CacheKey::new(module_hash, version, debug);
        let mut cache = cache!();

        if let Some(item) = cache.long_term.get(&key) {
            let data = item.data();
            cache.long_term_counters.hits += 1;
            return Some(data);
        }
        if long_term_tag == Self::ARBOS_TAG {
            cache.long_term_counters.misses += 1;
        }

        if let Some(item) = cache.lru.get(&key).cloned() {
            cache.lru_counters.hits += 1;
            if long_term_tag == Self::ARBOS_TAG {
                cache.long_term_size_bytes += item.entry_size_estimate_bytes;
                cache.long_term.insert(key, item.clone());
            }
            return Some(item.data());
        }
        cache.lru_counters.misses += 1;

        None
    }

    /// Insert a module into the cache.
    pub fn insert(
        module_hash: B256,
        module: &[u8],
        version: u16,
        long_term_tag: u32,
        debug: bool,
    ) -> eyre::Result<(Module, Store)> {
        let key = CacheKey::new(module_hash, version, debug);
        let mut cache = cache!();

        if let Some(item) = cache.long_term.get(&key) {
            return Ok(item.data());
        }
        if let Some(item) = cache.lru.get(&key).cloned() {
            if long_term_tag == Self::ARBOS_TAG {
                cache.long_term_size_bytes += item.entry_size_estimate_bytes;
                cache.long_term.insert(key, item.clone());
            }
            return Ok(item.data());
        }
        drop(cache);

        let (module, engine, entry_size_estimate_bytes) =
            deserialize_module(module, version, debug)?;
        let item = CacheItem::new(module, engine, entry_size_estimate_bytes);
        let data = item.data();

        let mut cache = cache!();
        if long_term_tag == Self::ARBOS_TAG {
            cache.long_term_size_bytes += entry_size_estimate_bytes;
            cache.long_term.insert(key, item);
        } else {
            // Simple eviction: if at capacity, remove an arbitrary entry
            if cache.lru.len() >= cache.lru_capacity {
                let first_key = cache.lru.keys().next().copied();
                if let Some(k) = first_key {
                    cache.lru.remove(&k);
                }
            }
            cache.lru.insert(key, item);
        }
        Ok(data)
    }

    /// Evict a module from the long-term cache.
    pub fn evict(module_hash: B256, version: u16, long_term_tag: u32, debug: bool) {
        if long_term_tag != Self::ARBOS_TAG {
            return;
        }
        let key = CacheKey::new(module_hash, version, debug);
        let mut cache = cache!();
        if let Some(item) = cache.long_term.remove(&key) {
            cache.long_term_size_bytes -= item.entry_size_estimate_bytes;
            cache.lru.insert(key, item);
        }
    }

    /// Clear the long-term cache, moving items to LRU.
    pub fn clear_long_term(long_term_tag: u32) {
        if long_term_tag != Self::ARBOS_TAG {
            return;
        }
        let mut cache = cache!();
        let drained: Vec<_> = cache.long_term.drain().collect();
        for (key, item) in drained {
            cache.lru.insert(key, item);
        }
        cache.long_term_size_bytes = 0;
    }

    /// Get cache metrics, resetting counters.
    pub fn get_metrics() -> CacheMetrics {
        let mut cache = cache!();
        let metrics = CacheMetrics {
            lru: LruCacheMetrics {
                size_bytes: cache.lru.len() as u64,
                count: cache.lru.len() as u32,
                hits: cache.lru_counters.hits,
                misses: cache.lru_counters.misses,
                does_not_fit: cache.lru_counters.does_not_fit,
            },
            long_term: LongTermCacheMetrics {
                size_bytes: cache.long_term_size_bytes as u64,
                count: cache.long_term.len() as u32,
                hits: cache.long_term_counters.hits,
                misses: cache.long_term_counters.misses,
            },
        };
        cache.lru_counters = LruCounters::default();
        cache.long_term_counters = LongTermCounters::default();
        metrics
    }

    /// Clear the LRU cache.
    pub fn clear_lru_cache() {
        let mut cache = cache!();
        cache.lru.clear();
        cache.lru_counters = LruCounters::default();
    }
}
