// Copyright 2022-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

use std::collections::HashMap;

use eyre::Result;
use thiserror::Error;

use super::api::Gas;
use crate::Bytes32;

/// Represents the EVM word at a given key.
#[derive(Debug)]
struct StorageWord {
    /// The current value of the slot.
    value: Bytes32,
    /// The value in Geth, if known.
    known: Option<Bytes32>,
}

impl StorageWord {
    fn unknown(value: Bytes32) -> Self {
        Self { value, known: None }
    }
}

#[derive(Debug, Error)]
#[error("storage cache limit exceeded")]
pub struct StorageCacheLimitExceeded;

pub struct StorageCache {
    clean: HashMap<Bytes32, Bytes32>,
    dirty: HashMap<Bytes32, StorageWord>,
    limit: usize,
    reads: usize,
    writes: usize,
}

impl StorageCache {
    pub const REQUIRED_ACCESS_GAS: Gas = Gas(10);

    /// Creates a cache that holds at most `limit` distinct storage slots. A limit of 0 disables
    /// the cap.
    pub(super) fn new(limit: u32) -> Self {
        Self {
            clean: HashMap::new(),
            dirty: HashMap::new(),
            limit: limit as usize,
            reads: 0,
            writes: 0,
        }
    }

    pub(super) fn read_gas(&mut self) -> Gas {
        self.reads += 1;
        match self.reads {
            0..=32 => Gas(0),
            33..=128 => Gas(2),
            _ => Gas(10),
        }
    }

    pub(super) fn write_gas(&mut self) -> Gas {
        self.writes += 1;
        match self.writes {
            0..=8 => Gas(0),
            9..=64 => Gas(7),
            _ => Gas(10),
        }
    }

    pub(super) fn get(&self, key: &Bytes32) -> Option<Bytes32> {
        self.dirty
            .get(key)
            .map(|word| word.value)
            .or_else(|| self.clean.get(key).copied())
    }

    pub(super) fn insert_known(&mut self, key: Bytes32, value: Bytes32) -> Result<()> {
        if self.get(&key).is_some() {
            return Ok(());
        }
        self.ensure_capacity()?;
        self.clean.insert(key, value);
        Ok(())
    }

    pub(super) fn cache(&mut self, key: Bytes32, value: Bytes32) -> Result<()> {
        if let Some(word) = self.dirty.get_mut(&key) {
            word.value = value;
            return Ok(());
        }

        if let Some(known) = self.clean.remove(&key) {
            if value == known {
                self.clean.insert(key, known);
            } else {
                self.dirty.insert(
                    key,
                    StorageWord {
                        value,
                        known: Some(known),
                    },
                );
            }
            return Ok(());
        }

        self.ensure_capacity()?;
        self.dirty.insert(key, StorageWord::unknown(value));
        Ok(())
    }

    /// Returns changed entries in ascending key order. Unless `clear` is set, their cached values
    /// become clean entries. Otherwise, the entire cache is emptied.
    pub(super) fn flush(&mut self, clear: bool) -> Vec<(Bytes32, Bytes32)> {
        let dirty = std::mem::take(&mut self.dirty);
        let mut entries = Vec::with_capacity(dirty.len());

        for (key, word) in dirty {
            if word.known != Some(word.value) {
                entries.push((key, word.value));
            }
            if !clear {
                self.clean.insert(key, word.value);
            }
        }
        if clear {
            self.clean.clear();
        }

        entries.sort_unstable_by_key(|(key, _)| *key);
        entries
    }

    fn len(&self) -> usize {
        self.clean.len() + self.dirty.len()
    }

    pub(super) fn ensure_capacity(&self) -> Result<()> {
        if self.limit > 0 && self.len() >= self.limit {
            return Err(StorageCacheLimitExceeded.into());
        }
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn limit_counts_clean_and_dirty_entries() {
        let mut cache = StorageCache::new(2);
        cache.insert_known(1u32.into(), 11u32.into()).unwrap();
        cache.cache(2u32.into(), 22u32.into()).unwrap();

        let error = cache.insert_known(3u32.into(), 33u32.into()).unwrap_err();
        assert!(error.downcast_ref::<StorageCacheLimitExceeded>().is_some());
        assert!(cache.cache(3u32.into(), 33u32.into()).is_err());

        cache.cache(1u32.into(), 111u32.into()).unwrap();
        cache.cache(2u32.into(), 222u32.into()).unwrap();
        assert_eq!(cache.len(), 2);
    }

    #[test]
    fn flush_separates_dirty_and_clean_entries() {
        let mut cache = StorageCache::new(3);
        cache.insert_known(1u32.into(), 11u32.into()).unwrap();
        cache.insert_known(2u32.into(), 22u32.into()).unwrap();
        cache.cache(2u32.into(), 222u32.into()).unwrap();
        cache.cache(3u32.into(), 333u32.into()).unwrap();

        assert_eq!(
            cache.flush(false),
            vec![(2u32.into(), 222u32.into()), (3u32.into(), 333u32.into())]
        );
        assert_eq!(cache.len(), 3);
        assert!(cache.flush(false).is_empty());

        cache.cache(1u32.into(), 111u32.into()).unwrap();
        assert_eq!(cache.flush(true), vec![(1u32.into(), 111u32.into())]);
        assert_eq!(cache.len(), 0);
    }

    #[test]
    fn restoring_known_value_keeps_entry_clean() {
        let mut cache = StorageCache::new(1);
        cache.insert_known(1u32.into(), 11u32.into()).unwrap();
        cache.cache(1u32.into(), 22u32.into()).unwrap();
        cache.cache(1u32.into(), 11u32.into()).unwrap();

        assert!(cache.flush(false).is_empty());
        assert_eq!(cache.get(&1u32.into()), Some(11u32.into()));
    }

    #[test]
    fn zero_limit_is_unlimited() {
        let mut cache = StorageCache::new(0);
        for key in 0..256u32 {
            cache.cache(key.into(), key.into()).unwrap();
        }
        assert_eq!(cache.len(), 256);
    }

    #[test]
    fn flush_sorts_entries_by_key() {
        let mut cache = StorageCache::new(0);
        let keys = [0x100u32, 7, 0xff, 3, 0x101, 1, 0x10000, 0x1000];
        for key in keys {
            cache.cache(key.into(), key.into()).unwrap();
        }

        let flushed: Vec<_> = cache.flush(false).into_iter().map(|(key, _)| key).collect();
        let expected = [1u32, 3, 7, 0xff, 0x100, 0x101, 0x1000, 0x10000]
            .map(Into::into)
            .to_vec();
        assert_eq!(flushed, expected);
    }
}
