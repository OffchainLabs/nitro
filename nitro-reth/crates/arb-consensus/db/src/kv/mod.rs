//! A minimal byte-oriented key-value store abstraction and its in-memory implementation.
//!
//! [`KvStore`] is the backend seam under [`crate::ConsensusDb`]: any engine that can
//! get/put/delete byte keys and iterate a prefix can back the consensus DB.

#[cfg(feature = "libmdbx")]
mod libmdbx;
mod memory;

#[cfg(feature = "libmdbx")]
pub use libmdbx::LibmdbxKvStore;
pub use memory::MemoryKvStore;

/// A borrowed key: a byte slice.
pub type Key<'a> = &'a [u8];
/// An owned key.
pub type KeyBuf = Vec<u8>;
/// An owned value.
pub type Value = Vec<u8>;

/// Marker for types usable as a [`KvStore::Error`]: [`std::error::Error`] plus
/// `Send + Sync + 'static`. Blanket-implemented for every qualifying type.
pub trait KvError: std::error::Error + Send + Sync + 'static {}
impl<T: std::error::Error + Send + Sync + 'static> KvError for T {}

/// A byte-oriented key-value store: the backend abstraction for [`crate::ConsensusDb`].
pub trait KvStore {
    /// The store's error type.
    type Error: KvError;

    /// Read the value at `key`, or `None` if absent.
    fn get(&self, key: Key) -> Result<Option<Value>, Self::Error>;
    /// Return whether `key` is present.
    fn has(&self, key: Key) -> Result<bool, Self::Error>;
    /// Write `value` at `key`, overwriting any existing value.
    fn put(&mut self, key: Key, value: Value) -> Result<(), Self::Error>;
    /// Delete `key` if present.
    fn delete(&mut self, key: Key) -> Result<(), Self::Error>;
    /// Apply `batch` atomically.
    fn write_batch(&mut self, batch: Batch) -> Result<(), Self::Error>;
    /// Iterate entries whose key begins with `prefix`, starting at `prefix ++ start`,
    /// in ascending key order.
    fn iter_prefix(
        &self,
        prefix: Key,
        start: impl AsRef<[u8]>,
    ) -> Result<Vec<(KeyBuf, Value)>, Self::Error>;

    /// Delete every key in the range `[start, end)`.
    fn delete_range(&mut self, start: Key, end: Key) -> Result<(), Self::Error>;
}

/// An ordered set of writes to be applied together via [`KvStore::write_batch`].
#[derive(Debug, Default)]
pub struct Batch {
    ops: Vec<Op>,
}

impl Batch {
    /// Create an empty batch.
    pub fn new() -> Self {
        Self::default()
    }

    /// Stage a write of `value` at `key`.
    pub fn put(&mut self, key: impl Into<KeyBuf>, value: Value) {
        self.ops.push(Op::Put(key.into(), value));
    }

    /// Stage a deletion of `key`.
    pub fn delete(&mut self, key: impl Into<KeyBuf>) {
        self.ops.push(Op::Delete(key.into()));
    }
}

/// A single staged operation within a [`Batch`].
#[derive(Debug)]
pub enum Op {
    /// Write a value at a key.
    Put(KeyBuf, Value),
    /// Delete a key.
    Delete(KeyBuf),
}
