//! An in-memory [`KvStore`], backed by a [`BTreeMap`]. Useful for tests and as the
//! reference implementation of the backend contract.

use std::collections::BTreeMap;

use crate::kv::{Batch, Key, KeyBuf, KvStore, Op, Value};

/// A [`KvStore`] backed by an in-memory [`BTreeMap`]. Its ordered keys give prefix
/// iteration and range deletion for free; operations are infallible.
#[derive(Debug, Default)]
pub struct MemoryKvStore {
    values: BTreeMap<KeyBuf, Value>,
}

impl MemoryKvStore {
    /// Create an empty store.
    pub fn new() -> Self {
        Self::default()
    }
}

impl KvStore for MemoryKvStore {
    type Error = std::convert::Infallible;

    fn get(&self, key: Key) -> Result<Option<Value>, Self::Error> {
        Ok(self.values.get(key).cloned())
    }

    fn has(&self, key: Key) -> Result<bool, Self::Error> {
        Ok(self.values.contains_key(key))
    }

    fn put(&mut self, key: Key, value: Value) -> Result<(), Self::Error> {
        self.values.insert(key.to_vec(), value);
        Ok(())
    }

    fn delete(&mut self, key: Key) -> Result<(), Self::Error> {
        self.values.remove(key);
        Ok(())
    }

    fn write_batch(&mut self, batch: Batch) -> Result<(), Self::Error> {
        for op in batch.ops.into_iter() {
            match op {
                Op::Put(key, value) => self.put(&key, value)?,
                Op::Delete(key) => self.delete(&key)?,
            }
        }
        Ok(())
    }

    fn iter_prefix(
        &self,
        prefix: Key,
        start: impl AsRef<[u8]>,
    ) -> impl Iterator<Item = Result<(KeyBuf, Value), Self::Error>> {
        self.values
            .range([prefix, start.as_ref()].concat()..)
            .take_while(move |(k, _)| k.starts_with(prefix))
            .map(|(k, v)| Ok((k.clone(), v.clone())))
    }

    fn delete_range(&mut self, start: Key, end: Key) -> Result<(), Self::Error> {
        // `extract_if` walks only `[start, end)` and removes in place; it is lazy,
        // so it must be consumed to take effect.
        self.values
            .extract_if(start.to_vec()..end.to_vec(), |_, _| true)
            .for_each(drop);
        Ok(())
    }
}
