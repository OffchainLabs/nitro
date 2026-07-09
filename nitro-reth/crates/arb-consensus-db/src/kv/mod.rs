mod memory;

pub use memory::MemoryKvStore;

pub type Key<'a> = &'a [u8];
pub type KeyBuf = Vec<u8>;
pub type Value = Vec<u8>;

pub trait KvError: std::error::Error + Send + Sync + 'static {}
impl<T: std::error::Error + Send + Sync + 'static> KvError for T {}

pub trait KvStore {
    type Error: KvError;

    fn get(&self, key: Key) -> Result<Option<Value>, Self::Error>;
    fn has(&self, key: Key) -> Result<bool, Self::Error>;
    fn put(&mut self, key: Key, value: Value) -> Result<(), Self::Error>;
    fn delete(&mut self, key: Key) -> Result<(), Self::Error>;
    fn write_batch(&mut self, batch: Batch) -> Result<(), Self::Error>;
}

#[derive(Debug, Default)]
pub struct Batch {
    ops: Vec<Op>,
}

impl Batch {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn put(&mut self, key: impl Into<KeyBuf>, value: Value) {
        self.ops.push(Op::Put(key.into(), value));
    }

    pub fn delete(&mut self, key: impl Into<KeyBuf>) {
        self.ops.push(Op::Delete(key.into()));
    }
}

#[derive(Debug)]
pub enum Op {
    Put(KeyBuf, Value),
    Delete(KeyBuf),
}
