mod memory;

pub use memory::MemoryKvStore;

pub type Key<'a> = &'a [u8];
pub type KeyBuf = Vec<u8>;
pub type Value = Vec<u8>;

pub trait KvStore {
    type Error;

    fn get(&self, key: Key) -> Result<Option<Value>, Self::Error>;
    fn has(&self, key: Key) -> Result<bool, Self::Error>;
    fn put(&mut self, key: Key, value: Value) -> Result<(), Self::Error>;
    fn delete(&mut self, key: Key) -> Result<(), Self::Error>;
    fn write_batch(&mut self, batch: Batch) -> Result<(), Self::Error>;
}

#[derive(Debug)]
pub struct Batch {
    pub ops: Vec<Op>,
}

#[derive(Debug)]
pub enum Op {
    Put(KeyBuf, Value),
    Delete(KeyBuf),
}
