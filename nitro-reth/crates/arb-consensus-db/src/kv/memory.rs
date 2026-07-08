use std::collections::BTreeMap;

use crate::kv::{Batch, Key, KeyBuf, KvStore, Op, Value};

#[derive(Debug)]
pub struct MemoryKvStore {
    values: BTreeMap<KeyBuf, Value>,
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
                Op::Put(key, value) => self.put(&key, value).unwrap(),
                Op::Delete(key) => self.delete(&key).unwrap(),
            }
        }
        Ok(())
    }
}
