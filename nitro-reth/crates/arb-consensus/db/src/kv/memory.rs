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
    ) -> Result<Vec<(KeyBuf, Value)>, Self::Error> {
        self.values
            .range([prefix, start.as_ref()].concat()..)
            .take_while(move |(k, _)| k.starts_with(prefix))
            .map(|(k, v)| Ok((k.clone(), v.clone())))
            .collect()
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

#[cfg(test)]
mod tests {
    use super::*;
    use crate::kv::{Batch, KvStore};

    #[test]
    fn put_get_has_delete() {
        let mut s = MemoryKvStore::new();
        assert_eq!(s.get(b"k").unwrap(), None);
        assert!(!s.has(b"k").unwrap());

        s.put(b"k", vec![1, 2]).unwrap();
        assert_eq!(s.get(b"k").unwrap(), Some(vec![1, 2]));
        assert!(s.has(b"k").unwrap());

        // put overwrites (upsert).
        s.put(b"k", vec![3]).unwrap();
        assert_eq!(s.get(b"k").unwrap(), Some(vec![3]));

        // delete, then delete again (idempotent).
        s.delete(b"k").unwrap();
        assert_eq!(s.get(b"k").unwrap(), None);
        s.delete(b"k").unwrap();
    }

    #[test]
    fn write_batch_applies_in_order() {
        let mut s = MemoryKvStore::new();
        let mut b = Batch::new();
        b.put(b"a".to_vec(), vec![1]);
        b.put(b"b".to_vec(), vec![2]);
        b.delete(b"a".to_vec()); // put then delete the same key within one batch -> gone
        s.write_batch(b).unwrap();
        assert_eq!(s.get(b"a").unwrap(), None);
        assert_eq!(s.get(b"b").unwrap(), Some(vec![2]));
    }

    #[test]
    fn empty_batch_is_noop() {
        let mut s = MemoryKvStore::new();
        s.put(b"k", vec![9]).unwrap();
        s.write_batch(Batch::new()).unwrap();
        assert_eq!(s.get(b"k").unwrap(), Some(vec![9]));
    }

    #[test]
    fn iter_prefix_is_ordered_and_bounded() {
        let mut s = MemoryKvStore::new();
        // insert out of order, plus an entry under a different prefix
        s.put(b"m3", vec![3]).unwrap();
        s.put(b"m1", vec![1]).unwrap();
        s.put(b"m2", vec![2]).unwrap();
        s.put(b"n1", vec![99]).unwrap();

        let got: Vec<_> = s
            .iter_prefix(b"m", b"")
            .unwrap()
            .into_iter()
            .map(|(_, v)| v)
            .collect();
        assert_eq!(got, vec![vec![1], vec![2], vec![3]]);
    }

    #[test]
    fn iter_prefix_respects_start() {
        let mut s = MemoryKvStore::new();
        s.put(b"m1", vec![1]).unwrap();
        s.put(b"m2", vec![2]).unwrap();
        s.put(b"m3", vec![3]).unwrap();

        // `start` is appended to the prefix, so iteration begins at "m2".
        let got: Vec<_> = s
            .iter_prefix(b"m", b"2")
            .unwrap()
            .into_iter()
            .map(|(_, v)| v)
            .collect();
        assert_eq!(got, vec![vec![2], vec![3]]);
    }

    #[test]
    fn delete_range_is_half_open() {
        let mut s = MemoryKvStore::new();
        for i in 0u8..5 {
            s.put(&[b'k', i], vec![i]).unwrap();
        }
        // [1, 3): removes 1 and 2; keeps 0, 3, 4.
        s.delete_range(&[b'k', 1], &[b'k', 3]).unwrap();
        assert!(s.has(&[b'k', 0]).unwrap());
        assert!(!s.has(&[b'k', 1]).unwrap());
        assert!(!s.has(&[b'k', 2]).unwrap());
        assert!(s.has(&[b'k', 3]).unwrap());
        assert!(s.has(&[b'k', 4]).unwrap());
    }
}
